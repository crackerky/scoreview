import type { EntryQuestion } from '@/api/entry.types'
import { entryLimits } from '@/api/entry.types'

// 質問 1 つの答えの欄。欄の名前は a{質問 ID}(entry.utils の draftFromForm が読む)。必須の印は
// 見出しに付けるだけにし、ブラウザの required は使わない(答えの検査は api の 1 か所で行う)。
export function EntryAnswerField({ question }: { question: EntryQuestion }) {
  const name = `a${question.id}`
  const label = `${question.label}${question.required ? '(必須)' : ''}`
  switch (question.kind) {
    case 'long_text':
      return (
        <label className="adm-field">
          <span>{label}</span>
          <textarea name={name} rows={4} maxLength={entryLimits.maxLongText} />
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
        <label className="adm-field">
          <span>{label}</span>
          <select name={name} defaultValue="">
            <option value="">(答えない)</option>
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
        <label className="adm-field">
          <span>{label}</span>
          <input type="number" name={name} step={1} inputMode="numeric" />
        </label>
      )
    case 'short_text':
      return (
        <label className="adm-field">
          <span>{label}</span>
          <input type="text" name={name} maxLength={entryLimits.maxShortText * 2} />
        </label>
      )
  }
}
