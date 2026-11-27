import 'server-only'
import { loginResponseSchema } from './admin.types'
import { apiFetch } from './client'
import {
  type EntryDraftInput,
  type EntryJoinInput,
  type EntryResubmitInput,
  entryDetailSchema,
  entryLinkMemberSchema,
  entryLinkSchema,
  issuedLinkSchema,
} from './entry.types'

// docs/api.md「LINE からの申込」を呼ぶ関数。ログインは無く、パスの合言葉で本人を確かめる。

const base = '/api/v1/entry-links'

export const entryLinks = {
  get: (token: string) => apiFetch(`${base}/${encodeURIComponent(token)}`, entryLinkSchema),
  submitTeam: (token: string, body: EntryDraftInput) =>
    apiFetch(`${base}/${encodeURIComponent(token)}/team`, entryDetailSchema, {
      method: 'POST',
      body,
    }),
  /** 差し戻された申込 entryId を直して送り直す */
  resubmitTeam: (token: string, entryId: number, body: EntryResubmitInput) =>
    apiFetch(`${base}/${encodeURIComponent(token)}/team/${entryId}`, entryDetailSchema, {
      method: 'POST',
      body,
    }),
  /** 参加が決まった申込 entryId のチームの記録係を選ぶ(memberId が null なら外す) */
  chooseScorer: (token: string, entryId: number, memberId: number | null) =>
    apiFetch(`${base}/${encodeURIComponent(token)}/team/${entryId}/scorer`, entryDetailSchema, {
      method: 'POST',
      body: { memberId },
    }),
  /** チームの記録係のリンクから、記録の画面に入るセッションを発行する(応答はログインと同じ形) */
  scorerSession: (token: string) =>
    apiFetch(`${base}/${encodeURIComponent(token)}/session`, loginResponseSchema, {
      method: 'POST',
      body: {},
    }),
  /** チームを探す画面で選んだチーム entryId への、メンバー登録のリンクを発行する */
  chooseTeam: (token: string, entryId: number) =>
    apiFetch(`${base}/${encodeURIComponent(token)}/find`, issuedLinkSchema, {
      method: 'POST',
      body: { entryId },
    }),
  /** 代表が、自分の申込 entryId から、招待リンクを使わずに登録したメンバー memberId を外す */
  removeMember: (token: string, entryId: number, memberId: number) =>
    apiFetch(
      `${base}/${encodeURIComponent(token)}/team/${entryId}/members/${memberId}/remove`,
      entryDetailSchema,
      { method: 'POST', body: {} },
    ),
  join: (token: string, body: EntryJoinInput) =>
    apiFetch(`${base}/${encodeURIComponent(token)}/member`, entryLinkMemberSchema, {
      method: 'POST',
      body,
    }),
}
