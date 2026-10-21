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

/**
 * api が断った操作の知らせ。記録係がそのまま次の行動を決められる言葉にする。reason は api の
 * エラーの message。
 */
export function failureMessage(op: ScoreOp, status: number, reason: string): string {
  if (status === 409 && op.type === 'status' && op.status === 'live') {
    // 終了の取り消しで、結果のある後続の試合の対戦が変わる(api は下流の扱いの承認を求める)。
    // 記録画面では承認させず、影響を見られる本部の画面で訂正してもらう
    return `試合終了を取り消せませんでした: ${reason}。本部の訂正の画面から訂正してください。`
  }
  switch (status) {
    case 403:
      return '記録できませんでした: この試合の担当ではありません。担当を本部に確認してください。'
    case 409:
      return `記録できませんでした: 試合の記録が別の端末で変わっています(${reason})。画面を最新にしました。`
    case 404:
      return '記録できませんでした: 試合か得点経過が見つかりません(削除された可能性があります)。'
    default:
      return `記録できませんでした: 入力を確かめてください(${reason})。`
  }
}

/** コートの表示。「コート1」のように既に「コート」が付いた値に重ねて付けない(レビュー L1) */
export function courtLabel(field: string | null): string {
  if (field === null || field === '') {
    return ''
  }
  return /^(コート|court)/i.test(field) ? field : `コート ${field}`
}
