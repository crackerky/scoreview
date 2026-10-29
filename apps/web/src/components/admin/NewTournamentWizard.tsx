'use client'

import { useState } from 'react'
import { finalTeamNames, type Replacement, type SourceDivision } from '@/admin/newTournament.utils'
import { createFromWizardAction, loadSourceTeams } from '@/app/admin/_actions/newTournament'

// 大会を作る 3 ステップ(Claude Design「Score View Admin」)。1. 元にする大会、2. チーム(出ないチームを
// 入れ替える)、3. 確認して公開。前回の大会をコピーするのが基本で、白紙から作るときは 2 を飛ばす。

export type WizardSource = { id: number; name: string; meta: string }

type Mode = 'copy' | 'blank'

const steps = ['元にする大会', 'チーム', '確認して公開'] as const

export function NewTournamentWizard({
  sources,
  defaultDate,
}: {
  sources: WizardSource[]
  defaultDate: string
}) {
  const [step, setStep] = useState(1)
  const [mode, setMode] = useState<Mode>(sources.length > 0 ? 'copy' : 'blank')
  const [sourceId, setSourceId] = useState(sources[0]?.id ?? 0)
  const [name, setName] = useState('')
  const [date, setDate] = useState(defaultDate)
  const [divisions, setDivisions] = useState<SourceDivision[] | null>(null)
  const [replaced, setReplaced] = useState<Map<string, string>>(new Map())
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const source = sources.find((item) => item.id === sourceId)
  const replacements: Replacement[] = [...replaced].map(([key, value]) => {
    const [division, team] = key.split(':').map(Number)
    return { division: division ?? 0, team: team ?? 0, name: value }
  })
  const unfilled = replacements.some((item) => item.name.trim() === '')

  const next = async () => {
    setError('')
    if (step === 1 && mode === 'blank') {
      setStep(3)
      return
    }
    if (step === 1) {
      setLoading(true)
      try {
        setDivisions(await loadSourceTeams(sourceId))
        setReplaced(new Map())
        setStep(2)
      } catch {
        // 読めなかった理由(通信、権限)は画面に出せる形で返らないので、やり直しを促す
        setError('コピー元のチームを読めませんでした。もう一度「次へ」を押してください')
      } finally {
        setLoading(false)
      }
      return
    }
    setStep(3)
  }

  const toggle = (key: string) => {
    setReplaced((current) => {
      const copy = new Map(current)
      if (copy.has(key)) {
        copy.delete(key)
      } else {
        copy.set(key, '')
      }
      return copy
    })
  }

  const rename = (key: string, value: string) => {
    setReplaced((current) => new Map(current).set(key, value))
  }

  const teamNames = mode === 'copy' && divisions ? finalTeamNames(divisions, replacements) : []
  const canNext = step === 1 ? name.trim() !== '' && date !== '' : step === 2 ? !unfilled : true

  return (
    <div className="adm-narrow adm-stack-lg">
      <ol className="adm-steps">
        {steps.map((label, index) => (
          <li
            key={label}
            aria-current={index + 1 === step ? 'step' : undefined}
            className={index + 1 < step ? 'adm-step done' : 'adm-step'}
          >
            {label}
          </li>
        ))}
      </ol>
      <span className="adm-kicker">ステップ {step} / 3</span>

      {step === 1 ? (
        <>
          <section>
            <h2 className="adm-block-title">作り方</h2>
            <div className="adm-radios">
              {sources.length > 0 ? (
                <Radio
                  checked={mode === 'copy'}
                  onSelect={() => setMode('copy')}
                  label="前回の大会をコピーする"
                  sub="部門、大会の形、時間割の設定、お知らせ欄、バナーを引き継ぎます。チームは次で入れ替えられます。"
                />
              ) : null}
              <Radio
                checked={mode === 'blank'}
                onSelect={() => setMode('blank')}
                label="白紙から作る"
                sub="部門と大会の形を一から決めます。"
              />
            </div>
          </section>
          {mode === 'copy' ? (
            <section>
              <h2 className="adm-block-title">コピー元</h2>
              <div className="adm-radios">
                {sources.map((item) => (
                  <Radio
                    key={item.id}
                    checked={item.id === sourceId}
                    onSelect={() => setSourceId(item.id)}
                    label={item.name}
                    sub={item.meta}
                  />
                ))}
              </div>
            </section>
          ) : null}
          <label className="adm-field">
            <span>大会名</span>
            <input type="text" value={name} onChange={(event) => setName(event.target.value)} />
          </label>
          <label className="adm-field">
            <span>日付</span>
            <input type="date" value={date} onChange={(event) => setDate(event.target.value)} />
          </label>
        </>
      ) : null}

      {step === 2 && divisions ? (
        <section className="adm-stack">
          <p className="adm-muted">
            前回のチームが入っています。出ないチームは「入れ替える」で新しいチーム名にします。チームの数と組分けは前回と同じです。
          </p>
          {divisions.map((division, divisionIndex) => (
            <div key={division.name}>
              {divisions.length > 1 ? <h2 className="adm-block-title">{division.name}</h2> : null}
              <ul className="adm-rows">
                {division.teams.map((team, teamIndex) => {
                  const key = `${divisionIndex}:${teamIndex}`
                  const value = replaced.get(key)
                  return (
                    <li key={key} className="adm-team-row">
                      <span className="adm-num adm-team-no">{teamIndex + 1}</span>
                      {value === undefined ? (
                        <span className="adm-team-name">{team}</span>
                      ) : (
                        <input
                          type="text"
                          className="adm-team-input"
                          value={value}
                          placeholder={`新しいチーム名(${team} の代わり)`}
                          onChange={(event) => rename(key, event.target.value)}
                        />
                      )}
                      <button type="button" className="adm-link-button" onClick={() => toggle(key)}>
                        {value === undefined ? '入れ替える' : '元に戻す'}
                      </button>
                    </li>
                  )
                })}
              </ul>
            </div>
          ))}
          <span className="adm-muted">
            {divisions.reduce((sum, division) => sum + division.teams.length, 0)}チーム ・ 入れ替え{' '}
            {replacements.filter((item) => item.name.trim() !== '').length}
          </span>
        </section>
      ) : null}

      {step === 3 ? (
        <section className="adm-stack">
          <dl className="adm-summary">
            <dt>大会名</dt>
            <dd>{name}</dd>
            <dt>日付</dt>
            <dd>{date}</dd>
            <dt>元にした大会</dt>
            <dd>{mode === 'copy' && source ? source.name : '白紙から'}</dd>
            {mode === 'copy' ? (
              <>
                <dt>チーム</dt>
                <dd>{teamNames.join('、')}</dd>
              </>
            ) : null}
          </dl>
          <p className="adm-muted">
            公開すると、閲覧ページの大会一覧に「エントリー受付中」で出ます。
            {mode === 'blank' ? '部門と大会の形は、作ったあとの大会ホームから決めます。' : ''}
          </p>
        </section>
      ) : null}

      {error !== '' ? (
        <p className="adm-notice error" role="alert">
          {error}
        </p>
      ) : null}

      <form action={createFromWizardAction} className="adm-wizard-actions">
        <input type="hidden" name="back" value="/admin/tournaments/new" />
        <input type="hidden" name="mode" value={mode} />
        <input type="hidden" name="sourceId" value={sourceId} />
        <input type="hidden" name="nameJa" value={name} />
        <input type="hidden" name="startDate" value={date} />
        <input type="hidden" name="replacements" value={JSON.stringify(replacements)} />
        {step > 1 ? (
          <button
            type="button"
            className="adm-link-button"
            onClick={() => setStep(step === 3 && mode === 'blank' ? 1 : step - 1)}
          >
            戻る
          </button>
        ) : null}
        {step === 3 ? (
          <>
            <button type="submit" name="publish" value="" className="adm-link-button">
              下書きのまま保存
            </button>
            <button type="submit" name="publish" value="1" className="adm-primary">
              公開する
            </button>
          </>
        ) : (
          <button
            type="button"
            className="adm-primary"
            disabled={!canNext || loading}
            onClick={next}
          >
            {loading ? '読み込み中…' : '次へ'}
          </button>
        )}
      </form>
    </div>
  )
}

function Radio({
  checked,
  onSelect,
  label,
  sub,
}: {
  checked: boolean
  onSelect: () => void
  label: string
  sub: string
}) {
  return (
    <button type="button" className="adm-radio" aria-pressed={checked} onClick={onSelect}>
      <span className="adm-radio-ring" aria-hidden="true" />
      <span className="adm-radio-text">
        <span className="adm-radio-label">{label}</span>
        <span className="adm-radio-sub">{sub}</span>
      </span>
    </button>
  )
}
