import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { matchResponseSchema } from '../src/api/public.types.ts'
import { MatchTopMenu, MatchView } from '../src/components/MatchView.tsx'
import { eventsNewestFirst } from '../src/components/matchEvents.utils.ts'
import { formatClockJst } from '../src/format/format.utils.ts'
import { loadFixture } from './fixtures/load.ts'
import { renderHtml } from './render.ts'

const match = matchResponseSchema.parse(loadFixture('match-9002'))
// 2026-10-21 15:43:31 JST
const now = new Date('2026-10-21T06:43:31Z')
const html = renderHtml(<MatchView data={match} now={now} lang="jp" />)
const en = renderHtml(<MatchView data={match} now={now} lang="en" />)

/** table#events のデータ行(見出し行を除く)を <tr> 単位で取り出す。 */
function eventRows(markup: string): string[] {
  const table = /<table id="events"><tbody>(.*?)<\/tbody><\/table>/.exec(markup)?.[1] ?? ''
  return [...table.matchAll(/<tr>(.*?)<\/tr>/g)].map((row) => row[1] ?? '').slice(1)
}

/** 1 行のセルの中身(タグを除いた文字列)。 */
function cellTexts(row: string): string[] {
  return [...row.matchAll(/<td[^>]*>(.*?)<\/td>/g)].map((cell) =>
    (cell[1] ?? '').replace(/<[^>]+>/g, ''),
  )
}

describe('得点経過の並べ替え(clone-spec D 2.3)', () => {
  const event = (elapsedSec: number, leftScore: number, rightScore: number, tag: string) => ({
    elapsedSec,
    leftScore,
    rightScore,
    tag,
  })

  test('新しい順。同時刻は累計点の大きい方が上、同点は api の順の逆', () => {
    const sorted = eventsNewestFirst([
      event(0, 0, 0, 'pull'),
      event(120, 1, 0, 'goal-1'),
      event(120, 1, 1, 'goal-2'),
      event(120, 1, 1, 'timeout'),
      event(300, 1, 1, 'end'),
    ])
    assert.deepEqual(
      sorted.map((item) => item.tag),
      ['end', 'timeout', 'goal-2', 'goal-1', 'pull'],
    )
  })

  test('入力の配列を変更しない', () => {
    const input = [event(0, 0, 0, 'a'), event(60, 1, 0, 'b')]
    eventsNewestFirst(input)
    assert.deepEqual(
      input.map((item) => item.tag),
      ['a', 'b'],
    )
  })
})

describe('試合ページ: スコアボード(clone-spec D 2.2)', () => {
  test('CD / グループ名 / "YYYY/MM/DD&nbsp;HH:MM&nbsp; @Field:&nbsp;コート"', () => {
    assert.match(
      html,
      /^<div class="TBL"><table><tbody><tr><td class="LM TG_MD1" width="16%">A6<\/td><td class="LM TG_MD1" width="48%">Qualifying pool A<\/td><td class="CM TG_MD1" width="36%">2026\/10\/18 11:20  @Field: 2<\/td><\/tr><\/tbody><\/table>/,
    )
  })

  test('短縮名のチームリンクと得点、-対戦結果- と表示時刻(JST)', () => {
    assert.match(
      html,
      /<tr style="height:50px"><td class="CM TG_VSTITLE" width="41%"><span class="teamimg_big_match"><a href="\/pscore\/scv_team\/open\/5004\?lang=jp">架空チームD<\/a><\/span><\/td><td class="CM TG_POINT" width="7%">3<\/td><td class="CM TG_VSTITLE" width="6%">VS<\/td><td class="CM TG_POINT" width="7%">4<\/td><td class="CM TG_VSTITLE" width="41%"><span class="teamimg_big_match"><a href="\/pscore\/scv_team\/open\/5001\?lang=jp">架空チームA<\/a>/,
    )
    assert.match(
      html,
      /<td class="CM TG_MD1" width="16%">-対戦結果-<\/td><td class="LM TG_MD1" width="68%"><\/td><td class="CM TG_MD1" width="16%">15:43:31 JST<\/td>/,
    )
    assert.match(en, />--Result--</)
  })

  test('JST の表示時刻は 0 時を 00 で出す', () => {
    assert.equal(formatClockJst(new Date('2026-10-21T15:00:05Z')), '00:00:05 JST')
  })
})

