import { sideName } from '@/admin/labels.utils'
import { resultTypeLabels } from '@/admin/scoreOps.utils'
import type { AdminEvent, AdminPlayer, MatchDetail } from '@/api/admin.types'
import type { ResultType, Side } from '@/api/common.types'

// 紙のスコアシート(A4 縦 1 枚。docs/operations/operator-experience.md「紙を上回ることが前提」1)。
// 停電や通信の途絶に備えて、記録係の画面と同じ項目を紙でも書けるようにする。試合が進んでいれば
// 記録済みの得点経過と結果を印字し、控えとしても使える。

const sides: Side[] = ['left', 'right']
const sideLabel: Record<Side, string> = { left: '左', right: '右' }
// 得点経過の記入欄の行数。1 試合の得点は参照サイトの実データで多くても 30 前後(15 点先取の両チーム合計)
const TALLY_ROWS = 30
// 名簿の記入欄の最小行数(名簿が未登録のチームでも手書きできるように)
const MIN_ROSTER_ROWS = 12
const resultTypes: ResultType[] = [
  'normal',
  'forfeit',
  'double_forfeit',
  'no_contest',
  'abandoned',
  'unplayed',
]

function players(roster: AdminPlayer[]): AdminPlayer[] {
  return roster
    .filter((player) => player.role === null)
    .sort((a, b) => Number(a.number) - Number(b.number) || a.number.localeCompare(b.number))
}

function Box({ checked }: { checked: boolean }) {
  return <span className="pr-box">{checked ? '✓' : ''}</span>
}

function Roster({ detail, side }: { detail: MatchDetail; side: Side }) {
  const list = players(detail.rosters[side])
  const blanks = Math.max(0, MIN_ROSTER_ROWS - list.length)
  const played = new Set(
    detail.playerStats.filter((stat) => stat.side === side).map((stat) => stat.playerId),
  )
  return (
    <table className="pr-table pr-roster">
      <caption>
        {sideLabel[side]}: {sideName(detail.match[side])}
      </caption>
      <thead>
        <tr>
          <th className="num">背番号</th>
          <th>名前</th>
          <th className="mark">出場</th>
        </tr>
      </thead>
      <tbody>
        {list.map((player) => (
          <tr key={player.id}>
            <td className="num">{player.number}</td>
            <td>{player.name.ja}</td>
            <td className="mark">
              <Box checked={played.has(player.id)} />
            </td>
          </tr>
        ))}
        {Array.from({ length: blanks }, (_, index) => (
          // 手書き用の空行。行は増減しないので位置を key にしてよい
          // biome-ignore lint/suspicious/noArrayIndexKey: 並べ替えない空行
          <tr key={`blank-${index}`}>
            <td className="num" />
            <td />
            <td className="mark">
              <Box checked={false} />
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}

function TallyRows({ goals, from, to }: { goals: AdminEvent[]; from: number; to: number }) {
  return (
    <table className="pr-table pr-tally">
      <thead>
        <tr>
          <th className="num">点</th>
          <th className="mark">左</th>
          <th className="mark">右</th>
          <th>得点 #</th>
          <th>アシスト #</th>
          <th className="score">左-右</th>
        </tr>
      </thead>
      <tbody>
        {Array.from({ length: to - from }, (_, offset) => {
          const index = from + offset
          const goal = goals[index]
          return (
            <tr key={index}>
              <td className="num">{index + 1}</td>
              <td className="mark">{goal?.side === 'left' ? '○' : ''}</td>
              <td className="mark">{goal?.side === 'right' ? '○' : ''}</td>
              <td>{goal?.scorer?.number ?? ''}</td>
              <td>{goal?.assist?.number ?? ''}</td>
              <td className="score">{goal ? `${goal.leftScore}-${goal.rightScore}` : ''}</td>
            </tr>
          )
        })}
      </tbody>
    </table>
  )
}

export function ScoreSheet({ detail }: { detail: MatchDetail }) {
  const { match, context } = detail
  const goals = detail.events.filter((event) => event.kind === 'goal')
  const firstPull = detail.events.find((event) => event.kind === 'pull')
  const timeouts = (side: Side) =>
    detail.events.filter((event) => event.kind === 'timeout' && event.side === side).length
  const finished = match.status === 'finished'
  const half = TALLY_ROWS / 2
  return (
    <section className="pr-page pr-sheet">
      <header className="pr-head">
        <div>
          <p className="pr-small">
            {context.tournamentName} / {context.divisionName}
            {context.groupName ? ` / ${context.groupName}` : ''}
          </p>
          <h1>スコアシート</h1>
        </div>
        <dl className="pr-meta">
          <div>
            <dt>試合 CD</dt>
            <dd className="pr-code">{match.code || '　'}</dd>
          </div>
          <div>
            <dt>日付</dt>
            <dd>{match.date ?? '　'}</dd>
          </div>
          <div>
            <dt>時刻</dt>
            <dd>{match.time ?? '　'}</dd>
          </div>
          <div>
            <dt>コート</dt>
            <dd>{match.field ?? '　'}</dd>
          </div>
        </dl>
      </header>

      <div className="pr-two">
        {sides.map((side) => (
          <Roster key={side} detail={detail} side={side} />
        ))}
      </div>

      <p className="pr-line">
        最初のプル: <Box checked={firstPull?.side === 'left'} /> 左{' '}
        <Box checked={firstPull?.side === 'right'} /> 右
        <span className="pr-gap" />
        タイムアウト 左: <Box checked={timeouts('left') >= 1} />
        <Box checked={timeouts('left') >= 2} /> 右: <Box checked={timeouts('right') >= 1} />
        <Box checked={timeouts('right') >= 2} />
      </p>

      <h2>得点経過(得点したチームに ○、背番号を記入)</h2>
      <div className="pr-two">
        <TallyRows goals={goals} from={0} to={half} />
        <TallyRows goals={goals} from={half} to={TALLY_ROWS} />
      </div>

      <h2>結果</h2>
      <p className="pr-line">
        最終スコア 左 <span className="pr-fill">{finished ? (match.left.score ?? '') : ''}</span> -{' '}
        <span className="pr-fill">{finished ? (match.right.score ?? '') : ''}</span> 右
        <span className="pr-gap" />
        勝ち: <Box checked={finished && isWinner(detail, 'left')} /> 左{' '}
        <Box checked={finished && isWinner(detail, 'right')} /> 右{' '}
        <Box checked={finished && isDraw(detail)} /> 引き分け
      </p>
      <p className="pr-line">
        結果の種類:{' '}
        {resultTypes.map((type) => (
          <span key={type} className="pr-choice">
            <Box checked={finished && match.resultType === type} />
            {type === 'abandoned' ? '途中打ち切り' : resultTypeLabels[type]}
          </span>
        ))}
      </p>

      <div className="pr-signs">
        <div>記録係</div>
        <div>左チーム 主将</div>
        <div>右チーム 主将</div>
      </div>
      <p className="pr-small">
        記入したら本部へ提出してください。記録画面で入力した試合は、本部の画面に反映済みです。
      </p>
    </section>
  )
}

function isWinner(detail: MatchDetail, side: Side): boolean {
  const { match } = detail
  if (match.winnerSide !== null) {
    return match.winnerSide === side
  }
  const own = match[side].score
  const other = match[side === 'left' ? 'right' : 'left'].score
  return own !== null && other !== null && own > other
}

function isDraw(detail: MatchDetail): boolean {
  const { match } = detail
  return (
    match.resultType === 'normal' &&
    match.winnerSide === null &&
    match.left.score !== null &&
    match.left.score === match.right.score
  )
}
