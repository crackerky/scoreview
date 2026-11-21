// api のエラー(docs/api.md「エラー」の code と message)を、管理画面に出す運営の言葉にする共通の変換表。
// 画面に「bad_request:」などの接頭辞や英語の理由をそのまま出さない(レビュー H6)。
// bad_request の message は「項目: 理由」(domain.ValidationError)、conflict は「conflict: 日本語の理由」。

/** text の関数は理由に合った部分と、項目に合った部分(「members[2].name」の 2 など)を受け取る */
type Rule = {
  field: RegExp
  reason?: RegExp
  text: string | ((match: RegExpMatchArray, fieldMatch: RegExpMatchArray) => string)
}

/** 項目と理由の組から文を作る規則。上から順に最初に当てはまったものを使う */
const rules: readonly Rule[] = [
  {
    field: /^winnerSide$/,
    reason: /required for a tied match in a bracket/,
    text: 'トーナメントは引き分けにできません。勝ったチームを選んでください',
  },
  {
    field: /^winnerSide$/,
    reason: /contradicts the score/,
    text: '選んだ勝者がスコアと合いません。スコアか勝ったチームを直してください',
  },
  { field: /^winnerSide$/, reason: /is required/, text: '勝ったチームを選んでください' },
  {
    field: /^winnerSide$/,
    reason: /must be null/,
    text: 'この結果の種類では勝ったチームを選べません',
  },
  {
    field: /^winnerSide$/,
    reason: /only when finishing/,
    text: '勝ったチームは試合を終えるときだけ選べます',
  },
  {
    field: /^forfeitScore$/,
    reason: /the forfeit score is (\d+)-(\d+)/,
    text: (m) =>
      `不戦勝のスコアは大会の規定(${m[1]}-${m[2]})で記録します。スコアの欄を空にしてください`,
  },
  { field: /^score$/, reason: /required/, text: 'スコアを入れてください' },
  { field: /^score$/, reason: /0 or more/, text: 'スコアは 0 以上の数で入れてください' },
  {
    field: /^score$/,
    reason: /derived from the events/,
    text: 'この試合のスコアは得点経過から決まります。スコア入力画面で直してください',
  },
  {
    field: /^minutes$/,
    text: 'ずらす分数は -360〜360 分(0 以外)で入れてください',
  },
  { field: /^downstream$/, text: '既に結果のある下流の試合の扱いを選んでください' },
  { field: /^(teams|teamIds)$/, text: '並んだチームを、すべて 1 回ずつ選んでください' },
  { field: /^reason$/, text: '決め方(理由)を選んでください' },
  { field: /^resumeAt$/, text: '再開予定は 13:30 のように時刻で入れてください' },
  { field: /^matchIds$/, text: '対象の試合が変わりました。画面を開き直してください' },
  { field: /^clientId$/, text: '操作の番号が正しくありません。画面を開き直してください' },
  { field: /^note$/, reason: /is required/, text: '代表に直してほしいことを書いてください' },
  {
    field: /^members\[(\d+)\]\.name$/,
    text: (_, f) => `メンバー表の ${Number(f[1]) + 1} 人目の名前を確かめてください`,
  },
  {
    field: /^members\[(\d+)\]\.number$/,
    reason: /another member/,
    text: (_, f) => `メンバー表の ${Number(f[1]) + 1} 人目の背番号が、ほかの人と重なっています`,
  },
  { field: /^members$/, text: 'メンバーは 60 人までです' },
  { field: /^answers\[\d+\]$/, reason: /is required/, text: '必須の質問に答えてください' },
  { field: /^answers\[\d+\]$/, reason: /whole number/, text: '数の質問には整数で答えてください' },
  { field: /^answers\[\d+\]$/, text: '質問への答えを確かめてください' },
  {
    field: /^questions\[(\d+)\]\.options/,
    text: (_, f) =>
      `質問 ${Number(f[1]) + 1} の選択肢を確かめてください(選択式は 1 つ以上、同じ選択肢は 1 回だけ)`,
  },
  { field: /^questions\[(\d+)\]/, text: (_, f) => `質問 ${Number(f[1]) + 1} を確かめてください` },
  { field: /^questions$/, text: '質問は 30 問までです' },
  { field: /^body$/, reason: /is required/, text: '送る文を書いてください' },
  { field: /^body$/, reason: /at most/, text: '送る文は 2000 字までです' },
  { field: /^body$/, text: '送る文に使えない文字が入っています。貼り付けた文を確かめてください' },
  {
    field: /^audience$/,
    reason: /no one who can receive/,
    text: 'この条件で送れる人がいません(ブロックしている人には送れません)',
  },
  // kind と status はほかの画面でも使う項目名なので、宛先の条件の理由(reps を含む)に限る
  { field: /^(audience|kind|status)$/, reason: /reps/, text: '宛先の条件を選び直してください' },
  { field: /^requestKey$/, text: '送る画面を開き直してから送ってください' },
]