describe('試合ページ: 得点経過(clone-spec D 2.3)', () => {
  const rows = eventRows(html)

  test('見出し 8 列(jp / en)と table#events の入れ物', () => {
    assert.match(html, /<hr\/><div class="TBL TG_EventList"><table id="events">/)
    assert.match(
      html,
      /<td class="MD1 CM" width="6%">経過<\/td><td class="MD1 CM" width="17%">アシスト選手<\/td><td class="MD1 CM" width="17%">ゴール選手<\/td><td class="MD1 CM" width="3%">点<\/td><td class="MD1 CM" width="14%">イベント<\/td>/,
    )
    assert.match(en, />Time<\/td><td class="MD1 CM" width="17%">Assist Player</)
    assert.match(en, />P\.<\/td><td class="MD1 CM" width="14%">Event</)
  })

  test('新しい順: 試合終了が最上行、試合開始ｽﾛｰｵﾌが最下行、同時刻は点数の大きい方が上', () => {
    assert.deepEqual(
      rows.map((row) => cellTexts(row).slice(0, 6).join('|')),
      [
        '10:00||||試合終了|',
        '10:00||||得点|4',
        '07:00||||得点|3',
        '05:00|#8 仮想 八重|#21 仮想 二郎|3|得点|',
        '05:00||||得点|2',
        '04:00||#3 仮想 三太|2|キャラハンゴール|',
        '04:00||||タイムアウト|★',
        '03:00|#21 仮想 二郎|#8 仮想 八重|1|得点|',
        '01:00||||得点|1',
        '00:00|||★|試合開始ｽﾛｰｵﾌ|',
      ],
    )
  })

  test('得点行の左右ミラー: 左はアシスト→ゴール→点、右は点→ゴール→アシスト', () => {
    const leftGoal = rows[3] ?? ''
    assert.deepEqual(cellTexts(leftGoal), [
      '05:00',
      '#8 仮想 八重',
      '#21 仮想 二郎',
      '3',
      '得点',
      '',
      '',
      '',
    ])
    const rightGoal = rows[1] ?? ''
    assert.deepEqual(cellTexts(rightGoal), [
      '10:00',
      '',
      '',
      '',
      '得点',
      '4',
      '#7 架空 太郎',
      '#00 仮名 四郎',
    ])
    assert.match(
      rightGoal,
      /<td class="TG_EVMOJI LM"><a class="playerLinkM" href="\/pscore\/scv_player\/open\/70001\?lang=jp">#7 架空 太郎<\/a><\/td>/,
    )
  })

  test('得点行は TG_EVMOJI、イベント行は全セルに EVT', () => {
    assert.match(
      rows[1] ?? '',
      /^<td class="TG_EVMOJI CM">10:00<\/td><td class="TG_EVMOJI LM"><\/td>/,
    )
    for (const index of [0, 6, 9]) {
      const classes = [...(rows[index] ?? '').matchAll(/class="(TG_EVMOJI [A-Z]+[^"]*)"/g)]
      assert.equal(classes.length, 8)
      for (const [, name] of classes) {
        assert.match(name ?? '', / EVT$/)
      }
    }
  })

  test('★ はプルした側・タイムアウトを取った側の点列(左プルなら左の点列)', () => {
    assert.deepEqual(cellTexts(rows[9] ?? ''), ['00:00', '', '', '★', '試合開始ｽﾛｰｵﾌ', '', '', ''])
    assert.deepEqual(cellTexts(rows[6] ?? ''), ['04:00', '', '', '', 'タイムアウト', '★', '', ''])
  })

  test('キャラハン: 得点行のままイベント名だけ変わり、アシストは空', () => {
    assert.match(
      rows[5] ?? '',
      /^<td class="TG_EVMOJI CM">04:00<\/td><td class="TG_EVMOJI LM"><\/td>/,
    )
    assert.match(
      rows[5] ?? '',
      /<a class="playerLink" href="\/pscore\/scv_player\/open\/70203\?lang=jp">#3 仮想 三太<\/a>/,
    )
    const enRows = eventRows(en)
    assert.equal(cellTexts(enRows[5] ?? '')[4], 'Callahan Goal')
  })

  test('lang=en: Goal / 1st Pull / Timeout / End、★ は *、英語名が空なら番号だけ', () => {
    const enRows = eventRows(en).map((row) => cellTexts(row))
    assert.deepEqual(
      enRows.map((cells) => cells[4]),
      [
        'End',
        'Goal',
        'Goal',
        'Goal',
        'Goal',
        'Callahan Goal',
        'Timeout',
        'Goal',
        'Goal',
        '1st Pull',
      ],
    )
    assert.equal(enRows[9]?.[3], '*')
    assert.equal(enRows[1]?.[7], '#00 ')
  })
})

