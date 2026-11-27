// 管理画面「試合を組む」(大会の部門ごとの段の表と、部門の作業場)の計算。部門ごとに、チーム → 組分け →
// 時間割の順に進める。組分けは、チームを選んで組に入れるか、Excel の表を貼るか、蛇行で配る。決めた組は
// 下書き(api の pool-draft)に残し、形を作るときにそのまま予選の組にする(api の poolSeeds)。

/** 組分けで扱うチーム */
export type BuildTeam = { id: number; fullName: string; shortName: string }

/** 組の名前(0 番目が A) */
export function poolLetter(index: number): string {
  return String.fromCharCode(65 + index)
}

/**
 * 組分けから、形を作るときのシード順(チームの id の並び)と、組ごとのシード番号を作る。シード番号は
 * 各組の 1 番目を A、B、C… の順に 1、2、3…、続けて各組の 2 番目…と振る(組の上から順に強い扱い)。
 */
export function seedsFromPools(pools: readonly (readonly number[])[]): {
  seeds: number[]
  poolSeeds: number[][]
} {
  const seeds: number[] = []
  const poolSeeds: number[][] = pools.map(() => [])
  const depth = Math.max(0, ...pools.map((pool) => pool.length))
  for (let row = 0; row < depth; row++) {
    pools.forEach((pool, index) => {
      const team = pool[row]
      if (team !== undefined) {
        seeds.push(team)
        poolSeeds[index]?.push(seeds.length)
      }
    })
  }
  return { seeds, poolSeeds }
}

/** チームを並びの順に、count 個の組へ蛇行(A、B、C、C、B、A…)で配る(おまかせ) */
export function snakePools(teamIds: readonly number[], count: number): number[][] {
  const pools: number[][] = Array.from({ length: Math.max(1, count) }, () => [])
  teamIds.forEach((id, i) => {
    const row = Math.floor(i / pools.length)
    const col = i % pools.length
    pools[row % 2 === 0 ? col : pools.length - 1 - col]?.push(id)
  })
  return pools
}

/** 組分けの進み具合。complete は全チームがちょうど 1 回入り、どの組も 2 チーム以上 */
export function draftState(
  pools: readonly (readonly number[])[],
  teamIds: readonly number[],
): { placed: number; total: number; complete: boolean } {
  const known = new Set(teamIds)
  const placed = new Set(pools.flat().filter((id) => known.has(id)))
  const complete =
    pools.length > 0 &&
    placed.size === teamIds.length &&
    pools.flat().length === placed.size &&
    pools.every((pool) => pool.length >= 2)
  return { placed: placed.size, total: teamIds.length, complete }
}

/** 名前を比べる形(全角と半角、大文字と小文字、空白の違いを見ない) */
function nameKey(text: string): string {
  return text.normalize('NFKC').toLowerCase().replace(/\s+/g, '')
}

/** 組の名前の欄を組の名前にする(「A組」「プールA」「Pool A」「Aブロック」はどれも A) */
function poolName(cell: string): string {
  const key = cell.normalize('NFKC').trim()
  const stripped = key
    .replace(/^(pool|プール|ブロック|block)\s*/i, '')
    .replace(/\s*(組|プール|ブロック|pool|block)$/i, '')
    .trim()
  return stripped === '' ? key : stripped.toUpperCase()
}

/** 申込に無い名前に近い名前(同じ文字が多いもの。2 文字以上同じでなければ無し) */
function nearestName(name: string, teams: readonly BuildTeam[]): string | null {
  const key = nameKey(name)
  let best: { name: string; score: number } | null = null
  for (const team of teams) {
    const other = nameKey(team.fullName)
    let score = 0
    for (const ch of new Set(key)) {
      if (other.includes(ch)) {
        score++
      }
    }
    if (score >= 2 && (best === null || score > best.score)) {
      best = { name: team.fullName, score }
    }
  }
  return best?.name ?? null
}

/** 貼り付けた組分けを読んだ結果 */
export type PoolPaste = {
  /** 組の名前(表の順) */
  names: string[]
  /** 組ごとのチームの id(names と同じ順) */
  pools: number[][]
  /** 申込のチームと合わなかった名前と、近い名前 */
  unknown: Array<{ pool: string; name: string; near: string | null }>
  /** 2 回出てきたチーム */
  duplicates: string[]
  /** 表に無かった申込のチーム */
  missing: number[]
}

/**
 * Excel やスプレッドシートからコピーした組分け(タブ区切り)か CSV(カンマ区切り)を読み、申込のチームと
 * 照らす。形は 2 つ読む。1 行目が組の名前でその下にチーム名(組ごとに縦に並べる)と、「組, チーム名」の
 * 2 列を 1 行 1 チーム(1 行目が「組, チーム」の見出しなら飛ばす)。チーム名は正式名か短縮名で、全角と
 * 半角、大文字と小文字、空白の違いを見ない。
 */
