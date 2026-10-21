package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"reflect"
	"strings"

	"scoreview/api/internal/auth"
	"scoreview/api/internal/domain"
	"scoreview/api/internal/store"
)

// 認証と管理用エンドポイントの共通部分(docs/api.md「管理用エンドポイント」)。

// maxBodyBytes は管理用のボディの上限。自由記述 HTML(大会トップ)が最大の項目で、参照サイトの
// 大会で最大でも数十 KB なので 1 MiB に収まる。
const maxBodyBytes = 1 << 20

// policy は管理用エンドポイントの認可の規則。
type policy struct {
	// staffTournament が nil ならadmin 専用。nil でなければ、staff はこれが返す大会の担当なら許す
	staffTournament func(*Server, *http.Request) (int64, error)
	// anyUser なら役割を問わずログインしていればよい(応答の絞り込みはハンドラが行う)
	anyUser bool
	// scoreMatch なら、staff はパスの {id} の試合を覆う割り当て(staff_assignments)も要る
	scoreMatch bool
}

// adminOnly は管理者だけに許す。
func adminOnly() policy { return policy{} }

// anyUser はログインしていれば誰にでも許す。
func anyUser() policy { return policy{anyUser: true} }

// staffOfTournament は記録係にも、パスの {id} の大会の担当なら許す。
func staffOfTournament() policy {
	return policy{
		staffTournament: func(_ *Server, r *http.Request) (int64, error) { return pathID(r) },
	}
}

// staffOfDivision は記録係にも、パスの {id} のディビジョンの大会の担当なら許す。
func staffOfDivision() policy {
	return policy{staffTournament: func(s *Server, r *http.Request) (int64, error) {
		id, err := pathID(r)
		if err != nil {
			return 0, err
		}
		return store.TournamentOfDivision(r.Context(), s.db, id)
	}}
}

// staffOfMatch は記録係にも、パスの {id} の試合の大会の担当なら許す。
func staffOfMatch() policy {
	return policy{staffTournament: func(s *Server, r *http.Request) (int64, error) {
		id, err := pathID(r)
		if err != nil {
			return 0, err
		}
		return store.TournamentOfMatch(r.Context(), s.db, id)
	}}
}

// scorerOfMatch は記録係にも、パスの {id} の試合の大会の担当で、割り当てがその試合を覆うなら許す
// (担当外の試合は入力できない。docs/operations/operator-experience.md「記録係」5)。
func scorerOfMatch() policy {
	rule := staffOfMatch()
	rule.scoreMatch = true
	return rule
}

// adminFunc は認可済みの利用者を受け取るハンドラ。
type adminFunc func(r *http.Request, user domain.User) (any, error)

// admin はセッションと権限を確かめてから fn を呼ぶルートを登録する。成功すると status で応答し、
// 204 なら本文を書かない。
func (s *Server) admin(pattern string, rule policy, status int, fn adminFunc) {
	s.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		user, err := s.authenticate(r)
		if err == nil {
			err = s.authorize(r, user, rule)
		}
		var body any
		if err == nil {
			body, err = fn(r, user)
		}
		if err != nil {
			writeError(w, s.log, err)
			return
		}
		if status == http.StatusNoContent {
			w.WriteHeader(status)
			return
		}
		writeJSON(w, status, body)
	})
}

// bearerToken は Authorization: Bearer {token} のトークンを返す。無ければ空文字。
func bearerToken(r *http.Request) string {
	const prefix = "Bearer "
	header := r.Header.Get("Authorization")
	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return ""
	}
	return header[len(prefix):]
}

func (s *Server) authenticate(r *http.Request) (domain.User, error) {
	return auth.Authenticate(r.Context(), s.db, bearerToken(r), s.now())
}

func (s *Server) authorize(r *http.Request, user domain.User, rule policy) error {
	if user.Role == domain.RoleAdmin || rule.anyUser {
		return nil
	}
	if user.Role != domain.RoleStaff || rule.staffTournament == nil {
		return fmt.Errorf("%w: admin only", domain.ErrForbidden)
	}
	tournamentID, err := rule.staffTournament(s, r)
	if err != nil {
		return err
	}
	ok, err := store.IsTournamentStaff(r.Context(), s.db, user.ID, tournamentID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%w: not a staff of tournament %d", domain.ErrForbidden, tournamentID)
	}
	if !rule.scoreMatch {
		return nil
	}
	matchID, err := pathID(r)
	if err != nil {
		return err
	}
	ok, err = store.CanScoreMatch(r.Context(), s.db, user.ID, matchID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%w: match %d is not assigned to this staff", domain.ErrForbidden, matchID)
	}
	return nil
}

func actorOf(
	user domain.User,
) domain.Actor {
	return domain.Actor{UserID: user.ID, Role: user.Role}
}

// decodeBody はボディの JSON を dst に読み込む。dst に既に入っている値は、ボディに無いキーの分だけ残る
// (PATCH は現在の値を入れてから呼ぶ)。未知のキー、型の違い、複数の値は検証エラーにする。
func decodeBody(r *http.Request, dst any) error {
	data, err := io.ReadAll(http.MaxBytesReader(nil, r.Body, maxBodyBytes))
	if err != nil {
		return domain.Invalid("body", "too large or unreadable")
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return jsonBodyError(err)
	}
	if dec.More() {
		return domain.Invalid("body", "must be a single JSON value")
	}
	return nil
}

