import { notFound } from 'next/navigation'
import { parseMatchId } from '@/admin/relay.utils'
import { statusWord, winnerOf } from '@/live/live.utils'
import { loadLiveMatch } from '@/live/liveData'
import { requireScoreBoard, scoreRevalidateSeconds } from '@/score/score.load'
import { scorePaths } from '@/score/score.paths'
import { slotText } from '@/score/score.utils'

// 試合の画面。何の試合か、点、勝敗を大きく出す。得点の流れはたたんでおく(見るのは分析する人くらい)。

/** ビルドでは描かない(大会の外枠と同じ。最初に開かれたときに描いて使い回す) */
export function generateStaticParams(): Array<{ matchId: string }> {
  return []
}

export default async function MatchPage({
  params,
}: {
  params: Promise<{ id: string; matchId: string }>
}) {
  const { id: rawId, matchId: rawMatch } = await params
  const { board, data } = await requireScoreBoard(rawId)
  const matchId = parseMatchId(rawMatch)
  const match = data.matches.find((item) => item.id === matchId)
  if (!match) {
    notFound()
  }
  const id = board.tournament.id
  const detail = await loadLiveMatch(id, match.id, { revalidateSeconds: scoreRevalidateSeconds })
  const winner = winnerOf(match)
  const back =
    match.groupId !== null && match.pool
      ? { href: scorePaths.pool(id, match.groupId), label: match.stage }
      : { href: scorePaths.schedule(id), label: 'タイムスケジュール' }
  const side = (key: 'left' | 'right') => (
    <div className={`sc-board-side${winner === key ? ' sc-won' : winner ? ' sc-lost' : ''}`}>
      <span className="sc-team-bar" style={{ background: match[key].color }} />
      <span className="sc-board-name">{match[key].name}</span>
      {winner === key ? <span className="sc-win-chip">勝ち</span> : null}
    </div>
  )
  return (
    <>
      <title>{`${match.left.name} 対 ${match.right.name} - ${board.tournament.name.ja}`}</title>
      <a className="sc-back" href={back.href}>
        <span aria-hidden="true">‹ </span>
        {back.label}
      </a>
      <article className={`sc-board sc-${match.status}`}>
        <p className="sc-board-stage">
          {[match.divisionName, match.stage].filter(Boolean).join(' ・ ')}
        </p>
        <div className="sc-board-row">
          {side('left')}
          <div className="sc-board-score">
            {match.status === 'sched' ? (
              <span className="sc-board-vs">対</span>
            ) : (
              <>
                <span>{match.left.score ?? 0}</span>
                <span className="sc-board-dash">-</span>
                <span>{match.right.score ?? 0}</span>
              </>
            )}
          </div>
          {side('right')}
        </div>
        <p className="sc-board-meta">
          <span className="sc-status">{statusWord(match)}</span>
          {slotText(match) !== '' ? <span>{slotText(match)}</span> : null}
        </p>
      </article>
      {detail && detail.events.length > 0 ? (
        <details className="sc-flow">
          <summary>得点の流れ({detail.events.length} 点)</summary>
          <ol>
            {detail.events.map((event, index) => (
              // biome-ignore lint/suspicious/noArrayIndexKey: 得点の順番が識別子
              <li key={index} className={`sc-flow-${event.side}`}>
                <span className="sc-flow-score">
                  {event.leftScore}-{event.rightScore}
                </span>
                <span className="sc-flow-team">{match[event.side].name}</span>
                <span className="sc-flow-who">
                  {[event.scorer?.name, event.assist ? `アシスト ${event.assist.name}` : '']
                    .filter(Boolean)
                    .join(' ・ ')}
                  {event.callahan ? ' (キャラハン)' : ''}
                </span>
                <span className="sc-flow-time">{event.elapsed}</span>
              </li>
            ))}
          </ol>
        </details>
      ) : null}
    </>
  )
}
