package httpapi

import (
	"time"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/store"
)

// 管理用エンドポイント(docs/api.md「管理用エンドポイント」)のボディと応答。キーは閲覧用の型と揃えた
// camelCase で、日英は I18nText。*InputDTO は作成(POST)と部分更新(PATCH)のボディで、応答は
// それに id と親の id を足した形。PATCH は現在の値を *InputDTO にしてからボディを上から読み込むので、
// ボディに無いキーは変わらない。

// 認証

type loginBodyDTO struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type adminUserDTO struct {
	ID            int64   `json:"id"`
	Email         string  `json:"email"`
	DisplayName   string  `json:"displayName"`
	Role          string  `json:"role"`
	TournamentIDs []int64 `json:"tournamentIds"`
}

type loginDTO struct {
	Token     string       `json:"token"`
	ExpiresAt string       `json:"expiresAt"` // RFC 3339(UTC)
	User      adminUserDTO `json:"user"`
}

type meDTO struct {
	User adminUserDTO `json:"user"`
}

func toAdminUser(user domain.User, tournamentIDs []int64) adminUserDTO {
	if tournamentIDs == nil {
		tournamentIDs = []int64{}
	}
	return adminUserDTO{
		ID: user.ID, Email: user.Email, DisplayName: user.DisplayName, Role: string(user.Role),
		TournamentIDs: tournamentIDs,
	}
}

func formatInstant(at time.Time) string { return at.UTC().Format(time.RFC3339) }

// ユーザー

type userInputDTO struct {
	Email       string  `json:"email"`
	DisplayName string  `json:"displayName"`
	Role        string  `json:"role"`
	Password    *string `json:"password"` // 作成では必須。更新では null か省略なら変えない
}

type userListDTO struct {
	Users []adminUserDTO `json:"users"`
}

type userTournamentsBodyDTO struct {
	TournamentIDs []int64 `json:"tournamentIds"`
}

// 大会とバナー

type tournamentInputDTO struct {
	Name         i18nDTO   `json:"name"`
	ShortName    string    `json:"shortName"`
	Status       string    `json:"status"`
	Listed       bool      `json:"listed"`
	StartDate    *string   `json:"startDate"`
	LogoURL      *string   `json:"logoUrl"`
	TopHTML      i18nDTO   `json:"topHtml"`
	Labels       labelsDTO `json:"labels"`
	ShowSchedule bool      `json:"showSchedule"`
	// 不戦勝の規定スコア。null は未設定
	ForfeitScore *forfeitScoreDTO `json:"forfeitScore"`
}

type forfeitScoreDTO struct {
	Winner int `json:"winner"`
	Loser  int `json:"loser"`
}

func toForfeitScoreDTO(score *domain.ForfeitScore) *forfeitScoreDTO {
	if score == nil {
		return nil
	}
	return &forfeitScoreDTO{Winner: score.Winner, Loser: score.Loser}
}

func (dto *forfeitScoreDTO) toDomain() *domain.ForfeitScore {
	if dto == nil {
		return nil
	}
	return &domain.ForfeitScore{Winner: dto.Winner, Loser: dto.Loser}
}

type adminTournamentDTO struct {
	ID int64 `json:"id"`
	tournamentInputDTO
}

type adminTournamentListDTO struct {
	Tournaments []adminTournamentDTO `json:"tournaments"`
}

// newTournamentInput は作成時の既定値(大会一覧に載る受付中の大会、ナビは標準のラベル)。
func newTournamentInput() tournamentInputDTO {
	return tournamentInputDTO{
		Status: string(domain.TournamentEntryOpen), Listed: true,
		Labels: labelsDTO{TeamList: "teams", Groups: "groups"}, ShowSchedule: true,
	}
}

func toTournamentInput(record store.TournamentRecord) tournamentInputDTO {
	return tournamentInputDTO{
		Name:      toI18n(record.Name),
		ShortName: record.ShortName,
		Status:    string(record.Status),
		Listed:    record.Listed,
		StartDate: record.StartDate,
		LogoURL:   record.LogoURL,
		TopHTML:   toI18n(record.TopHTML),
		Labels: labelsDTO{
			TeamList: record.TeamListLabel,
			Groups:   record.GroupLabel,
		},
		ShowSchedule: record.ShowSchedule,
		ForfeitScore: toForfeitScoreDTO(record.ForfeitScore),
	}
}

