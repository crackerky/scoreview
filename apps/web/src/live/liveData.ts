import { ApiError } from '@/api/client'
import { getGroup, getMatch, getTournamentSchedule, getTournaments } from '@/api/public'
import type { LiveData, LiveMatchDetail } from './live.types'
import { buildLiveData, flowEvents } from './live.utils'

// 観戦アプリのデータを api から集める(ページの初回の描画と、ブラウザが取り直すデータの口で共有)。
// スケジュール(全日程)と、ナビに出す組(順位表と試合の組の名前)だけを読む。

export type SwitchTournament = { id: number; name: string; meta: string }

const sectionWords = {
  in_progress: '開催中',
  entry_open: 'エントリー受付中',
  entry_closed: 'エントリー締切り',
  finished: '終了',
} as const

/** 大会を切り替えるメニューの大会(開催中、受付中、締切り、終了の順。公開の一覧と同じ) */
export async function loadSwitchTournaments(): Promise<{
  list: SwitchTournament[]
  dates: Map<number, string | null>
}> {
  const response = await getTournaments()
  const list: SwitchTournament[] = []
  const dates = new Map<number, string | null>()
  for (const section of response.sections) {
    for (const tournament of section.tournaments) {
      dates.set(tournament.id, tournament.startDate)
      list.push({
        id: tournament.id,
        name: tournament.name.ja,
        meta: [tournament.startDate?.slice(5).replace('-', '/'), sectionWords[section.section]]
          .filter(Boolean)
          .join(' ・ '),
      })
    }
  }
  return { list, dates }
}

export async function loadLiveData(tournamentId: number, date: string | null): Promise<LiveData> {
  const schedule = await getTournamentSchedule(tournamentId, { ymd: '' })
  const groupIds = schedule.frame.divisions.flatMap((division) =>
    division.groups.filter((group) => group.linked).map((group) => group.id),
  )
  const groups = await Promise.all(groupIds.map((groupId) => getGroup(groupId)))
  return buildLiveData(schedule, groups, date)
}

/** 試合の得点の流れ。試合がその大会のものでなければ null(別の大会の試合を出さない) */
export async function loadLiveMatch(
  tournamentId: number,
  matchId: number,
): Promise<LiveMatchDetail | null> {
  try {
    const detail = await getMatch(matchId)
    return detail.frame.id === tournamentId ? { events: flowEvents(detail) } : null
  } catch (error) {
    if (error instanceof ApiError && error.status === 404) {
      return null
    }
    throw error
  }
}
