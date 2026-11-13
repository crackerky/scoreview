import { buildTimetable, type TimetableDay, type TimetableSlot } from '@/admin/timetable.utils'
import { japaneseDate } from '@/admin/tournamentList.utils'
import type { BoardResponse } from '@/api/public.types'
import { type BracketView, bracketViews, type PoolMatrix, poolMatrix } from '@/live/live.tables'
import type { LiveData, LiveMatch, LivePool } from '@/live/live.types'

// スコアサイト(/tournaments/{id}。クラブジュニアのスコアサイトに近い構成)の、大会の掲示板を画面の
// 形に直す決まり。試合、組の順位、表の作り方は観戦アプリと同じもの(live.utils、live.tables)を使う。

/** スコアサイトの部門。組の星取表とトーナメント */
export type ScoreDivision = {
  id: number
  name: string
  pools: PoolMatrix[]
  brackets: BracketView[]
}

/**
 * 部門ごとの星取表とトーナメント。星取表の見せ方(記号、列、並び、試合中のマス、進出圏の色)は
 * 管理者が「観戦アプリの表」で決めたものに従う。組もトーナメントも無い部門は出さない。
 */
export function scoreDivisions(board: BoardResponse, data: LiveData): ScoreDivision[] {
  const pools = new Map(data.pools.map((pool) => [pool.id, pool]))
  return board.divisions.flatMap((division) => {
    const tables = division.pools.flatMap((item) => {
      const pool = pools.get(item.groupId)
      return pool ? [poolMatrix(pool, data.matches, data.tables.config)] : []
    })
    const brackets = bracketViews(
      data.tables.brackets.filter((bracket) => bracket.divisionId === division.id),
      data.matches,
    )
    if (tables.length === 0 && brackets.length === 0) {
      return []
    }
    return [{ id: division.id, name: division.name.ja, pools: tables, brackets }]
  })
}

/** 時間割の枠に置く試合 */
export type ScoreSlot = TimetableSlot & { match: LiveMatch }

/**
 * 日ごとの時間割(時刻 × コート)。枠の位置は掲示板の日付、時刻、コート(並びに使う元の値)で決め、
 * 中身は観戦アプリの形の試合(決まっていないチームを「準決勝(1コート)の勝者」と呼ぶなど)にする。
 */
export function scoreDays(board: BoardResponse, data: LiveData): TimetableDay<ScoreSlot>[] {
  const byId = new Map(data.matches.map((match) => [match.id, match]))
  const slots = board.matches.flatMap((row) => {
    const match = byId.get(row.matchId)
    return match ? [{ date: row.date, time: row.time, field: row.field, match }] : []
  })
  return buildTimetable(slots)
}

/** 日付の見出し(「11月3日（火）」)。日付の無い試合の日は「日付未定」 */
export function dayLabel(date: string | null): string {
  return date === null ? '日付未定' : japaneseDate(date)
}

/** 日に試合中の試合があるか */
export function dayHasLive(day: TimetableDay<ScoreSlot>): boolean {
  return [...day.cells.values(), day.unplaced].some((list) =>
    list.some((slot) => slot.match.status === 'live'),
  )
}

/** 最初に開く日の位置。今日があれば今日、無ければ試合中の試合がある日、それも無ければ最初の日 */
export function initialDayIndex(
  days: ReadonlyArray<{ date: string | null; live: boolean }>,
  today: string,
): number {
  const todayAt = days.findIndex((day) => day.date === today)
  if (todayAt >= 0) {
    return todayAt
  }
  return Math.max(
    days.findIndex((day) => day.live),
    0,
  )
}

/** 組の画面に出すもの。組が掲示板に無ければ null */
export function poolPage(
  data: LiveData,
  groupId: number,
): { pool: LivePool; table: PoolMatrix; matches: LiveMatch[] } | null {
  const pool = data.pools.find((item) => item.id === groupId)
  if (!pool) {
    return null
  }
  return {
    pool,
    table: poolMatrix(pool, data.matches, data.tables.config),
    matches: data.matches.filter((match) => match.groupId === groupId),
  }
}

/** 試合の時刻とコート(「09:00 ・ 1コート」)。決まっていなければ空 */
export function slotText(match: LiveMatch): string {
  return [match.time, match.court].filter((part) => part !== '').join(' ・ ')
}
