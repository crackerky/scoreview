import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import {
  galleryResponseSchema,
  teamListResponseSchema,
  tournamentListResponseSchema,
  tournamentTopResponseSchema,
} from '../src/api/public.types.ts'
import { NotFoundScreen, RouteNotFoundScreen } from '../src/components/ErrorScreen.tsx'
import { GalleryView } from '../src/components/GalleryView.tsx'
import { currentLabel, stepIndex } from '../src/components/lightbox.utils.ts'
import { SanitizedHtml } from '../src/components/SanitizedHtml.tsx'
import { TeamListView } from '../src/components/TeamListView.tsx'
import { TournamentListView } from '../src/components/TournamentListView.tsx'
import { TournamentTopView } from '../src/components/TournamentTopView.tsx'
import { tournamentListTitle, tournamentPageTitle } from '../src/components/title.utils.ts'
import { loadFixture } from './fixtures/load.ts'
import { renderHtml } from './render.ts'

const list = tournamentListResponseSchema.parse(loadFixture('tournaments'))
const top1 = tournamentTopResponseSchema.parse(loadFixture('tournament-1'))
const top2 = tournamentTopResponseSchema.parse(loadFixture('tournament-2'))
const teams1 = teamListResponseSchema.parse(loadFixture('teams-1'))
const teams2 = teamListResponseSchema.parse(loadFixture('teams-2'))
const gallery = galleryResponseSchema.parse(loadFixture('gallery-201'))

describe('<title>(clone-spec A 1.7)', () => {
  test('大会ページは "{接頭辞} {大会名} "、英語名が空なら空のまま', () => {
    assert.equal(tournamentPageTitle(top1.frame, 'jp'), '[SCV] 2026 架空ビーチカップ ')
    assert.equal(tournamentPageTitle(top1.frame, 'en'), '[SCV] 2026 Fictional Beach Cup ')
    assert.equal(tournamentPageTitle(top2.frame, 'en'), '[SCV]  ')
    assert.equal(
      tournamentPageTitle(top1.frame, 'jp', '-全スケジュール-'),
      '[SCV] 2026 架空ビーチカップ -全スケジュール-',
    )
  })
  test('大会一覧は lang に関係なく同じ', () => {
    assert.equal(tournamentListTitle(), '[SCV] SCORE VIEW大会一覧')
  })
})

describe('大会一覧(clone-spec A 3)', () => {
  test('区分ごとに clear の空 div、TeamListMIDA 見出し、FTitle の帯', () => {
    const html = renderHtml(<TournamentListView data={list} lang="jp" />)
    assert.match(
      html,
      /^<div style="clear:both;height:10px"><\/div><div style="margin:2px;padding:1px;border-color:#cccccc;border-width:0px;border-style:solid"><div class="TeamListMIDA">エントリー受付中<\/div><\/div>/,
    )
    assert.match(
      html,
      /<div style="margin-left:14px;margin-top:10px"><div class="TeamListBLOCK" style="margin:2px 0px;width:98%"><div style="padding:4px;color:white" class="px12 FTitle"><span class="teamimg_rnk"><a href="\/scview\/3\?lang=jp">2027 架空フリースタイル<\/a><\/span><\/div><\/div><\/div>/,
    )
    assert.match(html, /<div class="TeamListMIDA">大会終了<\/div>/)
    assert.doesNotMatch(html, /エントリー締切り/)
  })

  test('lang=en: 区分見出しは日本語のまま、英語名が空の大会はリンクの中身が空', () => {
    const html = renderHtml(<TournamentListView data={list} lang="en" />)
    assert.match(html, /<div class="TeamListMIDA">大会終了<\/div>/)
    assert.match(html, /<a href="\/scview\/1\?lang=en">2026 Fictional Beach Cup<\/a>/)
    assert.match(html, /<a href="\/scview\/2\?lang=en"><\/a>/)
  })
})

