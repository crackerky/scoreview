import type { Assignment } from '@/api/admin.types'
import type { GroupProgress, Progress, ProgressMatch, ProgressSide, Todo } from '@/api/format.types'
import { describeSide } from './formatWizard.utils'
import { courtLabel } from './scoreKeeper.utils'

// 大会単位の本部(Claude Design「Score View Admin」の本部)。部門ごとの進行状況(api の progress)を
// 1 画面にまとめる。知らせは同じ原因のものを 1 件にし、そこから直接その作業の画面へ進めるようにする
// (docs/operations/operator-experience.md「当日」1)。

export type DivisionBoard = {
  divisionId: number
  divisionName: string
  progress: Progress
  matches: ProgressMatch[]
}

export type Tone = 'live' | 'done' | 'sched' | 'warn' | 'ok'
export type AlertLink = { label: string; href: string }
export type HqAlert = {
  key: string
  chip: string
  tone: Tone
  title: string
  body: string
  links: AlertLink[]
}

/**
 * これより長い遅れは前の日から残った知らせとみなし、遅れとしては出さない(分。12 時間)。
 * 日付が変わった後に「6081 分遅れ」と出て、今日の本当の知らせが埋もれていた(管理画面の点検)
 */
export const staleDelayMinutes = 12 * 60

/** 知らせに並べる試合の数の上限(それより多いときは「ほか N 試合」) */
const listedMatches = 4

/** 試合の片側。チームが決まっていなければ「A組1位」「E2 の勝者」のように決まり方を出す */
export function sideName(side: ProgressSide): string {
  return side.team?.name.ja ?? describeSide(side.label)
}

export function hqPath(tournamentId: number): string {
  return `/admin/tournaments/${tournamentId}/hq`
}

export function resultHref(matchId: number, tournamentId: number): string {
  return `/admin/matches/${matchId}/result?next=${encodeURIComponent(hqPath(tournamentId))}`
}

function matchWhen(match: ProgressMatch): string {
  return [match.time, courtLabel(match.field), match.groupLabel].filter(Boolean).join(' ')
}

function matchList(list: ProgressMatch[]): string {
  const shown = list
    .slice(0, listedMatches)
    .map((match) => `${matchWhen(match)}「${sideName(match.left)} 対 ${sideName(match.right)}」`)
  const rest = list.length - shown.length
  return rest > 0 ? `${shown.join('、')} ほか ${rest} 試合` : shown.join('、')
}

/** 知らせの中の結果の入力へのリンク。同じコートの試合が並んでも見分けられるよう試合番号を付ける */
function resultLinkLabel(match: ProgressMatch): string {
  const court = courtLabel(match.field)
  return `${match.code}${court ? `(${court})` : ''}の結果を入れる`
}

function byClock(a: ProgressMatch, b: ProgressMatch): number {
  return (
    (a.date ?? '').localeCompare(b.date ?? '') ||
    (a.time ?? '').localeCompare(b.time ?? '') ||
    (a.field ?? '').localeCompare(b.field ?? '', 'ja', { numeric: true })
  )
}

function allMatches(boards: DivisionBoard[]): ProgressMatch[] {
  return boards.flatMap((board) => board.matches).sort(byClock)
}

/**
 * 割り当てが試合を覆うか。api が記録を受ける決まり(internal/domain/assignment.go の Covers)と同じ:
 * 試合への割り当てはその試合だけ、そうでなければコート、日付、時刻の範囲(開始は含み終わりは含まない)が
 * すべて合うこと。範囲の条件があるのに試合の側が未定なら覆わない。
 */
export function coversMatch(assignment: Assignment, match: ProgressMatch): boolean {
  if (assignment.matchId !== null) {
    return assignment.matchId === match.id
  }
  const equal = (condition: string | null, value: string | null) =>
    condition === null || (value !== null && condition === value)
  if (!equal(assignment.field, match.field) || !equal(assignment.date, match.date)) {
    return false
  }
  // "HH:MM" は辞書順と時刻の順が一致する
  if (assignment.from !== null && (match.time === null || match.time < assignment.from)) {
    return false
  }
  return !(assignment.to !== null && (match.time === null || match.time >= assignment.to))
}

/** 試合を記録できる記録係がいるか */
function hasKeeper(match: ProgressMatch, assignments: Assignment[]): boolean {
  return assignments.some((assignment) => coversMatch(assignment, match))
}

