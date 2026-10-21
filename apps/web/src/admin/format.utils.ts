import type { GroupState, Proposal, Sport } from '@/api/format.types'

// 大会の形のウィザードと本部の進行画面で使う、表示と入力の整形(Next.js と api に依存しない)。

export const sportOptions: ReadonlyArray<{ value: Sport; label: string }> = [
  { value: 'four', label: '4on4' },
  { value: 'seven', label: '7 人制' },
  { value: 'beach', label: 'ビーチ' },
]

export const tiebreakOptions = [
  { value: 'current', label: '現行方式(勝ち点 → 直接対決 → 得失点差 → 得点)' },
  { value: 'current_no_h2h', label: '現行方式から直接対決を外す' },
  { value: 'jfda', label: 'JFDA 競技規則(勝利数 → 該当チーム間 → 棄権数 …)' },
] as const

/** 候補の要約「この形で 24 試合、12:40 終了、全 12 位まで決まります」(2 日目なら「2 日目の 15:00 終了」) */
export function proposalSentence(proposal: Proposal, teams: number): string {
  const day = proposal.endDay > 1 ? `${proposal.endDay} 日目の ` : ''
  const head = `この形で ${proposal.matches} 試合、${day}${proposal.end} 終了`
  if (proposal.unique === teams) {
    return `${head}、全 ${teams} 位まで決まります`
  }
  if (proposal.ranked === teams) {
    return `${head}、全チームの順位が決まります(${teams - proposal.unique} チームは同順位)`
  }
  return `${head}、${teams} チーム中 ${proposal.ranked} チームの順位が決まります`
}

/** 配列の 2 つの位置を入れ替えた新しい配列(シード順の並べ替えとプール間の入れ替え) */
export function swapAt<T>(list: readonly T[], a: number, b: number): T[] {
  const out = [...list]
  const first = out[a]
  const second = out[b]
  if (first === undefined || second === undefined) {
    return out
  }
  out[a] = second
  out[b] = first
  return out
}

/** from の要素を to の位置へ動かした新しい配列(シード順のドラッグ) */
export function moveTo<T>(list: readonly T[], from: number, to: number): T[] {
  const out = [...list]
  const [item] = out.splice(from, 1)
  if (item === undefined) {
    return [...list]
  }
  out.splice(to, 0, item)
  return out
}

/** 遅れの分数を 5 分単位に切り上げる(「このコートを N 分遅らせる」の初期値) */
export function roundUpToFive(minutes: number): number {
  return Math.max(5, Math.ceil(minutes / 5) * 5)
}

export const groupStateLabels: Record<GroupState, string> = {
  waiting: 'チーム待ち',
  playing: '試合中',
  tied: '同順位が未決',
  ready: '確定待ち',
  confirmed: '確定済み',
  finished: '終了',
}

/** 未決の同順位の決め方の選択肢(group_rank_decisions.reason) */
export const decisionReasons = [
  'ブリックマーク投げ',
  'ディスクフリップ',
  'じゃんけん',
  '運営の協議',
]
