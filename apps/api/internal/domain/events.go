package domain

// Event は試合の得点経過の 1 行。Side は goal なら得点した側、timeout なら取った側、
// pull ならプルした側。end は nil。
type Event struct {
	ElapsedSec   int
	Kind         EventKind
	Side         *Side
	FromTurnover bool
	ScorerID     *int64
	AssistID     *int64
}

// Score はある時点の累計スコア。
type Score struct {
	Left  int
	Right int
}

// RunningScores は各イベント時点の累計スコアを返す。events は時系列の昇順(経過時間、同時刻は入力順)で
// 渡すこと。戻り値は events と同じ長さと順。各時点のスコアは保存せず、イベントの順番から計算する
// (docs/architecture.md データモデル)。
func RunningScores(events []Event) []Score {
	out := make([]Score, len(events))
	var cur Score
	for i, ev := range events {
		if ev.Kind == EventGoal && ev.Side != nil {
			if *ev.Side == SideLeft {
				cur.Left++
			} else {
				cur.Right++
			}
		}
		out[i] = cur
	}
	return out
}

// SideStats は試合ページの統計の片側(clone-spec D 2.5)。
type SideStats struct {
	Goals        int
	FromOffense  int
	FromDefense  int
	FromTurnover int
	Timeouts     int
}

// MatchStats は試合ページの統計の左右。
type MatchStats struct {
	Left  SideStats
	Right SideStats
}

func (m *MatchStats) side(s Side) *SideStats {
	if s == SideLeft {
		return &m.Left
	}
	return &m.Right
}

// ComputeMatchStats は得点経過から統計を計算する。events は時系列の昇順で渡すこと。
//
// オフェンス開始かどうかはプルの向きで決まる。プルした側がそのポイントをディフェンスで始め、
// 受けた側がオフェンスで始める。最初のプルの side がディフェンス開始で、得点の後は得点した側が
// 次にプルする(= 次のポイントはディフェンス開始)。途中に pull イベントがあれば(後半開始など)
// その向きで置き換える。最初のプルより前の得点は開始側が分からないので、オフェンスにも
// ディフェンスにも数えない(推測)。ターンオーバーからの得点はプルから計算できないので
// イベントの FromTurnover を数える。
func ComputeMatchStats(events []Event) MatchStats {
	var stats MatchStats
	// defenseStart は現在のポイントをディフェンスで始めた側(= プルした側)。nil は不明
	var defenseStart *Side
	for _, ev := range events {
		if ev.Side == nil {
			continue
		}
		evSide := *ev.Side
		switch ev.Kind {
		case EventPull:
			defenseStart = &evSide
		case EventTimeout:
			stats.side(evSide).Timeouts++
		case EventGoal:
			st := stats.side(evSide)
			st.Goals++
			if ev.FromTurnover {
				st.FromTurnover++
			}
			if defenseStart != nil {
				if *defenseStart == evSide {
					st.FromDefense++
				} else {
					st.FromOffense++
				}
			}
			// 得点した側が次にプルする
			defenseStart = &evSide
		case EventEnd:
		}
	}
	return stats
}
