import type { AdminTournament, AdminUser } from '@/api/admin.types'
import type { ChipTone } from '@/components/admin/AdminParts'
import { entryPaths } from '@/entry/entry.paths'
import { notifyPaths } from '@/notify/notify.paths'
import { japaneseDate } from './tournamentList.utils'

// 大会ホームの段のカード。大会の段取りの順(準備、公式 LINE、試合を組む、当日、その他)に並べ、各段に
// 今の様子を 1 文と、次にやることを 1 つだけ出す。入口は段の中にまとめる。待っていること(開催日が
// 未定、確認待ちの申込、形の決まっていない部門など)がある最初の段を「今の段」にし、画面の主な操作にする。

export type StageKey = 'setup' | 'line' | 'build' | 'dayof' | 'more'

export type StageAction = { label: string; href: string }

export type StageLink = {
  label: string
  href: string
  sub?: string
  chip?: { label: string; tone: ChipTone }
}

/** 当日の数字(前日から出す) */
export type DayNumbers = { live: number; finished: number; total: number; alerts: number }

export type Stage = {
  key: StageKey
  title: string
  /** 今の様子を 1 文で */
  status: string
  /** 段の状態の札(済み、受付中など) */
  chip: { label: string; tone: ChipTone } | null
  /** 次にやること。済んだ段や、前の段が済むまで手を付けられない段は null */
  next: StageAction | null
  /** 先に片付けたいことがある(この段を今の段にする) */
  waiting: boolean
  /** 次にやることの前に知っておくこと */
  note: string | null
  /** 当日の数字。前日より前は null */
  numbers: DayNumbers | null
  links: StageLink[]
}

export type HomeDivision = {
  id: number
  name: string
  formatted: boolean
  /** チームの数(記録係には読めないので null) */
  teams: number | null
  matches: number
}

export type HomeInput = {
  tournamentId: number
  tournament: Pick<AdminTournament, 'startDate' | 'listed' | 'status'>
  /** 日本時間の今日("YYYY-MM-DD") */
  today: string
  role: AdminUser['role']
  divisions: HomeDivision[]
  /** 申込の数(管理者だけ) */
  /** noScorer は参加が決まったチームのうち、まだ記録係を選んでいないチームの数 */
  entries: { total: number; pending: number; approved: number; noScorer: number } | null
  /** 予約してまだ送っていない知らせの数(管理者だけ) */
  scheduled: number | null
  keepers: number
  /** 次の試合があるのに記録係のいないコート */
  freeCourts: string[]
  hasBoard: boolean
  day: DayNumbers
}

/** "YYYY-MM-DD" の前の日 */
export function dayBefore(date: string): string {
  const [year, month, day] = date.split('-').map(Number)
  if (year === undefined || month === undefined || day === undefined) {
    return date
  }
  return new Date(Date.UTC(year, month - 1, day - 1)).toISOString().slice(0, 10)
}

/**
 * 当日の数字を出すか。開催日の前日から出す(前日は記録係の割り当てと本部の確かめに使う)。開催日が
 * 未定でも、試合が始まっていれば出す。
 */
export function showsDayNumbers(startDate: string | null, today: string, day: DayNumbers): boolean {
  if (day.live > 0 || day.finished > 0) {
    return true
  }
  return startDate !== null && today >= dayBefore(startDate)
}

/** 待っていることがある最初の段(無ければ null) */
export function currentStage(stages: readonly Stage[]): Stage | null {
  return stages.find((stage) => stage.waiting && stage.next !== null) ?? null
}

/**
 * 大会ホームの段のカード。管理者には 5 つの段を出す。本部スタッフと記録係は申込と大会の形を
 * 扱わないので、当日の段と部門の一覧だけを出す。
 */
export function tournamentStages(input: HomeInput): Stage[] {
  if (input.role !== 'admin') {
    return [dayofStage(input), divisionsStage(input)]
  }
  return [
    setupStage(input),
    lineStage(input),
    buildStage(input),
    dayofStage(input),
    moreStage(input),
  ]
}

function base(input: HomeInput): string {
  return `/admin/tournaments/${input.tournamentId}`
}

function setupStage(input: HomeInput): Stage {
  const { startDate } = input.tournament
  const count = input.divisions.length
  const date = startDate === null ? null : `${japaneseDate(startDate)}に開催`
  const next: StageAction | null =
    startDate === null
      ? { label: '開催日を決める', href: `${base(input)}/settings` }
      : count === 0
        ? { label: '部門を作る', href: `${base(input)}/divisions/new` }
        : null
  return {
    key: 'setup',
    title: '大会の準備',
    status:
      date === null
        ? '開催日がまだ決まっていません'
        : count === 0
          ? `${date}。部門はまだありません`
          : `${date}。部門は ${input.divisions.map((division) => division.name).join('、')}`,
    chip: next === null ? { label: '済み', tone: 'ok' } : null,
    next,
    waiting: next !== null,
    note: null,
    numbers: null,
    links: [
      {
        label: '大会の設定',
        href: `${base(input)}/settings`,
        sub: '名前、開催日、公開、お知らせ',
      },
      {
        label: '部門を作る',
        href: `${base(input)}/divisions/new`,
        sub: 'オープン、女子など、順位を別に出す部門を足します',
      },
    ],
  }
}

