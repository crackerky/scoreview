import type { AuditEntry, AuditSnapshot } from '@/api/admin.types'
import type { Side } from '@/api/common.types'
import { formatElapsed } from './labels.utils'
import { resultTypeLabels } from './scoreOps.utils'

// 監査ログ(docs/api.md「監査ログ」)の 1 行を、本部が読んで「誰がいつ何を何から何へ」を言える文にする。
// 内部の値(status、winner_side など)はそのまま出さない。

export type AuditLine = {
  /** 何をしたか(例「得点を記録」) */
  what: string
  /** 対象(例「架空A #7 選手1」)。無ければ空文字 */
  target: string
  /** 変更前と変更後(例「2-1 試合中」→「3-1 試合中」) */
  before: string
  after: string
}

type Names = Record<Side, string>

const actionLabels: Record<AuditEntry['action'], string> = {
  'event.add': '記録',
  'event.remove': '取り消し',
  'event.update': '得点者の変更',
  'event.side': '得点のチームの訂正',
  'match.status': '試合の進行',
  'match.result': '結果の種類',
  'team.final_rank': '最終順位',
  'match.align_teams': '対戦チームの差し替え(本部)',
  'match.withdraw': '棄権の一括処理',
}

const kindLabels = { goal: '得点', timeout: 'タイムアウト', pull: 'プル', end: '試合終了' } as const

function player(value: { number: string; name: string } | null): string {
  if (value === null) {
    return 'なし'
  }
  return `${value.number ? `#${value.number} ` : ''}${value.name}`.trim() || '(削除された選手)'
}

function matchState(snapshot: AuditSnapshot, names: Names): string {
  const state = snapshot.match
  if (state === null) {
    return ''
  }
  const score =
    state.leftScore === null || state.rightScore === null
      ? 'スコアなし'
      : `${state.leftScore}-${state.rightScore}`
  const status = { scheduled: '試合前', live: '試合中', finished: '終了' }[state.status]
  const result = state.resultType === 'normal' ? '' : ` ${resultTypeLabels[state.resultType]}`
  const winner = state.winnerSide === null ? '' : ` ${names[state.winnerSide]}の勝ち`
  return `${status}${result} ${score}${winner}`
}

function eventTarget(snapshot: AuditSnapshot, names: Names): string {
  const event = snapshot.event
  if (event === null) {
    return ''
  }
  const team = event.side === null ? '' : `${names[event.side]} `
  return `${formatElapsed(event.elapsedSec)} ${team}${kindLabels[event.kind]}`
}

function scorerText(snapshot: AuditSnapshot): string {
  const event = snapshot.event
  if (event === null) {
    return ''
  }
  const assist = event.assist === null ? '' : ` ← ${player(event.assist)}`
  return `${player(event.scorer)}${assist}${event.callahan ? ' キャラハン' : ''}`
}

/** 監査ログの 1 行を文にする。names は試合の左右のチーム名 */
export function describeAudit(entry: AuditEntry, names: Names): AuditLine {
  const what = actionLabels[entry.action]
  switch (entry.action) {
    case 'event.add':
    case 'event.remove': {
      const snapshot = entry.action === 'event.add' ? entry.after : entry.before
      return {
        what,
        target: eventTarget(snapshot, names),
        before: matchState(entry.before, names),
        after: matchState(entry.after, names),
      }
    }
    case 'event.update':
      return {
        what,
        target: eventTarget(entry.after, names),
        before: scorerText(entry.before),
        after: scorerText(entry.after),
      }
    case 'event.side': {
      // 得点を相手チームの得点に付け替えた 1 つの訂正。チームとスコアの変化を並べる
      const team = (snapshot: AuditSnapshot) =>
        snapshot.event?.side ? `${names[snapshot.event.side]}の得点` : ''
      return {
        what,
        target: eventTarget(entry.before, names),
        before: `${team(entry.before)} ${matchState(entry.before, names)}`.trim(),
        after: `${team(entry.after)} ${matchState(entry.after, names)}`.trim(),
      }
    }
    case 'match.align_teams':
      // 差し替えは試合のスコアと状態を変えない(差し替えたチームは試合の画面に出ている)
      return { what, target: '', before: '', after: '' }
    case 'match.status':
    case 'match.result':
    case 'match.withdraw':
      return {
        what,
        // 補足(「得点経過(9-8)と違う最終スコア(11-9)で本部が確定」「大会の締め(未消化)」)
        target: entry.after.note ?? '',
        before: matchState(entry.before, names),
        after: matchState(entry.after, names),
      }
    case 'team.final_rank': {
      const rank = (value: number | null) => (value === null ? 'なし' : `${value} 位`)
      return {
        what,
        target: '',
        before: rank(entry.before.finalRank),
        after: rank(entry.after.finalRank),
      }
    }
  }
}

const timeFormat = new Intl.DateTimeFormat('ja-JP', {
  timeZone: 'Asia/Tokyo',
  month: 'numeric',
  day: 'numeric',
  hour: '2-digit',
  minute: '2-digit',
  second: '2-digit',
})

/** 記録した時刻(日本時間。大会の会場の時計に合わせる) */
export function formatAuditTime(at: string): string {
  return timeFormat.format(new Date(at))
}

/** 操作した人(消された利用者は「削除された利用者」) */
export function auditUser(entry: AuditEntry): string {
  if (entry.user === null) {
    return '不明'
  }
  if (entry.user.role === null) {
    return `削除された利用者 #${entry.user.id}`
  }
  return `${entry.user.name}(${entry.user.role === 'admin' ? '管理者' : '記録係'})`
}

/** 1 回の訂正(終了 → 試合中 → 終了)とみなす、同じ人の 2 つの操作の間隔の上限(ミリ秒) */
const correctionWindowMs = 60_000

function reopened(entry: AuditEntry): boolean {
  return (
    (entry.action === 'match.status' || entry.action === 'event.remove') &&
    entry.before.match?.status === 'finished' &&
    entry.after.match?.status === 'live'
  )
}

function refinished(entry: AuditEntry): boolean {
  return (
    (entry.action === 'match.status' || entry.action === 'event.add') &&
    entry.before.match?.status === 'live' &&
    entry.after.match?.status === 'finished'
  )
}

/**
 * 監査ログの行を履歴の表の行にする。同じ人が 1 分以内に「終了 → 試合中」「試合中 → 終了」と続けた
 * 2 行は、1 回の訂正として「結果の訂正 終了 11-6 → 終了 11-9」の 1 行にまとめる(レビュー L2)。
 * entries は新しい順(api の順)。
 */
export function auditLines(
  entries: readonly AuditEntry[],
  names: Names,
): Array<{ entry: AuditEntry; line: AuditLine }> {
  const out: Array<{ entry: AuditEntry; line: AuditLine }> = []
  for (let index = 0; index < entries.length; index++) {
    const newer = entries[index]
    const older = entries[index + 1]
    if (newer === undefined) {
      continue
    }
    const sameUser = older !== undefined && older.user?.id === newer.user?.id
    const close =
      older !== undefined &&
      Math.abs(new Date(newer.at).getTime() - new Date(older.at).getTime()) <= correctionWindowMs
    if (older !== undefined && sameUser && close && refinished(newer) && reopened(older)) {
      out.push({
        entry: newer,
        line: {
          what: '結果の訂正',
          target: newer.after.note ?? '',
          before: matchState(older.before, names),
          after: matchState(newer.after, names),
        },
      })
      index++
      continue
    }
    out.push({ entry: newer, line: describeAudit(newer, names) })
  }
  return out
}
