package httpapi

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/entry"
	"scoreview/api/internal/notify"
)

// 管理画面から公式アカウントで送る(docs/api.md「公式 LINE で送る」)。宛先は申込の代表とメンバーなので、
// 申込と同じく管理者だけ。決まりは notify、記録は store に任せる。

func (s *Server) notificationRoutes() {
	const base = "/api/v1/admin"
	ok, created := http.StatusOK, http.StatusCreated
	s.admin("GET "+base+"/tournaments/{id}/notifications", adminOnly(), ok, withPathID(s.listNotifications))
	s.admin("GET "+base+"/tournaments/{id}/notifications/preview", adminOnly(), ok,
		withPathID(s.previewNotification))
	s.admin("POST "+base+"/tournaments/{id}/notifications", adminOnly(), created, withPathID(s.sendNotification))
	s.admin("GET "+base+"/notifications/{id}", adminOnly(), ok, withPathID(s.getNotification))
	s.admin("POST "+base+"/notifications/{id}/resend", adminOnly(), ok, withPathID(s.resendNotification))
	s.admin("POST "+base+"/notifications/{id}/cancel", adminOnly(), ok, withPathID(s.cancelNotification))
	// 定期の呼び出し(web の /cron/notify が CRON_SECRET を確かめてから呼ぶ)。利用者のログインは無く、
	// ほかの口と同じく web との共有秘密(X-Internal-Token)だけで受ける
	s.handle("POST /api/v1/internal/notifications/dispatch", s.dispatchNotifications)
}

// audienceDTO は宛先の条件。status と divisionId は絞り込まないとき空と null。
type audienceDTO struct {
	Kind       string `json:"kind"`
	Status     string `json:"status"`
	DivisionID *int64 `json:"divisionId"`
}

type recipientDTO struct {
	Name      string   `json:"name"`
	Teams     []string `json:"teams"`
	Reachable bool     `json:"reachable"`
}

type quotaDTO struct {
	Limit *int64 `json:"limit"`
	Used  int64  `json:"used"`
}

type notificationPreviewDTO struct {
	Heading     string         `json:"heading"`
	Recipients  []recipientDTO `json:"recipients"`
	Reachable   int            `json:"reachable"`
	Unreachable int            `json:"unreachable"`
	Quota       *quotaDTO      `json:"quota"`
	CanSend     bool           `json:"canSend"`
	Variables   []variableDTO  `json:"variables"`
	SampleName  string         `json:"sampleName"`
	Everyone    bool           `json:"everyone"`
}

// variableDTO は文に書ける差し込み({name})と、送る画面の見本の人の値。
type variableDTO struct {
	Name    string `json:"name"`
	Example string `json:"example"`
}

type notificationCountsDTO struct {
	Pending int `json:"pending"`
	Sent    int `json:"sent"`
	Skipped int `json:"skipped"`
	Failed  int `json:"failed"`
}

type notificationSummaryDTO struct {
	ID           int64                   `json:"id"`
	TournamentID int64                   `json:"tournamentId"`
	Audience     audienceDTO             `json:"audience"`
	Body         string                  `json:"body"`
	Automatic    bool                    `json:"automatic"`
	SentBy       string                  `json:"sentBy"`
	CreatedAt    time.Time               `json:"createdAt"`
	SendAt       *time.Time              `json:"sendAt"`
	State        string                  `json:"state"`
	Note         string                  `json:"note"`
	Counts       notificationCountsDTO   `json:"counts"`
	CanResend    bool                    `json:"canResend"`
	Buttons      []notificationButtonDTO `json:"buttons"`
}

// notificationButtonDTO は届く文の下のボタン。url は site と url のときの行き先(entry は空)。
type notificationButtonDTO struct {
	Kind  string `json:"kind"`
	Label string `json:"label"`
	URL   string `json:"url"`
}

type notificationJobDTO struct {
	Name   string     `json:"name"`
	Teams  string     `json:"teams"`
	Status string     `json:"status"`
	Error  string     `json:"error"`
	SentAt *time.Time `json:"sentAt"`
}

type notificationDetailDTO struct {
	notificationSummaryDTO
	Jobs []notificationJobDTO `json:"jobs"`
}

type notificationListDTO struct {
	Notifications []notificationSummaryDTO `json:"notifications"`
	CanSend       bool                     `json:"canSend"`
}

type notificationInputDTO struct {
	Audience   audienceDTO             `json:"audience"`
	Body       string                  `json:"body"`
	Buttons    []notificationButtonDTO `json:"buttons"`
	ConfirmAll bool                    `json:"confirmAll"`
	RequestKey string                  `json:"requestKey"`
	SendAt     *time.Time              `json:"sendAt"`
}

type dispatchDTO struct {
	Sent    int `json:"sent"`
	Skipped int `json:"skipped"`
}

func (a audienceDTO) audience() notify.Audience {
	out := notify.Audience{Kind: notify.AudienceKind(a.Kind), Status: entry.Status(a.Status)}
	if a.DivisionID != nil {
		out.DivisionID = *a.DivisionID
	}
	return out
}

func toAudienceDTO(audience notify.Audience) audienceDTO {
	out := audienceDTO{Kind: string(audience.Kind), Status: string(audience.Status)}
	if audience.DivisionID != 0 {
		out.DivisionID = &audience.DivisionID
	}
	return out
}

