import { existsSync } from 'node:fs'
import { createServer, type ServerResponse } from 'node:http'
import { loadFixture } from '../test/fixtures/load.ts'

// 開発用のモック api。api(Go)が動かなくても `next dev` で画面を確かめられるよう、
// docs/api.md の閲覧用エンドポイントに test/fixtures の架空データを返す。
// 使い方: INTERNAL_API_TOKEN=... MOCK_API_PORT=8080 pnpm --filter @scoreview/web mock-api
// web 側は BACKEND_INTERNAL_URL=http://localhost:8080 で向ける。

const defaultPort = 8080

type Route = {
  pattern: RegExp
  /** fixture のファイル名(id を受け取る) */
  fixture: (id: string) => string
  /** not_found の message に使う種別(docs/api.md「エラー」) */
  idKind: string
}

const routes: Route[] = [
  { pattern: /^\/api\/v1\/tournaments$/, fixture: () => 'tournaments', idKind: 'TOURNAMENT_ID' },
  {
    pattern: /^\/api\/v1\/tournaments\/(\d+)$/,
    fixture: (id) => `tournament-${id}`,
    idKind: 'TOURNAMENT_ID',
  },
  {
    pattern: /^\/api\/v1\/tournaments\/(\d+)\/teams$/,
    fixture: (id) => `teams-${id}`,
    idKind: 'TOURNAMENT_ID',
  },
  {
    pattern: /^\/api\/v1\/tournaments\/(\d+)\/schedule$/,
    fixture: (id) => `schedule-tournament-${id}`,
    idKind: 'TOURNAMENT_ID',
  },
  {
    pattern: /^\/api\/v1\/divisions\/(\d+)\/schedule$/,
    fixture: (id) => `schedule-division-${id}`,
    idKind: 'EVENT_DIVISION_ID',
  },
  {
    pattern: /^\/api\/v1\/divisions\/(\d+)\/standings$/,
    fixture: (id) => `standings-${id}`,
    idKind: 'EVENT_DIVISION_ID',
  },
  { pattern: /^\/api\/v1\/groups\/(\d+)$/, fixture: (id) => `group-${id}`, idKind: 'GROUP_ID' },
  { pattern: /^\/api\/v1\/teams\/(\d+)$/, fixture: (id) => `team-${id}`, idKind: 'ENTRY_TEAM_ID' },
  { pattern: /^\/api\/v1\/matches\/(\d+)$/, fixture: (id) => `match-${id}`, idKind: 'MATCH_ID' },
  {
    pattern: /^\/api\/v1\/players\/(\d+)$/,
    fixture: (id) => `player-${id}`,
    idKind: 'ENTRY_PLAYER_ID',
  },
  {
    pattern: /^\/api\/v1\/galleries\/(\d+)$/,
    fixture: (id) => `gallery-${id}`,
    idKind: 'TGALLERY_ID',
  },
]

function sendJson(res: ServerResponse, status: number, body: unknown): void {
  res.writeHead(status, { 'Content-Type': 'application/json; charset=utf-8' })
  res.end(JSON.stringify(body))
}

function sendError(res: ServerResponse, status: number, code: string, message: string): void {
  sendJson(res, status, { error: { code, message } })
}

function fixtureExists(name: string): boolean {
  return existsSync(new URL(`../test/fixtures/${name}.json`, import.meta.url))
}

const token = process.env.INTERNAL_API_TOKEN
if (!token) {
  throw new Error('INTERNAL_API_TOKEN is not set')
}
const port = Number(process.env.MOCK_API_PORT ?? defaultPort)

const server = createServer((req, res) => {
  if (req.headers['x-internal-token'] !== token) {
    sendError(res, 401, 'unauthorized', 'invalid internal token')
    return
  }
  if (req.method !== 'GET') {
    sendError(res, 400, 'bad_request', 'mock api supports GET only')
    return
  }
  const path = new URL(req.url ?? '/', 'http://localhost').pathname
  for (const route of routes) {
    const match = route.pattern.exec(path)
    if (match === null) {
      continue
    }
    const id = match[1] ?? ''
    const name = route.fixture(id)
    if (!fixtureExists(name)) {
      sendError(res, 404, 'not_found', `${route.idKind} [${id}]`)
      return
    }
    sendJson(res, 200, loadFixture(name))
    return
  }
  sendError(res, 404, 'not_found', `no route: ${path}`)
})

server.listen(port, () => {
  console.log(`mock api listening on http://localhost:${port}`)
})
