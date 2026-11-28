import 'server-only'
import type { z } from 'zod'
import { apiFetch } from './client'
import {
  type EntryAction,
  type EntryDraftInput,
  type EntryPersonalAsk,
  type EntryQuestionInput,
  entryAdminDetailSchema,
  entryExportSchema,
  entryListSchema,
  entryPersonalAskSchema,
  entryQuestionListSchema,
} from './entry.types'

// docs/api.md「申込」を呼ぶ関数。管理者だけ。応答は zod で parse して返す。

const admin = '/api/v1/admin'

type Method = 'GET' | 'POST' | 'PUT'

function call<T>(
  token: string,
  method: Method,
  path: string,
  schema: z.ZodType<T>,
  body?: unknown,
) {
  return apiFetch(path, schema, { method, body, sessionToken: token })
}

export const entries = {
  questions: (token: string, tournamentId: number) =>
    call(
      token,
      'GET',
      `${admin}/tournaments/${tournamentId}/entry-questions`,
      entryQuestionListSchema,
    ),
  saveQuestions: (token: string, tournamentId: number, questions: EntryQuestionInput[]) =>
    call(
      token,
      'PUT',
      `${admin}/tournaments/${tournamentId}/entry-questions`,
      entryQuestionListSchema,
      { questions },
    ),
  /** メンバー本人に聞く項目(性別、生年月日)を決める */
  savePersonal: (token: string, tournamentId: number, ask: EntryPersonalAsk) =>
    call(
      token,
      'PUT',
      `${admin}/tournaments/${tournamentId}/entry-personal`,
      entryPersonalAskSchema,
      ask,
    ),
  list: (token: string, tournamentId: number) =>
    call(token, 'GET', `${admin}/tournaments/${tournamentId}/entries`, entryListSchema),
  /** 電話や紙で受けた申込を運営が代わりに入れる */
  create: (token: string, tournamentId: number, body: EntryDraftInput) =>
    call(
      token,
      'POST',
      `${admin}/tournaments/${tournamentId}/entries`,
      entryAdminDetailSchema,
      body,
    ),
  /** 書き出し(CSV)用に、質問と申込の全部を中身つきで読む */
  export: (token: string, tournamentId: number) =>
    call(token, 'GET', `${admin}/tournaments/${tournamentId}/entries/export`, entryExportSchema),
  get: (token: string, id: number) =>
    call(token, 'GET', `${admin}/entries/${id}`, entryAdminDetailSchema),
  decide: (token: string, id: number, action: EntryAction, note: string) =>
    call(token, 'POST', `${admin}/entries/${id}/decision`, entryAdminDetailSchema, {
      action,
      note,
    }),
}
