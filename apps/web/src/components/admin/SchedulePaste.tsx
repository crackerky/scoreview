'use client'

import { useRouter } from 'next/navigation'
import { type ChangeEvent, useState, useTransition } from 'react'
import type { BuildTeam } from '@/admin/build.utils'
import {
  type PasteMatch,
  parseSchedule,
  type SchedulePaste as Result,
} from '@/admin/schedulePaste.utils'
import type { Violation } from '@/api/format.types'
import { importScheduleAction } from '@/app/admin/_actions/format'

// 時間割の貼り付け。Excel で決めた日付、時刻、コートの表を貼るか CSV を選び、部門の試合に当てはめた結果
// (試合ごとの今 → 新しい時刻とコート)を確かめてから書く。当てはまらない行や重なりがあれば書けない。
// 書いた後は、api が時間割を確かめ直した結果(休みが足りない、など)を出す。

type Match = PasteMatch & { date: string | null; time: string | null; field: string | null }

type Props = {
  divisionId: number
  matches: Match[]
  teams: BuildTeam[]
  /** 日付の列が無い表の日付(初期値) */
  defaultDate: string
}

const placeholder = '時刻\tコート1\tコート2\n8:30\tA1\tA2\n9:20\t熱海 - 伊東\t沼津 - 三島'

export function SchedulePaste({ divisionId, matches, teams, defaultDate }: Props) {
  const router = useRouter()
  const [text, setText] = useState('')
  const [date, setDate] = useState(defaultDate)
  const [result, setResult] = useState<Result | null>(null)
  const [done, setDone] = useState<{ updated: number; violations: Violation[] | null } | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [pending, startTransition] = useTransition()
  const byId = new Map(matches.map((match) => [match.id, match]))
  const teamName = new Map(teams.map((team) => [team.id, team.shortName]))
  const side = (id: number | null) => (id === null ? '未定' : (teamName.get(id) ?? '未定'))

  function read(source: string) {
    setText(source)
    setDone(null)
    setError(null)
    setResult(parseSchedule(source, date, matches, teams))
  }

  function readFile(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0]
    if (file) {
      file.text().then(read, () => setError('ファイルを読めませんでした'))
    }
  }

  function write() {
    if (result === null) {
      return
    }
    const items = result.rows.map((row) => ({
      matchId: row.matchId,
      date: row.date,
      time: row.time,
      field: row.field,
    }))
    startTransition(async () => {
      const saved = await importScheduleAction(divisionId, items)
      if (saved.ok) {
        setDone(saved.data)
        setResult(null)
        router.refresh()
      } else {
        setError(saved.error)
      }
    })
  }

  const writable = result !== null && result.problems.length === 0 && result.rows.length > 0
  return (
    <div className="adm-stack">
      <p className="adm-muted">
        Excel の表をそのままコピーして貼れます。1 行目にコートの名前を並べ、1
        列目に時刻、枠に試合番号か「熱海 -
        伊東」を書いた形(日付は下で選ぶ)と、「日付、時刻、コート、試合番号(またはチーム、チーム)」の見出しを付けて
        1 行 1 試合にした形の、どちらでも読みます。表に無い試合は今の時刻のままです。
      </p>
      <div className="bld-split">
        <div className="adm-stack">
          <label className="adm-field">
            <span>日付(日付の列が無い表のとき)</span>
            <input type="date" value={date} onChange={(event) => setDate(event.target.value)} />
          </label>
          <label className="adm-field">
            <span>時間割の表</span>
            <textarea
              rows={10}
              value={text}
              placeholder={placeholder}
              onChange={(event) => setText(event.target.value)}
            />
          </label>
          <div className="adm-actions">
            <button type="button" onClick={() => read(text)} disabled={text.trim() === ''}>
              読み込む
            </button>
            <label className="bld-file">
              CSV を選ぶ
              <input type="file" accept=".csv,.tsv,.txt,text/csv" onChange={readFile} />
            </label>
          </div>
        </div>
        <div className="adm-stack" aria-live="polite">
          {error ? <p className="adm-notice error">{error}</p> : null}
          {done ? (
            <div className={done.violations?.length ? 'adm-notice error' : 'adm-notice ok'}>
              {done.updated} 試合の時刻とコートを書きました。
              {done.violations && done.violations.length > 0 ? (
                <ul>
                  {done.violations.map((violation) => (
                    <li key={`${violation.kind}-${violation.codes.join(',')}`}>
                      {violation.detail}
                    </li>
                  ))}
                </ul>
              ) : null}
            </div>
          ) : null}
          {result === null ? (
            done ? null : (
              <p className="adm-muted">読み込むと、試合ごとの今と新しい時刻がここに出ます。</p>
            )
          ) : (
            <>
              {result.problems.length > 0 ? (
                <div className="adm-notice error">
                  直してから読み込み直してください({result.problems.length} 件)
                  <ul>
                    {result.problems.map((problem) => (
                      <li key={problem}>{problem}</li>
                    ))}
                  </ul>
                </div>
              ) : null}
              <div className="adm-scroll">
                <table className="adm-table">
                  <thead>
                    <tr>
                      <th>試合</th>
                      <th>対戦</th>
                      <th>今</th>
                      <th>新しい</th>
                    </tr>
                  </thead>
                  <tbody>
                    {result.rows.map((row) => {
                      const match = byId.get(row.matchId)
                      return (
                        <tr key={row.matchId}>
                          <td>{row.code}</td>
                          <td>
                            {side(match?.leftTeamId ?? null)} - {side(match?.rightTeamId ?? null)}
                          </td>
                          <td className="adm-muted">
                            {match?.date ? match.date.slice(5) : '—'} {match?.time ?? ''}{' '}
                            {match?.field ? `コート ${match.field}` : ''}
                          </td>
                          <td>
                            {row.date.slice(5)} {row.time} コート {row.field}
                          </td>
                        </tr>
                      )
                    })}
                  </tbody>
                </table>
              </div>
              {result.untouched.length > 0 ? (
                <p className="adm-muted">
                  表に無い試合(今の時刻のまま): {result.untouched.join('、')}
                </p>
              ) : null}
              <div className="adm-actions">
                <button type="button" onClick={write} disabled={!writable || pending}>
                  この時間割にする({result.rows.length} 試合)
                </button>
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  )
}