describe('大会トップ(clone-spec A 4)', () => {
  test('見出し、既定ロゴ、ディビジョン表、Select_Link と Dummy_Link、自由記述', () => {
    const html = renderHtml(<TournamentTopView data={top1} lang="jp" />)
    assert.match(html, /^<div class="text_title">2026 架空ビーチカップ<\/div>/)
    assert.match(html, /<img src="\/brand\/logo-default-tournament.svg" alt="ロゴ"\/>/)
    assert.match(
      html,
      /<div class="TBL" style="margin-left:80px;margin-right:100px"><table style="width:100%"><tbody><tr><td class="MD1 CM" width="40%">開催ディビジョン<\/td><td class="MD1 CM" width="40%">大会情報<\/td><\/tr>/,
    )
    assert.match(
      html,
      /<tr><td> ビーチミックス<\/td><td class="CM"><a class="Select_Link" href="\/pscore\/scv_search_div\/open\/101\?lang=jp">\[スケジュール\]<\/a>   <a class="Select_Link" href="\/pscore\/scv_division\/open\/101\?lang=jp">\[最終順位\]<\/a><\/td><\/tr>/,
    )
    assert.match(html, /<span class="Dummy_Link">\[最終順位\]<\/span>/)
    assert.match(
      html,
      /<br\/><hr\/><br\/><div id="SCV_TopPage" style="margin-left:10px"><p><span class="SCV_MIDA_M">【Result】<\/span><\/p>/,
    )
  })

  test('lang=en: 見出しは日本語名のまま、文言と英語名、英語の自由記述が空なら空の div', () => {
    const html = renderHtml(<TournamentTopView data={top1} lang="en" />)
    assert.match(html, /<div class="text_title">2026 架空ビーチカップ<\/div>/)
    assert.match(html, />Division<\/td>.*>Information<\/td>/)
    assert.match(html, / Beach Mixed<\/td>/)
    assert.match(html, />\[Schedule\]<\/a>.*>\[Standings\]<\/a>/)
    assert.match(html, /<div id="SCV_TopPage" style="margin-left:10px"><\/div>$/)
  })

  test('スケジュールも最終順位もないディビジョンは両方 Dummy_Link、登録ロゴを使う', () => {
    const html = renderHtml(<TournamentTopView data={top2} lang="jp" />)
    assert.match(html, /<img src="\/brand\/banner-sample.svg" alt="ロゴ"\/>/)
    assert.match(
      html,
      /<span class="Dummy_Link">\[スケジュール\]<\/span>   <span class="Dummy_Link">\[最終順位\]<\/span>/,
    )
    assert.doesNotMatch(html, /Select_Link/)
  })
})

