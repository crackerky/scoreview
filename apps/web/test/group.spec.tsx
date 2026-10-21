import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { type GroupResponse, groupResponseSchema } from '../src/api/public.types.ts'
import { GroupShapeError, GroupView } from '../src/components/GroupView.tsx'
import { tournamentPageTitle } from '../src/components/title.utils.ts'
import { loadFixture } from './fixtures/load.ts'
import { renderHtml } from './render.ts'

const pool = groupResponseSchema.parse(loadFixture('group-1001'))
const bracketSmall = groupResponseSchema.parse(loadFixture('group-1002'))
const imageGroup = groupResponseSchema.parse(loadFixture('group-1003'))
const bracket = groupResponseSchema.parse(loadFixture('group-1004'))

function poolOf(data: GroupResponse) {
  assert.ok(data.pool)
  return data.pool
}

describe('プール型(clone-spec C 2.1)', () => {
  const html = renderHtml(<GroupView data={pool} lang="jp" />)

  test('見出しと節の並び', () => {
    assert.match(
      html,
      /^<span class="SCV_MIDA_M">&lt; ビーチミックス - Qualifying pool A &gt;<\/span><br\/><div class="TBL">/,
    )
    assert.match(
      html,
      /<\/table><\/div><hr\/><span class="SCV_MIDA_M">&lt; リーグ結果 &gt;<\/span><br\/><div class="TBL">/,
    )
    assert.match(
      html,
      /<\/table><\/div><hr\/><span class="SCV_MIDA_M">&lt; 結果一覧 &gt;<\/span><br\/><div class="TBL">/,
    )
    assert.match(html, /<\/table><\/div><hr\/><div class="TBL"> <\/div>$/)
  })

  test('マトリクス: 見出し 22% と 78/N%、対角・下三角・試合セル', () => {
    assert.match(
      html,
      /<tr><td class="RM_MD1 CM" width="22%">- MATRIX -<\/td><td class="RM_MD1 CM" width="39%"><span class="teamimg_rnk"><a href="\/pscore\/scv_team\/open\/5001\?lang=jp">架空チームA<\/a><\/span><\/td>/,
    )
    assert.match(
      html,
      /<tr><td class="RM_MD1"><span class="teamimg_rnk"><a href="\/pscore\/scv_team\/open\/5001\?lang=jp">架空チームA<\/a><\/span><\/td><td class="RM_AIDA CM"><\/td><td class="RM_ARI {2}CM"><a class="RM_ARI_Link" href="\/pscore\/scv_match\/open\/9001\?lang=jp">9-3<\/a><\/td><\/tr>/,
    )
    assert.match(html, /<td class="RM_NASI CM"><\/td><td class="RM_AIDA CM"><\/td><\/tr>/)
  })

  test('マトリクス: 詳細のない試合はリンクなし、試合中は RM_NOW(推測)、試合なしは空', () => {
    const base = poolOf(pool)
    const variants: Array<[GroupResponse, RegExp]> = [
      [
        { ...pool, matches: pool.matches.map((match) => ({ ...match, hasDetail: false })) },
        /<td class="RM_ARI {2}CM">9-3<\/td>/,
      ],
      [
        {
          ...pool,
          pool: {
            ...base,
            matrix: [
              [{ kind: 'self' }, { kind: 'match', matchId: 9001, status: 'live', text: '2-1' }],
              [{ kind: 'lower' }, { kind: 'self' }],
            ],
          },
        },
        /<td class="RM_NOW {2}CM"><a class="RM_NOW_Link" href="\/pscore\/scv_match\/open\/9001\?lang=jp">2-1<\/a><\/td>/,
      ],
      [
        {
          ...pool,
          pool: {
            ...base,
            matrix: [
              [{ kind: 'self' }, { kind: 'empty' }],
              [{ kind: 'lower' }, { kind: 'self' }],
            ],
          },
        },
        /<td class="RM_AIDA CM"><\/td><td class="RM_ARI {2}CM"><\/td><\/tr>/,
      ],
    ]
    for (const [data, pattern] of variants) {
      assert.match(renderHtml(<GroupView data={data} lang="jp" />), pattern)
    }
  })

  test('リーグ結果: 22% と 8.667% の 9 列、勝ち点の列なし、末尾の空白、順位は HL2', () => {
    assert.match(
      html,
      /<tr><td class="MD1 CM" width="22%">- RESULT -<\/td><td class="MD1 CM" width="8.667%">試合数<\/td>/,
    )
    assert.match(
      html,
      /<td class="MD1 CM" width="8.667%">得失差<\/td><td class="MD1 CM" width="8.667%">順位<\/td><\/tr>/,
    )
    assert.doesNotMatch(html, /勝ち点/)
    assert.match(
      html,
      /<tr><td class="MD1 LM"><span class="teamimg_rnk"><a href="\/pscore\/scv_team\/open\/5002\?lang=jp">架空チームB<\/a><\/span><\/td><td class="CM HL1">1<\/td><td class="CM">0<\/td><td class="CM">0<\/td><td class="CM">1<\/td><td class="CM">3 <\/td><td class="CM">9 <\/td><td class="CM HL1">-6 <\/td><td class="CM HL2">2 <\/td><\/tr>/,
    )
  })

  test('結果一覧は全角 Ｘ の 10 列', () => {
    assert.match(html, /<td class="MD1 CM" width="10%">CD<\/td>/)
    assert.match(html, /<td class="MC_END CM">Ｘ<\/td>/)
  })

  test('英語: 山括弧に空白なし、Reague Result の誤綴り', () => {
    const en = renderHtml(<GroupView data={pool} lang="en" />)
    assert.match(en, /^<span class="SCV_MIDA_M">&lt;Beach Mixed - Qualifying pool A&gt;<\/span>/)
    assert.match(en, /&lt;Reague Result&gt;/)
    assert.match(en, /&lt;GameList for the Group&gt;/)
    assert.match(en, />- MATRIX -<\/td>.*>- RESULT -<\/td>/)
  })
})

