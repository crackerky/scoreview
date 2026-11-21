import 'server-only'
import { apiFetch } from './client'
import {
  type EntryDraftInput,
  type EntryJoinInput,
  type EntryResubmitInput,
  entryDetailSchema,
  entryLinkMemberSchema,
  entryLinkSchema,
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
  join: (token: string, body: EntryJoinInput) =>
    apiFetch(`${base}/${encodeURIComponent(token)}/member`, entryLinkMemberSchema, {
      method: 'POST',
      body,
    }),
}
