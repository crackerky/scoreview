import type { MatchSide } from '@/api/common.types'
import type { GroupResponse, MatchResponse, ScheduleResponse } from '@/api/public.types'
import type { FlowEvent, LiveData, LiveMatch, LivePool, LiveSide, LiveStatus } from './live.types'

// 観戦アプリ(/live/{大会ID}。Claude Design「StageCam」)の、api の応答を画面の形に直す決まり。
// 参照サイトの模倣ページとは別の、本サイト独自の観戦用の画面。

/** チームの色(api にチームの色は無いので、ID から決まった色を割り当てる。白文字が読める濃さ) */
const teamPalette = [
  '#1747a6',
  '#0f7a4a',
  '#c8323c',
  '#a36f00',
  '#6b3fa0',
  '#00807a',
  '#1d2b6b',
  '#b5446e',
  '#2b7fc4',
  '#7a5230',
] as const

export function teamColor(teamId: number | null): string {
  if (teamId === null) {
    return '#2f343c'
  }
  return teamPalette[teamId % teamPalette.length] ?? '#2f343c'
}

const statusOf: Record<'scheduled' | 'live' | 'finished', LiveStatus> = {
  scheduled: 'sched',
  live: 'live',
  finished: 'done',
}

/** 決まっていないチームの表記(「E2 の勝者」「E3 の負けチーム」) */
function pendingName(side: MatchSide): string {
  if (side.placeholder) {
    return side.placeholder.outcome === 'win'
      ? `${side.placeholder.code} の勝者`
      : `${side.placeholder.code} の負けチーム`
  }
  return '未定'
}

function liveSide(side: MatchSide): LiveSide {
  const team = side.team
  return {
    teamId: team?.id ?? null,
    name: team ? team.name.ja : pendingName(side),
    pending: team === null,
    color: teamColor(team?.id ?? null),
    score: side.score,
  }
}

export function courtName(field: string | null): string {
  if (field === null || field === '') {
    return ''
  }
  return /^(コート|court)/i.test(field) ? field : `${field}コート`
}

/**
 * 大会のスケジュール(全日程)と組の応答から、観戦アプリの試合と順位表を作る。試合は日付、時刻、
 * コートの順(スケジュールの既定の並び)のまま。
 */
export function buildLiveData(
  schedule: ScheduleResponse,
  groups: GroupResponse[],
  tournamentDate: string | null,
): LiveData {
  const stageOf = new Map<number, { name: string; pool: boolean }>()
  for (const group of groups) {
    for (const row of group.matches) {
      stageOf.set(row.matchId, { name: group.group.name, pool: group.group.kind === 'pool' })
    }
  }
  const matches: LiveMatch[] = schedule.rows.map((row) => {
    const stage = stageOf.get(row.matchId)
    return {
      id: row.matchId,
      code: row.code,
      date: row.date,
      time: row.time ?? '',
      court: courtName(row.field),
      status: statusOf[row.status],
      stage: stage?.name ?? row.division.name.ja,
      pool: stage?.pool ?? false,
      divisionName: row.division.name.ja,
      left: liveSide(row.left),
      right: liveSide(row.right),
    }
  })
  const pools: LivePool[] = groups
    .filter((group) => group.pool !== null)
    .map((group) => ({
      id: group.group.id,
      name: group.group.name,
      done: group.matches.length > 0 && group.matches.every((row) => row.status === 'finished'),
      // 組のページの結果は登録順なので、順位の順に並べ直す(順位の無いチームは最後)
      rows: [...(group.pool?.results ?? [])]
        .sort((a, b) => (a.rank ?? Number.MAX_SAFE_INTEGER) - (b.rank ?? Number.MAX_SAFE_INTEGER))
        .map((row) => ({
          rank: row.rank,
          teamId: row.team.id,
          name: row.team.name.ja,
          color: teamColor(row.team.id),
          points: row.points,
        })),
    }))
  const banner = schedule.frame.banners[0]
  return {
    tournament: { id: schedule.frame.id, name: schedule.frame.name.ja, date: tournamentDate },
    sponsor: banner ? { imageUrl: banner.imageUrl, linkUrl: banner.linkUrl } : null,
    matches,
    pools,
  }
}

