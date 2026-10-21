'use client'

import { useRouter } from 'next/navigation'
import { type DragEvent, useState, useTransition } from 'react'
import {
  moveTo,
  proposalSentence,
  sportOptions,
  swapAt,
  tiebreakOptions,
} from '@/admin/format.utils'
import {
  adminBracketText,
  buildFormatInput,
  buildPreviewDays,
  type Chosen,
  type Conditions,
  copyFormat,
  describeSide,
  endText,
  initialConditions,
  maxDays,
  sameInput,
  scheduleOf,
  shiftConditions,
  type WizardForm,
  withDayCount,
  withSport,
} from '@/admin/formatWizard.utils'
import type { TeamRef } from '@/api/common.types'
import type {
  FormatInput,
  FormatPreview,
  PreviewGroup,
  Proposal,
  SavedFormat,
  Tiebreak,
} from '@/api/format.types'
import {
  type ActionResult,
  applyFormatAction,
  previewFormatAction,
  proposeFormatsAction,
} from '@/app/admin/_actions/format'
import { BracketTable } from '@/components/BracketTable'

// 形の決定ウィザード(docs/operations/operator-experience.md「大会の前」)。条件(日数、各日の時間、
// 昼休み、休み)を入れると形の候補が並び、候補を選ぶとプレビュー(組、トーナメント表、日ごとの
// 時間割、違反)が出る。チームはタップかドラッグでプール間の入れ替えができ、そのたびにプレビューを
// 作り直す。「この形で作る」は、最後にプレビューした入力そのものを保存する(プレビューと作成の
// 食い違いを起こさない)。

export type WizardProps = {
  divisionId: number
  teams: TeamRef[]
  initialSeeds: number[]
  defaultDate: string
  saved: SavedFormat | null
  /** 大会のコピーで引き継いだ形(この部門にまだ形が無いとき) */
  inherited: SavedFormat | null
  copies: SavedFormat[]
  /** 結果の入った試合の数。1 以上なら作り直せない */
  startedMatches: number
}

function teamsInParams(params: Chosen['params']): number | null {
  return typeof params.teams === 'number' ? params.teams : null
}

function chosenOf(saved: SavedFormat | null): Chosen | null {
  return saved ? { template: saved.template, params: saved.params, title: saved.title } : null
}

function initialForm(props: WizardProps): WizardForm {
  const base = props.saved ?? props.inherited
  let cond = initialConditions(base?.schedule ?? null, props.defaultDate)
  if (props.saved === null && props.inherited !== null) {
    // 引き継いだ時間割の設定は、日付だけ今回の大会の開催日に合わせる
    cond = shiftConditions(cond, props.defaultDate)
  }
  const inheritedFits =
    base !== null && teamsInParams(base.params) === props.initialSeeds.length ? base : null
  return {
    cond,
    chosen: chosenOf(props.saved ?? inheritedFits),
    seeds: props.initialSeeds,
    tiebreak: base?.tiebreak ?? 'current',
    forfeit: base?.forfeit ?? { winner: 15, loser: 0 },
  }
}