describe('トーナメント型(clone-spec C 2.3)', () => {
  const html = renderHtml(<GroupView data={bracket} lang="jp" />)

  test('見出しの後に div.TN_TMPL の table、その後 <br/> と結果一覧', () => {
    assert.match(
      html,
      /^<span class="SCV_MIDA_M">&lt; ビーチミックス - 1st placement tournament &gt;<\/span><br\/><div class="TN_TMPL"><table><tbody><tr><td class="WDiv3 txt_yen {3}">Semifinals<\/td><td class="WDiv3 txt_yen {3}">FINAL<\/td><td class="WDiv3 txt_sharp ">Placement<\/td><\/tr>/,
    )
    assert.match(
      html,
      /<\/table><\/div><br\/><span class="SCV_MIDA_M">&lt; 結果一覧 &gt;<\/span><br\/><div class="TBL">/,
    )
    assert.match(html, /<hr\/><div class="TBL"> <\/div>$/)
    assert.doesNotMatch(html, /MATRIX|RESULT/)
  })

  test('チームのセルは短縮名のリンク、試合のセルは tnmt_game_link、空セルは &nbsp;', () => {
    assert.match(
      html,
      /<tr><td class="WDiv3 BdBtm team "><span class="teamimg_tn"><a href="\/pscore\/scv_team\/open\/5001\?lang=jp">架空チームA<\/a><\/span><\/td><td class="WDiv3 "> <\/td><td class="WDiv3 "> <\/td><\/tr>/,
    )
    assert.match(
      html,
      /<td class="WDiv3 BdRight BdTop game "><a class="tnmt_game_link" href="\/pscore\/scv_match\/open\/9101\?lang=jp">8-3<\/a><\/td><td class="WDiv3 BdLeft BdBtm win {2}"><span class="teamimg_tn">/,
    )
    assert.match(
      html,
      /<td class="WDiv3 BdBtm lose "><span class="teamimg_tn"><a href="\/pscore\/scv_team\/open\/5001\?lang=jp">/,
    )
    assert.match(html, /<td class="WDiv3 BdLeft BdTop txt_sharp ">1st<\/td>/)
    assert.match(html, /<tr><td class="WDiv3 txt_and {3}">3rd - 4th<\/td>/)
  })

  test('試合中の試合セルは tnmt_play_now(推測)、未試合はリンクなし(推測)', () => {
    assert.ok(bracketSmall.bracket)
    const rows = bracketSmall.bracket.rows.map((row) =>
      row.map((cell) => (cell.kind === 'game' ? { ...cell, status: 'live' as const } : cell)),
    )
    const live = renderHtml(
      <GroupView
        data={{ ...bracketSmall, bracket: { ...bracketSmall.bracket, rows } }}
        lang="jp"
      />,
    )
    assert.match(
      live,
      /<a class="tnmt_play_now" href="\/pscore\/scv_match\/open\/9001\?lang=jp">9-3<\/a>/,
    )
    const scheduledRows = bracketSmall.bracket.rows.map((row) =>
      row.map((cell) =>
        cell.kind === 'game' ? { ...cell, status: 'scheduled' as const, text: '' } : cell,
      ),
    )
    const scheduled = renderHtml(
      <GroupView
        data={{ ...bracketSmall, bracket: { ...bracketSmall.bracket, rows: scheduledRows } }}
        lang="jp"
      />,
    )
    assert.match(scheduled, /<td class="WDiv2 BdRight BdTop game "> <\/td>/)
  })

  test('英語はチームの英語名、Placement 列などは英語固定', () => {
    const en = renderHtml(<GroupView data={bracket} lang="en" />)
    assert.match(en, />Fictional A<\/a>/)
    assert.match(en, />Placement<\/td>/)
  })
})

