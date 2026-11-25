import type {
  EntryDraftInput,
  EntryKind,
  EntryQuestion,
  EntryQuestionInput,
  EntryScope,
  EntryStatus,
  EntrySummary,
} from '@/api/entry.types'
import { entryKinds, entryScopes, entryStatuses } from '@/api/entry.types'

// 申込の管理画面の、画面の形への直し方とフォームの読み方。純粋な関数だけを置く。

export const statusLabels: Readonly<Record<EntryStatus, string>> = {
  pending: '確認待ち',
  returned: '直し待ち',
  approved: '参加決定',
  cancelled: '取消',
}

export const statusTones = {
  pending: 'warn',
  // 代表が直すのを待っている。参加決定(ok)と見分けられるよう赤にする
  returned: 'live',
  approved: 'ok',
  cancelled: 'done',
} as const satisfies Record<EntryStatus, 'live' | 'done' | 'sched' | 'warn' | 'ok'>

export const kindLabels: Readonly<Record<EntryKind, string>> = {
  short_text: '短い文',
  long_text: '長い文',
  choice: '選択',
  number: '数',
  yes_no: 'はい/いいえ',
}

export const scopeLabels: Readonly<Record<EntryScope, string>> = {
  team: 'チーム(代表が答える)',
  member: 'メンバー(各自が答える)',
}

export const channelLabels = { line: 'LINE', desk: '運営が入力' } as const

/** 状態ごとの申込の数(一覧の上の数字とタブ) */
export function countByStatus(list: readonly EntrySummary[]): Record<EntryStatus, number> {
  const counts: Record<EntryStatus, number> = { pending: 0, returned: 0, approved: 0, cancelled: 0 }
  for (const item of list) {
    counts[item.status] += 1
  }
  return counts
}

/** クエリの status を状態にする(知らない値は絞らない) */
export function parseStatus(value: string | string[] | undefined): EntryStatus | null {
  return entryStatuses.find((status) => status === value) ?? null
}

/** メンバーが LINE で登録した進み具合(「8 人中 6 人が LINE で登録済み」)。メンバーが 0 人なら空 */
export function registeredText(members: number, registered: number): string {
  return members === 0 ? 'メンバー未記入' : `${members} 人中 ${registered} 人が LINE で登録済み`
}

/** 答えの見せ方(はい/いいえを日本語に) */
export function answerText(question: Pick<EntryQuestion, 'kind'>, value: string): string {
  if (question.kind === 'yes_no') {
    return value === 'yes' ? 'はい' : value === 'no' ? 'いいえ' : value
  }
  return value
}

/**
 * メンバー表の欄(1 行 1 人)を読む。行の先頭が背番号(数字 1〜3 桁、"00" も可)なら背番号、残りを
 * 名前にする。空行は飛ばす。名前の中の空白(姓と名の間)はそのまま残す。
 */
export function parseMembers(text: string): EntryDraftInput['members'] {
  const members: EntryDraftInput['members'] = []
  for (const raw of text.split(/\r?\n/)) {
    const line = raw.trim()
    if (line === '') {
      continue
    }
    const found = /^#?([0-9]{1,3})[\s　.、,:：]+(.+)$/.exec(line)
    members.push(
      found
        ? { number: found[1] ?? '', name: (found[2] ?? '').trim() }
        : { number: '', name: line },
    )
  }
  return members
}

/** Server Action が読む、平らにしたフォーム(admin/form.utils の formToObject) */
export type FormFields = Record<string, string | string[]>

/** フォームの欄の文字の値(無いか複数なら '') */
export function formText(form: FormFields, name: string): string {
  const value = form[name]
  return typeof value === 'string' ? value : ''
}

/** 質問の答えの欄の名前(EntryAnswerField と、答えを読む関数が同じ名前を使う) */
export function answerFieldName(questionId: number): string {
  return `a${questionId}`
}

/** questions への答えを、答えの欄から読む。前後の空白を除き、空の答えは送らない */
export function answersFromForm(
  form: FormFields,
  questions: readonly EntryQuestion[],
): Array<{ questionId: number; value: string }> {
  return questions
    .map((question) => ({
      questionId: question.id,
      value: formText(form, answerFieldName(question.id)).trim(),
    }))
    .filter((answer) => answer.value !== '')
}

/**
 * 質問の設定のフォームを読む。行は q{i}.label などの欄で、ラベルが空の行と「消す」に印のある行は
 * 捨て、「順」の数の小さい順(同じなら画面の順)に並べる。選択肢は 1 行に 1 つ。
 */
export function questionsFromForm(form: FormFields, rows: number): EntryQuestionInput[] {
  const picked: Array<{ order: number; index: number; question: EntryQuestionInput }> = []
  for (let index = 0; index < rows; index++) {
    const key = (name: string) => formText(form, `q${index}.${name}`)
    const label = key('label').trim()
    if (label === '' || key('remove') !== '') {
      continue
    }
    const id = Number(key('id'))
    const scope = entryScopes.find((value) => value === key('scope')) ?? 'team'
    const kind = entryKinds.find((value) => value === key('kind')) ?? 'short_text'
    const options =
      kind === 'choice'
        ? key('options')
            .split(/\r?\n/)
            .map((option) => option.trim())
            .filter((option) => option !== '')
        : []
    const order = Number(key('order'))
    picked.push({
      order: Number.isFinite(order) && key('order').trim() !== '' ? order : index + 1,
      index,
      question: {
        id: Number.isInteger(id) && id > 0 ? id : null,
        scope,
        label,
        kind,
        required: key('required') !== '',
        options,
      },
    })
  }
  picked.sort((left, right) => left.order - right.order || left.index - right.index)
  return picked.map((item) => item.question)
}

/** 代わりに入れる申込のフォームを読む。答えは a{質問 ID} の欄で、空の答えは送らない */
export function draftFromForm(
  form: FormFields,
  questions: readonly EntryQuestion[],
): EntryDraftInput {
  return {
    divisionId: Number(formText(form, 'divisionId')),
    fullName: formText(form, 'fullName'),
    shortName: formText(form, 'shortName'),
    repName: formText(form, 'repName'),
    repPhone: formText(form, 'repPhone'),
    repEmail: formText(form, 'repEmail'),
    members: parseMembers(formText(form, 'members')),
    answers: answersFromForm(
      form,
      questions.filter((question) => question.scope === 'team'),
    ),
  }
}

/** 日本時間を足した UTC の時刻(UTC の読み方で日本の暦と時計が読める) */
const jstOffsetMs = 9 * 60 * 60 * 1000

/** 時刻(RFC 3339。申込を受けた時刻やリンクの期限)を日本時間の「11月13日 14:05」にする */
export function dateTimeText(iso: string): string {
  const jst = new Date(new Date(iso).getTime() + jstOffsetMs)
  const clock = `${String(jst.getUTCHours()).padStart(2, '0')}:${String(jst.getUTCMinutes()).padStart(2, '0')}`
  return `${jst.getUTCMonth() + 1}月${jst.getUTCDate()}日 ${clock}`
}