export function FormatWizard(props: WizardProps) {
  const { divisionId, teams } = props
  const router = useRouter()
  const [form, setForm] = useState<WizardForm>(() => initialForm(props))
  const [proposals, setProposals] = useState<Proposal[] | null>(null)
  const [preview, setPreview] = useState<FormatPreview | null>(null)
  // previewed は今出ているプレビューを作った入力。作成はこれをそのまま送る
  const [previewed, setPreviewed] = useState<FormatInput | null>(null)
  const [notes, setNotes] = useState<string[]>([])
  const [error, setError] = useState<string | null>(null)
  const [pending, startTransition] = useTransition()
  const teamById = new Map(teams.map((team) => [team.id, team]))
  const { cond } = form
  const locked = props.startedMatches > 0
  const current = buildFormatInput(form)
  const stale = preview !== null && !sameInput(current, previewed)

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

  /** 入力を next に変え、形が選ばれていれば next の入力でプレビューを作り直す */
  function update(next: WizardForm) {
    setForm(next)
    const input = buildFormatInput(next)
    if (input === null) {
      return
    }
    run(
      () => previewFormatAction(divisionId, input),
      (data) => {
        setPreview(data)
        setPreviewed(input)
      },
    )
  }

  function setCond(next: Conditions) {
    setForm({ ...form, cond: next })
  }

  function propose() {
    run(
      () =>
        proposeFormatsAction(divisionId, {
          teams: form.seeds.length,
          courts: cond.courts,
          slotMinutes: cond.slotMinutes,
          date: cond.days[0]?.date ?? props.defaultDate,
          start: cond.days[0]?.start ?? '09:00',
          sport: cond.sport,
          schedule: scheduleOf(cond),
        }),
      setProposals,
    )
  }

  function copyFrom(saved: SavedFormat) {
    const count = teamsInParams(saved.params)
    if (count !== form.seeds.length) {
      setError(
        `「${saved.tournamentName} ${saved.divisionName}」は ${count ?? '?'} チームの形です。この部門は ${form.seeds.length} チームです`,
      )
      return
    }
    const copied = copyFormat(form, saved)
    setNotes(copied.notes)
    update(copied.form)
  }

  function apply() {
    if (previewed === null) {
      return
    }
    const input = previewed
    run(
      () => applyFormatAction(divisionId, input),
      () => {
        router.push(
          `/admin/divisions/${divisionId}/progress?ok=${encodeURIComponent('大会の形を作りました')}`,
        )
      },
    )
  }

  const chosen = form.chosen
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
        <ConditionFields
          cond={cond}
          teams={form.seeds.length}
          onChange={setCond}
          defaultDate={props.defaultDate}
        />
        <div className="adm-actions">
          <button type="button" onClick={propose} disabled={pending || form.seeds.length < 2}>
            形の候補を出す
          </button>
          {chosen !== null ? (
            <button
              type="button"
              className="secondary"
              onClick={() => update(form)}
              disabled={pending}
            >
              この条件で時間割を作り直す
            </button>
          ) : null}
        </div>
        <CopySelect copies={props.copies} onCopy={copyFrom} />
        {notes.length > 0 ? (
          <ul className="fmt-notes" aria-live="polite">
            {notes.map((note) => (
              <li key={note}>{note}</li>
            ))}
          </ul>
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
                  className={`fmt-card${chosen?.title === proposal.title ? ' chosen' : ''}${
                    proposal.overflow > 0 ? ' overflow' : ''
                  }`}
                >
                  <h3>
                    {proposal.title}
                    {proposal.recommended ? <span className="fmt-tag">おすすめ</span> : null}
                  </h3>
                  <p className="fmt-big">{proposalSentence(proposal, form.seeds.length)}</p>
                  {proposal.fitNote ? <p className="fmt-warn">{proposal.fitNote}</p> : null}
                  <p className="adm-muted">{proposal.note}</p>
                  {proposal.violations > 0 ? (
                    <p className="adm-muted">時間割の注意が {proposal.violations} 件あります</p>
                  ) : null}
                  <button
                    type="button"
                    disabled={pending}
                    onClick={() => {
                      setNotes([])
                      update({
                        ...form,
                        chosen: {
                          template: proposal.template,
                          params: proposal.params,
                          title: proposal.title,
                        },
                      })
                    }}
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
          <button type="button" onClick={() => update(form)} disabled={pending}>
            「{chosen.title}」のプレビューを出す
          </button>
        </p>
      ) : null}
      {pending ? <p className="adm-muted">計算しています…</p> : null}

      {preview !== null && chosen !== null ? (
        <PreviewSection
          preview={preview}
          seeds={form.seeds}
          teamById={teamById}
          tiebreak={form.tiebreak}
          pending={pending}
          stale={stale}
          locked={locked}
          startedMatches={props.startedMatches}
          rebuilding={props.saved !== null}
          onSeeds={(seeds) => update({ ...form, seeds })}
          onTiebreak={(tiebreak) => update({ ...form, tiebreak })}
          onRefresh={() => update(form)}
          onApply={apply}
        />
      ) : null}
    </div>
  )
}