/** 本部の知らせ。部門をまたいで同じ原因(同じコートの遅れ、結果待ち)を 1 件にまとめる */
export function buildAlerts(
  boards: DivisionBoard[],
  tournamentId: number,
  assignments: Assignment[],
): HqAlert[] {
  const hq = hqPath(tournamentId)
  const matches = allMatches(boards)
  const alerts: HqAlert[] = []

  const delays = new Map<string, number>()
  for (const board of boards) {
    for (const todo of board.progress.todo) {
      if (todo.kind === 'court_delay' && todo.court !== null && todo.minutes <= staleDelayMinutes) {
        delays.set(todo.court, Math.max(delays.get(todo.court) ?? 0, todo.minutes))
      }
    }
  }
  for (const [court, minutes] of [...delays].sort(([a], [b]) => a.localeCompare(b, 'ja'))) {
    const affected = matches.filter(
      (match) => match.field === court && match.status === 'scheduled',
    )
    alerts.push({
      key: `delay-${court}`,
      chip: '遅れ',
      tone: 'warn',
      title: `${courtLabel(court)}が ${minutes} 分遅れています`,
      body:
        affected.length > 0
          ? `このあとの ${affected.length} 試合(${matchList(affected)})に影響します。`
          : 'このコートのあとの試合はありません。',
      links: [{ label: '時間をずらす', href: `${hq}/shift?court=${encodeURIComponent(court)}` }],
    })
  }

  const overdueIds = new Set(
    boards.flatMap((board) =>
      board.progress.todo
        .filter((todo) => todo.kind === 'overdue' && todo.matchId !== null)
        .map((todo) => todo.matchId),
    ),
  )
  const overdue = matches.filter((match) => overdueIds.has(match.id))
  if (overdue.length > 0) {
    alerts.push({
      key: 'overdue',
      chip: '結果待ち',
      tone: 'live',
      title: `終わる時刻を過ぎて結果が入っていない試合が ${overdue.length} 件あります`,
      body: `${matchList(overdue)}。`,
      links: overdue.slice(0, listedMatches).map((match) => ({
        label: overdue.length > 1 ? resultLinkLabel(match) : '結果を入れる',
        href: resultHref(match.id, tournamentId),
      })),
    })
  }

  const live = matches.filter((match) => match.status === 'live')
  const noKeeper = live.filter((match) => !hasKeeper(match, assignments))
  if (noKeeper.length > 0) {
    alerts.push({
      key: 'no-keeper',
      chip: '記録係なし',
      tone: 'live',
      title: `記録係のいない試合が ${noKeeper.length} 件あります`,
      body: `${matchList(noKeeper)}。終わったら本部で結果を入れてください。`,
      links: noKeeper.slice(0, listedMatches).map((match) => ({
        label: noKeeper.length > 1 ? resultLinkLabel(match) : '結果を入れる',
        href: resultHref(match.id, tournamentId),
      })),
    })
  }

  for (const board of boards) {
    const groups = new Map(board.progress.groups.map((group) => [group.id, group]))
    for (const todo of board.progress.todo) {
      const alert = groupAlert(todo, groups, board, hq)
      if (alert) {
        alerts.push(alert)
      }
    }
  }
  return alerts
}

/** 組と段の知らせ(確定待ち、同順位)と、本部の進行画面で直す知らせ */
function groupAlert(
  todo: Todo,
  groups: Map<number, GroupProgress>,
  board: DivisionBoard,
  hq: string,
): HqAlert | null {
  const progressPage = `/admin/divisions/${board.divisionId}/progress`
  const group = todo.groupId === null ? undefined : groups.get(todo.groupId)
  const key = `${todo.kind}-${board.divisionId}-${todo.groupId}-${todo.stageId}-${todo.matchId}-${todo.teamId}`
  switch (todo.kind) {
    case 'confirm_group':
      return group
        ? {
            key,
            chip: '確定待ち',
            tone: 'ok',
            title: `${group.label}の全試合が終わりました`,
            body: `確定すると、次の組み合わせに${group.label}の順位が入ります。`,
            links: [{ label: `${group.label}を確定する`, href: `${hq}/confirm/${group.id}` }],
          }
        : null
    case 'confirm_stage':
      return {
        key,
        chip: '確定待ち',
        tone: 'ok',
        title: todo.text,
        body: '段の全部の組が終わりました。確定すると次の段の組み合わせが決まります。',
        links: [{ label: '段を確定する', href: progressPage }],
      }
    case 'tie':
      return {
        key,
        chip: '同順位',
        tone: 'warn',
        title: group ? `${group.label}に規程で決まらない同順位があります` : todo.text,
        body: todo.text,
        links: [{ label: '順位を決める', href: progressPage }],
      }
    case 'mismatch':
    case 'overlap':
    case 'withdrawn':
      return {
        key,
        chip: '要確認',
        tone: 'warn',
        title: todo.text,
        body: `${board.divisionName}の進行で直します。`,
        links: [{ label: `${board.divisionName}の進行で直す`, href: progressPage }],
      }
    default:
      return null
  }
}

