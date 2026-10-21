'use client'

import { useRouter } from 'next/navigation'
import { type DragEvent, useState, useTransition } from 'react'
import {
  buildPreviewGrid,
  defaultSlotMinutes,
  moveTo,
  proposalSentence,
  sportOptions,
  swapAt,
  tiebreakOptions,
} from '@/admin/format.utils'
import type { TeamRef } from '@/api/common.types'
import type {
  FormatInput,
  FormatParams,
  FormatPreview,
  PreviewGroup,
  Proposal,
  SavedFormat,
  Sport,
  Tiebreak,
} from '@/api/format.types'
import {
  type ActionResult,
  applyFormatAction,
  previewFormatAction,
  proposeFormatsAction,
} from '@/app/admin/_actions/format'
import { BracketTable } from '@/components/BracketTable'

// 形の決定ウィザード(docs/operations/operator-experience.md「大会の前」)。条件を入れると形の候補が
// 並び、候補を選ぶとプレビュー(プール、トーナメント表、時間割、違反)が出る。チームはドラッグで
// 並べ替えとプール間の入れ替えができ、そのたびにプレビューを作り直す。「この形で作る」で保存する。

type Conditions = {
  courts: number
  slotMinutes: number
  date: string
  start: string
  sport: Sport
}

type Chosen = { template: string; params: FormatParams; title: string }

export type WizardProps = {
  divisionId: number
  teams: TeamRef[]
  initialSeeds: number[]
  defaultDate: string
  saved: SavedFormat | null
  copies: SavedFormat[]
}

function teamsInParams(params: FormatParams): number | null {
  return typeof params.teams === 'number' ? params.teams : null
}

