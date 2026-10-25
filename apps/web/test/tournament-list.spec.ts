import assert from 'node:assert/strict'
import { test } from 'node:test'
import type { TournamentSummary } from '../src/api/public.types.ts'
import {
  fillTemplate,
  recentFinishedCount,
  splitFinished,
} from '../src/components/tournamentList.utils.ts'

const tournament = (id: number, startDate: string | null): TournamentSummary => ({
  id,
  name: { ja: `大会${id}`, en: '' },
  startDate,
  teamCount: 0,
})

test('終了した大会が 5 件以下なら、たたむものはない', () => {
  const list = [tournament(1, '2026-10-01'), tournament(2, '2025-10-01')]
  assert.deepEqual(splitFinished(list), { recent: list, older: [] })
})

test('6 件目からを api の並びのまま年ごとにまとめ、開催日未定は null の年にする', () => {
  const list = [
    ...Array.from({ length: recentFinishedCount }, (_, index) => tournament(index, '2026-05-01')),
    tournament(10, '2026-01-01'),
    tournament(11, '2025-12-01'),
    tournament(12, '2025-03-01'),
    tournament(13, null),
  ]
  const { recent, older } = splitFinished(list)
  assert.equal(recent.length, recentFinishedCount)
  assert.deepEqual(
    older.map((group) => [group.year, group.tournaments.map((item) => item.id)]),
    [
      [2026, [10]],
      [2025, [11, 12]],
      [null, [13]],
    ],
  )
})

test('テンプレートの {n} などを埋め、知らない名前はそのまま残す', () => {
  assert.equal(fillTemplate('{year}年({n} 大会)', { year: 2025, n: 3 }), '2025年(3 大会)')
  assert.equal(fillTemplate('{n} と {x}', { n: 1 }), '1 と {x}')
})
