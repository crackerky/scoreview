import assert from 'node:assert/strict'
import { test } from 'node:test'
import {
  currentStage,
  dayBefore,
  type HomeInput,
  type Stage,
  showsDayNumbers,
  tournamentStages,
} from '../src/admin/tournamentHome.utils.ts'

// 大会ホームの段のカード(src/admin/tournamentHome.utils.ts)

const noDay = { live: 0, finished: 0, total: 0, alerts: 0 }

function input(overrides: Partial<HomeInput> = {}): HomeInput {
  return {
    tournamentId: 5,
    tournament: { startDate: '2027-01-17', listed: true, status: 'entry_open' },
    today: '2026-12-01',
    role: 'admin',
    divisions: [
      { id: 11, name: 'オープン', formatted: true, teams: 8, matches: 20 },
      { id: 12, name: '女子', formatted: false, teams: 4, matches: 0 },
    ],
    entries: { total: 12, pending: 0, approved: 12 },
    scheduled: 0,
    keepers: 0,
    freeCourts: [],
    hasBoard: true,
    day: noDay,
    ...overrides,
  }
}

function stage(stages: Stage[], key: Stage['key']): Stage {
  const found = stages.find((item) => item.key === key)
  assert.ok(found, `段 ${key} がない`)
  return found
}

test('管理者には段を準備、公式 LINE、試合を組む、当日、その他の順に出す', () => {
  assert.deepEqual(
    tournamentStages(input()).map((item) => item.key),
    ['setup', 'line', 'build', 'dayof', 'more'],
  )
})

test('本部スタッフと記録係には当日の段と部門の一覧だけを出し、記録係には記録係の入口を出さない', () => {
  const hq = tournamentStages(input({ role: 'hq' }))
  assert.deepEqual(
    hq.map((item) => item.title),
    ['当日', '部門'],
  )
  assert.ok(stage(hq, 'dayof').links.some((link) => link.label === '記録係'))
  const staff = tournamentStages(input({ role: 'staff' }))
  assert.equal(
    stage(staff, 'dayof').links.some((link) => link.label === '記録係'),
    false,
  )
  assert.equal(stage(staff, 'build').links[1]?.chip, undefined)
})

test('開催日が未定なら、準備の段で開催日を決めさせ、そこを今の段にする', () => {
  const stages = tournamentStages(
    input({ tournament: { startDate: null, listed: false, status: 'entry_open' } }),
  )
  const setup = stage(stages, 'setup')
  assert.equal(setup.next?.label, '開催日を決める')
  assert.equal(setup.next?.href, '/admin/tournaments/5/settings')
  assert.equal(currentStage(stages)?.key, 'setup')
})

test('部門が無ければ部門を作らせ、試合を組む段は手を付けられない', () => {
  const stages = tournamentStages(input({ divisions: [], hasBoard: false }))
  assert.equal(stage(stages, 'setup').next?.label, '部門を作る')
  assert.equal(stage(stages, 'build').next, null)
  assert.equal(stage(stages, 'dayof').next, null)
})

test('準備が済んでいれば済みの札を付け、次にやることは出さない', () => {
  const setup = stage(tournamentStages(input()), 'setup')
  assert.equal(setup.chip?.label, '済み')
  assert.equal(setup.next, null)
  assert.match(setup.status, /1月17日（日）に開催。部門は オープン、女子/)
})

test('確認待ちの申込があれば公式 LINE を今の段にし、確認待ちだけの一覧へ送る', () => {
  const stages = tournamentStages(input({ entries: { total: 14, pending: 2, approved: 12 } }))
  const line = stage(stages, 'line')
  assert.equal(line.next?.href, '/admin/tournaments/5/entries?status=pending')
  assert.equal(line.links.find((link) => link.label === '申込')?.chip?.label, '確認待ち 2')
  assert.equal(currentStage(stages)?.key, 'line')
  // 形はその時点のチームで作るので、参加を決める前に作ると組から漏れることを知らせる
  assert.match(stage(stages, 'build').note ?? '', /参加を決めてから形を作ると/)
})

test('受付中で非公開なら、公式 LINE から申し込めないことを伝えて公開させる', () => {
  const line = stage(
    tournamentStages(
      input({ tournament: { startDate: '2027-01-17', listed: false, status: 'entry_open' } }),
    ),
    'line',
  )
  assert.equal(line.next?.label, '大会を公開する')
  assert.equal(line.waiting, true)
})

