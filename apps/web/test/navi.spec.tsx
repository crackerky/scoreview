import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { type TournamentFrame, tournamentFrameSchema } from '../src/api/common.types.ts'
import { buildNavi } from '../src/components/navi.utils.ts'
import { ScoreNavi } from '../src/components/ScoreNavi.tsx'
import { loadFixture } from './fixtures/load.ts'
import { renderHtml } from './render.ts'

const standard = tournamentFrameSchema.parse(loadFixture('frame-1'))

function variant(overrides: Partial<TournamentFrame>): TournamentFrame {
  return { ...standard, ...overrides }
}

function labels(frame: TournamentFrame, lang: 'jp' | 'en' = 'jp'): string[] {
  return buildNavi(frame, lang).items.map((item) => item.label)
}

const noGallery = { galleries: [] }
const entries = { labels: { teamList: 'entries', groups: 'results' } } as const
const noSchedule = { showSchedule: false }

// clone-spec E 3 のナビ構成 7 パターン
describe('大会ナビの 7 パターン', () => {
  test('1. 標準(ギャラリーなし): 5 項目 + 空 li 172px', () => {
    const frame = variant(noGallery)
    assert.deepEqual(labels(frame), [
      '大会トップ',
      'チーム一覧',
      'スケジュール',
      'ディビジョン成績',
      'グループ成績　',
    ])
    assert.equal(buildNavi(frame, 'jp').fillerWidthPx, 172)
  })

  test('2. 標準 + ギャラリー: ギャラリーの li が 172px、空 li なし', () => {
    const navi = buildNavi(standard, 'jp')
    assert.equal(navi.items.length, 6)
    const gallery = navi.items[5]
    assert.equal(gallery?.label, 'ギャラリー')
    assert.equal(gallery?.kind === 'menu' && gallery.widthPx, 172)
    assert.equal(navi.fillerWidthPx, null)
  })

  test('3. エントリー一覧 / 成績 + 空 li 495px', () => {
    const frame = variant({ ...entries, ...noSchedule, ...noGallery })
    assert.deepEqual(labels(frame), ['大会トップ', 'エントリー一覧', '成績'])
    assert.equal(buildNavi(frame, 'jp').fillerWidthPx, 495)
  })

  test('4. エントリー一覧 / 成績 / ギャラリー + 空 li 321px', () => {
    const frame = variant({ ...entries, ...noSchedule })
    assert.deepEqual(labels(frame), ['大会トップ', 'エントリー一覧', '成績', 'ギャラリー'])
    assert.equal(buildNavi(frame, 'jp').fillerWidthPx, 321)
  })

  test('5. チーム一覧 / グループ成績 / ギャラリー + 空 li 321px', () => {
    const frame = variant(noSchedule)
    assert.deepEqual(labels(frame), ['大会トップ', 'チーム一覧', 'グループ成績　', 'ギャラリー'])
    assert.equal(buildNavi(frame, 'jp').fillerWidthPx, 321)
  })

  test('6. チーム一覧 / グループ成績 + 空 li 495px', () => {
    const frame = variant({ ...noSchedule, ...noGallery })
    assert.deepEqual(labels(frame), ['大会トップ', 'チーム一覧', 'グループ成績　'])
    assert.equal(buildNavi(frame, 'jp').fillerWidthPx, 495)
  })

  test('7. エントリー一覧 / スケジュール / ディビジョン成績 / 成績 + 空 li 172px', () => {
    const frame = variant({ ...entries, ...noGallery })
    assert.deepEqual(labels(frame), [
      '大会トップ',
      'エントリー一覧',
      'スケジュール',
      'ディビジョン成績',
      '成績',
    ])
    assert.equal(buildNavi(frame, 'jp').fillerWidthPx, 172)
  })

  test('日程のないディビジョンはスケジュールとディビジョン成績に出ず、グループ成績には出る', () => {
    const [first, second] = standard.divisions
    assert.ok(first && second)
    const frame = variant({ divisions: [first, { ...second, scheduleEnabled: false }] })
    const navi = buildNavi(frame, 'jp')
    const entryLabels = navi.items.map((item) =>
      item.kind === 'menu' ? item.entries.map((entry) => entry.label) : [],
    )
    assert.deepEqual(entryLabels[2], ['全ディビジョン', 'ビーチミックス'])
    assert.deepEqual(entryLabels[3], ['ビーチミックス'])
    assert.deepEqual(entryLabels[4], ['ビーチミックス', 'ウィメン'])
  })

  test('英語のラベル', () => {
    assert.deepEqual(labels(standard, 'en'), [
      'Tournament Top',
      'Team List',
      'Schedule',
      'Division Score',
      'Group Score',
      'Gallery',
    ])
    assert.deepEqual(labels(variant(entries), 'en').slice(1, 2), ['Entry List'])
    assert.equal(labels(variant(entries), 'en').at(4), 'Result')
  })
})

