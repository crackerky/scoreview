import assert from 'node:assert/strict'
import { test } from 'node:test'
import {
  addMinutes,
  buildAlerts,
  confirmEffects,
  courtCards,
  type DivisionBoard,
  divisionsOnCourt,
  groupCards,
  matchesByTime,
  shiftPreview,
  staleDelayMinutes,
  suggestedShift,
} from '../src/admin/hqBoard.utils.ts'
import type { Assignment } from '../src/api/admin.types.ts'
import type { GroupProgress, Progress, ProgressMatch, Todo } from '../src/api/format.types.ts'

// 大会単位の本部のまとめ方(src/admin/hqBoard.utils.ts)

const team = (id: number, name: string) => ({ id, name: { ja: name, en: '' } })

function match(
  id: number,
  field: string,
  time: string,
  status: ProgressMatch['status'],
  teams: [string, string] | null = ['湘南', '港北'],
): ProgressMatch {
  return {
    id,
    code: `M${id}`,
    groupName: 'A',
    groupLabel: '予選 A 組',
    date: '2026-10-25',
    time,
    field,
    status,
    left: { label: 'A1', team: teams ? team(id * 10, teams[0]) : null, score: null },
    right: { label: 'A2', team: teams ? team(id * 10 + 1, teams[1]) : null, score: null },
  }
}

function todo(kind: Todo['kind'], overrides: Partial<Todo> = {}): Todo {
  return {
    kind,
    text: `${kind} の知らせ`,
    groupId: null,
    stageId: null,
    matchId: null,
    teamId: null,
    court: null,
    minutes: 0,
    ...overrides,
  }
}

function group(id: number, state: GroupProgress['state'], label = 'A 組'): GroupProgress {
  return {
    id,
    letter: 'A',
    name: 'A',
    label,
    kind: 'pool',
    stage: 1,
    state,
    confirmedUnplayed: false,
    remaining: state === 'playing' ? 2 : 0,
    slots: [],
    rows: [
      { ...tally(), team: team(1, '湘南'), rank: 1, points: 6, reason: '', unresolved: false },
      { ...tally(), team: team(2, '港北'), rank: 2, points: 3, reason: '', unresolved: false },
    ],
    ties: [],
    conditions: [],
  }
}

function tally() {
  return { games: 2, wins: 1, draws: 0, losses: 1, goalsFor: 10, goalsAgainst: 9 }
}

function board(
  divisionId: number,
  todos: Todo[],
  matches: ProgressMatch[],
  groups: GroupProgress[] = [],
): DivisionBoard {
  const progress: Progress = {
    divisionId,
    title: '',
    version: 1,
    tiebreak: 'current',
    ruleText: [],
    advancement: [],
    slotMinutes: 20,
    courts: [...new Set(matches.map((item) => item.field ?? ''))],
    now: '2026-10-25T02:00:00Z',
    todo: todos,
    stages: [],
    groups,
    live: matches.filter((item) => item.status === 'live'),
    next: matches.filter((item) => item.status === 'scheduled'),
    final: [],
    withdrawn: [],
    closedAt: null,
    suspension: null,
  }
  return { divisionId, divisionName: `部門${divisionId}`, progress, matches }
}

const today = '2026-10-25'

test('同じコートの遅れは部門をまたいで 1 件にし、長い方の分数を出す。前の日から残った遅れは出さない', () => {
  const boards = [
    board(
      1,
      [todo('court_delay', { court: '2', minutes: 12 })],
      [match(1, '2', '11:20', 'scheduled')],
    ),
    board(
      2,
      [
        todo('court_delay', { court: '2', minutes: 8 }),
        todo('court_delay', { court: '3', minutes: staleDelayMinutes + 1 }),
      ],
      [match(2, '2', '11:40', 'scheduled'), match(3, '3', '11:40', 'scheduled')],
    ),
  ]
  const alerts = buildAlerts(boards, 9, [], today)
  assert.deepEqual(
    alerts.map((alert) => alert.title),
    ['コート 2が 12 分遅れています'],
  )
  assert.match(alerts[0]?.body ?? '', /このあとの 2 試合/)
  assert.equal(alerts[0]?.links[0]?.href, '/admin/tournaments/9/hq/shift?court=2')
})

test('結果待ちは 1 件にまとめ、試合ごとに結果の入力へのリンクを付ける', () => {
  const boards = [
    board(
      1,
      [todo('overdue', { matchId: 1 }), todo('overdue', { matchId: 2 })],
      [match(1, '1', '10:00', 'scheduled'), match(2, '2', '10:00', 'scheduled')],
    ),
  ]
  const [alert] = buildAlerts(boards, 9, [], today)
  assert.equal(alert?.title, '終わる時刻を過ぎて結果が入っていない試合が 2 件あります')
  assert.deepEqual(
    alert?.links.map((link) => link.label),
    ['M1(コート 1)の結果を入れる', 'M2(コート 2)の結果を入れる'],
  )
  assert.equal(
    alert?.links[0]?.href,
    '/admin/matches/1/result?next=%2Fadmin%2Ftournaments%2F9%2Fhq',
  )
})

