// Package storetest はテスト用の fixture ビルダー。行は GORM の Create で入れる(Raw/Exec を使わない)。
// store と httpapi のテストから使う。本番コードから import しない。
package storetest

import (
	"context"
	"testing"
	"time"

	"gorm.io/gorm"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/store"
)

// Builder は既定値つきで行を作る。既定値は各メソッドの opts で上書きする。
// 失敗したら tb.Fatal で止める(fixture の失敗はテストの前提が崩れたことなので続けない)。
type Builder struct {
	tb  testing.TB
	db  *gorm.DB
	seq map[int64]int // 試合ごとのイベントの seq
}

// New は Builder を作る。
func New(tb testing.TB, db *gorm.DB) *Builder {
	tb.Helper()
	return &Builder{tb: tb, db: db.WithContext(context.Background()), seq: map[int64]int{}}
}

// DB は Builder が書き込む DB を返す。
func (b *Builder) DB() *gorm.DB { return b.db }

func create[T any](b *Builder, row *T, opts []func(*T)) T {
	b.tb.Helper()
	for _, opt := range opts {
		opt(row)
	}
	if err := b.db.Create(row).Error; err != nil {
		b.tb.Fatalf("storetest: create %T: %v", row, err)
	}
	return *row
}

// Ptr は値のポインタを返す(fixture の nullable 列を埋めるため)。
func Ptr[T any](v T) *T { return &v }

// Date は "YYYY-MM-DD" を DATE 列の値にする。
func Date(tb testing.TB, value string) *time.Time {
	tb.Helper()
	day, err := time.Parse("2006-01-02", value)
	if err != nil {
		tb.Fatalf("storetest: parse date %q: %v", value, err)
	}
	return &day
}

// Tournament は終了済みで一覧に載る大会を作る。
func (b *Builder) Tournament(opts ...func(*store.Tournament)) store.Tournament {
	b.tb.Helper()
	return create(b, &store.Tournament{
		NameJa: "テスト大会", NameEn: "Test Tournament", ShortName: "Test",
		Status: domain.TournamentFinished, Listed: true,
		TeamListLabel: "teams", GroupLabel: "groups", ShowSchedule: true,
	}, opts)
}

// Banner はサイドバーのバナーを作る。
func (b *Builder) Banner(
	tournamentID int64,
	opts ...func(*store.TournamentBanner),
) store.TournamentBanner {
	b.tb.Helper()
	return create(
		b,
		&store.TournamentBanner{TournamentID: tournamentID, ImageURL: "/banner.png"},
		opts,
	)
}

// Division はチーム戦のアルティメットのディビジョンを作る。
func (b *Builder) Division(tournamentID int64, opts ...func(*store.Division)) store.Division {
	b.tb.Helper()
	return create(b, &store.Division{
		TournamentID: tournamentID, NameJa: "ミックス", NameEn: "Mixed", Kind: domain.DivisionMixed,
		ScheduleEnabled: true, EntryKind: domain.EntryTeam, StatsKind: domain.StatsUltimate,
	}, opts)
}

// Group は公開でリンクありのグループを作る。bracket 型は格子、image 型は画像 URL を opts で入れること。
func (b *Builder) Group(
	divisionID int64,
	kind domain.GroupKind,
	opts ...func(*store.Group),
) store.Group {
	b.tb.Helper()
	return create(b, &store.Group{
		DivisionID: divisionID, Name: "Qualifying pool A", Kind: kind, Linked: true, Published: true,
	}, opts)
}

// Team はディビジョンのチームを作る。
func (b *Builder) Team(division store.Division, opts ...func(*store.Team)) store.Team {
	b.tb.Helper()
	return create(b, &store.Team{
		TournamentID: division.TournamentID, DivisionID: division.ID,
		NameFullJa: "テストチーム", NameShortJa: "テスト", NameFullEn: "Test Team", NameShortEn: "Test",
		BarStyle: "default",
	}, opts)
}

