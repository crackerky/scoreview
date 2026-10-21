import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { teamResponseSchema } from '../src/api/public.types.ts'
import { TeamView } from '../src/components/TeamView.tsx'
import { loadFixture } from './fixtures/load.ts'
import { renderHtml, renderHtmlWithNotes } from './render.ts'

const team = teamResponseSchema.parse(loadFixture('team-5001'))
const rosterOnly = teamResponseSchema.parse(loadFixture('team-5003'))

describe('チームページ(clone-spec D 1.1)', () => {
  const html = renderHtml(<TeamView data={team} lang="jp" />)

  test('見出しは正式名 + <br>、紹介コメントは見出し(と本サイト独自の追記)の直後', () => {
    assert.match(
      html,
      /^<span class="SCV_MIDA_M">架空チームA フライングクラブ<br\/><\/span><div class="sv_team_notes" data-scoreview-addition="">.*?<\/div><div class="SCV_TEAM_COMM">架空のチーム紹介です。<\/div><div class="TBL">/,
    )
  })

  test('追記: 次の試合(日時、コート、相手、コード)と所属グループへのリンク', () => {
    assert.match(
      html,
      /<p>次の試合: 10\/19 10:40 コート 3 vs 架空チームB\(G2\)<\/p><p>所属グループ: <a href="\/pscore\/scv_group\/open\/1001\?lang=jp">Qualifying pool A<\/a><\/p>/,
    )
    const en = renderHtml(<TeamView data={team} lang="en" />)
    assert.match(en, /<p>Next match: 10\/19 10:40 Field 3 vs Fictional B \(G2\)<\/p><p>Group: <a /)
  })

  test('追記: 試合中なら「試合中」、未確定の相手は決まり方、無ければ追記ごと出さない', () => {
    const matches = team.matches.map((row) =>
      row.matchId === 9003
        ? {
            ...row,
            status: 'live' as const,
            right: {
              ...row.right,
              team: null,
              placeholder: { code: 'A1', outcome: 'win' as const },
            },
          }
        : row,
    )
    const live = renderHtml(<TeamView data={{ ...team, matches }} lang="jp" />)
    assert.match(live, /<p>試合中: 10\/19 10:40 コート 3 vs \[A1-勝ち\]\(G2\)<\/p>/)
    const none = renderHtml(
      <TeamView data={{ ...team, nextMatchId: null, currentGroup: null }} lang="jp" />,
    )
    assert.doesNotMatch(none, /sv_team_notes/)
    const unlinked = renderHtml(
      <TeamView
        data={{
          ...team,
          nextMatchId: null,
          currentGroup: { id: 1002, name: { ja: '順位決定戦', en: '' }, linked: false },
        }}
        lang="jp"
      />,
    )
    assert.match(unlinked, /<p>所属グループ: 順位決定戦<\/p><\/div>/)
  })

  test('紹介コメントが空なら div ごと出さない', () => {
    const en = renderHtml(<TeamView data={team} lang="en" />)
    assert.doesNotMatch(en, /SCV_TEAM_COMM/)
  })

  test('成績表: 9 列 9.444%、得点・失点・得失差・順位の後ろに空白 2 つ、勝率', () => {
    assert.match(html, /<td class="MD1 CM" width="9.444%">試合数<\/td>/)
    assert.match(
      html,
      /<tr><td class="CM">5<\/td><td class="CM">3<\/td><td class="CM">0<\/td><td class="CM">2<\/td><td class="CM">27 {2}<\/td><td class="CM">28 {2}<\/td><td class="CM">-1 {2}<\/td><td class="CM">60%<\/td><td class="CM">4 {2}<\/td><\/tr>/,
    )
  })

  test('試合 0 なら勝率 --%、順位なしは空白 2 つだけ', () => {
    const empty = {
      ...team,
      record: { ...team.record, games: 0, wins: 0, losses: 0, rank: null },
    }
    const out = renderHtml(<TeamView data={empty} lang="jp" />)
    assert.match(out, /<td class="CM">--%<\/td><td class="CM"> {2}<\/td><\/tr>/)
  })

  test('チーム対戦一覧: 見出しの後に <br />、終了はリンク、未試合はリンクなしで得点空', () => {
    assert.match(
      html,
      /<span class="SCV_MIDA_M">&lt; チーム対戦一覧 &gt;<\/span><br\/><div class="TBL">/,
    )
    assert.match(
      html,
      /<a class="Select_Link" href="\/pscore\/scv_match\/open\/9001\?lang=jp">\[対戦結果\]<\/a>/,
    )
    assert.match(
      html,
      /<td class="MC_MAE CM"><\/td><td class="MC_MAE CM">Ｘ<\/td><td class="MC_MAE CM"><\/td>/,
    )
    assert.match(html, /<td class="MC_MAE CM">未試合<\/td>/)
    assert.doesNotMatch(html, /scv_match\/open\/9003/)
  })

  test('選手一覧: api の順のまま、トータルに空白 2 つ、ブロック 0 は空白だけ、-- 列は空', () => {
    const numbers = [...html.matchAll(/<tr><td class="CM">([0-9]+)<\/td><td class="LM">/g)].map(
      (match) => match[1],
    )
    assert.deepEqual(numbers, ['7', '4', '10', '00'])
    assert.match(
      html,
      /<td class="CM">7<\/td><td class="LM"><a class="playerLinkM" href="\/pscore\/scv_player\/open\/70001\?lang=jp">架空 太郎<\/a><\/td><td class="CM">5<\/td><td class="CM">6<\/td><td class="CM">4<\/td><td class="CM">10 {2}<\/td><td class="CM"> {2}<\/td><td class="CM"><\/td><\/tr>/,
    )
    assert.match(html, /class="playerLinkF" href="\/pscore\/scv_player\/open\/70003\?lang=jp"/)
    assert.match(html, /<td class="CM">4 {2}<\/td><td class="CM">2 {2}<\/td>/)
  })

  test('SOTG スコア一覧: 対戦一覧の後に見出しと凡例、試合ごとの行(値は空)、平均の行', () => {
    const out = renderHtml(<TeamView data={{ ...team, showSotg: true }} lang="jp" />)
    assert.match(
      out,
      /<hr\/><div class=""><table class="WMAX"><tbody><tr><td class="LM" width="26%"><span class="SCV_MIDA_M">&lt; SOTGｽｺｱ一覧 &gt;<\/span><\/td><td class="LM" width="74%"><span class="SCV_SOTG_COMM">※１･･･ルールの理解/,
    )
    assert.match(out, /<td class="MD1 CM" width="11.6%">※１<\/td>/)
    assert.match(out, /<td class="MD1 CM" colSpan="2">&lt; SOTG 平均 &gt;<\/td>/i)
    const rows = [
      ...out.matchAll(/<tr><td class="(MC_\w+) LM">([^<]*)<\/td><td class="MC_\w+ LM">/g),
    ]
    assert.equal(rows.length, team.matches.length, '対戦一覧と同じ試合が SOTG 表にも並ぶ')
    assert.match(out, /<\/table><\/div><hr\/><span class="SCV_MIDA_M">&lt; 選手一覧 &gt;/)
    assert.doesNotMatch(renderHtml(<TeamView data={team} lang="jp" />), /SOTG/)
  })

  test('選手のいないチームは選手一覧を見出しごと出さない', () => {
    const out = renderHtml(
      <TeamView data={{ ...team, matches: [], players: [], staff: [] }} lang="jp" />,
    )
    assert.doesNotMatch(out, /選手一覧/)
    assert.match(out, /<\/table><\/div><hr\/><br\/><br\/><br\/>$/)
  })

  test('試合のないチームは対戦一覧を見出しごと出さない', () => {
    const out = renderHtml(<TeamView data={{ ...team, matches: [] }} lang="jp" />)
    assert.doesNotMatch(out, /チーム対戦一覧/)
    assert.match(out, /<\/table><\/div><hr\/><span class="SCV_MIDA_M">&lt; 選手一覧 &gt;<\/span>/)
  })

  test('欠場の選手は行の全セルに ABS を付ける', () => {
    const players = team.players.map((line, index) =>
      index === 3 ? { ...line, absent: true } : line,
    )
    const out = renderHtml(<TeamView data={{ ...team, players }} lang="jp" />)
    assert.match(
      out,
      /<tr><td class="CM ABS">00<\/td><td class="LM ABS"><a class="playerLinkM" href="[^"]*">仮名 四郎<\/a><\/td>(<td class="CM ABS">[^<]*<\/td>){6}<\/tr>/,
    )
    assert.equal([...out.matchAll(/ABS/g)].length, 8)
  })

  test('スタッフ一覧は表 70%、役割と名前のリンク、その後 <hr /> と <br /> 3 つで終わる', () => {
    assert.match(
      html,
      /<span class="SCV_MIDA_M">&lt; スタッフ一覧 &gt;<\/span><div class="TBL"><table style="width:70%"><tbody><tr><td class="MD1 CM" width="30%">役割<\/td><td class="MD1 CM" width="70%">スタッフ<\/td><\/tr><tr><td class="CM">マネージャ<\/td><td class="LM"><a class="playerLinkM" href="\/pscore\/scv_player\/open\/70009\?lang=jp">見本 次郎<\/a><\/td><\/tr><\/tbody><\/table><\/div><hr\/><br\/><br\/><br\/>$/,
    )
  })

  test('スタッフがいなければ選手一覧の <hr /> の後に <br /> 3 つ', () => {
    const out = renderHtml(<TeamView data={{ ...team, staff: [] }} lang="jp" />)
    assert.doesNotMatch(out, /スタッフ一覧/)
    assert.match(out, /<\/table><\/div><hr\/><br\/><br\/><br\/>$/)
  })

  test('lang=en: 見出し、Goals<br />for、Tot. と小文字の block、英語名が空なら空', () => {
    const en = renderHtml(<TeamView data={team} lang="en" />)
    assert.match(en, /^<span class="SCV_MIDA_M">Fictional A Flying Club<br\/><\/span>/)
    assert.match(en, />Goals<br\/>for</)
    assert.match(en, /&lt;GameList for the Team&gt;/)
    assert.match(en, /&lt;Players List&gt;/)
    assert.match(en, />Tot\.<\/td><td class="MD1 CM" width="10%">block<\/td>/)
    assert.match(en, /&lt;Staffs List&gt;/)
    assert.match(en, />Task<\/td><td class="MD1 CM" width="70%">Staff</)
    assert.match(en, /href="\/pscore\/scv_player\/open\/70004\?lang=en"><\/a>/)
  })
})

