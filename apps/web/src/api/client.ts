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
    cache: 'no-store',
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