describe('ScoreNavi の描画', () => {
  test('直リンク、p.ar のドロップダウン、ディビジョン → グループの 2 階層', () => {
    const html = renderHtml(<ScoreNavi frame={standard} lang="jp" />)
    assert.match(html, /^<ul id="score_navi">/)
    assert.match(html, /<li><a href="\/pscore\/scv_top\/open\/1\?lang=jp">大会トップ<\/a><\/li>/)
    assert.match(html, /<p class="ar">スケジュール<\/p><ul><li><a href="\/pscore\/scv_search_tn/)
    assert.match(html, /href="\/pscore\/scv_search_div\/open\/101\?lang=jp">ビーチミックス</)
    assert.match(html, /href="\/pscore\/scv_division\/open\/102\?lang=jp">ウィメン</)
    assert.match(
      html,
      /<li><p>ビーチミックス<\/p><ul><li><a href="\/pscore\/scv_group\/open\/1001\?lang=jp">Qualifying pool A<\/a><\/li>/,
    )
    // グループのないディビジョンも名前と空の ul が出る
    assert.match(html, /<li><p>ウィメン<\/p><ul><\/ul><\/li>/)
    assert.match(html, /<li style="width:172px"><p class="ar">ギャラリー<\/p>/)
    assert.match(html, /href="\/pscore\/scv_tgallery\/open\/201\?lang=jp">Day1-AM</)
    assert.doesNotMatch(html, /&nbsp;| /)
  })

  test('リンクなしのグループは名前だけ、末尾に幅調整の空 li', () => {
    const frame = tournamentFrameSchema.parse(loadFixture('frame-2'))
    const html = renderHtml(<ScoreNavi frame={frame} lang="jp" />)
    assert.match(html, /エントリー一覧/)
    assert.match(html, /<p class="ar">成績<\/p>/)
    assert.match(
      html,
      /<li><p>ＤＧシングルス（日曜日）<\/p><ul><li><p>最終結果<\/p><\/li><\/ul><\/li>/,
    )
    assert.match(html, /<li style="width:495px"><p> <\/p><\/li><\/ul>$/)
    assert.doesNotMatch(html, /scv_search|scv_division|scv_group/)
  })

  test('lang=en ではグループとギャラリーも英語名("---" もそのまま)', () => {
    const html = renderHtml(<ScoreNavi frame={standard} lang="en" />)
    assert.match(html, /href="\/pscore\/scv_group\/open\/1002\?lang=en">Tournament A</)
    assert.match(html, /href="\/pscore\/scv_tgallery\/open\/201\?lang=en">---</)
  })

  test('名前の空のグループへのリンクはリンク先を文字にする(参照サイトの CakePHP と同じ)', () => {
    const [first, second] = standard.divisions
    assert.ok(first && second)
    const groups = first.groups.map((group) => ({ ...group, name: { ...group.name, en: '' } }))
    const frame = variant({ divisions: [{ ...first, groups }, second] })
    const html = renderHtml(<ScoreNavi frame={frame} lang="en" />)
    assert.match(
      html,
      /<a href="\/pscore\/scv_group\/open\/1001\?lang=en">\/pscore\/scv_group\/open\/1001\?lang=en<\/a>/,
    )
  })

  test('lang=en はリンクに引き継がれ、英語名が空のディビジョンは空のまま', () => {
    const frame = tournamentFrameSchema.parse(loadFixture('frame-2'))
    const html = renderHtml(<ScoreNavi frame={frame} lang="en" />)
    assert.match(html, /href="\/pscore\/scv_team_list\/open\/2\?lang=en">Entry List</)
    assert.match(html, /<li><p><\/p><ul><\/ul><\/li>/)
  })
})