// jsonBodyError はボディの JSON の読み込みの失敗を、利用者向けの検証エラーにする。encoding/json の
// 文面は Go の構造体名と型名(httpapi.tournamentBodyDTO、int など)を含むので、そのまま返さない
// (セキュリティ点検 I-2)。
func jsonBodyError(err error) error {
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		field := typeErr.Field
		if field == "" {
			field = "body"
		}
		return domain.Invalid(field, "must be "+jsonKindOf(typeErr.Type))
	}
	if key, ok := strings.CutPrefix(err.Error(), "json: unknown field "); ok {
		return domain.Invalid("body", "unknown key "+key)
	}
	return domain.Invalid("body", "must be a single valid JSON value")
}

// jsonKindOf は Go の型を、JSON での書き方の説明にする。
func jsonKindOf(goType reflect.Type) string {
	kind := goType.Kind()
	for elem := goType; kind == reflect.Pointer; kind = elem.Kind() {
		elem = elem.Elem()
	}
	switch kind {
	case reflect.String:
		return "a string"
	case reflect.Bool:
		return "true or false"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "an integer in range"
	case reflect.Float32, reflect.Float64:
		return "a number"
	case reflect.Slice, reflect.Array:
		return "an array"
	case reflect.Struct, reflect.Map:
		return "an object"
	default:
		return "a valid value"
	}
}

// withPathID はパスの {id} を読んでから fn を呼ぶ adminFunc にする。
func withPathID(
	fn func(ctx context.Context, id int64, r *http.Request, user domain.User) (any, error),
) adminFunc {
	return func(r *http.Request, user domain.User) (any, error) {
		id, err := pathID(r)
		if err != nil {
			return nil, err
		}
		return fn(r.Context(), id, r, user)
	}
}

// 認証

func (s *Server) authRoutes() {
	s.handle("POST /api/v1/auth/login", s.login)
	s.admin("POST /api/v1/auth/logout", anyUser(), http.StatusNoContent, s.logout)
	s.admin("GET /api/v1/auth/me", anyUser(), http.StatusOK, s.me)
	s.admin("POST /api/v1/auth/password", anyUser(), http.StatusNoContent, s.changePassword)
}

func (s *Server) login(r *http.Request) (any, error) {
	var body loginBodyDTO
	if err := decodeBody(r, &body); err != nil {
		return nil, err
	}
	// パスワードの長さの上限だけは照合の前に確かめる(巨大な入力で argon2 を回させない)
	if len(body.Password) > 4*auth.MaxPasswordLen {
		return nil, auth.ErrInvalidCredentials
	}
	attempt := auth.Attempt{
		Email:    body.Email,
		Password: body.Password,
		ClientIP: clientIP(r),
	}
	session, err := auth.Login(r.Context(), s.db, s.passwords, attempt, s.now())
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			s.log.Info("login failed", "clientIp", attempt.ClientIP)
		}
		return nil, err
	}
	ids, err := store.StaffTournamentIDs(r.Context(), s.db, session.User.ID)
	if err != nil {
		return nil, err
	}
	return loginDTO{
		Token:     session.Token,
		ExpiresAt: formatInstant(session.ExpiresAt),
		User:      toAdminUser(session.User, ids),
	}, nil
}

// clientIPHeader は web が利用者の IP を入れて渡すヘッダー(docs/api.md「認証」)。api は web からしか
// 呼ばれない(内部トークンで確かめ済み)ので、この値を信じる。
const clientIPHeader = "X-Client-IP"

// ipv6PrefixBits は IPv6 の利用者を同じ人として数える範囲。家庭や携帯の回線には /64 がまとめて
// 割り当てられ、その中のアドレスは自由に変えられるので、アドレス単位では数えない。
const ipv6PrefixBits = 64

// clientIP は X-Client-IP の IP を正規化して返す(IPv4 はそのまま、IPv6 は /64 の範囲)。
// 無いか IP として読めなければ空文字(IP 単位の制限を掛けない)。
func clientIP(r *http.Request) string {
	addr, err := netip.ParseAddr(strings.TrimSpace(r.Header.Get(clientIPHeader)))
	if err != nil {
		return ""
	}
	addr = addr.Unmap().WithZone("")
	if addr.Is4() {
		return addr.String()
	}
	prefix, err := addr.Prefix(ipv6PrefixBits)
	if err != nil {
		return "" // Is4 でない有効なアドレスは IPv6 なので /64 は必ず取れる
	}
	return prefix.String()
}

// changePassword は本人がパスワードを変える。このセッションは残し、他の端末のセッションを消す。
func (s *Server) changePassword(r *http.Request, user domain.User) (any, error) {
	var body passwordChangeBodyDTO
	if err := decodeBody(r, &body); err != nil {
		return nil, err
	}
	// 今のパスワードの長さの上限だけは照合の前に確かめる(巨大な入力で argon2 を回させない)
	if len(body.CurrentPassword) > 4*auth.MaxPasswordLen {
		return nil, domain.Invalid("currentPassword", "is incorrect")
	}
	change := auth.PasswordChange{
		Current:  body.CurrentPassword,
		New:      body.NewPassword,
		Token:    bearerToken(r),
		ClientIP: clientIP(r),
	}
	return nil, auth.ChangePassword(r.Context(), s.db, s.passwords, user, change, s.now())
}

func (s *Server) logout(r *http.Request, _ domain.User) (any, error) {
	return nil, auth.Logout(r.Context(), s.db, bearerToken(r))
}

func (s *Server) me(r *http.Request, user domain.User) (any, error) {
	ids, err := store.StaffTournamentIDs(r.Context(), s.db, user.ID)
	if err != nil {
		return nil, err
	}
	return meDTO{User: toAdminUser(user, ids)}, nil
}
