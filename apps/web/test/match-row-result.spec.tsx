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
  winnerSide: 'left',
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

  test('不戦勝は勝ったチームの名前と種類を出し、試合ページがあればリンクにする', () => {
    const html = render({ ...base, resultType: 'forfeit' })
    assert.match(
      html,
      /<a class="Select_Link" href="\/pscore\/scv_match\/open\/9\?lang=jp">架空A の不戦勝<\/a>/,
    )
    assert.doesNotMatch(html, /対戦結果/)
    const right = render({ ...base, resultType: 'forfeit', winnerSide: 'right' }, 'en')
    assert.match(right, />Forfeit win: B<\/a>/)
  })

  test('勝者が分からない不戦勝は種類だけ、両者不戦敗と打切は種類だけ', () => {
    assert.match(render({ ...base, resultType: 'forfeit', winnerSide: null }), />不戦勝<\/a>/)
    assert.match(
      render({ ...base, resultType: 'double_forfeit', winnerSide: null }),
      />両者不戦敗<\/a>/,
    )
    assert.match(render({ ...base, resultType: 'abandoned' }, 'en'), />Abandoned<\/a>/)
  })

  test('不成立は勝ったチームを括弧で添える', () => {
    const html = render({ ...base, resultType: 'no_contest', winnerSide: 'right' })
    assert.match(html, />不成立\(架空B の勝ち\)<\/a>/)
  })

  test('スケジュールでは名前を行のチーム名と同じ幅(16)で切り詰める', () => {
    const long = { id: 1, name: { ja: 'とても長い架空のチームの名前', en: 'A' } }
    const html = renderHtml(
      <table>
        <tbody>
          <MatchRowLine
            row={{ ...base, resultType: 'forfeit', left: { ...base.left, team: long } }}
            lang="jp"
            variant={{
              kind: 'schedule',
              division: { name: { ja: 'ミックス', en: '' }, kind: 'mixed' },
            }}
          />
        </tbody>
      </table>,
    )
    assert.match(html, />とても長い架空の の不戦勝<\/a>/)
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
