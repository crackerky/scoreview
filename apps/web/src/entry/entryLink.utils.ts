import { validationParts } from '@/admin/errorText.utils'
import type { EntryDetail, EntryJoinInput, EntryQuestion } from '@/api/entry.types'
import { entryLimits } from '@/api/entry.types'
import { answerFieldName, answersFromForm, type FormFields, formText } from './entry.utils'

// 本人用のリンクで開く画面(/entry/{合言葉})の、フォームの読み方と、参加者に見せる文。

/** 合言葉の形(api の domain.NewToken。base64url の 43 文字)。違う形は api に聞かずに「見つからない」 */
export function isLinkToken(token: string): boolean {
  return /^[A-Za-z0-9_-]{43}$/.test(token)
}

/**
 * 招待リンクをチームの LINE グループに送るための、LINE の「送る先を選ぶ」画面の URL
 * (https://developers.line.biz/ja/docs/messaging-api/using-line-url-scheme/ のテキストメッセージを送る)。
 */
export function shareToLineUrl(team: string, inviteUrl: string): string {
  const text = `${team} のメンバー登録のお願いです。\n下のリンクを開いて、そのまま「参加」のメッセージを送ってください。\n${inviteUrl}`
  return `https://line.me/R/share?text=${encodeURIComponent(text)}`
}

/**
 * メンバー登録のフォームを読む。member は代表が書いた行の ID か、new(名簿に無い)。もう登録した人は
 * 自分の行の ID を送る(api は登録済みの人の行と照らす)。答えは a{質問 ID} の欄で、空の答えは送らない。
 */
export function joinFromForm(
  form: FormFields,
  questions: readonly EntryQuestion[],
): EntryJoinInput {
  const picked = Number(formText(form, 'member'))
  return {
    memberId: Number.isInteger(picked) && picked > 0 ? picked : null,
    name: formText(form, 'name'),
    number: formText(form, 'number'),
    answers: answersFromForm(form, questions),
  }
}

/**
 * 送り直しのフォームの registered の欄(フォームを開いたときに本人が登録していたメンバーの行の ID)を読む。
 * 数でない値は捨てる。
 */
export function registeredIdsFromForm(form: FormFields): number[] {
  const value = form.registered
  const values = value === undefined ? [] : Array.isArray(value) ? value : [value]
  return values.map(Number).filter((id) => Number.isInteger(id) && id > 0)
}

/**
 * 公開側のフォームの送信の結果。error は画面に出す文。成功したときは同じ画面へ移るので使わない。
 */
export type LinkFormState = { error: string | null }

/** メンバー表の欄(1 行 1 人、背番号は名前の前。entry.utils の parseMembers が読む形)に戻す */
export function membersText(members: ReadonlyArray<{ name: string; number: string }>): string {
  return members
    .map((member) => (member.number ? `${member.number} ${member.name}` : member.name))
    .join('\n')
}

/**
 * 差し戻された申込を、送り直しの欄に入れる値にする。メンバー表には、本人がまだ登録していない人だけを
 * 入れる(本人が登録した人は本人が直すので、代表の欄には出さない。api もそう扱う)。
 */
export function resubmitValues(entry: EntryDetail): Record<string, string> {
  const values: Record<string, string> = {
    divisionId: String(entry.division.id),
    fullName: entry.fullName,
    shortName: entry.shortName,
    repName: entry.repName,
    repPhone: entry.repPhone,
    repEmail: entry.repEmail,
    members: membersText(entry.memberList.filter((member) => !member.registered)),
  }
  for (const answer of entry.answers) {
    values[answerFieldName(answer.questionId)] = answer.value
  }
  return values
}

/**
 * リンクが無いか期限が切れたときの案内(送信のエラーと、app/entry/[token]/not-found.tsx の画面)。代表と
 * メンバーで、新しいリンクの受け取り方が違う。
 */
export const linkUnusableText =
  'リンクの期限が切れたか、リンクが正しくありません。代表の方は公式 LINE にもう一度「エントリー」と送ると、新しいリンクが届きます。メンバーの方は、代表の方から届いた招待リンクを開き直してください。'

/** 項目の参加者向けの呼び方 */
function fieldName(field: string): string {
  const member = /^members\[(\d+)\]\.(name|number)$/.exec(field)
  if (member) {
    return `メンバーの ${Number(member[1]) + 1} 人目の${member[2] === 'name' ? '名前' : '背番号'}`
  }
  const labels: Readonly<Record<string, string>> = {
    fullName: 'チーム名(正式)',
    shortName: 'チーム名(短縮)',
    repName: '代表の名前',
    repPhone: '電話番号',
    repEmail: 'メールアドレス',
    name: '名前',
    number: '背番号',
  }
  return labels[field] ?? '入力'
}

function badRequestText(message: string): string {
  const { field, reason } = validationParts(message)
  if (/^answers\[\d+\]$/.test(field)) {
    if (/is required/.test(reason)) {
      return '必須の質問に答えてください'
    }
    return /whole number/.test(reason)
      ? '数の質問には整数で答えてください'
      : '質問への答えを確かめてください'
  }
  if (field === 'divisionId') {
    return '部門を選んでください'
  }
  if (field === 'members') {
    return `メンバーは ${entryLimits.maxMembers} 人までです`
  }
  if (field === 'memberId') {
    return '名簿が変わりました。画面を開き直してから、もう一度選んでください'
  }
  if (/another member/.test(reason)) {
    return `${fieldName(field)}が、同じチームのほかの人と重なっています`
  }
  if (/line breaks/.test(reason)) {
    return `${fieldName(field)}は、改行を入れずに 1 行で書いてください`
  }
  const length = /at most (\d+) characters/.exec(reason)
  if (length) {
    return `${fieldName(field)}は ${length[1]} 文字までです`
  }
  if (/is required/.test(reason)) {
    return `${fieldName(field)}を入れてください`
  }
  return '入力した内容を確かめてください'
}

/** conflict の理由(api の英語)と、参加者向けの文 */
const conflictTexts: ReadonlyArray<{ reason: RegExp; text: string }> = [
  {
    reason: /a team with the same name has already applied/,
    text: 'この部門には、同じチーム名の申込がもうあります。チーム名を確かめてください',
  },
  { reason: /not accepting entries/, text: 'この大会の申込は締め切りました' },
  { reason: /tournament has finished/, text: 'この大会は終わりました' },
  {
    reason: /someone else has already registered/,
    text: 'その名前では、ほかの人がもう登録しました。名簿から選び直すか、「名簿に自分がいない」を選んでください',
  },
  { reason: /has been cancelled/, text: 'このチームの申込は取り消されています' },
  {
    reason: /members of the entry changed/,
    text: 'この間にメンバーが本人登録をしました。画面を開き直してから、もう一度直してください',
  },
  {
    reason: /changed by someone else|only a returned entry/,
    text: 'この間に申込の状態が変わりました。画面を開き直して、今の状態を確かめてください',
  },
]

/**
 * api のエラーを、本人用のリンクの画面に出す参加者向けの 1 文にする。運営向けの変換表
 * (admin/errorText.utils)は管理画面の言葉(申込一覧など)を使うので、ここでは使わない。
 */
export function participantErrorText(code: string, message: string): string {
  switch (code) {
    case 'bad_request':
      return badRequestText(message)
    case 'conflict':
      return (
        conflictTexts.find((rule) => rule.reason.test(message))?.text ??
        'ほかの操作と重なりました。画面を開き直してから、もう一度送ってください'
      )
    case 'not_found':
      return linkUnusableText
  }
  return '混み合っています。少し待ってから、もう一度送ってください'
}
