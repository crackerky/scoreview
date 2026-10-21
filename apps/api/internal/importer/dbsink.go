package importer

import (
	"context"
	"fmt"
	"io"

	"gorm.io/gorm"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/store"
)

// DBSink は Bundle を store を通して DB に書く Sink。1 大会を 1 トランザクションで書き、同じ大会 ID の
// 既存データは消してから入れる(再取込しても重複しない)。
//
// 試合のスコアと状態は行の作成時の初期値として入れる。試合更新用の関数を通さないので通知を積まない
// 経路である(scoreview-design 8)。
type DBSink struct {
	DB *gorm.DB
	// Log は推測で決めた箇所と取りこぼしの警告、書いた件数を書く先。nil なら書かない。
	Log io.Writer
}

// Write は b を DB に書く。最初に自動採番の範囲を GeneratedIDFloor 以上に進め、参照サイトの ID と
// 自動採番の ID が衝突しないようにする。
func (s DBSink) Write(ctx context.Context, b *Bundle) error {
	plan, err := BuildPlan(b)
	if err != nil {
		return err
	}
	if err := store.ReserveGeneratedIDRange(ctx, s.DB); err != nil {
		return err
	}
	if err := store.ReplaceTournament(ctx, s.DB, plan.Import); err != nil {
		return fmt.Errorf("tournament %d: %w", b.TournamentID, err)
	}
	if s.Log != nil {
		s.logPlan(b.TournamentID, plan)
	}
	return nil
}

func (s DBSink) logPlan(tid int64, plan *Plan) {
	// 書き込みは済んでいる。ログの書き込み失敗で取込の結果は変わらないので戻り値を見ない
	_, _ = fmt.Fprintf(s.Log, "importer: tournament %d written: %s\n", tid, plan.Summary())
	for _, w := range plan.Warnings {
		_, _ = fmt.Fprintf(s.Log, "importer: tournament %d warning: %s\n", tid, w)
	}
}

// Summary は書く行の件数を 1 行で返す。
func (p *Plan) Summary() string {
	imp := p.Import
	var groups, teams, matches, finished int
	for _, d := range imp.Divisions {
		groups += len(d.Groups)
		teams += len(d.Teams)
		matches += len(d.Matches)
		for _, m := range d.Matches {
			if m.Status == domain.MatchFinished {
				finished++
			}
		}
	}
	photos := 0
	for _, g := range imp.Galleries {
		photos += len(g.Photos)
	}
	return fmt.Sprintf("divisions=%d groups=%d teams=%d players=%d matches=%d (finished %d) "+
		"events=%d player_stats=%d galleries=%d photos=%d banners=%d",
		len(imp.Divisions), groups, teams, len(imp.Players), matches, finished,
		len(imp.Events), len(imp.Stats), len(imp.Galleries), photos, len(imp.Banners))
}