func (in tournamentInputDTO) fields() store.TournamentFields {
	return store.TournamentFields{
		Name: fromI18n(
			in.Name,
		),
		ShortName:     in.ShortName,
		Status:        domain.TournamentStatus(in.Status),
		Listed:        in.Listed,
		StartDate:     in.StartDate,
		LogoURL:       in.LogoURL,
		TopHTML:       fromI18n(in.TopHTML),
		TeamListLabel: in.Labels.TeamList,
		GroupLabel:    in.Labels.Groups,
		ShowSchedule:  in.ShowSchedule,
		ForfeitScore:  in.ForfeitScore.toDomain(),
	}
}

func toAdminTournament(record store.TournamentRecord) adminTournamentDTO {
	return adminTournamentDTO{ID: record.ID, tournamentInputDTO: toTournamentInput(record)}
}

func fromI18n(text i18nDTO) domain.I18nText { return domain.I18nText{Ja: text.Ja, En: text.En} }

type bannersDTO struct {
	Banners []bannerDTO `json:"banners"`
}

func toBanners(banners []store.BannerFields) bannersDTO {
	out := bannersDTO{Banners: make([]bannerDTO, len(banners))}
	for i, banner := range banners {
		out.Banners[i] = bannerDTO{ImageURL: banner.ImageURL, LinkURL: banner.LinkURL}
	}
	return out
}

// ディビジョン

type divisionInputDTO struct {
	Name               i18nDTO `json:"name"`
	Kind               string  `json:"kind"`
	SortOrder          int     `json:"sortOrder"`
	ScheduleEnabled    bool    `json:"scheduleEnabled"`
	StandingsPublished bool    `json:"standingsPublished"`
	EntryKind          string  `json:"entryKind"`
	StatsKind          string  `json:"statsKind"`
}

type adminDivisionDTO struct {
	ID           int64 `json:"id"`
	TournamentID int64 `json:"tournamentId"`
	divisionInputDTO
}

type adminDivisionListDTO struct {
	Divisions []adminDivisionDTO `json:"divisions"`
}

func newDivisionInput() divisionInputDTO {
	return divisionInputDTO{
		Kind: string(domain.DivisionNone), ScheduleEnabled: true,
		EntryKind: string(domain.EntryTeam), StatsKind: string(domain.StatsUltimate),
	}
}

func toDivisionInput(record store.DivisionRecord) divisionInputDTO {
	return divisionInputDTO{
		Name: toI18n(record.Name), Kind: string(record.Kind), SortOrder: record.SortOrder,
		ScheduleEnabled: record.ScheduleEnabled, StandingsPublished: record.StandingsPublished,
		EntryKind: string(record.EntryKind), StatsKind: string(record.StatsKind),
	}
}

func (in divisionInputDTO) fields() store.DivisionFields {
	return store.DivisionFields{
		Name: fromI18n(in.Name), Kind: domain.DivisionKind(in.Kind), SortOrder: in.SortOrder,
		ScheduleEnabled: in.ScheduleEnabled, StandingsPublished: in.StandingsPublished,
		EntryKind: domain.EntryKind(in.EntryKind), StatsKind: domain.StatsKind(in.StatsKind),
	}
}

func toAdminDivision(record store.DivisionRecord) adminDivisionDTO {
	return adminDivisionDTO{
		ID:               record.ID,
		TournamentID:     record.TournamentID,
		divisionInputDTO: toDivisionInput(record),
	}
}

// グループ

type groupImageDTO struct {
	URL    string `json:"url"`
	Width  *int   `json:"width"`
	Height *int   `json:"height"`
}

type groupInputDTO struct {
	Name      string         `json:"name"`
	Kind      string         `json:"kind"`
	SortOrder int            `json:"sortOrder"`
	Linked    bool           `json:"linked"`
	Published bool           `json:"published"`
	Image     *groupImageDTO `json:"image"`
}

type groupTeamDTO struct {
	TeamID       int64 `json:"teamId"`
	RankOverride *int  `json:"rankOverride"`
}

type groupBracketDTO struct {
	Pattern string `json:"pattern"`
	PtnName string `json:"ptnName"`
}

type adminGroupDTO struct {
	ID         int64 `json:"id"`
	DivisionID int64 `json:"divisionId"`
	groupInputDTO
	Bracket *groupBracketDTO `json:"bracket"`
	Teams   []groupTeamDTO   `json:"teams"`
}

