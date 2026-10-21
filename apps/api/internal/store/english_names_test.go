package store_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/store"
	"scoreview/api/internal/store/storetest"
	"scoreview/api/internal/testdb"
)

// 英語名の補い(通し確認: 名簿の貼り付けが日本語名しか入れず、英語の閲覧ページでチーム名が空になった)。

func TestCreateFillsEnglishNames(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	b := storetest.New(t, db)
	division := b.Division(b.Tournament().ID)

	team, err := store.CreateTeam(ctx, db, division.ID, store.TeamFields{
		FullName:  domain.I18nText{Ja: "架空フライヤーズ東京"},
		ShortName: domain.I18nText{Ja: "フライヤーズ"},
		BarStyle:  "default",
	})
	require.NoError(t, err)
	assert.Equal(t, domain.I18nText{Ja: "架空フライヤーズ東京", En: "架空フライヤーズ東京"}, team.FullName)
	assert.Equal(t, domain.I18nText{Ja: "フライヤーズ", En: "フライヤーズ"}, team.ShortName)

	// 英語の正式名だけ指定すれば、英語の短縮名はそれにする(名簿の英語名の列)
	english, err := store.CreateTeam(ctx, db, division.ID, store.TeamFields{
		FullName:  domain.I18nText{Ja: "架空ベイ", En: "Fictional Bay"},
		ShortName: domain.I18nText{Ja: "ベイ"},
		BarStyle:  "default",
	})
	require.NoError(t, err)
	assert.Equal(t, "Fictional Bay", english.FullName.En)
	assert.Equal(t, "Fictional Bay", english.ShortName.En)

	player, err := store.CreatePlayer(ctx, db, team.ID, store.PlayerFields{
		Number: "7", Name: domain.I18nText{Ja: "架空 一郎"},
	})
	require.NoError(t, err)
	assert.Equal(t, "架空 一郎", player.Name.En)
	named, err := store.CreatePlayer(ctx, db, team.ID, store.PlayerFields{
		Number: "8", Name: domain.I18nText{Ja: "架空 二郎", En: "Jiro KAKU"},
	})
	require.NoError(t, err)
	assert.Equal(t, "Jiro KAKU", named.Name.En)

	// 更新では補わない(英語名を空にしたいときに空のまま保存できる)
	cleared := team.TeamFields
	cleared.FullName.En = ""
	updated, err := store.UpdateTeam(ctx, db, domain.Actor{}, team.ID, cleared)
	require.NoError(t, err)
	assert.Empty(t, updated.FullName.En)
}

func TestGroupEnglishNameFollowsName(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	b := storetest.New(t, db)
	division := b.Division(b.Tournament().ID)

	englishOf := func(id int64) string {
		var row store.Group
		require.NoError(t, db.Where("id = ?", id).First(&row).Error)
		return row.NameEn
	}
	group, err := store.CreateGroup(ctx, db, division.ID, store.GroupFields{
		Name: "予選 A", Kind: domain.GroupPool, Linked: true, Published: true,
	})
	require.NoError(t, err)
	assert.Equal(t, "予選 A", englishOf(group.ID))

	_, err = store.UpdateGroup(ctx, db, group.ID, store.GroupFields{
		Name: "予選 A 組", Kind: domain.GroupPool, Linked: true, Published: true,
	})
	require.NoError(t, err)
	assert.Equal(t, "予選 A 組", englishOf(group.ID), "日本語名に合わせた英語名は名前の変更に付いていく")

	// 取込ツールの組のように英語名が別の値なら、名前を変えても英語名は変えない
	imported := b.Group(division.ID, domain.GroupPool, func(g *store.Group) { g.NameEn = "---" })
	_, err = store.UpdateGroup(ctx, db, imported.ID, store.GroupFields{
		Name: "Pool B", Kind: domain.GroupPool, Linked: true, Published: true,
	})
	require.NoError(t, err)
	assert.Equal(t, "---", englishOf(imported.ID))
}

func TestCopyTournamentFillsEnglishNames(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	b := storetest.New(t, db)
	source := b.Tournament()
	division := b.Division(source.ID)
	team := b.Team(division, func(row *store.Team) { row.NameFullEn, row.NameShortEn = "", "" })
	b.Player(team.ID, func(row *store.Player) { row.NameJa, row.NameEn = "架空 三子", "" })

	record, err := store.GetAdminTournament(ctx, db, source.ID)
	require.NoError(t, err)
	fields := record.TournamentFields
	date := "2027-10-18"
	fields.StartDate = &date
	copied, err := store.CopyTournament(ctx, db, source.ID, fields, true)
	require.NoError(t, err)

	var teams []store.Team
	require.NoError(t, db.Where("tournament_id = ?", copied.ID).Find(&teams).Error)
	require.Len(t, teams, 1)
	assert.Equal(t, team.NameFullJa, teams[0].NameFullEn)
	assert.Equal(t, team.NameShortJa, teams[0].NameShortEn)
	var players []store.Player
	require.NoError(t, db.Where("team_id = ?", teams[0].ID).Find(&players).Error)
	require.Len(t, players, 1)
	assert.Equal(t, "架空 三子", players[0].NameEn)
}

