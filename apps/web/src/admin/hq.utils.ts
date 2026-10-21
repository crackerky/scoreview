import type { Withdrawal } from '@/api/format.types'

// 本部の進行画面と、そこから開く確認画面(棄権、大会の締め、遅れの調整)の文の組み立て。Next.js と api に
// 依存しない(テストで確かめる)。

/** スマートフォンで本部画面を分けるタブ(レビュー M7)。パソコンでは全部を縦に並べる */
export const hqTabs = [
  { id: 'todo', label: 'いまやること' },
  { id: 'matches', label: '進行中と次' },
  { id: 'groups', label: 'プールとトーナメント' },
  { id: 'delay', label: '遅れの調整' },
] as const

export type HqTab = (typeof hqTabs)[number]['id']

/** クエリの tab を読む。知らない値は「いまやること」 */
export function hqTabOf(value: unknown): HqTab {
  const found = hqTabs.find((tab) => tab.id === value)
  return found ? found.id : 'todo'
}

const clockFormat = new Intl.DateTimeFormat('ja-JP', {
  timeZone: 'Asia/Tokyo',
  hour: '2-digit',
  minute: '2-digit',
  second: '2-digit',
  hourCycle: 'h23',
})

/** 「最終更新 13:42:10」の時刻(日本時間。会場の時計に合わせる) */
export function updatedAtText(at: Date): string {
  return `最終更新 ${clockFormat.format(at)}`
}

/** 遅れの調整の確認「コート 3 の 8 試合を 15 分ずらします」 */
export function shiftSentence(court: string | null, moved: number, minutes: number): string {
  const where = court === null ? '全コートの' : `コート ${court} の`
  const how = minutes > 0 ? `${minutes} 分遅らせます` : `${-minutes} 分早めます`
  return `${where}まだ始まっていない ${moved} 試合を ${how}`
}

/** 棄権の確認「残り 2 試合(C3、G3)を相手の不戦勝にします」 */
export function withdrawalSentence(withdrawal: Withdrawal): string {
  const targets = withdrawal.targets
  if (targets.length === 0) {
    return '今すぐ記録する試合はありません'
  }
  const codes = targets.map((match) => match.code).join('、')
  const doubles = targets.filter((match) => match.result === 'double_forfeit').length
  const tail =
    doubles === 0
      ? '相手の不戦勝にします'
      : `相手の不戦勝にします(相手も棄権した ${doubles} 試合は両チーム不戦敗)`
  return `残り ${targets.length} 試合(${codes})を${tail}`
}

/** 不戦勝のスコアの説明。規定スコアが無ければ勝敗だけを記録する(スコアは空のまま) */
export function forfeitScoreText(score: { winner: number; loser: number } | null): string {
  return score === null
    ? '不戦勝の規定スコアが設定されていないので、勝敗だけを記録します(スコアは空のまま)'
    : `不戦勝は規定スコア ${score.winner}-${score.loser} で記録します`
}

/** 中断から何分たったかを、再開でずらす分数(5 分単位に切り上げ)にする */
export function suspendedMinutes(since: string, now: Date): number {
  const elapsed = (now.getTime() - new Date(since).getTime()) / 60000
  return elapsed <= 0 ? 0 : Math.ceil(elapsed / 5) * 5
}
