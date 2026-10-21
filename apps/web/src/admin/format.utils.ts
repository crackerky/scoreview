import type { GroupState, PreviewMatch, Proposal, Sport } from '@/api/format.types'

// 大会の形のウィザードと本部の進行画面で使う、表示と入力の整形(Next.js と api に依存しない)。

/** 競技ごとの 1 試合の枠の長さの初期値(分)。運営会社の実績の最頻値(R3 5-2) */
export const defaultSlotMinutes: Record<Sport, number> = { four: 20, seven: 60, beach: 25 }

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

/** 名簿の 1 行。短縮名は画面の表で使う名前(省略すると正式名と同じ) */
export type RosterLine = { fullName: string; shortName: string }

/**
 * 名簿の貼り付けを読む。1 行 1 チームで、「正式名」か「正式名<タブ or ,>短縮名」。空行は飛ばす。
 * 表計算からの貼り付け(タブ区切り)とメモ帳からの貼り付け(カンマ区切り)の両方を受ける。
 */
export function parseRoster(text: string): RosterLine[] {
  const out: RosterLine[] = []
  for (const raw of text.split(/\r?\n/)) {
    const line = raw.trim()
    if (line === '') {
      continue
    }
    const [full = '', short = ''] = line.split(/\t|,|、/).map((part) => part.trim())
    if (full === '') {
      continue
    }
    out.push({ fullName: full, shortName: short === '' ? full : short })
  }
  return out
}

/** 候補の要約「この形で 24 試合、12:40 終了、全 12 位まで決まります」 */
export function proposalSentence(proposal: Proposal, teams: number): string {
  const head = `この形で ${proposal.matches} 試合、${proposal.end} 終了`
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

/** プレビューの時間割(時刻 × コート)。キーは `${time}|${court}` */
export type PreviewGrid = { times: string[]; courts: string[]; cells: Map<string, PreviewMatch> }

export function buildPreviewGrid(matches: readonly PreviewMatch[], courts: readonly string[]) {
  const cells = new Map<string, PreviewMatch>()
  const times = new Set<string>()
  for (const match of matches) {
    if (match.time === null || match.field === null) {
      continue
    }
    times.add(match.time)
    cells.set(`${match.time}|${match.field}`, match)
  }
  return { times: [...times].sort(), courts: [...courts], cells } satisfies PreviewGrid
}

/** 未決の同順位の決め方の選択肢(group_rank_decisions.reason) */
export const decisionReasons = [
  'ブリックマーク投げ',
  'ディスクフリップ',
  'じゃんけん',
  '運営の協議',
]