type adminGroupListDTO struct {
	Groups []adminGroupDTO `json:"groups"`
}

type groupTeamsBodyDTO struct {
	Teams []groupTeamDTO `json:"teams"`
}

type matchSideInputDTO struct {
	TeamID      *int64          `json:"teamId"`
	Placeholder *placeholderDTO `json:"placeholder"`
}

type bracketBodyDTO struct {
	Template   string              `json:"template"`
	CodePrefix string              `json:"codePrefix"`
	Seeds      []matchSideInputDTO `json:"seeds"`
}

type bracketTemplateDTO struct {
	ID    string  `json:"id"`
	Name  i18nDTO `json:"name"`
	Teams int     `json:"teams"`
}

type bracketTemplateListDTO struct {
	Templates []bracketTemplateDTO `json:"templates"`
}

func newGroupInput() groupInputDTO {
	return groupInputDTO{Kind: string(domain.GroupPool), Linked: true, Published: true}
}

func toGroupInput(record store.GroupRecord) groupInputDTO {
	in := groupInputDTO{
		Name: record.Name, Kind: string(record.Kind), SortOrder: record.SortOrder,
		Linked: record.Linked, Published: record.Published,
	}
	if record.Image != nil {
		in.Image = &groupImageDTO{
			URL:    record.Image.URL,
			Width:  record.Image.Width,
			Height: record.Image.Height,
		}
	}
	return in
}

func (in groupInputDTO) fields() store.GroupFields {
	fields := store.GroupFields{
		Name: in.Name, Kind: domain.GroupKind(in.Kind), SortOrder: in.SortOrder,
		Linked: in.Linked, Published: in.Published,
	}
	if in.Image != nil {
		fields.Image = &store.GroupImage{
			URL:    in.Image.URL,
			Width:  in.Image.Width,
			Height: in.Image.Height,
		}
	}
	return fields
}

func toAdminGroup(record store.GroupRecord) adminGroupDTO {
	out := adminGroupDTO{
		ID: record.ID, DivisionID: record.DivisionID, groupInputDTO: toGroupInput(record),
		Teams: make([]groupTeamDTO, len(record.Teams)),
	}
	if record.Bracket != nil {
		out.Bracket = &groupBracketDTO{
			Pattern: record.Bracket.Pattern,
			PtnName: record.Bracket.PtnName,
		}
	}
	for i, team := range record.Teams {
		out.Teams[i] = groupTeamDTO{TeamID: team.TeamID, RankOverride: team.RankOverride}
	}
	return out
}

func (in matchSideInputDTO) fields() store.MatchSideFields {
	side := store.MatchSideFields{TeamID: in.TeamID}
	if in.Placeholder != nil {
		side.Placeholder = &domain.Placeholder{
			Code: in.Placeholder.Code, Outcome: domain.Outcome(in.Placeholder.Outcome),
		}
	}
	return side
}

// チームと選手

type teamInputDTO struct {
	FullName  i18nDTO `json:"fullName"`
	ShortName i18nDTO `json:"shortName"`
	Comment   i18nDTO `json:"comment"`
	BarStyle  string  `json:"barStyle"`
	SortOrder int     `json:"sortOrder"`
	FinalRank *int    `json:"finalRank"`
}

type adminTeamDTO struct {
	ID           int64 `json:"id"`
	TournamentID int64 `json:"tournamentId"`
	DivisionID   int64 `json:"divisionId"`
	teamInputDTO
}

type adminTeamListDTO struct {
	Teams []adminTeamDTO `json:"teams"`
}

type finalRankDTO struct {
	TeamID int64 `json:"teamId"`
	Rank   *int  `json:"rank"`
}

func newTeamInput() teamInputDTO { return teamInputDTO{BarStyle: "default"} }

func toTeamInput(record store.TeamRecord) teamInputDTO {
	return teamInputDTO{
		FullName: toI18n(
			record.FullName,
		),
		ShortName: toI18n(record.ShortName),
		Comment:   toI18n(record.Comment),
		BarStyle:  string(record.BarStyle),
		SortOrder: record.SortOrder,
		FinalRank: record.FinalRank,
	}
}

func (in teamInputDTO) fields() store.TeamFields {
	return store.TeamFields{
		FullName: fromI18n(
			in.FullName,
		),
		ShortName: fromI18n(in.ShortName),
		Comment:   fromI18n(in.Comment),
		BarStyle: domain.TeamBarStyle(
			in.BarStyle,
		),
		SortOrder: in.SortOrder,
		FinalRank: in.FinalRank,
	}
}