// GroupTeam はチームをグループに登録する。
func (b *Builder) GroupTeam(groupID, teamID int64, opts ...func(*store.GroupTeam)) store.GroupTeam {
	b.tb.Helper()
	return create(b, &store.GroupTeam{GroupID: groupID, TeamID: teamID}, opts)
}

// Player は選手を作る。スタッフにするなら RoleJa/RoleEn を入れる。
func (b *Builder) Player(teamID int64, opts ...func(*store.Player)) store.Player {
	b.tb.Helper()
	return create(b, &store.Player{TeamID: teamID, Number: "1", NameJa: "選手", NameEn: "Player"}, opts)
}

// Match は詳細ページのある未試合を作る。left/right が nil の側は未確定(placeholder は opts で)。
func (b *Builder) Match(
	division store.Division,
	left, right *store.Team,
	opts ...func(*store.Match),
) store.Match {
	b.tb.Helper()
	match := &store.Match{DivisionID: division.ID, Status: domain.MatchScheduled, HasDetail: true}
	if left != nil {
		match.LeftTeamID = &left.ID
	}
	if right != nil {
		match.RightTeamID = &right.ID
	}
	return create(b, match, opts)
}

// Finished は試合を終了済みにしてスコアを入れる opts。
func Finished(left, right int) func(*store.Match) {
	return func(match *store.Match) {
		match.Status = domain.MatchFinished
		match.LeftScore = &left
		match.RightScore = &right
	}
}

// At は試合の日付、時刻、コートを入れる opts。date は "YYYY-MM-DD"、clock は "HH:MM"。
func At(tb testing.TB, date, clock, field string) func(*store.Match) {
	tb.Helper()
	day := Date(tb, date)
	return func(match *store.Match) {
		match.MatchDate = day
		match.StartTime = &clock
		match.Field = &field
	}
}

// Event は試合のイベントを作る。seq は試合ごとに 1 から振る。
func (b *Builder) Event(matchID int64, kind domain.EventKind, side *domain.Side, elapsedSec int,
	opts ...func(*store.MatchEvent),
) store.MatchEvent {
	b.tb.Helper()
	b.seq[matchID]++
	return create(b, &store.MatchEvent{
		MatchID: matchID, Seq: b.seq[matchID], ElapsedSec: elapsedSec, Kind: kind, Side: side,
	}, opts)
}

// Goal は得点イベントを作る。assist は nil でよい。
func (b *Builder) Goal(
	matchID int64,
	side domain.Side,
	elapsedSec int,
	scorer, assist *store.Player,
	opts ...func(*store.MatchEvent),
) store.MatchEvent {
	b.tb.Helper()
	all := append([]func(*store.MatchEvent){func(ev *store.MatchEvent) {
		if scorer != nil {
			ev.ScorerID = &scorer.ID
		}
		if assist != nil {
			ev.AssistID = &assist.ID
		}
	}}, opts...)
	return b.Event(matchID, domain.EventGoal, &side, elapsedSec, all...)
}

// Stat は試合の出場記録を作る。
func (b *Builder) Stat(matchID, playerID int64, side domain.Side,
	opts ...func(*store.MatchPlayerStat),
) store.MatchPlayerStat {
	b.tb.Helper()
	return create(b, &store.MatchPlayerStat{MatchID: matchID, PlayerID: playerID, Side: side}, opts)
}

// Gallery はギャラリーを作る。
func (b *Builder) Gallery(tournamentID int64, opts ...func(*store.Gallery)) store.Gallery {
	b.tb.Helper()
	return create(b, &store.Gallery{TournamentID: tournamentID, Name: "Day1"}, opts)
}

// Photo はギャラリーの写真を作る。
func (b *Builder) Photo(galleryID int64, opts ...func(*store.GalleryPhoto)) store.GalleryPhoto {
	b.tb.Helper()
	return create(
		b,
		&store.GalleryPhoto{GalleryID: galleryID, ThumbURL: "/mini/1.jpg", FullURL: "/1.jpg"},
		opts,
	)
}
