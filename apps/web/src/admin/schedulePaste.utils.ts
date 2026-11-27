// 時間割の貼り付け(試合を組むの時間割)。運営が Excel で決めた日付、時刻、コートを読み、部門の今ある試合に
// 当てはめる。どの試合かは試合番号(A1、E2 など)か、対戦する 2 チームの名前で決める。表の形は 2 つ読む。
//
// 一覧の形: 1 行目が見出し(日付、時刻、コート、試合番号、チーム、チーム、または 対戦)で、1 行 1 試合。
// 格子の形: 1 行目が「時刻」とコートの名前、2 行目からは 1 列目が時刻で、枠に試合番号か「A - B」。
// 日付は画面で選ぶ(この形には日付の列が無い)。

import type { BuildTeam } from './build.utils'

/** 当てはめに使う試合 */
export type PasteMatch = {
  id: number
  code: string
  status: string
  leftTeamId: number | null
  rightTeamId: number | null
}

/** 当てはめた 1 試合 */
export type ScheduleRow = {
  matchId: number
  code: string
  date: string
  time: string
  field: string
}

/** 読んだ結果。problems があれば書かない */
export type SchedulePaste = {
  rows: ScheduleRow[]
  problems: string[]
  /** 表に無かった、まだ始まっていない試合の試合番号(今の時刻のまま残る) */
  untouched: string[]
}

function key(text: string): string {
  return text.normalize('NFKC').toLowerCase().replace(/\s+/g, '')
}

/** 時刻を HH:MM にする(8:30、08:30、8時30分、8時)。読めなければ null */
export function clockOf(text: string): string | null {
  const value = text.normalize('NFKC').trim()
  const match =
    value.match(/^(\d{1,2})(?::|時)(\d{2})?(?:分)?$/) ?? value.match(/^(\d{1,2}):(\d{2}):\d{2}$/)
  if (match === null) {
    return null
  }
  const hour = Number(match[1])
  const minute = Number(match[2] ?? '0')
  if (hour > 23 || minute > 59) {
    return null
  }
  return `${String(hour).padStart(2, '0')}:${String(minute).padStart(2, '0')}`
}

/** 日付を YYYY-MM-DD にする(2027-02-09、2027/2/9、2/9、2月9日。年が無ければ baseDate の年) */
export function dayOf(text: string, baseDate: string): string | null {
  const value = text.normalize('NFKC').trim()
  const full = value.match(/^(\d{4})[-/年](\d{1,2})[-/月](\d{1,2})日?$/)
  const short = value.match(/^(\d{1,2})[/月](\d{1,2})日?$/)
  const year = full?.[1] ?? baseDate.slice(0, 4)
  const month = full?.[2] ?? short?.[1]
  const day = full?.[3] ?? short?.[2]
  if (month === undefined || day === undefined) {
    return null
  }
  return `${year}-${month.padStart(2, '0')}-${day.padStart(2, '0')}`
}

/** コートの名前を試合に書く形にする(「コート3」「Court 3」「第3コート」は 3。それ以外は書いたまま) */
export function courtOf(text: string): string {
  const value = text.normalize('NFKC').trim()
  const numbered = value.match(/^(?:コート|court|c|第)?\s*([0-9]+|[a-z])\s*(?:コート)?$/i)
  return numbered?.[1]?.toUpperCase() ?? value
}

/** 「A - B」「A vs B」「A 対 B」を 2 つの名前にする。分けられなければ null */
function pairOf(text: string): [string, string] | null {
  const parts = text
    .normalize('NFKC')
    .split(/\s+(?:-|ー|vs\.?|VS\.?|対|×|x)\s+|\s*(?:vs\.?|VS\.?|対)\s*/)
    .map((part) => part.trim())
    .filter(Boolean)
  return parts.length === 2 && parts[0] !== undefined && parts[1] !== undefined
    ? [parts[0], parts[1]]
    : null
}

function splitRows(text: string): string[][] {
  return text
    .replace(/^﻿/, '')
    .split(/\r?\n/)
    .map((line) => line.split(line.includes('\t') ? '\t' : ',').map((cell) => cell.trim()))
    .filter((cells) => cells.some((cell) => cell !== ''))
}

type Entry = {
  date: string | null
  time: string | null
  field: string
  code: string
  pair: [string, string] | null
  source: string
}

const headers = {
  date: /^(日付|日にち|date|日)$/i,
  time: /^(時刻|開始|開始時刻|時間|time|start)$/i,
  field: /^(コート|会場|court|field)$/i,
  code: /^(試合番号|試合|番号|code|no\.?|match)$/i,
  pair: /^(対戦|カード|match ?up)$/i,
  team: /^(チーム|team)/i,
}

/** 一覧の形の見出しの位置。時刻とコートと、試合番号か 2 チームか対戦が無ければ null */
function listColumns(head: readonly string[]) {
  const find = (pattern: RegExp) =>
    head.findIndex((cell) => pattern.test(cell.normalize('NFKC').trim()))
  const teams = head.flatMap((cell, index) =>
    headers.team.test(cell.normalize('NFKC').trim()) ? [index] : [],
  )
  const columns = {
    date: find(headers.date),
    time: find(headers.time),
    field: find(headers.field),
    code: find(headers.code),
    pair: find(headers.pair),
    teams,
  }
  const identified = columns.code >= 0 || columns.pair >= 0 || teams.length >= 2
  return columns.time >= 0 && columns.field >= 0 && identified ? columns : null
}

