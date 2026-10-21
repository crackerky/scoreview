package importer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"scoreview/api/internal/importer/parse"
)

// DivisionKindFromSchedule は日程のディビジョン列の種別クラス(MC_MEN / MC_WOM / MC_MIX、
// なければ none)から divisions.kind を推定する。clone-spec B 6: クラスは名前ではなく
// ディビジョンの区分属性で決まる。行がないか、行ごとに種別が食い違うときは ok=false。
func DivisionKindFromSchedule(s *parse.Schedule) (kind string, ok bool) {
	if s == nil {
		return "", false
	}
	for _, r := range s.Rows {
		switch {
		case kind == "":
			kind = r.DivisionKind
		case kind != r.DivisionKind:
			return "", false
		}
	}
	return kind, kind != ""
}

// DivisionKindFromTeamList はチーム一覧の色帯(FTitle_*)から divisions.kind を推定する。
// FTitle_Mix → mixed、FTitle_Women → women。FTitle(青)はメンとオープン等の両方に使われ
// (clone-spec A 5)、メンか none かを決められないので ok=false。FTitle_Univ などの特定チームの
// 色は数えない。チームがないか、色が食い違うときも ok=false。
func DivisionKindFromTeamList(d parse.TeamListDivision) (kind string, ok bool) {
	for _, e := range d.Entries {
		var k string
		switch e.BarStyle {
		case "mixed":
			k = "mixed"
		case "women":
			k = "women"
		case "default":
			return "", false
		default:
			continue
		}
		switch {
		case kind == "":
			kind = k
		case kind != k:
			return "", false
		}
	}
	return kind, kind != ""
}

// JSONSink は Bundle を JSON で書く Sink。-dry-run の出力に使う。
type JSONSink struct {
	W io.Writer
}

// Write は Bundle を字下げした JSON で書く。HTML(自由記述)を読めるよう & < > をエスケープしない。
func (s JSONSink) Write(_ context.Context, b *Bundle) error {
	enc := json.NewEncoder(s.W)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(b); err != nil {
		return fmt.Errorf("encode bundle: %w", err)
	}
	return nil
}
