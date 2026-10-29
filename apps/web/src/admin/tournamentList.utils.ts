import type { AdminTournament } from '@/api/admin.types'

// 管理画面の大会一覧の分け方(Claude Design「Score View Admin」の大会一覧)。開催中、これからの大会、
// 終わった大会の 3 つに分け、各大会に状態の札を付ける。日付は会場の暦(日本時間)で比べる。

export type TournamentPhase = 'live' | 'upcoming' | 'done'
export type PhaseChip = { label: string; tone: 'live' | 'sched' | 'warn' | 'done' }

const jstDate = new Intl.DateTimeFormat('en-CA', {
  timeZone: 'Asia/Tokyo',
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
})

/** 日本時間の今日("YYYY-MM-DD") */
export function jstToday(now: Date): string {
  return jstDate.format(now)
}

const weekdays = ['日', '月', '火', '水', '木', '金', '土'] as const

/** "2026-10-25" を「10月25日（日）」にする(開催日の表示) */
export function japaneseDate(date: string): string {
  const [year, month, day] = date.split('-').map(Number)
  if (year === undefined || month === undefined || day === undefined) {
    return date
  }
  const weekday = weekdays[new Date(Date.UTC(year, month - 1, day)).getUTCDay()]
  return `${month}月${day}日（${weekday}）`
}

/**
 * 大会の段階。管理者が終了にした大会と開催日を過ぎた大会は終わった大会、開催日が今日なら開催中、
 * それ以外(開催日が先か未定)はこれからの大会。
 */
export function tournamentPhase(tournament: AdminTournament, today: string): TournamentPhase {
  if (tournament.status === 'finished') {
    return 'done'
  }
  if (tournament.startDate === null || tournament.startDate > today) {
    return 'upcoming'
  }
  return tournament.startDate === today ? 'live' : 'done'
}

/** 一覧の札。これからの大会は、非公開なら下書き、公開中なら受付中か締切り */
export function phaseChip(tournament: AdminTournament, phase: TournamentPhase): PhaseChip {
  if (phase === 'live') {
    return { label: '開催中', tone: 'live' }
  }
  if (phase === 'done') {
    return { label: '終了', tone: 'done' }
  }
  if (!tournament.listed) {
    return { label: '下書き', tone: 'warn' }
  }
  return tournament.status === 'entry_closed'
    ? { label: '締切り', tone: 'sched' }
    : { label: '受付中', tone: 'sched' }
}

export type TournamentGroup = { title: string; tournaments: AdminTournament[] }

const groupTitles: Record<TournamentPhase, string> = {
  live: '開催中',
  upcoming: 'これからの大会',
  done: '終わった大会',
}

/**
 * 開催中、これからの大会、終わった大会の順に分ける。これからの大会は近い順(未定は最後)、
 * 終わった大会は新しい順。大会の無い区分は返さない。
 */
export function groupTournaments(list: AdminTournament[], today: string): TournamentGroup[] {
  const byPhase: Record<TournamentPhase, AdminTournament[]> = { live: [], upcoming: [], done: [] }
  for (const tournament of list) {
    byPhase[tournamentPhase(tournament, today)].push(tournament)
  }
  const date = (tournament: AdminTournament) => tournament.startDate ?? ''
  byPhase.upcoming.sort((a, b) => {
    if (a.startDate === null || b.startDate === null) {
      return a.startDate === b.startDate ? b.id - a.id : a.startDate === null ? 1 : -1
    }
    return date(a).localeCompare(date(b)) || a.id - b.id
  })
  byPhase.done.sort((a, b) => date(b).localeCompare(date(a)) || b.id - a.id)
  return (['live', 'upcoming', 'done'] as const)
    .map((phase) => ({ title: groupTitles[phase], tournaments: byPhase[phase] }))
    .filter((group) => group.tournaments.length > 0)
}

/** 名前の部分一致(空白区切りの全部の語を含む)で絞る */
export function searchTournaments(list: AdminTournament[], query: string): AdminTournament[] {
  const words = query.trim().toLowerCase().split(/\s+/).filter(Boolean)
  return list.filter((tournament) => {
    const name = `${tournament.name.ja} ${tournament.name.en} ${tournament.shortName}`.toLowerCase()
    return words.every((word) => name.includes(word))
  })
}
