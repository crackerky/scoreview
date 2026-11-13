import type { LiveMatch } from '@/live/live.types'
import { statusWord, winnerOf } from '@/live/live.utils'
import { scorePaths } from '@/score/score.paths'
import { slotText } from '@/score/score.utils'

// 試合の一覧(組の画面、時刻かコートが未定の試合)。押すと試合の画面へ移る。

export function MatchList({
  tournamentId,
  matches,
}: {
  tournamentId: number
  matches: LiveMatch[]
}) {
  if (matches.length === 0) {
    return <p className="sc-empty">試合はまだありません。</p>
  }
  return (
    <ul className="sc-matches">
      {matches.map((match) => (
        <li key={match.id}>
          <MatchLine tournamentId={tournamentId} match={match} />
        </li>
      ))}
    </ul>
  )
}

/** 試合 1 つ。勝った側を太く、負けた側を薄くする */
export function MatchLine({ tournamentId, match }: { tournamentId: number; match: LiveMatch }) {
  const winner = winnerOf(match)
  const side = (key: 'left' | 'right') => {
    const team = match[key]
    const tone = winner === null ? '' : winner === key ? ' sc-won' : ' sc-lost'
    return (
      <span className={`sc-side${tone}${team.pending ? ' sc-pending' : ''}`}>
        <span className="sc-team-bar" style={{ background: team.color }} />
        <span className="sc-side-name">{team.name}</span>
        <span className="sc-side-score">{match.status === 'sched' ? '' : (team.score ?? '')}</span>
      </span>
    )
  }
  return (
    <a className={`sc-match sc-${match.status}`} href={scorePaths.match(tournamentId, match.id)}>
      <span className="sc-match-meta">
        <span className="sc-status">{statusWord(match)}</span>
        <span>{[slotText(match), match.stage].filter(Boolean).join(' ・ ')}</span>
      </span>
      {side('left')}
      {side('right')}
    </a>
  )
}