test('受付中で待っている申込が無ければ、公式 LINE で送ることを次にし、今の段にはしない', () => {
  const line = stage(tournamentStages(input()), 'line')
  assert.equal(line.chip?.label, '受付中')
  assert.equal(line.next?.label, '公式 LINE で送る')
  assert.equal(line.waiting, false)
})

test('予約があれば送った記録の入口に数を出す', () => {
  const line = stage(tournamentStages(input({ scheduled: 2 })), 'line')
  assert.equal(line.links.find((link) => link.label === '送った記録と予約')?.sub, '予約 2 件')
})

test('形の決まっていない部門があれば、その部門の形を決めさせる', () => {
  const stages = tournamentStages(input())
  const build = stage(stages, 'build')
  assert.equal(build.next?.label, '女子の形を決める')
  assert.equal(build.next?.href, '/admin/divisions/12/format')
  assert.equal(build.links.find((link) => link.label === '女子')?.chip?.label, '形を決める')
  assert.equal(currentStage(stages)?.key, 'build')
})

test('どの部門も形があれば、時間割を確かめることを次にする', () => {
  const build = stage(
    tournamentStages(
      input({ divisions: [{ id: 11, name: 'オープン', formatted: true, teams: 8, matches: 20 }] }),
    ),
    'build',
  )
  assert.equal(build.chip?.label, '済み')
  assert.equal(build.next?.label, '時間割を確かめる')
})

test('当日の数字は開催日の前日から出し、それまでは出さない', () => {
  assert.equal(dayBefore('2027-01-17'), '2027-01-16')
  assert.equal(dayBefore('2027-03-01'), '2027-02-28')
  assert.equal(showsDayNumbers('2027-01-17', '2027-01-15', noDay), false)
  assert.equal(showsDayNumbers('2027-01-17', '2027-01-16', noDay), true)
  assert.equal(showsDayNumbers(null, '2027-01-16', noDay), false)
  // 開催日が未定でも試合が始まっていれば出す
  assert.equal(showsDayNumbers(null, '2027-01-16', { ...noDay, finished: 1 }), true)

  const before = stage(tournamentStages(input()), 'dayof')
  assert.equal(before.numbers, null)
  assert.match(before.status, /1月16日（土）から/)
})

test('前日からは本部を開くことを次にし、今の段にする', () => {
  const divisions = [{ id: 11, name: 'オープン', formatted: true, teams: 8, matches: 20 }]
  const day = { live: 2, finished: 5, total: 20, alerts: 1 }
  const stages = tournamentStages(input({ divisions, today: '2027-01-17', day }))
  const dayof = stage(stages, 'dayof')
  assert.equal(dayof.next?.label, '本部を開く')
  assert.deepEqual(dayof.numbers, day)
  assert.equal(dayof.status, '本部の知らせが 1 件あります')
  assert.equal(currentStage(stages)?.key, 'dayof')
})

test('前日より前でも、記録係のいないコートがあれば記録係を決めさせる', () => {
  const dayof = stage(tournamentStages(input({ freeCourts: ['A', 'B'], keepers: 3 })), 'dayof')
  assert.equal(dayof.next?.label, '記録係を決める')
  assert.equal(dayof.status, 'A、B に記録係がいません')
  const none = stage(tournamentStages(input({ freeCourts: ['A', 'B'], keepers: 0 })), 'dayof')
  assert.equal(none.status, 'どのコートにも、まだ記録係がいません')
})

test('試合を組む前は、当日の段に次にやることを出さない', () => {
  const dayof = stage(tournamentStages(input({ hasBoard: false })), 'dayof')
  assert.equal(dayof.next, null)
  assert.equal(
    dayof.links.some((link) => link.label === '本部'),
    false,
  )
})

test('終わった大会は公式 LINE の段を終了にし、今の段にはしない', () => {
  const stages = tournamentStages(
    input({
      tournament: { startDate: '2026-11-01', listed: true, status: 'finished' },
      divisions: [{ id: 11, name: 'オープン', formatted: true, teams: 8, matches: 20 }],
      day: { live: 0, finished: 20, total: 20, alerts: 0 },
    }),
  )
  assert.equal(stage(stages, 'line').chip?.label, '終了')
  assert.equal(stage(stages, 'line').next, null)
  assert.equal(stage(stages, 'dayof').next, null)
  assert.equal(currentStage(stages), null)
})