export function parsePoolTable(text: string, teams: readonly BuildTeam[]): PoolPaste {
  const rows = text
    .replace(/^﻿/, '')
    .split(/\r?\n/)
    .map((line) => line.split(line.includes('\t') ? '\t' : ',').map((cell) => cell.trim()))
    .filter((cells) => cells.some((cell) => cell !== ''))
  const byKey = new Map<string, BuildTeam>()
  for (const team of teams) {
    byKey.set(nameKey(team.shortName), team)
    byKey.set(nameKey(team.fullName), team)
  }
  const entries: Array<{ pool: string; name: string }> = []
  const twoColumns =
    rows.length > 0 &&
    rows.every((cells) => cells.filter((cell) => cell !== '').length <= 2) &&
    new Set(rows.map((cells) => cells[0])).size < rows.length
  if (twoColumns) {
    const header =
      /^(組|プール|pool|group)/i.test(rows[0]?.[0] ?? '') && /チーム|team/i.test(rows[0]?.[1] ?? '')
    for (const cells of header ? rows.slice(1) : rows) {
      const [pool = '', name = ''] = cells
      if (pool !== '' && name !== '') {
        entries.push({ pool: poolName(pool), name })
      }
    }
  } else {
    const [head = [], ...body] = rows
    for (const cells of body) {
      cells.forEach((name, col) => {
        const pool = head[col] ?? ''
        if (name !== '' && pool !== '') {
          entries.push({ pool: poolName(pool), name })
        }
      })
    }
  }
  const names: string[] = []
  const pools: number[][] = []
  const unknown: PoolPaste['unknown'] = []
  const duplicates: string[] = []
  const used = new Set<number>()
  for (const { pool, name } of entries) {
    let index = names.indexOf(pool)
    if (index < 0) {
      names.push(pool)
      pools.push([])
      index = names.length - 1
    }
    const team = byKey.get(nameKey(name))
    if (team === undefined) {
      unknown.push({ pool, name, near: nearestName(name, teams) })
    } else if (used.has(team.id)) {
      duplicates.push(name)
    } else {
      used.add(team.id)
      pools[index]?.push(team.id)
    }
  }
  const missing = teams.filter((team) => !used.has(team.id)).map((team) => team.id)
  return { names, pools, unknown, duplicates, missing }
}

/** 部門の段の表の 1 行(大会の「試合を組む」) */
export type DivisionBuild = {
  id: number
  name: string
  teams: number
  /** 確認待ちの申込(参加を決めると組に入れられる) */
  pendingEntries: number
  /** 形を作った(組と試合がある) */
  formatted: boolean
  /** 形の予選の組の数(形が無ければ 0) */
  pools: number
  /** 組分けの下書き(無ければ null) */
  draft: { placed: number; complete: boolean; pools: number } | null
  matches: number
  /** 日時とコートの決まった試合 */
  scheduled: number
}

export type Step = { tone: 'ok' | 'warn' | 'todo'; label: string; sub: string }

/** 部門の 3 つの段(チーム、組分け、時間割)の状態と、次にやること */
export function divisionSteps(item: DivisionBuild): {
  teams: Step
  pools: Step
  times: Step
  next: { label: string; step: 'teams' | 'pools' | 'times' }
} {
  const teams: Step =
    item.teams < 2
      ? { tone: 'todo', label: `${item.teams} チーム`, sub: '参加が決まると、ここに入ります' }
      : item.pendingEntries > 0
        ? {
            tone: 'warn',
            label: `${item.teams} チーム`,
            sub: `確認待ちの申込 ${item.pendingEntries}`,
          }
        : { tone: 'ok', label: `${item.teams} チーム`, sub: '参加決定' }
  const pools: Step = item.formatted
    ? { tone: 'ok', label: '済み', sub: item.pools > 0 ? `${item.pools} 組` : '組と順位決定戦あり' }
    : item.draft?.complete
      ? { tone: 'ok', label: '決めた', sub: `${item.draft.pools} 組(時間割はまだ)` }
      : item.draft && item.draft.placed > 0
        ? { tone: 'warn', label: '途中', sub: `${item.draft.placed}/${item.teams} チーム` }
        : { tone: 'todo', label: 'まだ', sub: '組分けから' }
  const times: Step = !item.formatted
    ? { tone: 'todo', label: 'まだ', sub: item.draft?.complete ? '作れます' : '組分けの後' }
    : item.scheduled < item.matches
      ? { tone: 'warn', label: '途中', sub: `${item.scheduled}/${item.matches} 試合` }
      : { tone: 'ok', label: '済み', sub: `${item.matches} 試合` }
  const next =
    item.teams < 2
      ? { label: 'チームを確かめる', step: 'teams' as const }
      : !item.formatted && !item.draft?.complete
        ? { label: '組分けをする', step: 'pools' as const }
        : !item.formatted
          ? { label: '時間割を作る', step: 'times' as const }
          : { label: '組と時間割を見る', step: 'times' as const }
  return { teams, pools, times, next }
}