describe('画像のみ型(clone-spec E 4)', () => {
  test('見出しと div.TBL の中の img だけ', () => {
    assert.equal(
      renderHtml(<GroupView data={imageGroup} lang="jp" />),
      '<span class="SCV_MIDA_M">&lt; ビーチミックス - Temporary ranking &gt;</span><br/><div class="TBL"><img src="/brand/photo-sample.svg" alt="" style="width:500px;height:333px"/></div>',
    )
  })
  test('大きさが未登録なら style を付けない', () => {
    assert.ok(imageGroup.image)
    const data = { ...imageGroup, image: { ...imageGroup.image, width: null, height: null } }
    assert.match(renderHtml(<GroupView data={data} lang="jp" />), /<img src="[^"]+" alt=""\/>/)
  })
})

describe('契約違反と title', () => {
  test('kind と節が食い違えば GroupShapeError', () => {
    assert.throws(
      () => renderHtml(<GroupView data={{ ...pool, pool: null }} lang="jp" />),
      GroupShapeError,
    )
  })
  test('title は "-{グループ名}-"', () => {
    assert.equal(
      tournamentPageTitle(pool.frame, 'jp', `-${pool.group.name}-`),
      '[SCV] 2026 架空ビーチカップ -Qualifying pool A-',
    )
  })
})

describe('プールの補足(本サイト独自。requirements.md「意図的な差分」)', () => {
  const [first, second] = poolOf(pool).teams
  assert.ok(first && second)
  const withNotes: GroupResponse = {
    ...pool,
    pool: {
      ...poolOf(pool),
      notes: {
        reasons: [{ team: first, text: '勝ち点同数、得失点差で上位' }],
        conditions: [{ team: second, text: '次の A3 に勝てば 1 位確定' }],
      },
    },
  }

  test('リーグ結果の下に、勝ち上がり条件と順位の根拠を小さく出す', () => {
    const html = renderHtml(<GroupView data={withNotes} lang="jp" />)
    assert.match(
      html,
      /<\/table><\/div><div class="sv_pool_notes"><p class="sv_pool_notes_head">勝ち上がり条件/,
    )
    assert.match(html, /<li>架空チームA: 勝ち点同数、得失点差で上位<\/li>/)
    assert.match(html, /次の A3 に勝てば 1 位確定/)
  })

  test('補足が無い、または英語表示なら出さない', () => {
    assert.doesNotMatch(renderHtml(<GroupView data={pool} lang="jp" />), /sv_pool_notes/)
    assert.doesNotMatch(renderHtml(<GroupView data={withNotes} lang="en" />), /sv_pool_notes/)
  })
})
