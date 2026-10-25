import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { type StandingsResponse, standingsResponseSchema } from '../src/api/public.types.ts'
import { DivisionStandingsView } from '../src/components/DivisionStandingsView.tsx'
import { loadFixture } from './fixtures/load.ts'
import { renderHtml } from './render.ts'

const standings = standingsResponseSchema.parse(loadFixture('standings-101'))

describe('ディビジョン成績の順位表(clone-spec C 1.1〜1.4)', () => {
  const html = renderHtml(<DivisionStandingsView data={standings} lang="jp" />)

  test('見出しと 10 列の幅', () => {
    assert.match(
      html,
      /^<span class="SCV_MIDA_M">&lt; ビーチミックス 順位表 &gt;<\/span><div class="TBL"><table><tbody><tr><td class="MD1 CM" width="24%">チーム名<\/td><td class="MD1 CM" width="8.444%">試合数<\/td>/,
    )
    assert.match(
      html,
      /<td class="MD1 CM" width="8.444%">勝率<\/td><td class="MD1 CM" width="8.444%">順位<\/td><\/tr>/,
    )
  })

  test('行: チーム名セルは MD1、得点・失点・得失差・順位の末尾に空白、順位は HL2', () => {
    assert.match(
      html,
      /<tr><td class="MD1"><span class="teamimg_rnk"><a href="\/pscore\/scv_team\/open\/5001\?lang=jp">架空チームA<\/a><\/span> <\/td><td class="CM HL1">5<\/td><td class="CM">5<\/td><td class="CM">0<\/td><td class="CM">0<\/td><td class="CM">38 <\/td><td class="CM">19 <\/td><td class="CM HL1">19 <\/td><td class="CM HL1">100%<\/td><td class="CM HL2">1 <\/td><\/tr>/,
    )
  })

  test('勝率は引き分けを勝ちに数えない。試合数 0 は --%、順位なしは空', () => {
    assert.match(html, /<td class="CM HL1">10 <\/td><td class="CM HL1">66.67%<\/td>/)
    assert.match(
      html,
      /<td class="CM HL1">0 <\/td><td class="CM HL1">--%<\/td><td class="CM HL2"><\/td><\/tr>/,
    )
  })

  test('勝敗は「○勝○敗○分」と読む順に、勝ち数、負け数、分け数で並べる(参照サイトは勝・分・負)', () => {
    const [first, ...rest] = standings.standings
    assert.ok(first)
    const data = { ...standings, standings: [{ ...first, wins: 3, losses: 2, draws: 1 }, ...rest] }
    const out = renderHtml(<DivisionStandingsView data={data} lang="jp" />)
    assert.match(out, />勝ち数<\/td><td [^>]*>負け数<\/td><td [^>]*>分け数<\/td>/)
    assert.match(
      out,
      /<td class="CM">3<\/td><td class="CM">2<\/td><td class="CM">1<\/td><td class="CM">38 <\/td>/,
    )
  })

  test('英語の見出しは空白なしの山括弧と改行入りの列名', () => {
    const en = renderHtml(<DivisionStandingsView data={standings} lang="en" />)
    assert.match(en, /^<span class="SCV_MIDA_M">&lt;Beach Mixed Standings&gt;<\/span>/)
    assert.match(en, />Teams<\/td>.*>Games<\/td>.*>Draw<\/td>.*>Goals<br\/>for<\/td>.*>Win %<\/td>/)
  })
})

describe('スコアボードリーダー(clone-spec C 1.5)', () => {
  const html = renderHtml(<DivisionStandingsView data={standings} lang="jp" />)

  test('得点表: チーム名はリンクなし、# の幅は " 5%"、トータルは得点+アシスト', () => {
    assert.match(
      html,
      /<hr\/><span class="SCV_MIDA_M">&lt; スコアボードリーダー &gt;<\/span><div class="TBL"><table><tbody><tr><td class="MD1 CM" width="25%">チーム名<\/td><td class="MD1 CM" width=" 5%">#<\/td><td class="MD1 CM" width="30%">選手<\/td><td class="MD1 CM" width="10%">試合数<\/td><td class="MD1 CM" width="10%">アシスト<\/td><td class="MD1 CM" width="10%">ゴール<\/td><td class="MD1 CM" width="10%">トータル<\/td><\/tr>/,
    )
    assert.match(
      html,
      /<tr><td class="LM"><span class="teamimg">架空チームA<\/span> <\/td><td class="CM">7<\/td><td class="LM"><a class="playerLinkM" href="\/pscore\/scv_player\/open\/70001\?lang=jp">架空 太郎<\/a><\/td><td class="CM HL1">5<\/td><td class="CM">6<\/td><td class="CM">4<\/td><td class="CM HL1">10 <\/td><\/tr>/,
    )
  })

  test('ブロック表: 全角空白入りの見出し、最終列は "--" で行は空', () => {
    assert.match(html, /&lt; スコアボードリーダー　（ブロック） &gt;/)
    assert.match(
      html,
      /<td class="MD1 CM" width="10%">ブロック<\/td><td class="MD1 CM" width="20%">--<\/td><\/tr>/,
    )
    assert.match(
      html,
      /<a class="playerLinkF" href="\/pscore\/scv_player\/open\/70011\?lang=jp">見本 花子<\/a><\/td><td class="CM HL1">6<\/td><td class="CM HL1">3 <\/td><td class="CM HL1"><\/td><\/tr><\/tbody><\/table><\/div><hr\/>$/,
    )
    const en = renderHtml(<DivisionStandingsView data={standings} lang="en" />)
    assert.match(en, /&lt;Scoreboard Leaders \*Block\*&gt;/)
    assert.match(en, />Team<\/td>.*>Player<\/td>.*>Tot\.<\/td>/)
  })

  test('データのない節は出さない', () => {
    const onlyStandings: StandingsResponse = { ...standings, leaders: [], blockLeaders: [] }
    const html2 = renderHtml(<DivisionStandingsView data={onlyStandings} lang="jp" />)
    assert.doesNotMatch(html2, /スコアボードリーダー/)
    assert.match(html2, /<\/table><\/div><hr\/>$/)
  })

  test('性別のない選手は playerLink', () => {
    const [leader] = standings.leaders
    assert.ok(leader)
    const data: StandingsResponse = {
      ...standings,
      leaders: [{ ...leader, player: { ...leader.player, gender: null } }],
    }
    assert.match(
      renderHtml(<DivisionStandingsView data={data} lang="jp" />),
      /<a class="playerLink" href=/,
    )
  })
})
