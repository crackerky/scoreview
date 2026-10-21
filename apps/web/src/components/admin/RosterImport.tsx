'use client'

import { useState } from 'react'
import {
  columnCount,
  guessPlayerColumns,
  guessTeamColumns,
  type PlayerColumns,
  playerRowsOf,
  splitRows,
  type TeamColumns,
  teamRowsOf,
} from '@/admin/roster.utils'
import {
  createPlayersFromRosterAction,
  createTeamsFromRosterAction,
} from '@/app/admin/_actions/format'

// 名簿の貼り付けと登録前のプレビュー(運営者レビュー H4)。貼った直後に表で見せ、1 行目が見出しか、
// どの列がチーム名・略称(選手ならチーム名、背番号、氏名、性別)かを選んでから登録する。数字だけの名前や
// 重複には注意を出す。

type TeamOption = { id: number; fullName: string; shortName: string }

type Props =
  | { kind: 'teams'; back: string; divisionId: number; startOrder: number; existing: string[] }
  | { kind: 'players'; back: string; divisionId: number; teams: TeamOption[] }

function ColumnSelect({
  label,
  value,
  count,
  optional,
  onChange,
}: {
  label: string
  value: number | null
  count: number
  optional: boolean
  onChange: (value: number | null) => void
}) {
  return (
    <label className="adm-field">
      <span>{label}</span>
      <select
        value={value === null ? '' : String(value)}
        onChange={(event) =>
          onChange(event.target.value === '' ? null : Number(event.target.value))
        }
      >
        {optional ? <option value="">使わない</option> : null}
        {Array.from({ length: count }, (_, index) => index).map((index) => (
          <option key={index} value={index}>
            {index + 1} 列目
          </option>
        ))}
      </select>
    </label>
  )
}