describe('チームページ: ディスクゴルフ団体戦(clone-spec D 1.2)', () => {
  test('選手一覧(# と名前の 2 列、表 70%)だけを出し、<br /> 3 つで終わる', () => {
    const html = renderHtml(<TeamView data={rosterOnly} lang="jp" />)
    assert.equal(
      html,
      '<span class="SCV_MIDA_M">架空ディスクゴルフ部<br/></span>' +
        '<span class="SCV_MIDA_M">&lt; 選手一覧 &gt;</span>' +
        '<div class="TBL"><table style="width:70%"><tbody>' +
        '<tr><td class="MD1 CM" width="10%">#</td><td class="MD1 CM" width="90%">選手</td></tr>' +
        '<tr><td class="CM">1</td><td class="LM"><a class="playerLinkM" href="/pscore/scv_player/open/70101?lang=jp">仮名 一</a></td></tr>' +
        '<tr><td class="CM">2</td><td class="LM"><a class="playerLinkF" href="/pscore/scv_player/open/70102?lang=jp">仮名 二</a></td></tr>' +
        '</tbody></table></div><br/><br/><br/>',
    )
  })
})

describe('読み上げ用の補足(本サイト独自)', () => {
  test('選手名のリンクに性別(色だけで示している)を視覚的に隠して添える', () => {
    const html = renderHtmlWithNotes(<TeamView data={team} lang="jp" />)
    assert.match(
      html,
      /open\/70001\?lang=jp">架空 太郎<span class="sv_sr" data-scoreview-addition="">\(男性\)<\/span><\/a>/,
    )
    assert.match(html, /<span class="sv_sr" data-scoreview-addition="">\(女性\)<\/span>/)
  })
})
