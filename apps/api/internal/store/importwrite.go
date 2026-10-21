package store

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// importBatchSize は 1 回の INSERT に入れる行数。PostgreSQL のプレースホルダ上限(65535)を
// 列数の多い matches(約 20 列)でも超えない値。
const importBatchSize = 500

// TournamentImport は取込ツールが 1 大会分をまとめて書くための行の集合。ID が 0 の行は自動採番し、
// それ以外は参照サイトの ID のまま INSERT する。自動採番の行を参照できるのはディビジョンだけで
// (子の DivisionID と TournamentID は書き込み時に埋める)、ほかの参照先はすべて ID を持つ行に限る。
type TournamentImport struct {
	Tournament Tournament
	Banners    []TournamentBanner
	Divisions  []DivisionImport
	// GroupTeams、Players、Events、Stats は ID を持つグループ、チーム、試合だけを参照する。
	GroupTeams []GroupTeam
	Players    []Player
	Events     []MatchEvent
	Stats      []MatchPlayerStat
	Galleries  []GalleryImport
}

// DivisionImport はディビジョンと、そのディビジョンに属するグループ、チーム、試合。
type DivisionImport struct {
	Division Division
	Groups   []Group
	Teams    []Team
	Matches  []Match
}

// GalleryImport はギャラリーと写真。写真の ID は自動採番する。
type GalleryImport struct {
	Gallery Gallery
	Photos  []GalleryPhoto
}

// ReplaceTournament は大会 imp.Tournament.ID の既存データを消してから imp を書く。1 つの
// トランザクションで行い、途中で失敗すれば何も変えない。既存データの削除は tournaments の行を
// 消して ON DELETE CASCADE に任せる。何度呼んでも同じ結果になる(冪等)。
//
// 試合のスコア、状態、勝者(matches.left_score / right_score / status / winner_side)は行の作成時の
// 初期値として入れる。試合更新用の関数を通さないので通知を積まない経路である(scoreview-design 8)。
func ReplaceTournament(ctx context.Context, db *gorm.DB, imp *TournamentImport) error {
	tid := imp.Tournament.ID
	if tid <= 0 {
		return fmt.Errorf("replace tournament: id must be positive, got %d", tid)
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where(sqlByID, tid).Delete(&Tournament{}).Error; err != nil {
			return fmt.Errorf("delete tournament %d: %w", tid, err)
		}
		if err := tx.Create(&imp.Tournament).Error; err != nil {
			return fmt.Errorf("create tournament %d: %w", tid, err)
		}
		for i := range imp.Banners {
			imp.Banners[i].TournamentID = tid
		}
		if err := createRows(tx, imp.Banners, "tournament_banners"); err != nil {
			return err
		}
		var matches []Match
		for i := range imp.Divisions {
			if err := createDivision(tx, tid, &imp.Divisions[i]); err != nil {
				return err
			}
			matches = append(matches, imp.Divisions[i].Matches...)
		}
		if err := createRows(tx, imp.GroupTeams, "group_teams"); err != nil {
			return err
		}
		if err := createRows(tx, imp.Players, "players"); err != nil {
			return err
		}
		matchID := func(m Match) int64 { return m.ID }
		if err := createWithOptionalID(tx, matches, matchID, "matches"); err != nil {
			return err
		}
		if err := createRows(tx, imp.Events, "match_events"); err != nil {
			return err
		}
		if err := createRows(tx, imp.Stats, "match_player_stats"); err != nil {
			return err
		}
		for i := range imp.Galleries {
			g := &imp.Galleries[i]
			g.Gallery.TournamentID = tid
			if err := tx.Create(&g.Gallery).Error; err != nil {
				return fmt.Errorf("create gallery %d: %w", g.Gallery.ID, err)
			}
			for k := range g.Photos {
				g.Photos[k].GalleryID = g.Gallery.ID
			}
			if err := createRows(tx, g.Photos, "gallery_photos"); err != nil {
				return err
			}
		}
		return nil
	})
}

// createDivision はディビジョンを作り、自動採番された ID を子のグループ、チーム、試合に入れてから
// グループとチームを作る。試合は全ディビジョンの分をまとめて後で作る(チームと選手が先に要るため)。
func createDivision(tx *gorm.DB, tournamentID int64, d *DivisionImport) error {
	d.Division.TournamentID = tournamentID
	if err := tx.Create(&d.Division).Error; err != nil {
		return fmt.Errorf("create division %q: %w", d.Division.NameJa, err)
	}
	for i := range d.Groups {
		d.Groups[i].DivisionID = d.Division.ID
	}
	for i := range d.Teams {
		d.Teams[i].DivisionID = d.Division.ID
		d.Teams[i].TournamentID = tournamentID
	}
	for i := range d.Matches {
		d.Matches[i].DivisionID = d.Division.ID
	}
	groupID := func(g Group) int64 { return g.ID }
	if err := createWithOptionalID(tx, d.Groups, groupID, "groups"); err != nil {
		return err
	}
	return createWithOptionalID(tx, d.Teams, func(t Team) int64 { return t.ID }, "teams")
}

// createWithOptionalID は ID を持つ行と持たない行を分けて作る。1 回の複数行 INSERT に両方を
// 混ぜると、GORM が id 列を入れるかどうかを行ごとに変えられないため。
func createWithOptionalID[T any](tx *gorm.DB, rows []T, idOf func(T) int64, table string) error {
	var explicit, generated []T
	for _, row := range rows {
		if idOf(row) != 0 {
			explicit = append(explicit, row)
		} else {
			generated = append(generated, row)
		}
	}
	if err := createRows(tx, explicit, table); err != nil {
		return err
	}
	return createRows(tx, generated, table)
}

func createRows[T any](tx *gorm.DB, rows []T, table string) error {
	if len(rows) == 0 {
		return nil
	}
	if err := tx.CreateInBatches(rows, importBatchSize).Error; err != nil {
		return fmt.Errorf("create %d rows of %s: %w", len(rows), table, err)
	}
	return nil
}