export function RosterImport(props: Props) {
  const [text, setText] = useState('')
  const rows = splitRows(text)
  const count = columnCount(rows)
  const [teamColumns, setTeamColumns] = useState<TeamColumns | null>(null)
  const [playerColumns, setPlayerColumns] = useState<PlayerColumns | null>(null)
  const teamCols = teamColumns ?? guessTeamColumns(rows)
  const playerCols = playerColumns ?? guessPlayerColumns(rows)
  const header = props.kind === 'teams' ? teamCols.header : playerCols.header
  const preview = rows.slice(header ? 1 : 0, (header ? 1 : 0) + 3)

  const teamRows = props.kind === 'teams' ? teamRowsOf(rows, teamCols, props.existing) : []
  const playerRows = props.kind === 'players' ? playerRowsOf(rows, playerCols, props.teams) : []
  const registrable =
    props.kind === 'teams'
      ? teamRows.length
      : playerRows.filter((row) => row.teamId !== null).length
  const payload =
    props.kind === 'teams'
      ? teamRows.map(({ fullName, shortName, fullNameEn }) => ({ fullName, shortName, fullNameEn }))
      : playerRows.flatMap(({ teamId, number, name, nameEn, gender }) =>
          teamId === null ? [] : [{ teamId, number, name, nameEn, gender }],
        )

  const setHeader = (value: boolean) => {
    if (props.kind === 'teams') setTeamColumns({ ...teamCols, header: value })
    else setPlayerColumns({ ...playerCols, header: value })
  }

  return (
    <form
      action={props.kind === 'teams' ? createTeamsFromRosterAction : createPlayersFromRosterAction}
      className="roster-import"
    >
      <input type="hidden" name="back" value={props.back} />
      <input type="hidden" name="divisionId" value={props.divisionId} />
      {props.kind === 'teams' ? (
        <input type="hidden" name="startOrder" value={props.startOrder} />
      ) : null}
      <input type="hidden" name="rows" value={JSON.stringify(payload)} />
      <label className="adm-field">
        <span>
          {props.kind === 'teams'
            ? 'エクセルなどの名簿をそのまま貼り付けてください(1 行 1 チーム。上から順にシード順の初期値)'
            : '選手名簿を貼り付けてください(1 行 1 人。チーム名、背番号、氏名、性別の列)'}
        </span>
        <textarea
          name="roster"
          rows={8}
          value={text}
          onChange={(event) => {
            setText(event.target.value)
            setTeamColumns(null)
            setPlayerColumns(null)
          }}
        />
      </label>
      {rows.length > 0 ? (
        <div className="roster-preview">
          <h3>登録する前に確かめてください</h3>
          <label className="adm-check">
            <input
              type="checkbox"
              checked={header}
              onChange={(event) => setHeader(event.target.checked)}
            />
            1 行目は見出し(登録しない)
          </label>
          {rows[0] ? (
            <p className="adm-muted">
              1 行目: {rows[0].join(' | ')}
              {preview.length > 0 ? ` / 2 行目以降の例: ${preview[0]?.join(' | ')}` : ''}
            </p>
          ) : null}
          <div className="adm-grid">
            {props.kind === 'teams' ? (
              <>
                <ColumnSelect
                  label="チーム名(正式名)の列"
                  value={teamCols.name}
                  count={count}
                  optional={false}
                  onChange={(value) => setTeamColumns({ ...teamCols, name: value ?? 0 })}
                />
                <ColumnSelect
                  label="略称の列(使わなければ正式名と同じ)"
                  value={teamCols.short}
                  count={count}
                  optional
                  onChange={(value) => setTeamColumns({ ...teamCols, short: value })}
                />
                <ColumnSelect
                  label="英語名の列(使わなければ英語のページにも日本語名を出す)"
                  value={teamCols.nameEn}
                  count={count}
                  optional
                  onChange={(value) => setTeamColumns({ ...teamCols, nameEn: value })}
                />
              </>
            ) : (
              <>
                <ColumnSelect
                  label="チーム名の列"
                  value={playerCols.team}
                  count={count}
                  optional={false}
                  onChange={(value) => setPlayerColumns({ ...playerCols, team: value ?? 0 })}
                />
                <ColumnSelect
                  label="背番号の列"
                  value={playerCols.number}
                  count={count}
                  optional
                  onChange={(value) => setPlayerColumns({ ...playerCols, number: value })}
                />
                <ColumnSelect
                  label="氏名の列"
                  value={playerCols.name}
                  count={count}
                  optional={false}
                  onChange={(value) => setPlayerColumns({ ...playerCols, name: value ?? 0 })}
                />
                <ColumnSelect
                  label="性別の列"
                  value={playerCols.gender}
                  count={count}
                  optional
                  onChange={(value) => setPlayerColumns({ ...playerCols, gender: value })}
                />
                <ColumnSelect
                  label="英語名の列(使わなければ英語のページにも日本語名を出す)"
                  value={playerCols.nameEn}
                  count={count}
                  optional
                  onChange={(value) => setPlayerColumns({ ...playerCols, nameEn: value })}
                />
              </>
            )}
          </div>
          <div className="adm-scroll">
            <table className="adm-table roster-table">
              <thead>
                {props.kind === 'teams' ? (
                  <tr>
                    <th>#</th>
                    <th>正式名</th>
                    <th>略称</th>
                    <th>英語名</th>
                    <th>注意</th>
                  </tr>
                ) : (
                  <tr>
                    <th>#</th>
                    <th>チーム</th>
                    <th>背番号</th>
                    <th>氏名</th>
                    <th>英語名</th>
                    <th>性別</th>
                    <th>注意</th>
                  </tr>
                )}
              </thead>
              <tbody>
                {props.kind === 'teams'
                  ? teamRows.map((row, index) => (
                      <tr
                        // biome-ignore lint/suspicious/noArrayIndexKey: 貼り付けの行の順そのもの(並べ替えない)
                        key={`${index}-${row.fullName}`}
                        className={row.warnings.length ? 'warn' : ''}
                      >
                        <td>{index + 1}</td>
                        <td>{row.fullName}</td>
                        <td>{row.shortName}</td>
                        <td>{row.fullNameEn || row.fullName}</td>
                        <td>{row.warnings.join('。')}</td>
                      </tr>
                    ))
                  : playerRows.map((row, index) => (
                      <tr
                        // biome-ignore lint/suspicious/noArrayIndexKey: 貼り付けの行の順そのもの(並べ替えない)
                        key={`${index}-${row.name}`}
                        className={row.warnings.length ? 'warn' : ''}
                      >
                        <td>{index + 1}</td>
                        <td>{row.teamName}</td>
                        <td>{row.number}</td>
                        <td>{row.name}</td>
                        <td>{row.nameEn || row.name}</td>
                        <td>{row.gender === 'M' ? '男' : row.gender === 'F' ? '女' : ''}</td>
                        <td>{row.warnings.join('。')}</td>
                      </tr>
                    ))}
              </tbody>
            </table>
          </div>
        </div>
      ) : null}
      <button type="submit" disabled={registrable === 0}>
        {props.kind === 'teams' ? `${registrable} チームを登録する` : `${registrable} 人を登録する`}
      </button>
    </form>
  )
}
