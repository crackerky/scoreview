// Package viewing は公開のスコアサイト(大会の掲示板)の使い方を持つ。
//
// 依存の向きは httpapi → viewing ← store。httpapi は Reader だけを知り、記録の読み方(Source)は
// store が GORM で実装する。このパッケージは、読んだ記録から掲示板の形を組み立てる決まり(組の順位、
// 試合と組の結び付き、日程の並び)だけを持ち、HTTP も DB も知らない。そのため決まりは DB 無しで
// 確かめられ、読み方を替えても(使い回し、別の保存先)ハンドラと決まりは変わらない。
package viewing

import (
	"context"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/format"
)

// Reader は大会の掲示板を返す。httpapi が使う唯一の口で、Service と Cache が実装する。
type Reader interface {
	// Board は大会の掲示板を返す。大会が無ければ domain.NotFound。返す Board は呼び出し側で
	// 共有されうるので、変更しないこと。
	Board(ctx context.Context, tournamentID int64) (Board, error)
}

// Source は掲示板に要る公開の記録を読む。store が実装する。
type Source interface {
	// Snapshot は大会 1 つ分の公開の記録をまとめて読む。問い合わせの数は試合、組、チームの数に
	// よらないこと(観客が増えるとこの読み込みが繰り返されるため)。大会が無ければ domain.NotFound。
	Snapshot(ctx context.Context, tournamentID int64) (Snapshot, error)
}

// Snapshot は Source が読んだ記録。
type Snapshot struct {
	Tournament Tournament
	// Divisions は部門の並び順
	Divisions []Division
	// Groups は公開して案内する組(published かつ linked)。部門の中の並び順
	Groups []Group
	// Matches は部門の全試合。並びは問わない
	Matches []Match
	// Brackets は大会の形から作ったトーナメントの形(公開した組だけ)
	Brackets []Bracket
	Tables   TablesConfig
}

// Tournament は大会。
type Tournament struct {
	ID   int64
	Name domain.I18nText
	// Banner は大会の最初のバナー(協賛の広告)。無ければ nil
	Banner *Banner
}

// Banner は協賛の広告。LinkURL が nil なら押しても移らない。
type Banner struct {
	ImageURL string
	LinkURL  *string
}

// Division は部門。
type Division struct {
	ID   int64
	Name domain.I18nText
}

// Group は組。Entries は総当たりの組(Kind が pool)の登録順のチームで、それ以外は空。
type Group struct {
	ID         int64
	DivisionID int64
	Name       domain.I18nText
	Kind       domain.GroupKind
	// Advance は優勝を争うグループへ進む上位の数(大会の形から。0 は無し)
	Advance int
	Entries []Entry
}

// Entry は総当たりの組の 1 チーム。RankOverride は運営が決めた順位(0 は順位を出さない)。
type Entry struct {
	Team         domain.TeamRef
	RankOverride *int
}

// Match は試合。GroupID は Snapshot.Groups の組に入っている試合だけ入る(案内しない組へつながない)。
type Match struct {
	domain.ScheduleRow
	GroupID *int64
}

// Bracket はトーナメントの形。試合は部門と試合番号(Code)で Match と結ぶ。
type Bracket struct {
	DivisionID int64
	GroupID    int64
	// Name は運営の言葉の名前(「1〜4 位を決めるトーナメント」)
	Name string
	// Matches は試合番号順
	Matches []BracketMatch
}

// BracketMatch はトーナメントの 1 試合の形。WinnerPlace と LoserPlace はグループ内の順位(0 は次の
// 試合へ進む)。
type BracketMatch struct {
	Code        string
	Round       int
	WinnerPlace int
	LoserPlace  int
	Left        BracketSide
	Right       BracketSide
}

// BracketSide は試合の片側の出どころ。同じグループの前の試合の勝者、敗者なら From、組の順位なら
// Seed(「予選 A 組 1位」)。
type BracketSide struct {
	From *format.MatchRef
	Seed string
}

// TablesConfig は管理者が決めた星取表とトーナメント表の見せ方(docs/api.md「観戦アプリの表」)。
// Marks は maru / wdl / kanji、Order は entry / rank、Live は score / blank。
type TablesConfig struct {
	Shown        bool
	PoolShown    bool
	BracketShown bool
	Columns      TableColumns
	Marks        string
	Order        string
	Live         string
	Highlight    bool
}

// TableColumns は星取表の右に並べる列。
type TableColumns struct {
	WDL    bool
	Goals  bool
	Diff   bool
	Points bool
	Rank   bool
}
