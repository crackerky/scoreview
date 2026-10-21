package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/store"
)

// placeholderImage は架空データの画像 URL。実在の大会の素材を使わないため、寸法と文字だけの画像にする。
const placeholderImage = "https://placehold.co/%dx%d?text=%s"

// ErrNotEmpty は DB に大会が既にあることを表す(-replace なしで二重に投入しないため)。
var ErrNotEmpty = errors.New("database already has tournaments; use -replace to overwrite")

// seeder は 1 回の投入の状態。tx はトランザクション、events は試合ごとのイベントの seq。
type seeder struct {
	tx      *gorm.DB
	seq     map[int64]int
	summary []string
}

// seed は架空の大会一式を 1 つのトランザクションで投入し、作った大会の要約を返す。
// replace なら既存の大会(配下は ON DELETE CASCADE で消える)を先に削除する。
func seed(ctx context.Context, db *gorm.DB, replace bool) ([]string, error) {
	var summary []string
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&store.Tournament{}).Count(&count).Error; err != nil {
			return fmt.Errorf("count tournaments: %w", err)
		}
		if count > 0 {
			if !replace {
				return ErrNotEmpty
			}
			if err := tx.Where("id > ?", 0).Delete(&store.Tournament{}).Error; err != nil {
				return fmt.Errorf("delete tournaments: %w", err)
			}
		}
		s := &seeder{tx: tx, seq: map[int64]int{}}
		if err := s.all(); err != nil {
			return err
		}
		summary = s.summary
		return nil
	})
	if err != nil {
		return nil, err
	}
	return summary, nil
}

func (s *seeder) create(row any) error {
	if err := s.tx.Create(row).Error; err != nil {
		return fmt.Errorf("create %T: %w", row, err)
	}
	return nil
}

func ptr[T any](v T) *T { return &v }

func date(value string) *time.Time {
	day, err := time.Parse("2006-01-02", value)
	if err != nil {
		// 日付はこのファイル内の定数だけなので、失敗はプログラムの誤り
		panic(fmt.Sprintf("seed: bad date %q", value))
	}
	return &day
}

func image(width, height int, text string) string {
	return fmt.Sprintf(placeholderImage, width, height, text)
}

func (s *seeder) all() error {
	steps := []func() error{
		s.beachLeague, s.upcomingCup, s.nightSeries, s.unlisted, s.discgolf, s.noEnglishName,
	}
	for _, step := range steps {
		if err := step(); err != nil {
			return err
		}
	}
	return nil
}

func (s *seeder) tournament(row store.Tournament) (store.Tournament, error) {
	if row.TeamListLabel == "" {
		row.TeamListLabel = "teams"
	}
	if row.GroupLabel == "" {
		row.GroupLabel = "groups"
	}
	if err := s.create(&row); err != nil {
		return row, err
	}
	s.summary = append(
		s.summary,
		fmt.Sprintf("tournament %d %s (%s, listed=%t)", row.ID, row.NameJa, row.Status, row.Listed),
	)
	return row, nil
}

func (s *seeder) division(row store.Division) (store.Division, error) {
	if row.Kind == "" {
		row.Kind = domain.DivisionNone
	}
	if row.EntryKind == "" {
		row.EntryKind = domain.EntryTeam
	}
	if row.StatsKind == "" {
		row.StatsKind = domain.StatsUltimate
	}
	err := s.create(&row)
	return row, err
}

// teamNames は架空のチーム名(正式名、短縮名、英語名)。短縮名が正式名の切り詰めでない例を含める
// (clone-spec D 1.3)。
type teamName struct{ full, short, en string }