func toAdminTeam(record store.TeamRecord) adminTeamDTO {
	return adminTeamDTO{
		ID: record.ID, TournamentID: record.TournamentID, DivisionID: record.DivisionID,
		teamInputDTO: toTeamInput(record),
	}
}

func toAdminTeams(records []store.TeamRecord) adminTeamListDTO {
	out := adminTeamListDTO{Teams: make([]adminTeamDTO, len(records))}
	for i, record := range records {
		out.Teams[i] = toAdminTeam(record)
	}
	return out
}

type playerInputDTO struct {
	Number string   `json:"number"`
	Name   i18nDTO  `json:"name"`
	Gender *string  `json:"gender"`
	Role   *i18nDTO `json:"role"` // null なら選手、値があればスタッフ
}

type adminPlayerDTO struct {
	ID     int64 `json:"id"`
	TeamID int64 `json:"teamId"`
	playerInputDTO
}

type adminPlayerListDTO struct {
	Players []adminPlayerDTO `json:"players"`
}

func toPlayerInput(record store.PlayerRecord) playerInputDTO {
	in := playerInputDTO{Number: record.Number, Name: toI18n(record.Name)}
	if record.Gender != nil {
		gender := string(*record.Gender)
		in.Gender = &gender
	}
	if record.Role != nil {
		role := toI18n(*record.Role)
		in.Role = &role
	}
	return in
}

func (in playerInputDTO) fields() store.PlayerFields {
	fields := store.PlayerFields{Number: in.Number, Name: fromI18n(in.Name)}
	if in.Gender != nil {
		gender := domain.Gender(*in.Gender)
		fields.Gender = &gender
	}
	if in.Role != nil {
		role := fromI18n(*in.Role)
		fields.Role = &role
	}
	return fields
}

func toAdminPlayer(record store.PlayerRecord) adminPlayerDTO {
	return adminPlayerDTO{
		ID:             record.ID,
		TeamID:         record.TeamID,
		playerInputDTO: toPlayerInput(record),
	}
}

func toAdminPlayers(records []store.PlayerRecord) []adminPlayerDTO {
	out := make([]adminPlayerDTO, len(records))
	for i, record := range records {
		out[i] = toAdminPlayer(record)
	}
	return out
}

// 試合

type matchInputDTO struct {
	GroupID   *int64            `json:"groupId"`
	Code      string            `json:"code"`
	Date      *string           `json:"date"`
	Time      *string           `json:"time"`
	Field     *string           `json:"field"`
	HasDetail bool              `json:"hasDetail"`
	Left      matchSideInputDTO `json:"left"`
	Right     matchSideInputDTO `json:"right"`
}

type adminMatchDTO struct {
	ID         int64        `json:"id"`
	DivisionID int64        `json:"divisionId"`
	GroupID    *int64       `json:"groupId"`
	Code       string       `json:"code"`
	Date       *string      `json:"date"`
	Time       *string      `json:"time"`
	Field      *string      `json:"field"`
	HasDetail  bool         `json:"hasDetail"`
	Status     string       `json:"status"`
	Left       matchSideDTO `json:"left"`
	Right      matchSideDTO `json:"right"`
	WinnerSide *string      `json:"winnerSide"`
	ResultType string       `json:"resultType"`
}

type adminMatchListDTO struct {
	Matches []adminMatchDTO `json:"matches"`
}

type adminEventDTO struct {
	ID         int64         `json:"id"`
	ClientID   string        `json:"clientId"`
	Seq        int           `json:"seq"`
	ElapsedSec int           `json:"elapsedSec"`
	Kind       string        `json:"kind"`
	Side       *string       `json:"side"`
	Scorer     *playerRefDTO `json:"scorer"`
	Assist     *playerRefDTO `json:"assist"`
	Callahan   bool          `json:"callahan"`
	LeftScore  int           `json:"leftScore"`
	RightScore int           `json:"rightScore"`
}

type playerStatDTO struct {
	PlayerID  int64  `json:"playerId"`
	Side      string `json:"side"`
	Blocks    int    `json:"blocks"`
	Greatests int    `json:"greatests"`
}

type rostersDTO struct {
	Left  []adminPlayerDTO `json:"left"`
	Right []adminPlayerDTO `json:"right"`
}

