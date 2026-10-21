import type { AdminEvent, MatchDetail, ScorerClaim } from '@/api/admin.types'
import type { Side } from '@/api/common.types'
import { sideName } from './labels.utils'
import type { ScoreOp } from './scoreOps.utils'

// 記録係のスコア入力画面(ScoreKeeper.tsx)の、画面の部品から切り離せる判断。

/**
 * 取り消しの連打を無視する間隔(ミリ秒)。1 回目の取り消しで画面が変わり、2 回目の指が正しい得点を
 * 消すのを防ぐ(レビュー H1: 2-1 から連打で 1-0 まで消えた)。人が「もう 1 つ消す」と決めて押し直す
 * には足りる長さにする。
 */
export const UNDO_COOLDOWN_MS = 1000

/** 取り消しを受け付けるか。lastUndoAt は前回の取り消しの時刻(ミリ秒。無ければ null) */
export function undoAllowed(lastUndoAt: number | null, now: number): boolean {
  return lastUndoAt === null || now - lastUndoAt >= UNDO_COOLDOWN_MS
}

/** 取り消した得点経過を元に戻す操作(同じ内容の記録を新しい操作 ID で足し直す) */
export function restoreOp(event: AdminEvent, opId: string): ScoreOp {
  return {
    type: 'event.add',
    opId,
    kind: event.kind,
    side: event.side,
    elapsedSec: event.elapsedSec,
    scorerId: event.scorer?.id ?? null,
    assistId: event.assist?.id ?? null,
    callahan: event.callahan,
    winnerSide: null,
    expectedScore: null,
  }
}

/**
 * 試合時間を指定していないディビジョンで使う試合時間(分)。参照サイトの最長の枠 90 分
 * (store/format_state.go の maxSlotMinutes の根拠と同じ)。
 */
export const DEFAULT_MATCH_MINUTES = 90
/** 経過時間が試合時間の何倍を超えたら「大きく超えた」として確認するか(時計の止め忘れ、試合の取り違え) */
export const LONG_ELAPSED_FACTOR = 1.5

/** 経過時間(秒)が試合時間を大きく超えているか */
export function isLongElapsed(elapsedSec: number | null, matchMinutes: number | null): boolean {
  if (elapsedSec === null) {
    return false
  }
  return elapsedSec > (matchMinutes ?? DEFAULT_MATCH_MINUTES) * 60 * LONG_ELAPSED_FACTOR
}

/** 終了した試合の勝者の文(画面に出す)。null なら勝者を出さない(未消化など) */
export function winnerLine(detail: MatchDetail): string | null {
  const { match } = detail
  if (match.status !== 'finished') {
    return null
  }
  if (match.resultType === 'double_forfeit') {
    return '両チーム不戦敗'
  }
  if (match.resultType === 'unplayed') {
    return null
  }
  const left = match.left.score
  const right = match.right.score
  const tied = left !== null && right !== null && left === right
  let winner: Side | null = match.winnerSide
  if (winner === null && left !== null && right !== null && left !== right) {
    winner = left > right ? 'left' : 'right'
  }
  if (winner === null) {
    return tied ? '引き分け' : null
  }
  return `勝者: ${sideName(match[winner])}${tied ? '(同点)' : ''}`
}

/** この端末から見た試合の記録担当。other は他の端末が記録中(引き継ぐまで記録させない) */
export type ScorerState =
  | { kind: 'mine' }
  | { kind: 'free' }
  | { kind: 'other'; name: string; sameUser: boolean }

/**
 * api が返した記録担当(期限内のもの)を、この端末から見た状態にする。userId は画面を開いている利用者。
 * 他の端末の担当でも期限を過ぎていれば free(延長が止まった端末を待たない)。
 */
export function scorerStateOf(
  scorer: ScorerClaim | null,
  myDeviceId: string,
  userId: number,
  now: number,
): ScorerState {
  if (scorer === null || Date.parse(scorer.expiresAt) <= now) {
    return { kind: 'free' }
  }
  if (scorer.deviceId === myDeviceId) {
    return { kind: 'mine' }
  }
  return { kind: 'other', name: scorer.name, sameUser: scorer.userId === userId }
}

/** コートの表示。「コート1」のように既に「コート」が付いた値に重ねて付けない(レビュー L1) */
export function courtLabel(field: string | null): string {
  if (field === null || field === '') {
    return ''
  }
  return /^(コート|court)/i.test(field) ? field : `コート ${field}`
}