describe('チーム一覧(clone-spec A 5)', () => {
  test('見出し "-- N Teams --"、色帯の 187px と人数の 30px、barStyle のクラス', () => {
    const html = renderHtml(<TeamListView data={teams1} lang="jp" />)
    assert.match(html, /<div class="TeamListMIDA">ビーチミックス   -- 4 Teams --   <\/div>/)
    assert.match(
      html,
      /<div class="TeamListBLOCK" style="margin:2px 0px;width:187px"><div style="padding:4px;color:white" class="px12 FTitle_Mix"><span class="teamimg_rnk"><a href="\/pscore\/scv_team\/open\/5001\?lang=jp">架空チームA<\/a><\/span><\/div><\/div>/,
    )
    assert.match(
      html,
      /<div class="TeamListBLOCK" style="margin:2px 8px 2px 0px;width:30px"><div style="padding:4px;color:white" class="px12 AreaSepa RM">x11<\/div><\/div>/,
    )
    assert.match(html, /class="px12 FTitle_Univ"/)
    assert.match(html, />x0<\/div>/)
  })

  test('チーム 0 のディビジョンも見出しと空の入れ物が出る', () => {
    const html = renderHtml(<TeamListView data={teams1} lang="jp" />)
    assert.match(
      html,
      /ウィメン   -- 0 Teams --   <\/div><\/div><div style="margin-left:14px;margin-top:10px"><\/div><div style="clear:both;height:10px"><\/div>$/,
    )
  })

  test('個人戦: "Players"、名前はリンクなし、人数欄は空白', () => {
    const html = renderHtml(<TeamListView data={teams2} lang="jp" />)
    assert.match(html, /-- 2 Players --/)
    assert.match(html, /<span class="teamimg_rnk">架空選手一<\/span>/)
    assert.match(html, /class="px12 AreaSepa RM"> <\/div>/)
    assert.doesNotMatch(html, /scv_team\//)
  })

  test('lang=en ではチーム名とディビジョン名が英語、リンクに lang=en', () => {
    const html = renderHtml(<TeamListView data={teams1} lang="en" />)
    assert.match(html, />Beach Mixed   -- 4 Teams/)
    assert.match(html, /href="\/pscore\/scv_team\/open\/5002\?lang=en">Fictional B</)
  })
})

describe('ギャラリー(clone-spec A 6)', () => {
  test('SCV_MIDA_M のタイトルとサムネイルの並び(初期状態でライトボックスは閉じている)', () => {
    const html = renderHtml(<GalleryView data={gallery} />)
    assert.match(
      html,
      /^<span class="SCV_MIDA_M">&lt; Day1-AM &gt;<\/span><br\/><div class="CM" id="photo">/,
    )
    assert.equal(html.match(/<a href="\/brand\/photo-sample.svg" rel="photos">/g)?.length, 3)
    assert.match(html, /<img src="\/brand\/photo-sample-thumb.svg" alt="" style="margin:2px"\/>/)
    assert.doesNotMatch(html, /lightbox/)
  })

  test('前後の移動は端で循環し、位置は "image n of m"', () => {
    assert.equal(stepIndex(0, -1, 3), 2)
    assert.equal(stepIndex(2, 1, 3), 0)
    assert.equal(stepIndex(1, 1, 3), 2)
    assert.equal(currentLabel(0, 189), 'image 1 of 189')
  })
})

describe('エラー画面(clone-spec A 7)', () => {
  test('api の message を埋め、/scview だけメニューへの文と戻るリンクを付ける', () => {
    const withMenu = renderHtml(<NotFoundScreen message="TOURNAMENT_ID [999999]" withMenuLink />)
    assert.match(withMenu, /<title>Error<\/title>/)
    assert.match(
      withMenu,
      /<div id="container"><h1>An Error Was Encountered<\/h1><p>TOURNAMENT_ID \[999999\]での操作に失敗しました。<br\/>情報が既に削除されているか、システムエラーの可能性があります。<br\/>メニュー画面より、再操作し確認して下さい。<br\/><br\/><a href="\/auth\/login">←メニュー画面に戻る<\/a><\/p><\/div>$/,
    )
    const plain = renderHtml(<NotFoundScreen message="TGALLERY_ID [999]" withMenuLink={false} />)
    assert.match(
      plain,
      /<p>TGALLERY_ID \[999\]での操作に失敗しました。<br\/>情報が既に削除されているか、システムエラーの可能性があります。<br\/><\/p>/,
    )
  })

  test('ルート不一致は 404 Page Not Found', () => {
    const html = renderHtml(<RouteNotFoundScreen />)
    assert.match(html, /<title>404 Page Not Found<\/title>/)
    assert.match(html, /<h1>404 Page Not Found<\/h1><p>The page you requested was not found.<\/p>/)
  })
})

describe('SanitizedHtml', () => {
  test('無害化済みの HTML をそのまま埋め込む', () => {
    const html = renderHtml(<SanitizedHtml id="x" html="<b>太字</b>" />)
    assert.equal(html, '<div id="x"><b>太字</b></div>')
  })
})