export type LiveTab = 'mine' | 'live' | 'done'

/** ライブのタブに並べる、試合中のあとのこれからの試合の数 */
const upcomingOnLive = 3

const statusOrder: Record<LiveStatus, number> = { live: 0, sched: 1, done: 2 }

/**
 * タブの試合。ライブは試合中とこれからの試合を少し、結果は終わった試合を新しい順、マイチームは
 * 選んだチームの試合を試合中、これから、終わったの順。
 */
export function tabMatches(matches: LiveMatch[], tab: LiveTab, favorites: number[]): LiveMatch[] {
  if (tab === 'live') {
    return [
      ...matches.filter((match) => match.status === 'live'),
      ...matches.filter((match) => match.status === 'sched').slice(0, upcomingOnLive),
    ]
  }
  if (tab === 'done') {
    return matches.filter((match) => match.status === 'done').reverse()
  }
  const mine = new Set(favorites)
  return matches
    .filter(
      (match) =>
        (match.left.teamId !== null && mine.has(match.left.teamId)) ||
        (match.right.teamId !== null && mine.has(match.right.teamId)),
    )
    .sort((a, b) => statusOrder[a.status] - statusOrder[b.status])
}

export type TimeGroup = { time: string; indexes: number[] }

/** 下の帯: 並びの中で、続けて同じ時刻の試合をまとめる */
export function timeGroups(matches: LiveMatch[]): TimeGroup[] {
  const groups: TimeGroup[] = []
  matches.forEach((match, index) => {
    const last = groups.at(-1)
    if (last && last.time === match.time) {
      last.indexes.push(index)
    } else {
      groups.push({ time: match.time, indexes: [index] })
    }
  })
  return groups
}

export type ScoreHits = Record<number, 'left' | 'right'>

/** 取り直す前と後で点が増えた側(点が入った演出に使う)。前に無かった試合は数えない */
export function scoreHits(before: LiveMatch[], after: LiveMatch[]): ScoreHits {
  const previous = new Map(before.map((match) => [match.id, match]))
  const hits: ScoreHits = {}
  for (const match of after) {
    const old = previous.get(match.id)
    if (!old) {
      continue
    }
    if ((match.left.score ?? 0) > (old.left.score ?? 0)) {
      hits[match.id] = 'left'
    } else if ((match.right.score ?? 0) > (old.right.score ?? 0)) {
      hits[match.id] = 'right'
    }
  }
  return hits
}

/** 経過秒を「12:05」にする(試合の経過時間。時計の時刻ではない) */
export function elapsedText(seconds: number): string {
  const minutes = Math.floor(seconds / 60)
  return `${minutes}:${String(seconds % 60).padStart(2, '0')}`
}

/** 試合の得点の流れ(得点だけ。古い順) */
export function flowEvents(detail: MatchResponse): FlowEvent[] {
  return detail.events
    .filter((event) => event.kind === 'goal' && event.side !== null)
    .map((event) => ({
      elapsed: elapsedText(event.elapsedSec),
      side: event.side === 'left' ? 'left' : 'right',
      scorer: event.scorer ? { name: event.scorer.name.ja, number: event.scorer.number } : null,
      assist: event.assist ? { name: event.assist.name.ja, number: event.assist.number } : null,
      callahan: event.callahan,
      leftScore: event.leftScore,
      rightScore: event.rightScore,
    }))
}

export type LeadPoint = { x: number; y: number; side: 'left' | 'right' }

/**
 * 得点差の折れ線(上が左のチームのリード)。x は 0〜width、y は 0〜height(中央が同点)。
 * 階段状に結ぶので、点ごとに前の高さから縦に動く。
 */
export function leadPoints(events: FlowEvent[], width: number, height: number): LeadPoint[] {
  const maxLead = Math.max(
    2,
    ...events.map((event) => Math.abs(event.leftScore - event.rightScore)),
  )
  const middle = height / 2
  return events.map((event, index) => ({
    x: ((index + 1) * width) / events.length,
    y: middle - ((event.leftScore - event.rightScore) * middle) / maxLead,
    side: event.side,
  }))
}