type matchContextDTO struct {
	TournamentID   int64   `json:"tournamentId"`
	TournamentName string  `json:"tournamentName"`
	DivisionName   string  `json:"divisionName"`
	GroupName      *string `json:"groupName"`
}

type matchDetailDTO struct {
	Match        adminMatchDTO    `json:"match"`
	Context      matchContextDTO  `json:"context"`
	InBracket    bool             `json:"inBracket"`
	ForfeitScore *forfeitScoreDTO `json:"forfeitScore"`
	StartedAt    *string          `json:"startedAt"`
	Events       []adminEventDTO  `json:"events"`
	PlayerStats  []playerStatDTO  `json:"playerStats"`
	Rosters      rostersDTO       `json:"rosters"`
	// 記録画面のための値(記録担当、二重記録の疑い、試合時間)
	Scorer            *scorerClaimDTO       `json:"scorer"`
	ConcurrentRecords []concurrentRecordDTO `json:"concurrentRecords"`
	MatchMinutes      *int                  `json:"matchMinutes"`
}

type eventBodyDTO struct {
	ClientID   *string `json:"clientId"`
	Kind       string  `json:"kind"`
	Side       *string `json:"side"`
	ElapsedSec *int    `json:"elapsedSec"`
	ScorerID   *int64  `json:"scorerId"`
	AssistID   *int64  `json:"assistId"`
	Callahan   bool    `json:"callahan"`
	WinnerSide *string `json:"winnerSide"`
	// ExpectedScore は kind=end で記録係が確認したスコア
	ExpectedScore *scoreDTO `json:"expectedScore"`
}

type scoreDTO struct {
	Left  int `json:"left"`
	Right int `json:"right"`
}

// toDomain は nil なら nil を返す。
func (s *scoreDTO) toDomain() *domain.Score {
	if s == nil {
		return nil
	}
	return &domain.Score{Left: s.Left, Right: s.Right}
}

type statusBodyDTO struct {
	ClientID   *string   `json:"clientId"`
	Status     string    `json:"status"`
	WinnerSide *string   `json:"winnerSide"`
	Score      *scoreDTO `json:"score"`
	// ExpectedScore は終了にするときに記録係が確認したスコア
	ExpectedScore *scoreDTO `json:"expectedScore"`
}

type eventPlayersBodyDTO struct {
	ClientID *string `json:"clientId"`
	ScorerID *int64  `json:"scorerId"`
	AssistID *int64  `json:"assistId"`
	Callahan bool    `json:"callahan"`
}

type resultBodyDTO struct {
	ClientID     *string          `json:"clientId"`
	ResultType   string           `json:"resultType"`
	WinnerSide   *string          `json:"winnerSide"`
	ForfeitScore *forfeitScoreDTO `json:"forfeitScore"`
	// ExpectedScore は途中打ち切りで記録係が確認したスコア
	ExpectedScore *scoreDTO `json:"expectedScore"`
}

type playerStatsBodyDTO struct {
	Stats []playerStatDTO `json:"stats"`
}

func newMatchInput() matchInputDTO { return matchInputDTO{HasDetail: true} }

func toMatchInput(record store.MatchRecord) matchInputDTO {
	side := func(side domain.MatchSide) matchSideInputDTO {
		in := matchSideInputDTO{}
		if side.Team != nil {
			id := side.Team.ID
			in.TeamID = &id
		}
		if side.Placeholder != nil {
			in.Placeholder = &placeholderDTO{
				Code:    side.Placeholder.Code,
				Outcome: string(side.Placeholder.Outcome),
			}
		}
		return in
	}
	return matchInputDTO{
		GroupID:   record.GroupID,
		Code:      record.Code,
		Date:      record.Date,
		Time:      record.Time,
		Field:     record.Field,
		HasDetail: record.HasDetail,
		Left:      side(record.Left),
		Right:     side(record.Right),
	}
}

func (in matchInputDTO) fields() store.MatchFields {
	return store.MatchFields{
		GroupID: in.GroupID, Code: in.Code, Date: in.Date, Time: in.Time, Field: in.Field,
		HasDetail: in.HasDetail, Left: in.Left.fields(), Right: in.Right.fields(),
	}
}

func sidePtrString(side *domain.Side) *string {
	if side == nil {
		return nil
	}
	value := string(*side)
	return &value
}

func toSidePtr(value *string) *domain.Side {
	if value == nil {
		return nil
	}
	side := domain.Side(*value)
	return &side
}

