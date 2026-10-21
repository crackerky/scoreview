import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { playerResponseSchema } from '../src/api/public.types.ts'
import { PlayerView } from '../src/components/PlayerView.tsx'
import { loadFixture } from './fixtures/load.ts'
import { renderHtml } from './render.ts'

const player = playerResponseSchema.parse(loadFixture('player-70001'))
const staff = playerResponseSchema.parse(loadFixture('player-70009'))
const discGolf = playerResponseSchema.parse(loadFixture('player-70101'))

const html = renderHtml(<PlayerView data={player} lang="jp" />)
const nb2 = '  '

describe('選手ページ(clone-spec D 3.1)', () => {
  test('前後ナビ: "<=[#番号 : 名前]" / "[チーム短縮名]" / "[#番号 : 名前]=>"、スタッフは役割', () => {
    assert.match(
      html,
      /^<table class="WMAX"><tbody><tr><td width="30%" class="LM"><a class="playerLinkM" href="\/pscore\/scv_player\/open\/70009\?lang=jp">&lt;=\[マネージャ : 見本 次郎\]<\/a><\/td><td width="40%" class="CM"><a class="playerLink" href="\/pscore\/scv_team\/open\/5001\?lang=jp">\[架空チームA\]<\/a><\/td><td width="30%" class="RM"><a class="playerLinkM" href="\/pscore\/scv_player\/open\/70002\?lang=jp">\[#10 : 見本 三郎\]=&gt;<\/a><\/td><\/tr><\/tbody><\/table><hr\/>/,
    )
  })

  test('見出し "{チーム短縮名} - #{背番号} : {名前}"', () => {
    assert.match(
      html,
      /<hr\/><span class="SCV_MIDA_M">架空チームA - #7 : 架空 太郎<\/span><div class="TBL">/,
    )
  })

  test('成績表: 7 列、"値 &nbsp;&nbsp;(avg:平均)" の空白は列ごとに 1/3/2、ブロック 0 は空', () => {
    assert.match(
      html,
      /<td class="MD1 CM" width="10%">試合数<\/td><td class="MD1 CM" width="15%">アシスト<\/td>.*<td class="MD1 CM" width="15%">キャラハン<\/td><td class="MD1 CM" width="15%">グレイテスト<\/td><td class="MD1 CM" width="15%">ブロック<\/td>/,
    )
    const expected = [
      '5',
      `6 ${nb2}(avg:1.2)`,
      `4   ${nb2}(avg:0.8)`,
      `10  ${nb2}(avg:2)`,
      '0',
      '0',
      '',
    ]
    const row = expected.map((text) => `<td class="RM CM">${text}</td>`).join('')
    assert.ok(html.includes(`<tr>${row}</tr>`), html)
  })

  test('試合数 0 は (avg:--)、ブロックがあれば数字', () => {
    const zero = {
      ...player,
      stats: { games: 0, assists: 0, goals: 0, callahans: 0, greatests: 0, blocks: 3 },
    }
    const out = renderHtml(<PlayerView data={zero} lang="jp" />)
    assert.ok(out.includes(`<td class="RM CM">0 ${nb2}(avg:--)</td>`))
    assert.ok(out.includes('<td class="RM CM">3</td></tr>'))
  })

  test('イベント: TG_EVMOJ の行、相手は短縮名のリンク、選手は "#番号&nbsp;名前"、api の順', () => {
    assert.match(html, /<\/table><\/div><hr\/><span class="SCV_MIDA_M">&lt; イベント &gt;<\/span>/)
    assert.match(
      html,
      /<td class="MD1 CM" width="10%">CD<\/td><td class="MD1 CM" width="7%">日付<\/td><td class="MD1 CM" width="20%">対戦チーム<\/td><td class="MD1 CM" width="7%">経過<\/td><td class="MD1 CM" width="16%">イベント<\/td><td class="MD1 CM" width="20%">ゴール選手<\/td><td class="MD1 CM" width="20%">アシスト選手<\/td>/,
    )
    assert.match(
      html,
      /<tr><td class="TG_EVMOJ LM">A1<\/td><td class="TG_EVMOJ CM">10\/18<\/td><td class="TG_EVMOJ LM"><span class="teamLink"><a href="\/pscore\/scv_team\/open\/5002\?lang=jp">架空チームB<\/a><\/span><\/td><td class="TG_EVMOJ CM">04:00<\/td><td class="TG_EVMOJ CM">得点<\/td><td class="TG_EVMOJ LM"><a class="playerLinkM" href="\/pscore\/scv_player\/open\/70001\?lang=jp">#7 架空 太郎<\/a><\/td><td class="TG_EVMOJ LM"><a class="playerLinkM" href="\/pscore\/scv_player\/open\/70002\?lang=jp">#10 見本 三郎<\/a><\/td><\/tr><tr><td class="TG_EVMOJ LM">A6<\/td>/,
    )
    assert.match(
      html,
      /class="playerLinkF" href="\/pscore\/scv_player\/open\/70003\?lang=jp">#4 見本 花子</,
    )
  })

  test('イベントが 0 件なら成績表の <hr /> で終わる', () => {
    const out = renderHtml(<PlayerView data={{ ...player, events: [] }} lang="jp" />)
    assert.doesNotMatch(out, /イベント/)
    assert.match(out, /<\/table><\/div><hr\/>$/)
  })

  test('lang=en: 見出しと列名、英語名が空の選手は名前が空', () => {
    const en = renderHtml(<PlayerView data={player} lang="en" />)
    assert.match(en, /&lt;=\[Manager : Jiro MIHON\]/)
    assert.match(en, />Fictional A - #7 : Taro KAKU</)
    assert.match(en, />Games<\/td><td class="MD1 CM" width="15%">Assist<\/td>/)
    assert.match(en, />Tot\.<\/td><td class="MD1 CM" width="15%">Callahan<\/td>/)
    assert.match(en, /&lt;Game Events&gt;/)
    assert.match(en, />VS Team<\/td>/)
    assert.match(en, /<td class="TG_EVMOJ CM">Goal<\/td>/)
  })
})

describe('選手ページ: スタッフとディスクゴルフ(clone-spec D 3.1)', () => {
  test('スタッフは "{チーム短縮名} - {役割} : {名前}" の見出しで終わる', () => {
    const out = renderHtml(<PlayerView data={staff} lang="jp" />)
    assert.match(
      out,
      /<hr\/><span class="SCV_MIDA_M">架空チームA - マネージャ : 見本 次郎<\/span>$/,
    )
    assert.match(out, /&lt;=\[#00 : 仮名 四郎\]/)
  })

  test('ディスクゴルフ: 8 列(半角カナ)、値は 0 と (avg:--)、空白は参照サイトどおり', () => {
    const out = renderHtml(<PlayerView data={discGolf} lang="jp" />)
    assert.match(
      out,
      /<td class="MD1 CM" width="10%">試合数<\/td><td class="MD1 CM" width="12.8%">得点<\/td><td class="MD1 CM" width="12.8%">キャッチ<\/td><td class="MD1 CM" width="12.8%">スローミス<\/td><td class="MD1 CM" width="12.8%">トータル<\/td><td class="MD1 CM" width="12.8%">ｸﾘｰﾝｷｬｯﾁ<\/td><td class="MD1 CM" width="12.8%">ﾌｫﾛｰｷｬｯﾁ<\/td><td class="MD1 CM" width="12.8%">ﾌｯﾄﾌｧｰﾙ<\/td>/,
    )
    const expected = [
      '0',
      `0    ${nb2}(avg:--)`,
      `0   ${nb2}(avg:--)`,
      `0    ${nb2}(avg:--)`,
      `0   ${nb2}(avg:--)`,
      '0  ',
      '0  ',
      '0  ',
    ]
    const row = expected.map((text) => `<td class="RM CM">${text}</td>`).join('')
    assert.ok(out.includes(`<tr>${row}</tr></tbody></table></div><hr/>`), out)
    assert.match(out, /<span class="SCV_MIDA_M">架空DG - #1 : 仮名 一<\/span>/)
  })
})