function readEntries(text: string, baseDate: string): { entries: Entry[]; problems: string[] } {
  const rows = splitRows(text)
  const [head = [], ...body] = rows
  const entries: Entry[] = []
  const problems: string[] = []
  const list = listColumns(head)
  if (list !== null) {
    for (const cells of body) {
      const cell = (index: number) => (index >= 0 ? (cells[index] ?? '') : '')
      const left = list.teams[0] === undefined ? '' : cell(list.teams[0])
      const right = list.teams[1] === undefined ? '' : cell(list.teams[1])
      entries.push({
        date: list.date >= 0 ? dayOf(cell(list.date), baseDate) : baseDate,
        time: clockOf(cell(list.time)),
        field: courtOf(cell(list.field)),
        code: cell(list.code),
        pair: left !== '' && right !== '' ? [left, right] : pairOf(cell(list.pair)),
        source: cells.join(' '),
      })
    }
    return { entries, problems }
  }
  // 格子の形: 1 行目はコートの名前(1 列目は「時刻」か空)
  const courts = head.slice(1).map(courtOf)
  if (courts.length === 0 || body.length === 0) {
    problems.push(
      '表の形が読めません。1 行目に「時刻、コート、試合番号(またはチーム、チーム)」の見出しを付けるか、1 行目にコートの名前を並べてください',
    )
    return { entries, problems }
  }
  for (const cells of body) {
    const time = clockOf(cells[0] ?? '')
    cells.slice(1).forEach((cell, index) => {
      if (cell === '') {
        return
      }
      const pair = pairOf(cell)
      entries.push({
        date: baseDate,
        time,
        field: courts[index] ?? '',
        code: pair === null ? cell : '',
        pair,
        source: `${cells[0] ?? ''} ${head[index + 1] ?? ''} ${cell}`,
      })
    })
  }
  return { entries, problems }
}

/**
 * 貼り付けた時間割を読み、部門の試合に当てはめる。baseDate は日付の列が無いときの日付と、年の無い日付の
 * 年。当てはまらない行、読めない日付と時刻、同じ試合が 2 回、始まった試合、同じ日時とコートに 2 試合、
 * 同じ日時に同じチームが 2 試合は problems に書く。
 */
export function parseSchedule(
  text: string,
  baseDate: string,
  matches: readonly PasteMatch[],
  teams: readonly BuildTeam[],
): SchedulePaste {
  const { entries, problems } = readEntries(text, baseDate)
  const byCode = new Map(matches.map((match) => [key(match.code), match]))
  const teamByName = new Map<string, number>()
  for (const team of teams) {
    teamByName.set(key(team.shortName), team.id)
    teamByName.set(key(team.fullName), team.id)
  }
  const rows: ScheduleRow[] = []
  const used = new Set<number>()
  for (const entry of entries) {
    let match: PasteMatch | undefined
    if (entry.code !== '') {
      match = byCode.get(key(entry.code))
    } else if (entry.pair !== null) {
      const ids = entry.pair.map((name) => teamByName.get(key(name)))
      const found = matches.filter(
        (item) =>
          ids[0] !== undefined &&
          ids[1] !== undefined &&
          ((item.leftTeamId === ids[0] && item.rightTeamId === ids[1]) ||
            (item.leftTeamId === ids[1] && item.rightTeamId === ids[0])),
      )
      if (found.length > 1) {
        problems.push(
          `「${entry.source}」に当たる試合が ${found.length} つあります。試合番号で書いてください`,
        )
        continue
      }
      match = found[0]
    }
    if (match === undefined) {
      problems.push(`「${entry.source}」に当たる試合が見つかりません`)
      continue
    }
    if (entry.date === null || entry.time === null || entry.field === '') {
      problems.push(`${match.code}: 日付、時刻、コートのどれかが読めません(「${entry.source}」)`)
      continue
    }
    if (used.has(match.id)) {
      problems.push(`${match.code} が 2 回出てきます`)
      continue
    }
    if (match.status !== 'scheduled') {
      problems.push(`${match.code} はもう始まっているので、時刻を変えられません`)
      continue
    }
    used.add(match.id)
    rows.push({
      matchId: match.id,
      code: match.code,
      date: entry.date,
      time: entry.time,
      field: entry.field,
    })
  }
  problems.push(...clashes(rows, matches))
  const untouched = matches
    .filter((match) => match.status === 'scheduled' && !used.has(match.id))
    .map((match) => match.code)
  return { rows, problems, untouched }
}

/** 同じ日時とコートに 2 試合、同じ日時に同じチームが 2 試合 */
function clashes(rows: readonly ScheduleRow[], matches: readonly PasteMatch[]): string[] {
  const out: string[] = []
  const byId = new Map(matches.map((match) => [match.id, match]))
  const courts = new Map<string, string>()
  const teams = new Map<string, string>()
  for (const row of rows) {
    const slot = `${row.date} ${row.time}`
    const court = `${slot} ${row.field}`
    const taken = courts.get(court)
    if (taken !== undefined) {
      out.push(
        `${row.date.slice(5)} ${row.time} コート ${row.field} に ${taken} と ${row.code} が重なっています`,
      )
    }
    courts.set(court, row.code)
    const match = byId.get(row.matchId)
    for (const team of [match?.leftTeamId, match?.rightTeamId]) {
      if (team === null || team === undefined) {
        continue
      }
      const other = teams.get(`${slot} ${team}`)
      if (other !== undefined) {
        out.push(
          `${row.date.slice(5)} ${row.time} に同じチームが ${other} と ${row.code} に入っています`,
        )
      }
      teams.set(`${slot} ${team}`, row.code)
    }
  }
  return out
}
