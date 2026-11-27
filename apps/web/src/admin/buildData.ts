import 'server-only'
import { divisions as divisionApi, matches, teams } from '@/api/admin'
import { entries } from '@/api/entry'
import { formats } from '@/api/format'
import { type BuildTeam, type DivisionBuild, draftState } from './build.utils'
import type { PasteMatch } from './schedulePaste.utils'

// 「試合を組む」の段の表と作業場に要るものを、部門ごとに api から集める(チーム、確認待ちの申込、
// 形、組分けの下書き、試合)。

/** 形の予選の組の数(pool_bands と round_robin の pools。読めなければ 0) */
function poolsOf(params: Record<string, unknown>): number {
  return typeof params.pools === 'number' ? params.pools : 0
}

export type DivisionBuildData = {
  build: DivisionBuild
  teams: BuildTeam[]
  /** 組分けの下書き(無ければ null) */
  draft: number[][] | null
  /** 確認待ちと直し待ちの申込のチーム名 */
  waiting: string[]
  /** 試合(時間割の貼り付けで当てはめる) */
  matches: Array<PasteMatch & { date: string | null; time: string | null; field: string | null }>
  /** 形の 1 試合の枠の長さ(分。形が無ければ null)。時間割の確かめはこの長さで重なりを見る */
  slotMinutes: number | null
}

type EntryList = Awaited<ReturnType<typeof entries.list>>

/** 大会の部門ごとの「試合を組む」の状態 */
export async function loadTournamentBuild(
  token: string,
  tournamentId: number,
): Promise<DivisionBuildData[]> {
  const [divisionList, entryList] = await Promise.all([
    divisionApi.list(token, tournamentId),
    entries.list(token, tournamentId),
  ])
  return Promise.all(
    divisionList.divisions.map((division) =>
      loadDivisionBuild(token, { id: division.id, name: division.name.ja }, entryList),
    ),
  )
}

/** 部門 1 つの「試合を組む」の状態(entryList は大会の申込の一覧) */
export async function loadDivisionBuild(
  token: string,
  division: { id: number; name: string },
  entryList: EntryList,
): Promise<DivisionBuildData> {
  const [teamList, matchList, saved, draft] = await Promise.all([
    teams.list(token, division.id),
    matches.list(token, division.id),
    formats.get(token, division.id),
    formats.poolDraft(token, division.id),
  ])
  const buildTeams = teamList.teams.map((team) => ({
    id: team.id,
    fullName: team.fullName.ja,
    shortName: team.shortName.ja,
  }))
  const waiting = entryList.entries
    .filter(
      (item) =>
        item.division.id === division.id &&
        (item.status === 'pending' || item.status === 'returned'),
    )
    .map((item) => item.fullName)
  const state = draft.pools
    ? draftState(
        draft.pools,
        buildTeams.map((team) => team.id),
      )
    : null
  const build: DivisionBuild = {
    id: division.id,
    name: division.name,
    teams: buildTeams.length,
    pendingEntries: waiting.length,
    formatted: saved.format !== null,
    pools: saved.format ? poolsOf(saved.format.params) : 0,
    draft:
      draft.pools && state
        ? { placed: state.placed, complete: state.complete, pools: draft.pools.length }
        : null,
    matches: matchList.matches.length,
    scheduled: matchList.matches.filter((match) => match.date !== null && match.time !== null)
      .length,
  }
  const pasteMatches = matchList.matches.map((match) => ({
    id: match.id,
    code: match.code,
    status: match.status,
    leftTeamId: match.left.team?.id ?? null,
    rightTeamId: match.right.team?.id ?? null,
    date: match.date,
    time: match.time,
    field: match.field,
  }))
  return {
    build,
    teams: buildTeams,
    draft: draft.pools,
    waiting,
    matches: pasteMatches,
    slotMinutes: saved.format?.schedule?.slotMinutes ?? null,
  }
}