export function FormatWizard(props: WizardProps) {
  const { divisionId, teams } = props
  const router = useRouter()
  const savedSchedule = props.saved?.schedule ?? null
  const [cond, setCond] = useState<Conditions>({
    courts: savedSchedule?.courts.length ?? 4,
    slotMinutes: savedSchedule?.slotMinutes ?? defaultSlotMinutes.four,
    date: savedSchedule?.date ?? props.defaultDate,
    start: savedSchedule?.start ?? '09:00',
    sport: 'four',
  })
  const [proposals, setProposals] = useState<Proposal[] | null>(null)
  const [chosen, setChosen] = useState<Chosen | null>(
    props.saved
      ? { template: props.saved.template, params: props.saved.params, title: props.saved.title }
      : null,
  )
  const [seeds, setSeeds] = useState<number[]>(props.initialSeeds)
  const [tiebreak, setTiebreak] = useState<Tiebreak>(props.saved?.tiebreak ?? 'current')
  const [preview, setPreview] = useState<FormatPreview | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [pending, startTransition] = useTransition()
  const teamById = new Map(teams.map((team) => [team.id, team]))

  function input(next: { chosen: Chosen; seeds: number[]; tiebreak: Tiebreak; cond: Conditions }) {
    const body: FormatInput = {
      template: next.chosen.template,
      params: next.chosen.params,
      seeds: next.seeds,
      tiebreak: next.tiebreak,
      forfeit: props.saved?.forfeit ?? { winner: 15, loser: 0 },
      schedule: {
        date: next.cond.date,
        start: next.cond.start,
        slotMinutes: next.cond.slotMinutes,
        courts: Array.from({ length: next.cond.courts }, (_, index) => String(index + 1)),
        teamRestSlots: next.cond.sport === 'four' ? 0 : 1,
        stageGapSlots: 0,
        end: null,
      },
    }
    return body
  }

  function run<T>(task: () => Promise<ActionResult<T>>, done: (data: T) => void) {
    startTransition(async () => {
      const result = await task()
      if (result.ok) {
        setError(null)
        done(result.data)
      } else {
        setError(result.error)
      }
    })
  }

  function refresh(next: Partial<{ chosen: Chosen; seeds: number[]; tiebreak: Tiebreak }>) {
    const merged = {
      chosen: next.chosen ?? chosen,
      seeds: next.seeds ?? seeds,
      tiebreak: next.tiebreak ?? tiebreak,
      cond,
    }
    if (next.chosen) setChosen(next.chosen)
    if (next.seeds) setSeeds(next.seeds)
    if (next.tiebreak) setTiebreak(next.tiebreak)
    const target = merged.chosen
    if (target === null) {
      return
    }
    run(() => previewFormatAction(divisionId, input({ ...merged, chosen: target })), setPreview)
  }

  function propose() {
    run(
      () =>
        proposeFormatsAction(divisionId, {
          teams: seeds.length,
          courts: cond.courts,
          slotMinutes: cond.slotMinutes,
          date: cond.date,
          start: cond.start,
          sport: cond.sport,
        }),
      setProposals,
    )
  }

  function copyFrom(saved: SavedFormat) {
    const count = teamsInParams(saved.params)
    if (count !== seeds.length) {
      setError(
        `「${saved.tournamentName} ${saved.divisionName}」は ${count ?? '?'} チームの形です。この部門は ${seeds.length} チームです`,
      )
      return
    }
    const schedule = saved.schedule
    if (schedule) {
      setCond({
        ...cond,
        courts: schedule.courts.length,
        slotMinutes: schedule.slotMinutes,
        start: schedule.start,
      })
    }
    refresh({
      chosen: { template: saved.template, params: saved.params, title: saved.title },
      tiebreak: saved.tiebreak,
    })
  }

  function apply() {
    if (chosen === null) {
      return
    }
    run(
      () => applyFormatAction(divisionId, input({ chosen, seeds, tiebreak, cond })),
      () => {
        router.push(
          `/admin/divisions/${divisionId}/progress?ok=${encodeURIComponent('大会の形を作りました')}`,
        )
      },
    )
  }

  return (
    <div className="fmt-wizard">
      <ol className="fmt-steps">
        <li className={chosen === null ? 'on' : ''}>1. 条件を入れる</li>
        <li className={chosen !== null && preview === null ? 'on' : ''}>2. 形を選ぶ</li>
        <li className={preview !== null ? 'on' : ''}>3. 確かめて作る</li>
      </ol>
      {error ? (
        <p className="adm-notice error" role="alert">
          {error}
        </p>
      ) : null}

      <fieldset>
        <legend>大会の条件</legend>
        <div className="adm-grid">
          <label className="adm-field">
            <span>チーム数(登録済みのチーム)</span>
            <input type="number" value={seeds.length} readOnly />
          </label>
          <label className="adm-field">
            <span>競技</span>
            <select
              value={cond.sport}
              onChange={(event) => {
                const sport = sportOptions.find((option) => option.value === event.target.value)
                if (sport) {
                  setCond({
                    ...cond,
                    sport: sport.value,
                    slotMinutes: defaultSlotMinutes[sport.value],
                  })
                }
              }}
            >
              {sportOptions.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </select>
          </label>
          <label className="adm-field">
            <span>コート数</span>
            <input
              type="number"
              min={1}
              value={cond.courts}
              onChange={(event) => setCond({ ...cond, courts: Number(event.target.value) })}
            />
          </label>
          <label className="adm-field">
            <span>1 試合の枠の長さ(分)</span>
            <input
              type="number"
              min={5}
              value={cond.slotMinutes}
              onChange={(event) => setCond({ ...cond, slotMinutes: Number(event.target.value) })}
            />
          </label>
          <label className="adm-field">
            <span>日付</span>
            <input
              type="date"
              value={cond.date}
              onChange={(event) => setCond({ ...cond, date: event.target.value })}
            />
          </label>
          <label className="adm-field">
            <span>開始時刻</span>
            <input
              type="time"
              value={cond.start}
              onChange={(event) => setCond({ ...cond, start: event.target.value })}
            />
          </label>
        </div>
        <div className="adm-actions">
          <button type="button" onClick={propose} disabled={pending || seeds.length < 2}>
            形の候補を出す
          </button>
          {chosen !== null ? (
            <button
              type="button"
              className="secondary"
              onClick={() => refresh({})}
              disabled={pending}
            >
              この条件で時間割を作り直す
            </button>
          ) : null}
        </div>
        {props.copies.length > 0 ? (
          <label className="adm-field">
            <span>前回の大会の形をコピーする</span>
            <select
              defaultValue=""
              onChange={(event) => {
                const saved = props.copies.find(
                  (copy) => String(copy.divisionId) === event.target.value,
                )
                if (saved) copyFrom(saved)
              }}
            >
              <option value="">選んでください</option>
              {props.copies.map((copy) => (
                <option key={copy.divisionId} value={copy.divisionId}>
                  {copy.tournamentName} {copy.divisionName}: {copy.title}
                </option>
              ))}
            </select>
          </label>
        ) : null}
      </fieldset>

      {proposals !== null ? (
        <section>
          <h2>形の候補</h2>
          {proposals.length === 0 ? (
            <p>この条件で作れる形がありません。チーム数を確かめてください。</p>
          ) : (
            <div className="fmt-cards">
              {proposals.map((proposal) => (
                <div
                  key={proposal.id}
                  className={`fmt-card${chosen?.title === proposal.title ? ' chosen' : ''}`}
                >
                  <h3>
                    {proposal.title}
                    {proposal.recommended ? <span className="fmt-tag">おすすめ</span> : null}
                  </h3>
                  <p className="fmt-big">{proposalSentence(proposal, seeds.length)}</p>
                  <p className="adm-muted">{proposal.note}</p>
                  {proposal.violations > 0 ? (
                    <p className="adm-muted">時間割の注意が {proposal.violations} 件あります</p>
                  ) : null}
                  <button
                    type="button"
                    disabled={pending}
                    onClick={() =>
                      refresh({
                        chosen: {
                          template: proposal.template,
                          params: proposal.params,
                          title: proposal.title,
                        },
                      })
                    }
                  >
                    この形にする
                  </button>
                </div>
              ))}
            </div>
          )}
        </section>
      ) : null}

      {chosen !== null && preview === null ? (
        <p className="adm-actions">
          <button type="button" onClick={() => refresh({})} disabled={pending}>
            「{chosen.title}」のプレビューを出す
          </button>
        </p>
      ) : null}
      {pending ? <p className="adm-muted">計算しています…</p> : null}

      {preview !== null && chosen !== null ? (
        <PreviewSection
          preview={preview}
          seeds={seeds}
          teamById={teamById}
          tiebreak={tiebreak}
          pending={pending}
          onSeeds={(next) => refresh({ seeds: next })}
          onTiebreak={(next) => refresh({ tiebreak: next })}
          onApply={apply}
        />
      ) : null}
    </div>
  )
}

type PreviewProps = {
  preview: FormatPreview
  seeds: number[]
  teamById: Map<number, TeamRef>
  tiebreak: Tiebreak
  pending: boolean
  onSeeds: (seeds: number[]) => void
  onTiebreak: (tiebreak: Tiebreak) => void
  onApply: () => void
}

function PreviewSection(props: PreviewProps) {
  const { preview } = props
  const [copied, setCopied] = useState(false)
  const first = preview.groups.filter((group) => group.stage === 0)
  const later = preview.groups.filter((group) => group.stage > 0)
  const grid = buildPreviewGrid(preview.matches, preview.courts)
  const guide = [...preview.ruleText, ...preview.advancement].join('\n')
  return (
    <section>
      <h2>プレビュー: {preview.title}</h2>
      <p className="fmt-big">
        {preview.matches.length} 試合、{preview.end ?? '時刻未定'} 終了。
        {preview.unique === preview.teams
          ? `全 ${preview.teams} 位まで決まります。`
          : `${preview.ranked} チームの順位が決まります。`}
        試合番号は {preview.startLetter} から。
      </p>
      {preview.violations.length === 0 ? (
        <p className="fmt-ok">時間割の制約違反はありません。</p>
      ) : (
        <>
          <h3>時間割の注意({preview.violations.length} 件)</h3>
          <ul className="fmt-violations">
            {preview.violations.map((violation) => (
              <li key={`${violation.kind}-${violation.codes.join('-')}`}>{violation.detail}</li>
            ))}
          </ul>
        </>
      )}

      <h3>プール(チームをドラッグしてプール間で入れ替え)</h3>
      <PoolCards groups={first} seeds={props.seeds} onSeeds={props.onSeeds} />
      <SeedList
        seeds={props.seeds}
        teamById={props.teamById}
        onSeeds={props.onSeeds}
        disabled={props.pending}
      />

      <h3>順位の決め方と勝ち上がり(参加チームへの案内にそのまま使えます)</h3>
      <label className="adm-field">
        <span>同点のときの決め方</span>
        <select
          value={props.tiebreak}
          onChange={(event) => {
            const option = tiebreakOptions.find((item) => item.value === event.target.value)
            if (option) props.onTiebreak(option.value)
          }}
        >
          {tiebreakOptions.map((option) => (
            <option key={option.value} value={option.value}>
              {option.label}
            </option>
          ))}
        </select>
      </label>
      <div className="fmt-rules">
        {preview.ruleText.map((line) => (
          <p key={line}>{line}</p>
        ))}
        {preview.advancement.map((line) => (
          <p key={line}>{line}</p>
        ))}
      </div>
      <p className="adm-actions">
        <button
          type="button"
          className="secondary"
          onClick={() => {
            navigator.clipboard.writeText(guide).then(
              () => setCopied(true),
              () => setCopied(false),
            )
          }}
        >
          {copied ? 'コピーしました' : '案内文をコピー'}
        </button>
      </p>

      <h3>順位決定戦</h3>
      {later.map((group) => (
        <div key={group.index}>
          <h4>{group.name}</h4>
          {group.bracket ? (
            <div className="fmt-bracket">
              <BracketTable bracket={group.bracket} lang="jp" />
            </div>
          ) : (
            <p>{group.slots.map((slot) => slot.label).join('、')} の総当たり</p>
          )}
        </div>
      ))}

      <h3>時間割</h3>
      <div className="adm-scroll">
        <table className="fmt-grid">
          <thead>
            <tr>
              <th>時刻</th>
              {grid.courts.map((court) => (
                <th key={court}>コート {court}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {grid.times.map((time) => (
              <tr key={time}>
                <th>{time}</th>
                {grid.courts.map((court) => {
                  const match = grid.cells.get(`${time}|${court}`)
                  return (
                    <td key={court}>
                      {match ? (
                        <>
                          <span className="code">{match.code}</span>
                          <br />
                          {match.left.team?.name.ja ?? match.left.label} 対{' '}
                          {match.right.team?.name.ja ?? match.right.label}
                        </>
                      ) : null}
                    </td>
                  )
                })}
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <h2>この形で作る</h2>
      {preview.replacesGroups + preview.replacesMatches > 0 ? (
        <p className="adm-notice error">
          この部門の今のグループ {preview.replacesGroups} 個と試合 {preview.replacesMatches}{' '}
          件は消えて、この形に置き換わります。
        </p>
      ) : null}
      <p className="adm-actions">
        <button type="button" onClick={props.onApply} disabled={props.pending}>
          この形で作る
        </button>
      </p>
    </section>
  )
}

/** プール表。チームをドラッグして別のチームに落とすと、2 チームのシードを入れ替える */
function PoolCards({
  groups,
  seeds,
  onSeeds,
}: {
  groups: PreviewGroup[]
  seeds: number[]
  onSeeds: (seeds: number[]) => void
}) {
  const [over, setOver] = useState<number | null>(null)
  const drop = (event: DragEvent, target: number) => {
    event.preventDefault()
    setOver(null)
    const source = Number(event.dataTransfer.getData('text/plain'))
    if (Number.isInteger(source) && source > 0 && source !== target) {
      onSeeds(swapAt(seeds, source - 1, target - 1))
    }
  }
  return (
    <div className="fmt-pools">
      {groups.map((group) => (
        <div key={group.index} className="fmt-pool">
          <h4>{group.name}</h4>
          <ul className="fmt-pool-list">
            {group.slots.map((slot) => {
              const seed = slot.seed
              if (seed === null) {
                return (
                  <li key={slot.position} className="fmt-slot">
                    {slot.label}
                  </li>
                )
              }
              return (
                <li
                  key={slot.position}
                  className={`fmt-slot${over === seed ? ' over' : ''}`}
                  draggable
                  onDragStart={(event) => event.dataTransfer.setData('text/plain', String(seed))}
                  onDragOver={(event) => {
                    event.preventDefault()
                    setOver(seed)
                  }}
                  onDragLeave={() => setOver(null)}
                  onDrop={(event) => drop(event, seed)}
                  data-seed={seed}
                >
                  <span className="seed">シード{seed}</span>
                  <span className="grow">{slot.team?.name.ja ?? slot.label}</span>
                </li>
              )
            })}
          </ul>
        </div>
      ))}
    </div>
  )
}

/** シード順の一覧。ドラッグで並べ替え、スマートフォンでは上下のボタンで動かす */
function SeedList({
  seeds,
  teamById,
  onSeeds,
  disabled,
}: {
  seeds: number[]
  teamById: Map<number, TeamRef>
  onSeeds: (seeds: number[]) => void
  disabled: boolean
}) {
  return (
    <details>
      <summary>
        シード順を並べ替える(上が強いチーム。プールへは 1, 2, 3, 4, 4, 3, … の順に配ります)
      </summary>
      <ol className="fmt-seeds">
        {seeds.map((teamId, index) => (
          <li
            key={teamId}
            className="fmt-slot"
            draggable
            onDragStart={(event) => event.dataTransfer.setData('text/plain', String(index))}
            onDragOver={(event) => event.preventDefault()}
            onDrop={(event) => {
              event.preventDefault()
              const from = Number(event.dataTransfer.getData('text/plain'))
              if (Number.isInteger(from) && from !== index) {
                onSeeds(moveTo(seeds, from, index))
              }
            }}
          >
            <span className="seed">{index + 1}</span>
            <span className="grow">{teamById.get(teamId)?.name.ja ?? teamId}</span>
            <button
              type="button"
              className="secondary"
              aria-label="上へ"
              disabled={disabled || index === 0}
              onClick={() => onSeeds(moveTo(seeds, index, index - 1))}
            >
              ↑
            </button>
            <button
              type="button"
              className="secondary"
              aria-label="下へ"
              disabled={disabled || index === seeds.length - 1}
              onClick={() => onSeeds(moveTo(seeds, index, index + 1))}
            >
              ↓
            </button>
          </li>
        ))}
      </ol>
    </details>
  )
}