func TestFillDivisionEnglishNames(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	b := storetest.New(t, db)
	tournament := b.Tournament()
	division := b.Division(tournament.ID)
	other := b.Division(tournament.ID)
	blank := b.Team(division, func(row *store.Team) {
		row.NameFullJa, row.NameShortJa, row.NameFullEn, row.NameShortEn = "架空ウェーブ", "ウェーブ", "", ""
	})
	shortOnly := b.Team(division, func(row *store.Team) {
		row.NameFullEn, row.NameShortEn = "Fictional Storm", ""
	})
	named := b.Team(division) // 英語名あり(変えない)
	untouched := b.Team(other, func(row *store.Team) { row.NameFullEn, row.NameShortEn = "", "" })
	b.Player(blank.ID, func(row *store.Player) { row.NameJa, row.NameEn = "架空 四郎", "" })
	b.Player(named.ID) // 英語名あり
	b.Group(division.ID, domain.GroupPool, func(row *store.Group) { row.Name, row.NameEn = "A", "" })
	b.Group(division.ID, domain.GroupPool, func(row *store.Group) { row.NameEn = "---" })

	filled, err := store.FillDivisionEnglishNames(ctx, db, division.ID)
	require.NoError(t, err)
	assert.Equal(t, store.FilledEnglishNames{Teams: 2, Players: 1, Groups: 1}, filled)

	read := func(id int64) store.Team {
		var row store.Team
		require.NoError(t, db.Where("id = ?", id).First(&row).Error)
		return row
	}
	assert.Equal(t, "架空ウェーブ", read(blank.ID).NameFullEn)
	assert.Equal(t, "ウェーブ", read(blank.ID).NameShortEn)
	assert.Equal(t, "Fictional Storm", read(shortOnly.ID).NameShortEn)
	assert.Equal(t, "Test Team", read(named.ID).NameFullEn)
	assert.Empty(t, read(untouched.ID).NameFullEn, "他の部門は変えない")
	var groups []store.Group
	require.NoError(t, db.Where("division_id = ?", division.ID).Order("id").Find(&groups).Error)
	assert.Equal(t, []string{"A", "---"}, []string{groups[0].NameEn, groups[1].NameEn})

	again, err := store.FillDivisionEnglishNames(ctx, db, division.ID)
	require.NoError(t, err)
	assert.Equal(t, store.FilledEnglishNames{}, again, "2 回目は何も変えない")

	_, err = store.FillDivisionEnglishNames(ctx, db, 999999)
	require.ErrorIs(t, err, domain.ErrNotFound)
}

// TestImportKeepsEmptyEnglishNames は取込ツールの書き込み(ReplaceTournament)が英語名を補わない
// ことを確かめる(参照サイトの再現。英語名が空のチームは英語版で空表示する)。
func TestImportKeepsEmptyEnglishNames(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	imp := &store.TournamentImport{
		Tournament: store.Tournament{
			ID: 900001, NameJa: "取込の架空大会", Status: domain.TournamentFinished, Listed: true,
			TeamListLabel: "teams", GroupLabel: "groups", ShowSchedule: true,
		},
		Divisions: []store.DivisionImport{{
			Division: store.Division{
				NameJa: "オープン", Kind: domain.DivisionMixed, ScheduleEnabled: true,
				EntryKind: domain.EntryTeam, StatsKind: domain.StatsUltimate,
			},
			Groups: []store.Group{{ID: 900101, Name: "予選 A", Kind: domain.GroupPool, Linked: true}},
			Teams: []store.Team{{
				ID: 900201, NameFullJa: "取込チーム", NameShortJa: "取込", BarStyle: "default",
			}},
		}},
		Players: []store.Player{{ID: 900301, TeamID: 900201, NameJa: "取込 選手"}},
	}
	require.NoError(t, store.ReplaceTournament(ctx, db, imp))

	var team store.Team
	require.NoError(t, db.Where("id = ?", 900201).First(&team).Error)
	assert.Empty(t, team.NameFullEn)
	assert.Empty(t, team.NameShortEn)
	var player store.Player
	require.NoError(t, db.Where("id = ?", 900301).First(&player).Error)
	assert.Empty(t, player.NameEn)
	var group store.Group
	require.NoError(t, db.Where("id = ?", 900101).First(&group).Error)
	assert.Empty(t, group.NameEn)
}
