// api のエラー(docs/api.md「エラー」の code と message)を、管理画面に出す運営の言葉にする共通の変換表。
// 画面に「bad_request:」などの接頭辞や英語の理由をそのまま出さない(レビュー H6)。
// bad_request の message は「項目: 理由」(domain.ValidationError)、conflict は「conflict: 日本語の理由」。

type Rule = { field: RegExp; reason?: RegExp; text: string | ((match: RegExpMatchArray) => string) }

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
]

/** 項目の運営の言葉(どの規則にも当たらないときの文に使う) */
const fieldLabels: Readonly<Record<string, string>> = {
  'name.ja': '名前',
  name: '名前',
  shortName: '略称',
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

function badRequestText(message: string): string {
  const separator = message.indexOf(': ')
  const field = separator < 0 ? '' : message.slice(0, separator)
  const reason = separator < 0 ? message : message.slice(separator + 2)
  for (const rule of rules) {
    if (!rule.field.test(field)) {
      continue
    }
    const matched = rule.reason ? reason.match(rule.reason) : reason.match(/.*/)
    if (matched === null) {
      continue
    }
    return typeof rule.text === 'string' ? rule.text : rule.text(matched)
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