// teams は division にチームを作り、選手を perTeam 人ずつ登録する。staff が真なら最初のチームにスタッフを足す。
func (s *seeder) teams(
	division store.Division,
	names []teamName,
	bar domain.TeamBarStyle,
	perTeam int,
) (
	[]store.Team, map[int64][]store.Player, error,
) {
	teams := make([]store.Team, 0, len(names))
	players := map[int64][]store.Player{}
	for idx, name := range names {
		team := store.Team{
			TournamentID: division.TournamentID, DivisionID: division.ID,
			NameFullJa: name.full, NameShortJa: name.short, NameFullEn: name.en, NameShortEn: name.en,
			BarStyle: bar, SortOrder: idx,
		}
		if idx == 0 {
			team.CommentJa, team.CommentEn = "架空のチームです。", "A fictional team."
		}
		if err := s.create(&team); err != nil {
			return nil, nil, err
		}
		teams = append(teams, team)
		for num := 1; num <= perTeam; num++ {
			gender := "M"
			if num%3 == 0 {
				gender = "F"
			}
			player := store.Player{
				TeamID: team.ID, Number: fmt.Sprintf("%d", num*7%100),
				NameJa: fmt.Sprintf(
					"%s 選手%d",
					name.short,
					num,
				), NameEn: fmt.Sprintf("%s Player %d", name.en, num),
				Gender: ptr(gender),
			}
			if err := s.create(&player); err != nil {
				return nil, nil, err
			}
			players[team.ID] = append(players[team.ID], player)
		}
	}
	if len(teams) > 0 && perTeam > 0 {
		staff := store.Player{
			TeamID: teams[0].ID, NameJa: "架空 監督", NameEn: "Fictional Coach",
			RoleJa: ptr("マネージャ"), RoleEn: ptr("Manager"),
		}
		if err := s.create(&staff); err != nil {
			return nil, nil, err
		}
	}
	return teams, players, nil
}

// matchSpec は 1 試合の指定。left/right が nil なら未確定(placeholder を使う)。
type matchSpec struct {
	code        string
	group       *store.Group
	day, clock  string
	field       string
	left, right *store.Team
	leftHolder  *domain.Placeholder
	rightHolder *domain.Placeholder
	status      domain.MatchStatus
	scores      [2]int
	winner      *domain.Side
	noDetail    bool
	// result は結果の種類(空は通常)。noScore はスコアを記録しない結果(両者不戦敗、未消化)
	result  domain.ResultType
	noScore bool
}

// match は試合を作る。終了か試合中で詳細ありなら、スコアに合う得点経過と出場記録も作る。
func (s *seeder) match(
	division store.Division,
	spec matchSpec,
	players map[int64][]store.Player,
) (store.Match, error) {
	row := store.Match{
		DivisionID: division.ID, Code: spec.code, MatchDate: date(spec.day), StartTime: ptr(spec.clock),
		Field: ptr(spec.field), Status: spec.status, HasDetail: !spec.noDetail, WinnerSide: spec.winner,
	}
	if spec.group != nil {
		row.GroupID = &spec.group.ID
	}
	if spec.left != nil {
		row.LeftTeamID = &spec.left.ID
	} else if spec.leftHolder != nil {
		row.LeftPlaceholderCode, row.LeftPlaceholderOutcome = &spec.leftHolder.Code, &spec.leftHolder.Outcome
	}
	if spec.right != nil {
		row.RightTeamID = &spec.right.ID
	} else if spec.rightHolder != nil {
		row.RightPlaceholderCode, row.RightPlaceholderOutcome = &spec.rightHolder.Code, &spec.rightHolder.Outcome
	}
	// スコアと状態の変更は本来 store の試合更新用関数だけが行う(scoreview-design 8)。
	// シードでは行の作成時の初期値として入れる
	if spec.status != domain.MatchScheduled && !spec.noScore {
		row.LeftScore, row.RightScore = ptr(spec.scores[0]), ptr(spec.scores[1])
	}
	if spec.result != "" {
		row.ResultType = spec.result
	}
	if err := s.create(&row); err != nil {
		return row, err
	}
	if spec.status == domain.MatchScheduled || spec.noDetail || spec.noScore || spec.left == nil ||
		spec.right == nil {
		return row, nil
	}
	return row, s.progress(row, spec, players[spec.left.ID], players[spec.right.ID])
}

