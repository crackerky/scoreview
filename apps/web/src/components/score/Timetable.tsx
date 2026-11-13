import { cellKey, type TimetableDay } from '@/admin/timetable.utils'
import { courtName, winnerOf } from '@/live/live.utils'
import { scorePaths } from '@/score/score.paths'
import type { ScoreSlot } from '@/score/score.utils'
import { LiveDot } from './LiveDot'
import { MatchList } from './MatchList'

// 1 日の時間割(縦に時刻、横にコート)。会場に貼る紙の時間割と同じ並びにする。押すと試合の画面へ移る。

export function Timetable({
  tournamentId,
  day,
}: {
  tournamentId: number
  day: TimetableDay<ScoreSlot>
}) {
  return (
    <>
      {day.times.length > 0 ? (
        <div className="sc-tt-scroll">
          <table className="sc-tt">
            <thead>
              <tr>
                <th scope="col" className="sc-tt-time">
                  時刻
                </th>
                {day.fields.map((field) => (
                  <th key={field} scope="col">
                    {courtName(field)}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {day.times.map((time) => (
                <tr key={time}>
                  <th scope="row" className="sc-tt-time">
                    {time}
                  </th>
                  {day.fields.map((field) => (
                    <td key={field}>
                      {(day.cells.get(cellKey(time, field)) ?? []).map((slot) => (
                        <SlotCard key={slot.match.id} tournamentId={tournamentId} slot={slot} />
                      ))}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : null}
      {day.unplaced.length > 0 ? (
        <section className="sc-unplaced">
          <h3>時刻かコートが決まっていない試合</h3>
          <MatchList tournamentId={tournamentId} matches={day.unplaced.map((slot) => slot.match)} />
        </section>
      ) : null}
    </>
  )
}

function SlotCard({ tournamentId, slot }: { tournamentId: number; slot: ScoreSlot }) {
  const { match } = slot
  const winner = winnerOf(match)
  const line = (key: 'left' | 'right') => (
    <span
      className={`sc-slot-team${winner === key ? ' sc-won' : winner ? ' sc-lost' : ''}${
        match[key].pending ? ' sc-pending' : ''
      }`}
    >
      <span className="sc-team-bar" style={{ background: match[key].color }} />
      <span className="sc-side-name">{match[key].name}</span>
      <span className="sc-side-score">
        {match.status === 'sched' ? '' : (match[key].score ?? '')}
      </span>
    </span>
  )
  return (
    <a className={`sc-slot sc-${match.status}`} href={scorePaths.match(tournamentId, match.id)}>
      <span className="sc-slot-stage">
        {match.status === 'live' ? <LiveDot /> : null}
        {match.stage}
      </span>
      {line('left')}
      {line('right')}
    </a>
  )
}
