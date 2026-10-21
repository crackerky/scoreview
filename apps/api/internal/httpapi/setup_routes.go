package httpapi

import (
	"context"
	"net/http"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/store"
)

// 大会の準備の管理用エンドポイント(docs/api.md「大会の準備」): 大会のコピーと、試合の無い部門の
// チームの一括削除と、英語名が空のチーム、選手、組に日本語名を入れる操作。管理者だけ。

func (s *Server) setupRoutes() {
	const base = "/api/v1/admin"
	s.admin("POST "+base+"/tournaments/{id}/copy", adminOnly(), http.StatusCreated,
		withPathID(s.copyTournament))
	s.admin("DELETE "+base+"/divisions/{id}/teams", adminOnly(), http.StatusOK,
		withPathID(s.deleteDivisionTeams))
	s.admin("POST "+base+"/divisions/{id}/english-names", adminOnly(), http.StatusOK,
		withPathID(s.fillEnglishNames))
}

type filledEnglishNamesDTO struct {
	Teams   int `json:"teams"`
	Players int `json:"players"`
	Groups  int `json:"groups"`
}

type tournamentCopyInputDTO struct {
	Name      i18nDTO `json:"name"`
	StartDate *string `json:"startDate"`
	WithTeams bool    `json:"withTeams"`
}

type deletedTeamsDTO struct {
	Deleted int `json:"deleted"`
}

func (s *Server) copyTournament(ctx context.Context, id int64, r *http.Request, _ domain.User) (any, error) {
	var in tournamentCopyInputDTO
	if err := decodeBody(r, &in); err != nil {
		return nil, err
	}
	if in.StartDate == nil || *in.StartDate == "" {
		return nil, domain.Invalid("startDate", "新しい大会の開催日を入れてください")
	}
	source, err := store.GetAdminTournament(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	fields := source.TournamentFields
	if in.Name.Ja != "" {
		fields.Name = domain.I18nText{Ja: in.Name.Ja, En: in.Name.En}
	}
	fields.StartDate = in.StartDate
	// 新しい大会は準備中: 受付中で、公開の一覧には載せない(運営者レビュー M2)
	fields.Status, fields.Listed = domain.TournamentEntryOpen, false
	// 保存済みのお知らせ欄も、保存のときと同じ無害化を通してから写す
	fields.TopHTML = domain.I18nText{Ja: s.sanitizer.HTML(fields.TopHTML.Ja), En: s.sanitizer.HTML(fields.TopHTML.En)}
	record, err := store.CopyTournament(ctx, s.db, id, fields, in.WithTeams)
	if err != nil {
		return nil, err
	}
	return toAdminTournament(record), nil
}

func (s *Server) deleteDivisionTeams(ctx context.Context, id int64, _ *http.Request, _ domain.User) (any, error) {
	deleted, err := store.DeleteDivisionTeams(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	return deletedTeamsDTO{Deleted: deleted}, nil
}

func (s *Server) fillEnglishNames(ctx context.Context, id int64, _ *http.Request, _ domain.User) (any, error) {
	filled, err := store.FillDivisionEnglishNames(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	return filledEnglishNamesDTO{Teams: filled.Teams, Players: filled.Players, Groups: filled.Groups}, nil
}