func toAdminMatch(record store.MatchRecord) adminMatchDTO {
	return adminMatchDTO{
		ID:         record.ID,
		DivisionID: record.DivisionID,
		GroupID:    record.GroupID,
		Code:       record.Code,
		Date:       record.Date,
		Time:       record.Time,
		Field:      record.Field,
		HasDetail:  record.HasDetail,
		Status: string(
			record.Status,
		),
		Left:       toMatchSide(record.Left),
		Right:      toMatchSide(record.Right),
		WinnerSide: sidePtrString(record.WinnerSide),
		ResultType: resultTypeString(record.ResultType),
	}
}

// resultTypeString は結果の種類の文字列。値の無い行(取込の途中など)は normal にする。
func resultTypeString(value domain.ResultType) string {
	if value == "" {
		return string(domain.ResultNormal)
	}
	return string(value)
}

func toAdminMatches(records []store.MatchRecord) adminMatchListDTO {
	out := adminMatchListDTO{Matches: make([]adminMatchDTO, len(records))}
	for i, record := range records {
		out.Matches[i] = toAdminMatch(record)
	}
	return out
}

func toMatchDetail(detail store.MatchDetail) matchDetailDTO {
	out := matchDetailDTO{
		Match: toAdminMatch(detail.Match), InBracket: detail.InBracket,
		Context: matchContextDTO{
			TournamentID: detail.TournamentID, TournamentName: detail.TournamentName,
			DivisionName: detail.DivisionName, GroupName: detail.GroupName,
		},
		ForfeitScore: toForfeitScoreDTO(detail.ForfeitScore),
		Events:       make([]adminEventDTO, len(detail.Events)),
		PlayerStats:  make([]playerStatDTO, len(detail.PlayerStats)),
		Rosters: rostersDTO{
			Left: toAdminPlayers(detail.Rosters.Left), Right: toAdminPlayers(detail.Rosters.Right),
		},
	}
	for i, ev := range detail.Events {
		out.Events[i] = adminEventDTO{
			ID:         ev.ID,
			ClientID:   ev.ClientID,
			Seq:        ev.Seq,
			ElapsedSec: ev.ElapsedSec,
			Kind:       string(ev.Kind),
			Side:       sidePtrString(ev.Side),
			Scorer: toPlayerRefPtr(
				ev.Scorer,
			),
			Assist:     toPlayerRefPtr(ev.Assist),
			Callahan:   ev.Callahan,
			LeftScore:  ev.LeftScore,
			RightScore: ev.RightScore,
		}
	}
	if detail.StartedAt != nil {
		started := formatInstant(*detail.StartedAt)
		out.StartedAt = &started
	}
	out.Scorer = toScorerClaimDTO(detail.Scorer)
	out.ConcurrentRecords = toConcurrentRecordDTOs(detail.ConcurrentRecords)
	out.MatchMinutes = detail.MatchMinutes
	for i, stat := range detail.PlayerStats {
		out.PlayerStats[i] = playerStatDTO{
			PlayerID:  stat.PlayerID,
			Side:      string(stat.Side),
			Blocks:    stat.Blocks,
			Greatests: stat.Greatests,
		}
	}
	return out
}

// ギャラリー

type galleryInputDTO struct {
	Name      string `json:"name"`
	SortOrder int    `json:"sortOrder"`
}

// adminGalleryDTO の写真は閲覧用のギャラリーと同じ photoDTO。
type adminGalleryDTO struct {
	ID           int64 `json:"id"`
	TournamentID int64 `json:"tournamentId"`
	galleryInputDTO
	Photos []photoDTO `json:"photos"`
}

type adminGalleryListDTO struct {
	Galleries []adminGalleryDTO `json:"galleries"`
}

type photosBodyDTO struct {
	Photos []photoDTO `json:"photos"`
}

func toAdminGallery(record store.GalleryRecord) adminGalleryDTO {
	out := adminGalleryDTO{
		ID: record.ID, TournamentID: record.TournamentID,
		galleryInputDTO: galleryInputDTO{Name: record.Name, SortOrder: record.SortOrder},
		Photos:          make([]photoDTO, len(record.Photos)),
	}
	for i, photo := range record.Photos {
		out.Photos[i] = photoDTO{ThumbURL: photo.ThumbURL, FullURL: photo.FullURL}
	}
	return out
}
