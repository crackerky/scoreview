import assert from 'node:assert/strict'
import { test } from 'node:test'
import {
  groupTournaments,
  japaneseDate,
  jstToday,
  phaseChip,
  searchTournaments,
  tournamentPhase,
} from '../src/admin/tournamentList.utils.ts'
import type { AdminTournament } from '../src/api/admin.types.ts'

// 管理画面の大会一覧の分け方(src/admin/tournamentList.utils.ts)

function tournament(
  id: number,
  startDate: string | null,
  overrides: Partial<AdminTournament> = {},
): AdminTournament {
  return {
    id,
    name: { ja: `大会${id}`, en: '' },
    shortName: '',
    status: 'entry_open',
    listed: true,
    startDate,
    logoUrl: null,
    topHtml: { ja: '', en: '' },
    labels: { teamList: 'teams', groups: 'groups' },
    showSchedule: true,
    forfeitScore: null,
    ...overrides,
  }
}

const today = '2026-10-25'

test('今日は日本時間の暦の日(UTC では前の日の 15 時以降)', () => {
  assert.equal(jstToday(new Date('2026-10-24T15:30:00Z')), '2026-10-25')
  assert.equal(jstToday(new Date('2026-10-24T14:59:00Z')), '2026-10-24')
})

test('開催日を「10月25日（日）」の形にする', () => {
  assert.equal(japaneseDate('2026-10-25'), '10月25日（日）')
  assert.equal(japaneseDate('2027-05-03'), '5月3日（月）')
})

test('段階: 今日なら開催中、先か未定ならこれから、過ぎたか終了にしたなら終わった', () => {
  assert.equal(tournamentPhase(tournament(1, today), today), 'live')
  assert.equal(tournamentPhase(tournament(2, '2026-11-03'), today), 'upcoming')
  assert.equal(tournamentPhase(tournament(3, null), today), 'upcoming')
  assert.equal(tournamentPhase(tournament(4, '2026-10-21'), today), 'done')
  assert.equal(tournamentPhase(tournament(5, today, { status: 'finished' }), today), 'done')
})

test('札: これからの大会は非公開なら下書き、公開中なら受付中か締切り', () => {
  const upcoming = (overrides: Partial<AdminTournament>) =>
    phaseChip(tournament(1, '2026-11-03', overrides), 'upcoming').label
  assert.equal(upcoming({ listed: false }), '下書き')
  assert.equal(upcoming({}), '受付中')
  assert.equal(upcoming({ status: 'entry_closed' }), '締切り')
  assert.equal(phaseChip(tournament(1, today), 'live').label, '開催中')
  assert.equal(phaseChip(tournament(1, '2026-01-01'), 'done').label, '終了')
})

test('区分は開催中、これから(近い順、未定は最後)、終わった(新しい順)。空の区分は出さない', () => {
  const groups = groupTournaments(
    [
      tournament(1, '2026-09-01'),
      tournament(2, null),
      tournament(3, '2026-12-01'),
      tournament(4, '2026-11-03'),
      tournament(5, '2026-10-01'),
    ],
    today,
  )
  assert.deepEqual(
    groups.map((group) => [group.title, group.tournaments.map((item) => item.id)]),
    [
      ['これからの大会', [4, 3, 2]],
      ['終わった大会', [5, 1]],
    ],
  )
})

test('名前で絞る(全部の語を含む大会)', () => {
  const list = [
    tournament(1, null, { name: { ja: '架空ビーチリーグ 第3戦', en: '' } }),
    tournament(2, null, { name: { ja: '架空カップ', en: 'Fictional Cup' } }),
  ]
  assert.deepEqual(
    searchTournaments(list, 'ビーチ 第3').map((item) => item.id),
    [1],
  )
  assert.deepEqual(
    searchTournaments(list, 'cup').map((item) => item.id),
    [2],
  )
})