function lineStage(input: HomeInput): Stage {
  const { listed, status } = input.tournament
  const entries = input.entries ?? { total: 0, pending: 0, approved: 0, noScorer: 0 }
  const counts = `申込 ${entries.total} 件、参加決定 ${entries.approved} 件`
  const scorers = `記録係が決まったチーム ${entries.approved - entries.noScorer} / ${entries.approved}`
  const send: StageAction = {
    label: '公式 LINE で送る',
    href: notifyPaths.create(input.tournamentId),
  }
  const links: StageLink[] = [
    {
      label: '申込',
      href: entryPaths.list(input.tournamentId),
      sub: counts,
      chip:
        entries.pending > 0 ? { label: `確認待ち ${entries.pending}`, tone: 'warn' } : undefined,
    },
    {
      label: '申込で聞く質問',
      href: entryPaths.questions(input.tournamentId),
      sub: '学年や連絡先など、申込のときに答えてもらうこと',
    },
    ...(entries.approved > 0
      ? [
          {
            label: '記録係を選んでもらう',
            href: notifyPaths.scorerRequest(input.tournamentId),
            sub: scorers,
            chip:
              entries.noScorer > 0
                ? { label: `まだ ${entries.noScorer}`, tone: 'warn' as const }
                : { label: '済み', tone: 'ok' as const },
          },
        ]
      : []),
    {
      label: '送った記録と予約',
      href: notifyPaths.list(input.tournamentId),
      sub:
        input.scheduled !== null && input.scheduled > 0
          ? `予約 ${input.scheduled} 件`
          : '送った文と、届いた人数',
    },
  ]
  const shared = { key: 'line' as const, title: '公式 LINE', note: null, numbers: null, links }
  if (status === 'finished') {
    return {
      ...shared,
      status: `大会は終わりました。${counts}`,
      chip: { label: '終了', tone: 'done' },
      next: null,
      waiting: false,
    }
  }
  if (entries.pending > 0) {
    return {
      ...shared,
      status: `確認待ちの申込が ${entries.pending} 件あります`,
      // 数は今の様子と申込の入口に出るので、段の札は付けない
      chip: null,
      next: { label: '確認待ちの申込を見る', href: entryPaths.list(input.tournamentId, 'pending') },
      waiting: true,
    }
  }
  if (status === 'entry_open' && !listed) {
    return {
      ...shared,
      status: '非公開なので、まだ公式 LINE から申し込めません',
      chip: { label: '非公開', tone: 'warn' },
      next: { label: '大会を公開する', href: `${base(input)}/settings` },
      waiting: true,
    }
  }
  if (status === 'entry_open') {
    return {
      ...shared,
      status:
        entries.total === 0
          ? '受付中です。代表が公式 LINE から申し込むと、ここに届きます'
          : `受付中です。${counts}`,
      chip: { label: '受付中', tone: 'sched' },
      next: send,
      waiting: false,
    }
  }
  // 受付を締めたら、参加が決まったチームに記録係を選んでもらう(自分たちの試合のスコアを入れる人)
  if (entries.approved > 0 && entries.noScorer > 0) {
    return {
      ...shared,
      status: `受付を締め切りました。記録係を選んでいないチームが ${entries.noScorer} あります`,
      chip: { label: '締切り', tone: 'sched' },
      next: { label: '記録係を選んでもらう', href: notifyPaths.scorerRequest(input.tournamentId) },
      waiting: true,
    }
  }
  return {
    ...shared,
    status: `受付を締め切りました。${counts}`,
    chip: { label: '締切り', tone: 'sched' },
    next: send,
    waiting: false,
  }
}

function divisionLinks(input: HomeInput, withFormat: boolean): StageLink[] {
  return input.divisions.map((division) => ({
    label: division.name,
    href: `/admin/divisions/${division.id}`,
    sub: [division.teams === null ? null : `${division.teams}チーム`, `${division.matches}試合`]
      .filter(Boolean)
      .join(' ・ '),
    chip:
      withFormat && !division.formatted
        ? { label: 'まだ組んでいない', tone: 'warn' as const }
        : undefined,
  }))
}