export type CourtCard = {
  court: string
  name: string
  delay: number | null
  keeper: string | null
  /** いま試合中か次の試合があるのに、記録できる記録係がいない */
  needsKeeper: boolean
  live: ProgressMatch | null
  next: ProgressMatch | null
}

/** コートごとの今の試合、次の試合、遅れ、記録係 */
export function courtCards(boards: DivisionBoard[], assignments: Assignment[]): CourtCard[] {
  const courts = [...new Set(boards.flatMap((board) => board.progress.courts))].sort((a, b) =>
    a.localeCompare(b, 'ja', { numeric: true }),
  )
  const matches = allMatches(boards)
  return courts.map((court) => {
    const here = matches.filter((match) => match.field === court)
    const delays = boards.flatMap((board) =>
      board.progress.todo.filter(
        (todo) =>
          todo.kind === 'court_delay' && todo.court === court && todo.minutes <= staleDelayMinutes,
      ),
    )
    const live = here.find((match) => match.status === 'live') ?? null
    const next = here.find((match) => match.status === 'scheduled') ?? null
    // コートの記録係は、いま試合中か次の試合を記録できる人(全コートや試合ごとの割り当ても含む)
    const focus = live ?? next
    const keepers = focus
      ? [
          ...new Set(
            assignments
              .filter((assignment) => coversMatch(assignment, focus))
              .map((assignment) => assignment.userName),
          ),
        ]
      : []
    return {
      court,
      name: courtLabel(court),
      delay: delays.length > 0 ? Math.max(...delays.map((todo) => todo.minutes)) : null,
      keeper: keepers.length > 0 ? keepers.join('、') : null,
      needsKeeper: focus !== null && keepers.length === 0,
      live,
      next,
    }
  })
}

export type GroupCard = {
  id: number
  divisionId: number
  name: string
  chip: string
  tone: Tone
  sub: string
  link: AlertLink | null
}

/** 組の状態の札。確定を待つ組と同順位の組には、その作業へのリンクを付ける */
export function groupCards(boards: DivisionBoard[], tournamentId: number): GroupCard[] {
  const several = boards.length > 1
  return boards.flatMap((board) =>
    board.progress.groups.map((group) => {
      const name = several ? `${board.divisionName} ${group.label}` : group.label
      const top = group.rows
        .slice(0, 2)
        .map((row) => `${row.rank}位 ${row.team.name.ja}`)
        .join(' ・ ')
      const base = { id: group.id, divisionId: board.divisionId, name }
      switch (group.state) {
        case 'ready':
          return {
            ...base,
            chip: '確定待ち',
            tone: 'warn' as const,
            sub: top,
            link: { label: '確定する', href: `${hqPath(tournamentId)}/confirm/${group.id}` },
          }
        case 'tied':
          return {
            ...base,
            chip: '同順位',
            tone: 'warn' as const,
            sub: '規程で決まらない同順位があります',
            link: { label: '順位を決める', href: `/admin/divisions/${board.divisionId}/progress` },
          }
        case 'confirmed':
        case 'finished':
          return { ...base, chip: '確定', tone: 'ok' as const, sub: top, link: null }
        case 'playing':
          return {
            ...base,
            chip: '試合中',
            tone: 'live' as const,
            sub: `残り ${group.remaining} 試合`,
            link: null,
          }
        default:
          // waiting(前の段の確定を待っている)
          return {
            ...base,
            chip: '組み合わせ待ち',
            tone: 'done' as const,
            sub: '前の段の確定を待っています',
            link: null,
          }
      }
    }),
  )
}

export type MatchTimeGroup = {
  key: string
  date: string | null
  time: string
  rows: ProgressMatch[]
}

