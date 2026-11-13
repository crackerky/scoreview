import { loadAdmin, requireUser } from '@/admin/session'
import { tournaments } from '@/api/admin'
import { entries } from '@/api/entry'
import { type EntryQuestion, entryKinds, entryScopes } from '@/api/entry.types'
import type { IdPageProps } from '@/app/_lib/pageProps'
import { saveEntryQuestionsAction } from '@/app/admin/_actions/entry'
import { readAdminPage } from '@/app/admin/_lib/page'
import { AdminShell, Hidden } from '@/components/admin/AdminParts'
import { entryPaths } from '@/entry/entry.paths'
import { kindLabels, scopeLabels } from '@/entry/entry.utils'

// 申込で聞く質問(Lイベントの「アンケート」と参加フォームの追加項目にあたる)。部門、チーム名、代表の
// 連絡先、メンバーの名前と背番号はどの大会でも聞くので、ここには大会ごとに足す質問だけを置く。
// 1 行が 1 問。下の空の行に書くと足し、文言を消すか「消す」に印を付けると消す。順は「順」の数で並べ替える。

/** 1 回に足せる空の行の数 */
const blankRows = 3

export default async function EntryQuestionsPage(props: IdPageProps) {
  const { id, notice } = await readAdminPage(props)
  const { user, token } = await requireUser('admin')
  const [tournament, list] = await loadAdmin(() =>
    Promise.all([tournaments.get(token, id), entries.questions(token, id)]),
  )
  const rows: Array<EntryQuestion | null> = [
    ...list.questions,
    ...Array.from({ length: blankRows }, () => null),
  ]
  const formId = 'entry-questions-form'
  return (
    <AdminShell
      user={user}
      title="申込で聞く質問"
      kicker="チーム名、代表の連絡先、メンバーの名前と背番号は、どの大会でも聞きます"
      notice={notice}
      crumbs={[
        { href: `/admin/tournaments/${id}`, label: tournament.name.ja },
        { href: entryPaths.list(id), label: '申込' },
      ]}
      primary={{ label: '保存する', form: formId }}
    >
      <form id={formId} action={saveEntryQuestionsAction} className="adm-stack">
        <Hidden values={{ back: entryPaths.questions(id), tournamentId: id, rows: rows.length }} />
        {rows.map((question, index) => (
          <QuestionRow
            // 空の行は質問が無いので並びの位置で見分ける(保存すると画面ごと描き直す)
            key={question?.id ?? `new-${index}`}
            index={index}
            question={question}
          />
        ))}
      </form>
    </AdminShell>
  )
}

function QuestionRow({ index, question }: { index: number; question: EntryQuestion | null }) {
  const name = (key: string) => `q${index}.${key}`
  // 答えのある質問は、答え方と答える人を変えると集めた答えの意味が変わるので選ばせない
  const locked = question !== null && question.answered > 0
  return (
    <fieldset className="adm-card">
      <legend>
        {question === null ? '質問を足す' : `質問 ${index + 1}`}
        {locked ? `(答え ${question.answered} 件)` : ''}
      </legend>
      {question === null ? null : <input type="hidden" name={name('id')} value={question.id} />}
      <div className="adm-grid">
        <label className="adm-field">
          <span>質問の文</span>
          <input type="text" name={name('label')} defaultValue={question?.label ?? ''} />
        </label>
        <ChoiceField
          label="答える人"
          name={name('scope')}
          locked={locked}
          value={question?.scope ?? 'team'}
          options={entryScopes.map((scope) => ({ value: scope, label: scopeLabels[scope] }))}
        />
        <ChoiceField
          label="答え方"
          name={name('kind')}
          locked={locked}
          value={question?.kind ?? 'short_text'}
          options={entryKinds.map((kind) => ({ value: kind, label: kindLabels[kind] }))}
        />
        <label className="adm-field">
          <span>順</span>
          <input type="number" name={name('order')} defaultValue={index + 1} step={1} />
        </label>
      </div>
      <label className="adm-field">
        <span>選択肢(答え方が「選択」のとき。1 行に 1 つ)</span>
        <textarea
          name={name('options')}
          rows={3}
          defaultValue={question?.options.join('\n') ?? ''}
        />
      </label>
      <div className="adm-actions">
        <label className="adm-check">
          <input
            type="checkbox"
            name={name('required')}
            defaultChecked={question?.required ?? false}
          />
          <span>必須にする</span>
        </label>
        {question === null ? null : (
          <label className="adm-check">
            <input type="checkbox" name={name('remove')} />
            <span>この質問を消す{locked ? `(答え ${question.answered} 件も消えます)` : ''}</span>
          </label>
        )}
      </div>
    </fieldset>
  )
}

/** 選ぶ欄。locked なら選ばせず、今の値を hidden で送って呼び名だけを出す */
function ChoiceField({
  label,
  name,
  locked,
  value,
  options,
}: {
  label: string
  name: string
  locked: boolean
  value: string
  options: Array<{ value: string; label: string }>
}) {
  if (locked) {
    return (
      <div className="adm-field">
        <span>{label}</span>
        <input type="hidden" name={name} value={value} />
        <span>{options.find((option) => option.value === value)?.label ?? value}</span>
      </div>
    )
  }
  return (
    <label className="adm-field">
      <span>{label}</span>
      <select name={name} defaultValue={value}>
        {options.map((option) => (
          <option key={option.value} value={option.value}>
            {option.label}
          </option>
        ))}
      </select>
    </label>
  )
}
