import assert from 'node:assert/strict'
import { test } from 'node:test'
import { matchResponseSchema } from '../src/api/public.types.ts'
import { MatchView } from '../src/components/MatchView.tsx'
import { loadFixture } from './fixtures/load.ts'
import { renderHtml } from './render.ts'

const base = matchResponseSchema.parse(loadFixture('match-9001'))

test('不戦勝はスコアボード下の行に勝ったチームの名前つきで出す(本サイト独自)', () => {
  const forfeit = {
    ...base,
    match: { ...base.match, resultType: 'forfeit' as const, winnerSide: 'left' as const },
  }
  const html = renderHtml(<MatchView data={forfeit} lang="jp" now={new Date(0)} />)
  const name = base.match.left.team?.name.ja ?? ''
  assert.ok(html.includes(`${name} の不戦勝`))
  assert.match(html, /data-scoreview-addition/)
})

test('通常の試合は何も足さない', () => {
  const html = renderHtml(<MatchView data={base} lang="jp" now={new Date(0)} />)
  assert.doesNotMatch(html, /不戦勝/)
})