test('試合中の試合のコートに今日の記録係がいなければ知らせる', () => {
  const live = [match(1, '1', '11:00', 'live'), match(2, '2', '11:00', 'live')]
  const keeper: Assignment = {
    id: 1,
    userId: 5,
    userName: '佐藤',
    matchId: null,
    matchCode: null,
    field: '1',
    date: today,
    from: null,
    to: null,
  }
  const alerts = buildAlerts([board(1, [], live)], 9, [keeper], today)
  assert.deepEqual(
    alerts.map((alert) => [alert.chip, alert.title]),
    [['記録係なし', '記録係のいない試合が 1 件あります']],
  )
})

test('確定待ちの組は確定の画面へ、同順位は進行画面へ', () => {
  const boards = [
    board(
      4,
      [todo('confirm_group', { groupId: 7 }), todo('tie', { groupId: 8 })],
      [],
      [group(7, 'ready'), group(8, 'tied', 'B 組')],
    ),
  ]
  const alerts = buildAlerts(boards, 9, [], today)
  assert.deepEqual(
    alerts.map((alert) => [alert.chip, alert.links[0]?.href]),
    [
      ['確定待ち', '/admin/tournaments/9/hq/confirm/7'],
      ['同順位', '/admin/divisions/4/progress'],
    ],
  )
})

test('コートの札: 試合中の試合、次の試合、遅れ、記録係', () => {
  const boards = [
    board(
      1,
      [todo('court_delay', { court: '1', minutes: 5 })],
      [
        match(1, '1', '11:00', 'live'),
        match(2, '1', '11:20', 'scheduled'),
        match(3, '2', '11:20', 'scheduled'),
      ],
    ),
  ]
  const cards = courtCards(boards, [], today)
  assert.deepEqual(
    cards.map((card) => [card.name, card.live?.id ?? null, card.next?.id ?? null, card.delay]),
    [
      ['コート 1', 1, 2, 5],
      ['コート 2', null, 3, null],
    ],
  )
})

test('組の札: 部門が複数なら部門名を付け、確定待ちに確定のリンク', () => {
  const boards = [board(1, [], [], [group(7, 'ready')]), board(2, [], [], [group(8, 'playing')])]
  const cards = groupCards(boards, 9)
  assert.deepEqual(
    cards.map((card) => [card.name, card.chip, card.link?.label ?? null]),
    [
      ['部門1 A 組', '確定待ち', '確定する'],
      ['部門2 A 組', '試合中', null],
    ],
  )
  assert.equal(cards[0]?.sub, '1位 湘南 ・ 2位 港北')
})

test('試合は日付と時刻ごとにまとめ、時刻順に並べる', () => {
  const boards = [
    board(1, [], [match(2, '1', '11:00', 'scheduled'), match(1, '2', '10:00', 'live')]),
    board(2, [], [match(3, '3', '10:00', 'scheduled')]),
  ]
  assert.deepEqual(
    matchesByTime(boards).map((group) => [group.time, group.rows.map((row) => row.id)]),
    [
      ['10:00', [1, 3]],
      ['11:00', [2]],
    ],
  )
})

test('時間をずらす下見: そのコートのまだ始まっていない試合だけを、時刻を足して並べる', () => {
  const boards = [
    board(1, [], [match(1, '1', '09:40', 'live'), match(2, '1', '10:00', 'scheduled')]),
    board(2, [], [match(3, '1', '10:20', 'scheduled'), match(4, '2', '10:20', 'scheduled')]),
  ]
  assert.deepEqual(
    shiftPreview(boards, '1', 15).map((line) => [line.match.id, line.before, line.after]),
    [
      [2, '10:00', '10:15'],
      [3, '10:20', '10:35'],
    ],
  )
  assert.deepEqual(divisionsOnCourt(boards, '1'), [1, 2])
  assert.deepEqual(divisionsOnCourt(boards, '2'), [2])
})

test('時刻の足し算と、遅れに合うずらす時間の初めの選択', () => {
  assert.equal(addMinutes('09:50', 15), '10:05')
  assert.equal(addMinutes('23:50', 20), '24:10')
  assert.equal(suggestedShift(null), 10)
  assert.equal(suggestedShift(12), 15)
  assert.equal(suggestedShift(3), 5)
  assert.equal(suggestedShift(80), 20)
})

test('組を確定すると入る先: この組の順位を指す試合に、今の順位のチームを添える', () => {
  const pending = (id: number, left: string, right: string): ProgressMatch => ({
    ...match(id, '1', '10:40', 'scheduled', null),
    groupLabel: '1〜4 位を決めるトーナメント',
    left: { label: left, team: null, score: null },
    right: { label: right, team: null, score: null },
  })
  const target = board(1, [], [pending(1, 'A1', 'B2'), pending(2, 'C1', 'D2')], [group(7, 'ready')])
  const effects = confirmEffects(target, group(7, 'ready'))
  assert.deepEqual(
    effects.map((effect) => [effect.match.id, effect.left, effect.right]),
    [[1, '湘南(A組1位)', 'B組2位']],
  )
})
