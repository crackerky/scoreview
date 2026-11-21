import 'server-only'
import type { z } from 'zod'

/** api が返す {error:{code,message}} を表す。HTTP ステータスとコードを保持する。 */
export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly code: string,
    message: string,
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

function backendBaseUrl(): string {
  const url = process.env.BACKEND_INTERNAL_URL
  if (!url) {
    throw new Error('BACKEND_INTERNAL_URL is not set')
  }
  return url
}

function internalToken(): string {
  const token = process.env.INTERNAL_API_TOKEN
  if (!token) {
    throw new Error('INTERNAL_API_TOKEN is not set')
  }
  return token
}

type RequestOptions = {
  method?: 'GET' | 'POST' | 'PATCH' | 'PUT' | 'DELETE'
  query?: Record<string, string | undefined>
  body?: unknown
  sessionToken?: string
  /** 利用者の IP。X-Client-IP で api に渡す(docs/api.md「認証」。ログインの試行制限に使う) */
  clientIp?: string
  /**
   * 応答を Next.js のデータの使い回しに置く秒数。誰が見ても同じ公開データだけに使う(管理画面や
   * ログインの要求には付けない)。無ければ毎回 api を呼ぶ
   */
  revalidateSeconds?: number
}

/**
 * api を呼び、応答を schema で parse して返す。
 * 2xx 以外は ApiError を投げる。応答が schema に合わなければ ZodError を投げる(契約違反を握りつぶさない)。
 */
export async function apiFetch<T>(
  path: string,
  schema: z.ZodType<T>,
  options: RequestOptions = {},
): Promise<T> {
  const url = new URL(path, backendBaseUrl())
  for (const [key, value] of Object.entries(options.query ?? {})) {
    // undefined はキーごと送らない。空文字は「空を選択した」という意味があるので送る(api.md の ymd)
    if (value !== undefined) {
      url.searchParams.set(key, value)
    }
  }
  const headers: Record<string, string> = { 'X-Internal-Token': internalToken() }
  if (options.body !== undefined) {
    headers['Content-Type'] = 'application/json'
  }
  if (options.sessionToken) {
    headers.Authorization = `Bearer ${options.sessionToken}`
  }
  if (options.clientIp) {
    headers['X-Client-IP'] = options.clientIp
  }
  const res = await fetch(url, {
    method: options.method ?? 'GET',
    headers,
    body: options.body === undefined ? undefined : JSON.stringify(options.body),
    ...(options.revalidateSeconds === undefined
      ? { cache: 'no-store' as const }
      : { next: { revalidate: options.revalidateSeconds } }),
  })
  if (res.status === 204) {
    return schema.parse(undefined)
  }
  const json: unknown = await res.json()
  if (!res.ok) {
    const { code, message } = parseErrorBody(json)
    throw new ApiError(res.status, code, message)
  }
  return schema.parse(json)
}

function parseErrorBody(json: unknown): { code: string; message: string } {
  if (typeof json === 'object' && json !== null && 'error' in json) {
    const err = (json as { error: unknown }).error // 直前の in で error キーの存在を確認済み
    if (typeof err === 'object' && err !== null && 'code' in err && 'message' in err) {
      return { code: String(err.code), message: String(err.message) }
    }
  }
  return { code: 'internal', message: 'unexpected error body' }
}

/**
 * 本文を加工せずに api へ渡し、api の HTTP ステータスを返す。公式 LINE の Webhook の中継に使う(本文を
 * 読み直すと署名が合わなくなるため、JSON として扱わずにバイト列のまま送る)。
 */
export async function apiForward(
  path: string,
  body: ArrayBuffer,
  headers: Record<string, string>,
): Promise<number> {
  const res = await fetch(new URL(path, backendBaseUrl()), {
    method: 'POST',
    headers: { ...headers, 'X-Internal-Token': internalToken() },
    body,
    cache: 'no-store',
  })
  // 本文は使わないが、読み切って接続を返す
  await res.arrayBuffer()
  return res.status
}