function ConditionFields({
  cond,
  teams,
  onChange,
  defaultDate,
}: {
  cond: Conditions
  teams: number
  onChange: (cond: Conditions) => void
  defaultDate: string
}) {
  const number = (value: string, fallback: number) => {
    const parsed = Number(value)
    return Number.isFinite(parsed) ? parsed : fallback
  }
  return (
    <>
      <div className="adm-grid">
        <label className="adm-field">
          <span>チーム数(登録済みのチーム)</span>
          <input type="number" value={teams} readOnly />
        </label>
        <label className="adm-field">
          <span>競技</span>
          <select
            value={cond.sport}
            onChange={(event) => {
              const sport = sportOptions.find((option) => option.value === event.target.value)
              if (sport) onChange(withSport(cond, sport.value))
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
            onChange={(event) =>
              onChange({ ...cond, courts: number(event.target.value, cond.courts) })
            }
          />
        </label>
        <label className="adm-field">
          <span>1 試合の枠の長さ(分)</span>
          <input
            type="number"
            min={5}
            value={cond.slotMinutes}
            onChange={(event) =>
              onChange({ ...cond, slotMinutes: number(event.target.value, cond.slotMinutes) })
            }
          />
        </label>
        <label className="adm-field">
          <span>同じチームの試合の間に最低空ける枠</span>
          <input
            type="number"
            min={0}
            max={6}
            value={cond.restSlots}
            onChange={(event) =>
              onChange({ ...cond, restSlots: number(event.target.value, cond.restSlots) })
            }
          />
          <small className="adm-muted">4on4 は 0(連戦を許す)、7 人制は 1 が目安</small>
        </label>
        <label className="adm-field">
          <span>日数</span>
          <select
            value={cond.days.length}
            onChange={(event) => onChange(withDayCount(cond, Number(event.target.value)))}
          >
            {Array.from({ length: maxDays }, (_, index) => index + 1).map((count) => (
              <option key={count} value={count}>
                {count} 日
              </option>
            ))}
          </select>
        </label>
      </div>
      <div className="fmt-days">
        {cond.days.map((day, index) => {
          const set = (patch: Partial<typeof day>) =>
            onChange({
              ...cond,
              days: cond.days.map((item, at) => (at === index ? { ...item, ...patch } : item)),
            })
          return (
            // biome-ignore lint/suspicious/noArrayIndexKey: 日の行は並べ替えず、追加と削除は末尾だけ
            <div key={index} className="fmt-day">
              <strong>{index + 1} 日目</strong>
              <label className="adm-field">
                <span>日付</span>
                <input
                  type="date"
                  value={day.date || defaultDate}
                  onChange={(event) => set({ date: event.target.value })}
                />
              </label>
              <label className="adm-field">
                <span>開始</span>
                <input
                  type="time"
                  value={day.start}
                  onChange={(event) => set({ start: event.target.value })}
                />
              </label>
              <label className="adm-field">
                <span>終了</span>
                <input
                  type="time"
                  value={day.end}
                  onChange={(event) => set({ end: event.target.value })}
                />
              </label>
            </div>
          )
        })}
        <div className="fmt-day">
          <strong>昼休み</strong>
          <label className="adm-field">
            <span>開始</span>
            <input
              type="time"
              value={cond.lunchStart}
              onChange={(event) => onChange({ ...cond, lunchStart: event.target.value })}
            />
          </label>
          <label className="adm-field">
            <span>長さ(分。0 でなし)</span>
            <input
              type="number"
              min={0}
              step={5}
              value={cond.lunchMinutes}
              onChange={(event) =>
                onChange({ ...cond, lunchMinutes: number(event.target.value, cond.lunchMinutes) })
              }
            />
          </label>
        </div>
      </div>
      {cond.days.length > 1 ? (
        <label className="adm-check">
          <input
            type="checkbox"
            checked={cond.stageDays}
            onChange={(event) => onChange({ ...cond, stageDays: event.target.checked })}
          />
          段ごとに日を分ける(1 日目は予選、2 日目は順位決定戦)
        </label>
      ) : null}
    </>
  )
}

function CopySelect({
  copies,
  onCopy,
}: {
  copies: SavedFormat[]
  onCopy: (saved: SavedFormat) => void
}) {
  return (
    <div className="adm-field">
      {copies.length > 0 ? (
        <label className="adm-field">
          <span>前回の大会の形をコピーする(コート数、日付、時刻は今回の条件のまま)</span>
          <select
            defaultValue=""
            onChange={(event) => {
              const saved = copies.find((copy) => String(copy.divisionId) === event.target.value)
              if (saved) onCopy(saved)
            }}
          >
            <option value="">選んでください</option>
            {copies.map((copy) => (
              <option key={copy.divisionId} value={copy.divisionId}>
                {copy.tournamentName} {copy.divisionName}: {copy.title}
              </option>
            ))}
          </select>
        </label>
      ) : null}
      <small className="adm-muted">
        参照サイトから取り込んだ過去の大会は、組と試合の記録だけで形の設定(順位帯や勝ち上がりの決まり)を
        持たないため、ここには出ません。同じ形は「形の候補を出す」から選べます。
      </small>
    </div>
  )
}

type PreviewProps = {
  preview: FormatPreview
  seeds: number[]
  teamById: Map<number, TeamRef>
  tiebreak: Tiebreak
  pending: boolean
  /** 条件を変えたのにプレビューを作り直していない */
  stale: boolean
  locked: boolean
  startedMatches: number
  rebuilding: boolean
  onSeeds: (seeds: number[]) => void
  onTiebreak: (tiebreak: Tiebreak) => void
  onRefresh: () => void
  onApply: () => void
}

function PreviewSection(props: PreviewProps) {
  const { preview } = props
  const [copied, setCopied] = useState(false)
  const first = preview.groups.filter((group) => group.stage === 0)
  const later = preview.groups.filter((group) => group.stage > 0)
  const days = buildPreviewDays(preview.matches)
  const guide = [...preview.ruleText, ...preview.advancement].join('\n')
  return (
    <section>
      <h2>プレビュー: {preview.title}</h2>
      <p className="fmt-big">
        {preview.matches.length} 試合
        {preview.fitNote ? '' : `、${endText(preview.end, preview.endDay)}`}。
        {preview.unique === preview.teams
          ? `全 ${preview.teams} 位まで決まります。`
          : `${preview.ranked} チームの順位が決まります。`}
        試合番号は {preview.startLetter} から。
      </p>
      {preview.fitNote ? <p className="fmt-warn">{preview.fitNote}</p> : null}
      {preview.violations.length === 0 ? (
        <p className="fmt-ok">時間割の決まりに合わない試合はありません。</p>
      ) : (
        <>
          <h3 className="fmt-warn">時間割の注意({preview.violations.length} 件)</h3>
          <ul className="fmt-violations">
            {preview.violations.map((violation) => (
              <li key={`${violation.kind}-${violation.codes.join('-')}`}>{violation.detail}</li>
            ))}
          </ul>
        </>
      )}

      <h3>予選の組(チームを 2 回タップするか、ドラッグして組の間で入れ替え)</h3>
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
              <BracketTable
                bracket={{
                  ...group.bracket,
                  rows: group.bracket.rows.map((row) =>
                    row.map((cell) => ({ ...cell, text: adminBracketText(cell.text) })),
                  ),
                }}
                lang="jp"
              />
            </div>
          ) : (
            <p>{group.slots.map((slot) => describeSide(slot.label)).join('、')} の総当たり</p>
          )}
        </div>
      ))}

      <h3>時間割</h3>
      {days.map((day, index) => (
        <div key={day.date ?? 'none'}>
          <h4>
            {days.length > 1 ? `${index + 1} 日目 ` : ''}
            {day.date ?? '日付未定'}
          </h4>
          <div className="adm-scroll">
            <table className="fmt-grid">
              <thead>
                <tr>
                  <th>時刻</th>
                  {preview.courts.map((court) => (
                    <th key={court}>コート {court}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {day.times.map((time) => (
                  <tr key={time}>
                    <th>{time}</th>
                    {preview.courts.map((court) => {
                      const match = day.cells.get(`${time}|${court}`)
                      return (
                        <td key={court}>
                          {match ? (
                            <>
                              <span className="code">{match.code}</span>
                              <br />
                              {match.left.team?.name.ja ?? describeSide(match.left.label)} 対{' '}
                              {match.right.team?.name.ja ?? describeSide(match.right.label)}
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
        </div>
      ))}

      <h2>{props.rebuilding ? 'この形で作り直す' : 'この形で作る'}</h2>
      {props.locked ? (
        <p className="adm-notice error">
          結果の入った試合が {props.startedMatches}{' '}
          件あるので、形を作り直せません(作り直すと結果が消えるため)。日程の変更は本部の進行画面の「遅れの調整」か、試合の編集で行ってください。
        </p>
      ) : preview.replacesGroups + preview.replacesMatches > 0 ? (
        <p className="adm-notice error">
          この部門の今の組 {preview.replacesGroups} 個と試合 {preview.replacesMatches}{' '}
          件は消えて、この形に置き換わります。
        </p>
      ) : null}
      {props.stale && !props.locked ? (
        <p className="adm-notice error">
          条件を変えたので、作る前にプレビューを作り直してください(プレビューと違う時間割は作りません)。
        </p>
      ) : null}
      <p className="adm-actions">
        {props.stale ? (
          <button type="button" onClick={props.onRefresh} disabled={props.pending || props.locked}>
            この条件でプレビューを作り直す
          </button>
        ) : (
          <button type="button" onClick={props.onApply} disabled={props.pending || props.locked}>
            {props.rebuilding ? 'この形で作り直す' : 'この形で作る'}
          </button>
        )}
      </p>
    </section>
  )
}

/**
 * プール表。チームをタップしてから別のチームをタップすると 2 チームのシードを入れ替える(スマート
 * フォーン向け)。PC ではドラッグして別のチームに落としても入れ替わる
 */
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
  const [picked, setPicked] = useState<number | null>(null)
  const swap = (source: number, target: number) => {
    if (Number.isInteger(source) && source > 0 && source !== target) {
      onSeeds(swapAt(seeds, source - 1, target - 1))
    }
  }
  const drop = (event: DragEvent, target: number) => {
    event.preventDefault()
    setOver(null)
    swap(Number(event.dataTransfer.getData('text/plain')), target)
  }
  const tap = (seed: number) => {
    if (picked === null) {
      setPicked(seed)
      return
    }
    setPicked(null)
    swap(picked, seed)
  }
  return (
    <>
      {picked !== null ? (
        <p className="adm-muted" aria-live="polite">
          シード{picked}{' '}
          を選びました。入れ替える相手をタップしてください(同じチームをもう一度で取り消し)。
        </p>
      ) : null}
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
                      {describeSide(slot.label)}
                    </li>
                  )
                }
                const classes = [
                  'fmt-slot',
                  over === seed ? 'over' : '',
                  picked === seed ? 'picked' : '',
                ]
                return (
                  <li
                    key={slot.position}
                    className={classes.filter(Boolean).join(' ')}
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
                    <button
                      type="button"
                      className="fmt-pick"
                      aria-pressed={picked === seed}
                      onClick={() => tap(seed)}
                    >
                      <span className="seed">シード{seed}</span>
                      <span className="grow">{slot.team?.name.ja ?? slot.label}</span>
                    </button>
                  </li>
                )
              })}
            </ul>
          </div>
        ))}
      </div>
    </>
  )
}

/** シード順の一覧。ドラッグで並べ替え、スマートフォンでは上下、先頭へ、末尾へのボタンで動かす */
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
  const last = seeds.length - 1
  return (
    <details>
      <summary>
        シード順を並べ替える(上が強いチーム。組へは 1, 2, 3, 4, 4, 3, … の順に配ります)
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
              disabled={disabled || index === 0}
              onClick={() => onSeeds(moveTo(seeds, index, 0))}
            >
              先頭へ
            </button>
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
              disabled={disabled || index === last}
              onClick={() => onSeeds(moveTo(seeds, index, index + 1))}
            >
              ↓
            </button>
            <button
              type="button"
              className="secondary"
              disabled={disabled || index === last}
              onClick={() => onSeeds(moveTo(seeds, index, last))}
            >
              末尾へ
            </button>
          </li>
        ))}
      </ol>
    </details>
  )
}