function buildStage(input: HomeInput): Stage {
  const shared = { key: 'build' as const, title: '試合を組む', numbers: null }
  // 部門ごとの進み具合(チーム → 組分け → 時間割)の表を先頭に置く
  const links: StageLink[] = [
    {
      label: '部門ごとの進み具合',
      href: `${base(input)}/build`,
      sub: 'チーム → 組分け → 時間割',
    },
    ...divisionLinks(input, true),
  ]
  if (input.divisions.length === 0) {
    return {
      ...shared,
      status: '部門を作ると、ここで組と試合を作れます',
      chip: null,
      next: null,
      waiting: false,
      note: null,
      links,
    }
  }
  const unformatted = input.divisions.filter((division) => !division.formatted)
  const first = unformatted[0]
  if (first === undefined) {
    const matches = input.divisions.reduce((sum, division) => sum + division.matches, 0)
    return {
      ...shared,
      status: `どの部門も形が決まっています。全部で ${matches} 試合`,
      chip: { label: '済み', tone: 'ok' },
      next: { label: '時間割を確かめる', href: `${base(input)}/timetable` },
      waiting: false,
      note: null,
      links,
    }
  }
  const pending = input.entries?.pending ?? 0
  return {
    ...shared,
    status: `${unformatted.map((division) => division.name).join('、')}の形がまだ決まっていません`,
    chip: null,
    next: { label: `${first.name}を組む`, href: `/admin/divisions/${first.id}/build` },
    waiting: true,
    // 形は作った時点のチームで組を作る。あとで参加を決めたチームは組に入らない
    note:
      pending > 0
        ? `確認待ちの申込が ${pending} 件あります。参加を決めてから形を作ると、そのチームも組に入ります`
        : null,
    links,
  }
}

function dayofStage(input: HomeInput): Stage {
  const { startDate } = input.tournament
  const near = showsDayNumbers(startDate, input.today, input.day)
  const canRun = input.role !== 'staff'
  const links: StageLink[] = [
    ...(input.hasBoard
      ? [
          {
            label: '本部',
            href: `${base(input)}/hq`,
            sub: '結果の入力、遅れの調整、組の確定',
            chip:
              input.day.alerts > 0
                ? { label: `知らせ ${input.day.alerts}`, tone: 'warn' as const }
                : undefined,
          },
        ]
      : []),
    ...(canRun
      ? [
          {
            label: '記録係',
            href: `${base(input)}/staff`,
            sub: `割り当て ${input.keepers} 件。当日の交代もここから`,
            chip:
              input.hasBoard && input.freeCourts.length > 0
                ? { label: `${input.freeCourts.join('・')}が空き`, tone: 'live' as const }
                : undefined,
          },
        ]
      : []),
    {
      label: '時間割と印刷',
      href: `${base(input)}/timetable`,
      sub: 'コートごとの時間割と、掲示用の印刷',
    },
  ]
  const shared = { key: 'dayof' as const, title: '当日', note: null, links }
  if (!input.hasBoard) {
    return {
      ...shared,
      status: '試合を組むと、本部で当日の進行ができます',
      chip: null,
      next: null,
      waiting: false,
      numbers: null,
    }
  }
  if (input.tournament.status === 'finished') {
    return {
      ...shared,
      status: `大会は終わりました。終わった試合 ${input.day.finished} / ${input.day.total}`,
      chip: { label: '終了', tone: 'done' },
      next: null,
      waiting: false,
      numbers: null,
    }
  }
  if (near) {
    const { live, alerts } = input.day
    return {
      ...shared,
      status:
        alerts > 0
          ? `本部の知らせが ${alerts} 件あります`
          : live > 0
            ? `いま ${live} 試合が進んでいます`
            : '本部で当日の進行をします',
      chip: live > 0 ? { label: '試合中', tone: 'live' } : null,
      next: { label: '本部を開く', href: `${base(input)}/hq` },
      waiting: true,
      numbers: input.day,
    }
  }
  const later =
    startDate === null ? '開催日が決まると' : `${japaneseDate(dayBefore(startDate))}から`
  if (canRun && input.freeCourts.length > 0) {
    return {
      ...shared,
      status:
        input.keepers === 0
          ? 'どのコートにも、まだ記録係がいません'
          : `${input.freeCourts.join('、')} に記録係がいません`,
      chip: null,
      next: { label: '記録係を決める', href: `${base(input)}/staff` },
      waiting: true,
      numbers: null,
    }
  }
  return {
    ...shared,
    status: `${later}、試合中の数と本部の知らせをここに出します`,
    chip: null,
    next: null,
    waiting: false,
    numbers: null,
  }
}

function divisionsStage(input: HomeInput): Stage {
  return {
    key: 'build',
    title: '部門',
    status:
      input.divisions.length === 0
        ? '部門はまだありません'
        : `部門は ${input.divisions.map((division) => division.name).join('、')}`,
    chip: null,
    next: null,
    waiting: false,
    note: null,
    numbers: null,
    links: divisionLinks(input, false),
  }
}

function moreStage(input: HomeInput): Stage {
  return {
    key: 'more',
    title: 'その他',
    status: '観戦アプリの見せ方と、写真とバナー',
    chip: null,
    next: null,
    waiting: false,
    note: null,
    numbers: null,
    links: [
      {
        label: '観戦アプリの表',
        href: `${base(input)}/live-tables`,
        sub: '星取表とトーナメント表。見せる表、列、記号、並び順',
      },
      {
        label: '写真とバナー',
        href: `${base(input)}/media`,
        sub: 'スポンサーのバナーと、大会のギャラリー',
      },
      { label: '観戦アプリを開く', href: `/live/${input.tournamentId}`, sub: '観る人と同じ画面' },
    ],
  }
}