describe('試合ページ: 出場選手(clone-spec D 2.4)', () => {
  test('左右 2 表、見出しは jp でも英語、名前だけのリンク、ブロック 0 は空', () => {
    assert.match(
      html,
      /<hr\/><table style="width:100%"><tbody><tr><td width="50%"><div class="TBL TG_PlayersList_Ult"><table id="lplayers"><tbody><tr><td class="MD1 CM" width="10%">#<\/td><td class="MD1 CM" width="41%">Player<\/td><td class="MD1 CM" width="13%">Assist<\/td><td class="MD1 CM" width="13%">Goal<\/td><td class="MD1 CM" width="13%">Block<\/td><\/tr>/,
    )
    assert.match(
      html,
      /<tr><td class="CM">21<\/td><td class="LM"><a class="playerLinkM" href="\/pscore\/scv_player\/open\/70202\?lang=jp">仮想 二郎<\/a><\/td><td class="CM">1<\/td><td class="CM">1<\/td><td class="CM">2<\/td><\/tr><tr><td class="CM">8<\/td><td class="LM"><a class="playerLinkF"[^>]*>仮想 八重<\/a><\/td><td class="CM">1<\/td><td class="CM">1<\/td><td class="CM"><\/td><\/tr>/,
    )
    assert.match(html, /<td width="50%"><div class="TBL TG_PlayersList_Ult"><table id="rplayers">/)
  })

  test('並びは api の順のまま', () => {
    const right = /<table id="rplayers"><tbody>(.*?)<\/tbody>/.exec(html)?.[1] ?? ''
    const numbers = [...right.matchAll(/<tr><td class="CM">([0-9]+)<\/td>/g)].map((row) => row[1])
    assert.deepEqual(numbers, ['7', '4', '10', '00'])
  })
})