func toNotificationSummary(batch notify.Batch) notificationSummaryDTO {
	counts := batch.Counts
	return withButtons(notificationSummaryDTO{ID: batch.ID, TournamentID: batch.TournamentID,
		Audience: toAudienceDTO(batch.Audience), Body: batch.Body, Automatic: batch.Automatic,
		SentBy: batch.SentByName, CreatedAt: batch.CreatedAt, SendAt: batch.SendAt, State: string(batch.State),
		Note: string(batch.Note), Counts: notificationCountsDTO{
			Pending: counts.Pending, Sent: counts.Sent, Skipped: counts.Skipped, Failed: counts.Failed},
		CanResend: batch.CanResend, Buttons: make([]notificationButtonDTO, len(batch.Buttons))}, batch.Buttons)
}

func withButtons(out notificationSummaryDTO, buttons []notify.Button) notificationSummaryDTO {
	for i, button := range buttons {
		out.Buttons[i] = notificationButtonDTO{Kind: string(button.Kind), Label: button.Label, URL: button.URL}
	}
	return out
}

func toNotificationDetail(batch notify.Batch) notificationDetailDTO {
	out := notificationDetailDTO{notificationSummaryDTO: toNotificationSummary(batch),
		Jobs: make([]notificationJobDTO, len(batch.Jobs))}
	for i, job := range batch.Jobs {
		out.Jobs[i] = notificationJobDTO{Name: job.Name, Teams: job.Teams, Status: string(job.Status),
			Error: job.Error, SentAt: job.SentAt}
	}
	return out
}

func (s *Server) listNotifications(ctx context.Context, id int64, _ *http.Request, _ domain.User) (any, error) {
	history, err := s.notices.History(ctx, id)
	if err != nil {
		return nil, err
	}
	out := notificationListDTO{Notifications: make([]notificationSummaryDTO, len(history.Batches)),
		CanSend: history.CanSend}
	for i, batch := range history.Batches {
		out.Notifications[i] = toNotificationSummary(batch)
	}
	return out, nil
}

// previewNotification は宛先の条件をクエリ(kind、status、divisionId)で受ける。
func (s *Server) previewNotification(ctx context.Context, id int64, r *http.Request, _ domain.User) (any, error) {
	query := r.URL.Query()
	audience := audienceDTO{Kind: query.Get("kind"), Status: query.Get("status")}
	if text := query.Get("divisionId"); text != "" {
		division, err := strconv.ParseInt(text, 10, 64)
		if err != nil || division <= 0 {
			return nil, domain.Invalid("divisionId", "must be a positive integer")
		}
		audience.DivisionID = &division
	}
	preview, err := s.notices.Preview(ctx, id, audience.audience())
	if err != nil {
		return nil, err
	}
	out := notificationPreviewDTO{Heading: preview.Heading, Recipients: make([]recipientDTO, len(preview.Recipients)),
		Reachable: preview.Reachable, Unreachable: preview.Unreachable, CanSend: preview.CanSend,
		Variables: make([]variableDTO, len(preview.Variables)), SampleName: preview.SampleName,
		Everyone: preview.Everyone}
	for i, variable := range preview.Variables {
		out.Variables[i] = variableDTO{Name: variable.Name, Example: variable.Example}
	}
	for i, recipient := range preview.Recipients {
		out.Recipients[i] = recipientDTO{Name: recipient.Name, Teams: recipient.Teams, Reachable: recipient.Friend}
	}
	if preview.Quota != nil {
		out.Quota = &quotaDTO{Limit: preview.Quota.Limit, Used: preview.Quota.Used}
	}
	return out, nil
}

func (s *Server) sendNotification(ctx context.Context, id int64, r *http.Request, user domain.User) (any, error) {
	var body notificationInputDTO
	if err := decodeBody(r, &body); err != nil {
		return nil, err
	}
	request := notify.Request{Audience: body.Audience.audience(), Body: body.Body, RequestKey: body.RequestKey,
		SendAt: body.SendAt, Buttons: make([]notify.Button, len(body.Buttons)), ConfirmAll: body.ConfirmAll}
	for i, button := range body.Buttons {
		request.Buttons[i] = notify.Button{Kind: notify.ButtonKind(button.Kind), Label: button.Label, URL: button.URL}
	}
	batch, err := s.notices.Send(ctx, id, request, actorOf(user))
	if err != nil {
		return nil, err
	}
	return toNotificationDetail(batch), nil
}

func (s *Server) getNotification(ctx context.Context, id int64, _ *http.Request, _ domain.User) (any, error) {
	batch, err := s.notices.Batch(ctx, id)
	if err != nil {
		return nil, err
	}
	return toNotificationDetail(batch), nil
}

func (s *Server) resendNotification(ctx context.Context, id int64, _ *http.Request, _ domain.User) (any, error) {
	batch, err := s.notices.Resend(ctx, id)
	if err != nil {
		return nil, err
	}
	return toNotificationDetail(batch), nil
}

func (s *Server) cancelNotification(ctx context.Context, id int64, _ *http.Request, _ domain.User) (any, error) {
	batch, err := s.notices.Cancel(ctx, id)
	if err != nil {
		return nil, err
	}
	return toNotificationDetail(batch), nil
}

// dispatchNotifications は送る時刻の来た予約を送り、送った数と送れなかった数を返す。予約の扱いに
// 失敗があれば 500 にして、定期の呼び出しのログに残す(宛先を書く前に失敗した予約は予約のまま残り、
// 次の呼び出しでもう一度扱う。宛先を書いた後の失敗は、宛先の結果として記録に残り、送り直せる)。
func (s *Server) dispatchNotifications(r *http.Request) (any, error) {
	result, err := s.notices.Dispatch(r.Context())
	if err != nil {
		return nil, err
	}
	return dispatchDTO{Sent: result.Sent, Skipped: result.Skipped}, nil
}
