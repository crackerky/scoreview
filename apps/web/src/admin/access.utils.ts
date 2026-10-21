import type { AdminUser } from '@/api/admin.types'

// 管理画面のページを出してよいかの判定。auth/me の結果(未ログインなら null)と、ページが求める役割から決める。

/** ページが求める役割。'admin' は管理者だけ、'any' はログインしていれば誰でも(記録係を含む) */
export type Requirement = 'admin' | 'any'

export type AccessDecision =
  | { kind: 'login' } // ログイン画面へ
  | { kind: 'forbidden' } // 管理画面のトップへ戻して「権限がありません」を出す
  | { kind: 'ok'; user: AdminUser }

export function decideAccess(user: AdminUser | null, requirement: Requirement): AccessDecision {
  if (user === null) {
    return { kind: 'login' }
  }
  if (requirement === 'admin' && user.role !== 'admin') {
    return { kind: 'forbidden' }
  }
  return { kind: 'ok', user }
}

/** 記録係が大会を担当しているか(管理者は全大会) */
export function canScore(user: AdminUser, tournamentId: number): boolean {
  return user.role === 'admin' || user.tournamentIds.includes(tournamentId)
}