describe('試合ページ: 統計(clone-spec D 2.5)', () => {
  test('見出しは短縮名、オフェンス/ディフェンスからの得点は "数&nbsp;×3[割合]"、ターンオーバーは数だけ', () => {
    assert.match(
      html,
      /<hr\/><div class="TBL"><table style="width:80%"><tbody><tr><td class="MD1 CM" width="40%">統計情報<\/td><td class="MD1 CM" width="30%">架空チームD<\/td><td class="MD1 CM" width="30%">架空チームA<\/td><\/tr><tr><td class="LM">得点数<\/td><td class="LM">3<\/td><td class="LM">4<\/td><\/tr>/,
    )
    assert.match(
      html,
      /<td class="LM">オフェンスからの得点数<\/td><td class="LM">1   \[33.33%\]<\/td><td class="LM">3   \[75%\]<\/td>/,
    )
    assert.match(
      html,
      /<td class="LM">ディフェンスからの得点数<\/td><td class="LM">2   \[66.67%\]<\/td><td class="LM">1   \[25%\]<\/td>/,
    )
    assert.match(
      html,
      /<td class="LM">ターンオーバーからの得点数<\/td><td class="LM">2<\/td><td class="LM">1<\/td>/,
    )
    assert.match(
      html,
      /<td class="LM">タイムアウト数<\/td><td class="LM">0<\/td><td class="LM">1<\/td><\/tr><\/tbody><\/table><\/div>$/,
    )
  })

  test('得点 0 の側は [--%]', () => {
    const zero = { goals: 0, fromOffense: 0, fromDefense: 0, fromTurnover: 0, timeouts: 0 }
    const out = renderHtml(
      <MatchView data={{ ...match, stats: { ...match.stats, left: zero } }} now={now} lang="jp" />,
    )
    assert.match(out, /<td class="LM">オフェンスからの得点数<\/td><td class="LM">0   \[--%\]<\/td>/)
  })

  test('lang=en の行見出し(参照サイトの deffence の綴りのまま)', () => {
    assert.match(en, />Game Statistics</)
    assert.match(en, />Goals from starting on offence</)
    assert.match(en, />Goals from starting on deffence</)
    assert.match(en, />Goals from turnovers</)
    assert.match(en, />Time outs</)
  })
})

describe('試合ページ: 試合ごとの表示の違い', () => {
  const variant = {
    ...match,
    match: { ...match.match, note: '架空の備考', hideBlocks: true, hideStats: true },
    events: match.events.map((event) =>
      event.kind === 'end' ? { ...event, side: 'left' as const } : event,
    ),
  }
  const out = renderHtml(<MatchView data={variant} now={now} lang="jp" />)

  test('備考はフッター行の中央に出る', () => {
    assert.match(out, /<td class="LM TG_MD1" width="68%">架空の備考<\/td>/)
  })

  test('Block 列のない試合は 4 列(10/50/15/15%)', () => {
    assert.doesNotMatch(out, />Block</)
    assert.match(
      out,
      /<table id="lplayers"><tbody><tr><td class="MD1 CM" width="10%">#<\/td><td class="MD1 CM" width="50%">Player<\/td><td class="MD1 CM" width="15%">Assist<\/td><td class="MD1 CM" width="15%">Goal<\/td><\/tr>/,
    )
  })

  test('統計表のない試合は出場選手の後の <hr/> で終わる', () => {
    assert.doesNotMatch(out, /統計情報/)
    assert.match(out, /<\/table><hr\/>$/)
  })

  test('試合終了の行の ★', () => {
    assert.deepEqual(cellTexts(eventRows(out)[0] ?? '').slice(3, 5), ['★', '試合終了'])
  })
})

describe('試合ページ: 更新ボタン(clone-spec D 2.1)', () => {
  test('初期 HTML は両方 display:none、OK は refpos=1 付きの同じページ、alt は参照サイトの綴り', () => {
    const menu = renderHtml(<MatchTopMenu matchId={9002} lang="jp" scrollToNavi={false} />)
    assert.equal(
      menu,
      '<table style="width:100%"><tbody><tr><td class="RM" width="140px" style="height:33px">' +
        '<a id="Refresh_OK" style="display:none" href="/pscore/scv_match/open/9002?lang=jp&amp;refpos=1">' +
        '<img src="/brand/refresh.svg" alt="Reflesh_OK"/></a>' +
        '<a id="Refresh_NG" style="display:none" href="#"><img src="/brand/refresh-ng.svg" alt="RefleshWait"/></a>' +
        ' </td></tr></tbody></table>',
    )
  })
})