/** conflict の英語の理由を運営の言葉にする規則(申込など、理由を英語で返す口) */
const conflictRules: ReadonlyArray<{ reason: RegExp; text: string }> = [
  {
    reason: /a team with the same name has already applied/,
    text: 'この部門には同じチーム名の申込があります。申込一覧で確かめてください',
  },
  { reason: /tournament has finished/, text: '終わった大会には申込を入れられません' },
  { reason: /not accepting entries/, text: 'この大会は今、申込を受け付けていません' },
  {
    reason: /already has answers/,
    text: '答えのある質問は、答える人と答え方を変えられません。新しい質問として足してください',
  },
  {
    reason: /approved entry cannot be cancelled/,
    text: '承認した申込は取り消せません。部門のチームを棄権にしてください',
  },
  {
    reason: /changed by someone else|members of the entry changed/,
    text: 'この間に申込が変わりました。中身を確かめてから、もう一度押してください',
  },
  {
    reason: /cannot (approve|return|cancel) an entry/,
    text: 'この申込の今の状態では、その操作はできません。画面を開き直してください',
  },
  { reason: /LINE settings are missing/, text: '公式 LINE の設定が無いので送れません' },
  {
    reason: /are left this month/,
    text: '今月の残りの配信数が足りないので、送りませんでした。宛先を絞ってください',
  },
  { reason: /request key was used/, text: '送る画面を開き直してから送ってください' },
]

/** 項目の運営の言葉(どの規則にも当たらないときの文に使う) */
const fieldLabels: Readonly<Record<string, string>> = {
  'name.ja': '名前',
  name: '名前',
  shortName: '略称',
  fullName: 'チーム名(正式)',
  repName: '代表の名前',
  repPhone: '電話番号',
  repEmail: 'メールアドレス',
  divisionId: '部門',
  date: '日付',
  start: '開始時刻',
  time: '時刻',
  field: 'コート',
  code: '試合番号',
  score: 'スコア',
  minutes: '分数',
  reason: '理由',
  resumeAt: '再開予定',
  email: 'メールアドレス',
  password: 'パスワード',
}

const japanese = /[぀-ヿ一-鿿]/

/** score_mismatch の message「the score is now 11-9, not the confirmed 9-8」から今のスコア */
export function currentScoreOf(message: string): { left: number; right: number } | null {
  const found = /now (\d+)-(\d+)/.exec(message)
  if (!found) {
    return null
  }
  return { left: Number(found[1]), right: Number(found[2]) }
}

/**
 * bad_request の message(domain.ValidationError の「項目: 理由」)を項目と理由に分ける。項目の無い
 * message は全体を理由にする。
 */
export function validationParts(message: string): { field: string; reason: string } {
  const separator = message.indexOf(': ')
  return separator < 0
    ? { field: '', reason: message }
    : { field: message.slice(0, separator), reason: message.slice(separator + 2) }
}

function badRequestText(message: string): string {
  const { field, reason } = validationParts(message)
  for (const rule of rules) {
    if (!rule.field.test(field)) {
      continue
    }
    const matched = rule.reason ? reason.match(rule.reason) : reason.match(/.*/)
    if (matched === null) {
      continue
    }
    const fieldMatch = field.match(rule.field) ?? matched
    return typeof rule.text === 'string' ? rule.text : rule.text(matched, fieldMatch)
  }
  if (japanese.test(reason)) {
    return reason
  }
  const label = fieldLabels[field]
  return label ? `「${label}」の値を確かめてください` : '入力した値を確かめてください'
}

/**
 * api のエラーを運営の言葉の 1 文にする。code は docs/api.md のエラーコード、message は api の message。
 */
export function operatorErrorText(code: string, message: string): string {
  switch (code) {
    case 'bad_request':
      return badRequestText(message)
    case 'score_mismatch': {
      const current = currentScoreOf(message)
      return current
        ? `この間に結果が ${current.left}-${current.right} に変わりました。今の結果を確かめてから保存してください`
        : 'この間に結果が変わりました。今の結果を確かめてから保存してください'
    }
    case 'conflict': {
      const reason = message.replace(/^conflict: /, '')
      const known = conflictRules.find((rule) => rule.reason.test(reason))
      if (known) {
        return known.text
      }
      return japanese.test(reason)
        ? reason
        : 'ほかの操作と重なったため保存できませんでした。画面を開き直してください'
    }
    case 'not_found':
      return '対象が見つかりません。消されたか、別の大会のものです'
    case 'forbidden':
      return 'この操作をする権限がありません。担当の大会と試合を確かめてください'
    case 'unauthorized':
      return 'ログインが切れました。ログインし直してください'
    case 'too_many_requests':
      return '試行の回数が多すぎます。15 分ほど待ってからやり直してください'
    case 'unavailable':
      return '混み合っています。少し待ってから、もう一度押してください'
  }
  return japanese.test(message) ? message : '保存できませんでした。画面を開き直してください'
}