// progress はスコアに合う得点経過を決定的に作る。左が最初にプルし、得点を左右に交互に近い形で配る。
// 試合中ならプルと得点だけで、終了なら途中に双方のタイムアウトと最後に終了を入れる。
func (s *seeder) progress(match store.Match, spec matchSpec, left, right []store.Player) error {
	const minute = 60
	add := func(ev store.MatchEvent) error {
		s.seq[match.ID]++
		ev.MatchID, ev.Seq = match.ID, s.seq[match.ID]
		return s.create(&ev)
	}
	if err := add(store.MatchEvent{Kind: domain.EventPull, Side: ptr(domain.SideLeft)}); err != nil {
		return err
	}
	goalsLeft, goalsRight := spec.scores[0], spec.scores[1]
	total := goalsLeft + goalsRight
	madeLeft, madeRight := 0, 0
	for k := range total {
		// 残りの得点の多い側を優先し、同じなら交互にする(順番は決定的)
		side := domain.SideLeft
		if goalsLeft-madeLeft < goalsRight-madeRight ||
			(goalsLeft-madeLeft == goalsRight-madeRight && k%2 == 1) {
			side = domain.SideRight
		}
		roster := left
		if side == domain.SideRight {
			roster = right
			madeRight++
		} else {
			madeLeft++
		}
		ev := store.MatchEvent{Kind: domain.EventGoal, Side: ptr(side), ElapsedSec: (k + 1) * 2 * minute}
		if len(roster) > 0 {
			scorer := roster[(k*3+int(match.ID))%len(roster)]
			ev.ScorerID = &scorer.ID
			if len(roster) > 1 && k%5 != 4 {
				assist := roster[(k*3+int(match.ID)+1)%len(roster)]
				ev.AssistID = &assist.ID
			} else if k%5 == 4 {
				// アシストのない得点の一部をキャラハンゴールにする(試合 45310 の実例)
				ev.Callahan = true
			}
		}
		if err := add(ev); err != nil {
			return err
		}
		if k == total/2 {
			for _, timeoutSide := range []domain.Side{domain.SideLeft, domain.SideRight} {
				timeout := store.MatchEvent{
					Kind:       domain.EventTimeout,
					Side:       ptr(timeoutSide),
					ElapsedSec: ev.ElapsedSec,
				}
				if err := add(timeout); err != nil {
					return err
				}
			}
		}
	}
	if spec.status == domain.MatchFinished {
		if err := add(store.MatchEvent{Kind: domain.EventEnd, ElapsedSec: (total + 1) * 2 * minute}); err != nil {
			return err
		}
	}
	for sideIdx, roster := range [][]store.Player{left, right} {
		side := domain.SideLeft
		if sideIdx == 1 {
			side = domain.SideRight
		}
		for idx, player := range roster {
			stat := store.MatchPlayerStat{
				MatchID: match.ID, PlayerID: player.ID, Side: side,
				Blocks: (idx + int(match.ID)) % 3, Greatests: boolToInt(idx == 0 && match.ID%4 == 0),
			}
			if err := s.create(&stat); err != nil {
				return err
			}
		}
	}
	return nil
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func (s *seeder) group(row store.Group, teams []store.Team) (store.Group, error) {
	if err := s.create(&row); err != nil {
		return row, err
	}
	for idx, team := range teams {
		member := store.GroupTeam{GroupID: row.ID, TeamID: team.ID, SortOrder: idx}
		if err := s.create(&member); err != nil {
			return row, err
		}
	}
	return row, nil
}

// roundRobin はプールの総当たりを作る。results[k] は k 番目の組み合わせ(0-1, 0-2, ...)の状態とスコア。
func (s *seeder) roundRobin(
	division store.Division,
	group store.Group,
	teams []store.Team,
	prefix string,
	day string,
	results []matchSpec,
	players map[int64][]store.Player,
) error {
	k := 0
	for i := range teams {
		for j := i + 1; j < len(teams); j++ {
			spec := results[k]
			spec.code = fmt.Sprintf("%s%d", prefix, k+1)
			spec.group = &group
			spec.day = day
			spec.clock = fmt.Sprintf("%02d:00", 9+k/2)
			spec.field = fmt.Sprintf("%d", 1+k%2)
			spec.left, spec.right = &teams[i], &teams[j]
			if _, err := s.match(division, spec, players); err != nil {
				return err
			}
			k++
		}
	}
	return nil
}

func finished(left, right int) matchSpec {
	return matchSpec{status: domain.MatchFinished, scores: [2]int{left, right}}
}

// exceptional は通常以外の結果で終えた試合(閲覧ページの状況列、対戦表とブラケットの補足の確認用)。
// 不戦勝は大会の規定スコア 15-0 を勝者の側に入れる。打切のほかは試合ページ(得点経過)が無い。
func exceptional(result domain.ResultType, winner *domain.Side) matchSpec {
	spec := matchSpec{status: domain.MatchFinished, result: result, winner: winner}
	switch result {
	case domain.ResultForfeit:
		spec.scores, spec.noDetail = [2]int{15, 0}, true
		if winner != nil && *winner == domain.SideRight {
			spec.scores = [2]int{0, 15}
		}
	case domain.ResultDoubleForfeit, domain.ResultUnplayed, domain.ResultNoContest:
		// 試合をしていないので試合ページ(得点経過)も無い
		spec.noScore, spec.noDetail = true, true
	}
	return spec
}
