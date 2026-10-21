import { sideName } from '@/admin/labels.utils'
import { resultTypeLabels } from '@/admin/scoreOps.utils'
import { buildTimetable, cellKey, type TimetableMatch } from '@/admin/timetable.utils'

// 時間割(コート × 時刻の表。A4 横 1 日 1 枚)。会場に貼って、選手と記録係が次の試合のコートを探せる
// ようにする(docs/operations/operator-experience.md「紙を上回ることが前提」1)。

function MatchCell({ match }: { match: TimetableMatch }) {
  const score =
    match.status === 'finished'
      ? match.resultType === 'normal' || match.resultType === 'abandoned'
        ? `${match.left.score ?? ''}-${match.right.score ?? ''}`
        : resultTypeLabels[match.resultType]
      : ''
  return (
    <div className="pr-cell">
      <span className="pr-cell-code">{match.code}</span>
      <span className="pr-cell-div">{match.divisionName}</span>
      <span className="pr-cell-teams">
        {sideName(match.left)}
        <br />
        {sideName(match.right)}
      </span>
      {score ? <span className="pr-cell-score">{score}</span> : null}
    </div>
  )
}

export function Timetable({ title, matches }: { title: string; matches: TimetableMatch[] }) {
  const days = buildTimetable(matches)
  if (days.length === 0) {
    return <p>試合がありません。</p>
  }
  return (
    <>
      {days.map((day) => (
        <section key={day.date ?? 'undated'} className="pr-page pr-timetable">
          <h1>
            {title} 時間割 {day.date ?? '(日付未定)'}
          </h1>
          {day.times.length > 0 ? (
            <table className="pr-table pr-grid">
              <thead>
                <tr>
                  <th className="pr-time">時刻</th>
                  {day.fields.map((field) => (
                    <th key={field}>コート {field}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {day.times.map((time) => (
                  <tr key={time}>
                    <th className="pr-time">{time}</th>
                    {day.fields.map((field) => (
                      <td key={field}>
                        {(day.cells.get(cellKey(time, field)) ?? []).map((match) => (
                          <MatchCell key={match.id} match={match} />
                        ))}
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          ) : null}
          {day.unplaced.length > 0 ? (
            <>
              <h2>時刻かコートが決まっていない試合</h2>
              <ul className="pr-unplaced">
                {day.unplaced.map((match) => (
                  <li key={match.id}>
                    {match.code} {match.divisionName} {sideName(match.left)} -{' '}
                    {sideName(match.right)}
                    {match.time ? ` ${match.time}` : ''}
                    {match.field ? ` コート ${match.field}` : ''}
                  </li>
                ))}
              </ul>
            </>
          ) : null}
        </section>
      ))}
    </>
  )
}