/** 試合を日付と時刻ごとにまとめる(本部の「試合」タブ)。時刻の無い試合は最後に「時刻未定」 */
export function matchesByTime(boards: DivisionBoard[]): MatchTimeGroup[] {
  const groups: MatchTimeGroup[] = []
  for (const match of allMatches(boards)) {
    const time = match.time ?? '時刻未定'
    const key = `${match.date ?? ''} ${time}`
    const last = groups.at(-1)
    if (last && last.key === key) {
      last.rows.push(match)
    } else {
      groups.push({ key, date: match.date, time, rows: [match] })
    }
  }
  const undated = groups.filter((group) => group.time === '時刻未定')
  return [...groups.filter((group) => group.time !== '時刻未定'), ...undated]
}

/** 結果を入れられる試合(両チームが決まっている) */
export function decided(match: ProgressMatch): boolean {
  return match.left.team !== null && match.right.team !== null
}

/** "09:40" に分を足す(日をまたいだら 24 時以降のまま出す。大会は 1 日の中で終わる) */
export function addMinutes(time: string, minutes: number): string {
  const [hour, minute] = time.split(':').map(Number)
  if (hour === undefined || minute === undefined || Number.isNaN(hour) || Number.isNaN(minute)) {
    return time
  }
  const total = hour * 60 + minute + minutes
  return `${String(Math.floor(total / 60)).padStart(2, '0')}:${String(total % 60).padStart(2, '0')}`
}

export type ShiftLine = { match: ProgressMatch; before: string; after: string }

/** コートのまだ始まっていない試合と、ずらした後の時刻(時間をずらす画面の下見) */
export function shiftPreview(boards: DivisionBoard[], court: string, minutes: number): ShiftLine[] {
  return allMatches(boards)
    .filter((match) => match.field === court && match.status === 'scheduled' && match.time !== null)
    .map((match) => {
      const before = match.time ?? ''
      return { match, before, after: addMinutes(before, minutes) }
    })
}

/** コートに、まだ始まっていない試合のある部門(ずらす api は部門ごと) */
export function divisionsOnCourt(boards: DivisionBoard[], court: string): number[] {
  return boards
    .filter((board) =>
      board.matches.some((match) => match.field === court && match.status === 'scheduled'),
    )
    .map((board) => board.divisionId)
}

/** 遅れの分数に合う、ずらす時間の初めの選択(5 分刻みで切り上げ、最大 20 分) */
export function suggestedShift(delay: number | null): number {
  if (delay === null || delay <= 0) {
    return 10
  }
  return Math.min(20, Math.max(5, Math.ceil(delay / 5) * 5))
}

/** 時間をずらしたあとの「元に戻す」に要るもの(本部の画面の ?undoCourt= など) */
export type ShiftUndo = { court: string; minutes: number; divisionIds: number[] }

/**
 * 時間をずらした直後に本部の画面へ付けて戻すクエリから、元に戻す操作を読む。形が合わなければ null
 * (ずらす時間は 5〜20 分の選択肢、部門は数字の ID の並び)。
 */
export function shiftUndoOf(
  query: Record<string, string | string[] | undefined>,
): ShiftUndo | null {
  const court = query.undoCourt
  const minutes = Number(query.undoMinutes)
  const divisions = query.undoDivisions
  if (typeof court !== 'string' || court === '' || typeof divisions !== 'string') {
    return null
  }
  if (![5, 10, 15, 20].includes(minutes) || !/^\d+(,\d+)*$/.test(divisions)) {
    return null
  }
  return { court, minutes, divisionIds: divisions.split(',').map(Number) }
}

export type ConfirmEffect = { match: ProgressMatch; left: string; right: string }

/**
 * 組を確定すると入る先(組の確定の画面)。片側が「A組1位」のようにこの組の順位を指す、まだチームの
 * 決まっていない試合を挙げ、今の順位で入るチームを「湘南(A組1位)」の形で出す。
 */
export function confirmEffects(board: DivisionBoard, group: GroupProgress): ConfirmEffect[] {
  const pattern = new RegExp(`^${group.letter}(\\d+)$`)
  const sideWith = (side: ProgressSide): string | null => {
    const found = side.team === null ? pattern.exec(side.label) : null
    if (!found) {
      return null
    }
    const row = group.rows.find((item) => item.rank === Number(found[1]))
    return row ? `${row.team.name.ja}(${describeSide(side.label)})` : describeSide(side.label)
  }
  return board.matches
    .filter((match) => sideWith(match.left) !== null || sideWith(match.right) !== null)
    .sort(byClock)
    .map((match) => ({
      match,
      left: sideWith(match.left) ?? sideName(match.left),
      right: sideWith(match.right) ?? sideName(match.right),
    }))
}
