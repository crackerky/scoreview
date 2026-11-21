import type { EntryQuestion } from '@/api/entry.types'
import { entryLimits } from '@/api/entry.types'
import { answerFieldName } from '@/entry/entry.utils'

// 質問 1 つの答えの欄。欄の名前は a{質問 ID}(entry.utils の draftFromForm と entryLink.utils の
// joinFromForm が読む)。管理画面の代理入力では必須の印を見出しに付けるだけにし、答えの検査は api に
// 任せる(必須の答えが無ければ 400 の文が出る)。本人がスマートフォンで答える公開側では、送ってから
// 断られるより先に気付けるよう、required で送る前にも止める。

type Props = {
  question: EntryQuestion
  /** 欄の外側のクラス(管理画面 adm-field、公開側 entry-field) */
  fieldClass?: string
  /** 必須の質問をブラウザでも止めるか(公開側) */
  enforceRequired?: boolean
  defaultValue?: string
}

export function EntryAnswerField({
  question,
  fieldClass = 'adm-field',
  enforceRequired = false,
  defaultValue = '',
}: Props) {
  const name = answerFieldName(question.id)
  const label = `${question.label}${question.required ? '(必須)' : ''}`
  const required = enforceRequired && question.required
  switch (question.kind) {
    case 'long_text':
      return (
        <label className={fieldClass}>
          <span>{label}</span>
          <textarea
            name={name}
            rows={4}
            maxLength={entryLimits.maxLongText}
            required={required}
            defaultValue={defaultValue}
          />
        </label>
      )
    case 'choice':
    case 'yes_no': {
      const options =
        question.kind === 'yes_no'
          ? [
              { value: 'yes', label: 'はい' },
              { value: 'no', label: 'いいえ' },
            ]
          : question.options.map((option) => ({ value: option, label: option }))
      return (
        <label className={fieldClass}>
          <span>{label}</span>
          <select name={name} defaultValue={defaultValue} required={required}>
            <option value="">{required ? '選んでください' : '(答えない)'}</option>
            {options.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
        </label>
      )
    }
    case 'number':
      return (
        <label className={fieldClass}>
          <span>{label}</span>
          <input
            type="number"
            name={name}
            step={1}
            inputMode="numeric"
            required={required}
            defaultValue={defaultValue}
          />
        </label>
      )
    case 'short_text':
      return (
        <label className={fieldClass}>
          <span>{label}</span>
          <input
            type="text"
            name={name}
            maxLength={entryLimits.maxShortText * 2}
            required={required}
            defaultValue={defaultValue}
          />
        </label>
      )
  }
}
