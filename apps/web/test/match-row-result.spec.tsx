import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import type { MatchRow } from '../src/api/common.types.ts'
import { MatchRowLine } from '../src/components/MatchRowLine.tsx'
import { renderHtml } from './render.ts'

const base: MatchRow = {
  matchId: 9,
  hasDetail: true,
  code: 'A1',
  date: '2026-10-18',
  time: '09:00',
  field: '1',
  status: 'finished',
  resultType: 'normal',
  left: {
    team: { id: 1, name: { ja: '架空A', en: 'A' } },
    placeholder: null,
    score: 15,
    labelEn: '',
  },
  right: {
    team: { id: 2, name: { ja: '架空B', en: 'B' } },
    placeholder: null,
    score: 0,
    labelEn: '',
  },
}

function render(row: MatchRow, lang: 'jp' | 'en' = 'jp'): string {
  return renderHtml(
    <table>
      <tbody>
        <MatchRowLine row={row} lang={lang} variant={{ kind: 'list' }} />
      </tbody>
    </table>,
  )
}

describe('状況列の結果の種類', () => {
  test('通常の終了試合は [対戦結果] のリンク', () => {
    assert.match(render(base), /\[対戦結果\]<\/a>/)
  })

  test('不戦勝は種類を出し、試合ページがあればリンクにする', () => {
    const html = render({ ...base, resultType: 'forfeit' })
    assert.match(
      html,
      /<a class="Select_Link" href="\/pscore\/scv_match\/open\/9\?lang=jp">不戦勝<\/a>/,
    )
    assert.doesNotMatch(html, /対戦結果/)
  })

  test('試合ページのない未消化はリンクなしの文字', () => {
    const html = render(
      { ...base, hasDetail: false, status: 'scheduled', resultType: 'unplayed' },
      'en',
    )
    assert.match(html, />Unplayed</)
    assert.doesNotMatch(html, /scv_match/)
  })
})
