import assert from 'node:assert/strict'
import { existsSync } from 'node:fs'
import { describe, test } from 'node:test'
import { tournamentFrameSchema } from '../src/api/common.types.ts'
import { brand } from '../src/brand/brand.config.ts'
import { SiteFrame } from '../src/components/SiteFrame.tsx'
import { loadFixture } from './fixtures/load.ts'
import { renderHtml } from './render.ts'

const frame = tournamentFrameSchema.parse(loadFixture('frame-1'))

function renderTournament(topMenu?: null): string {
  return renderHtml(
    <SiteFrame variant={{ kind: 'tournament', frame, lang: 'jp' }} topMenu={topMenu} year={2026}>
      <p>本文</p>
    </SiteFrame>,
  )
}

describe('外枠', () => {
  test('大会ページ: 背景パターン、ナビ、サイドバー、本文の入れ子', () => {
    const html = renderTournament()
    assert.match(
      html,
      /^<style>body \{background: var\(--brand-body-color\) var\(--brand-body-bg\) repeat top;\}/,
    )
    assert.match(
      html,
      /<div id="top"><div id="wrapper"><header id="header"><div id="outer1"><div id="header">/,
    )
    assert.match(html, /<\/div><\/div><ul id="score_navi">/)
    assert.match(html, /<aside id="sub">/)
    // 本サイト独自: header / aside / main / footer 要素(ランドマーク)と、視覚的に隠した h1
    assert.match(
      html,
      /<div id="contents"><main id="main"><h1 class="sv_sr" data-scoreview-addition="">2026 架空ビーチカップ<\/h1><div id="score_parts"><div id="score_view"><p>本文<\/p><\/div><\/div><br\/><\/main><\/div>/,
    )
    assert.doesNotMatch(html, /score_topmenu/)
    assert.match(
      html,
      /<div id="footer_ue"><\/div><footer id="footer"><div class="copyright">Copyright\(c\) 2026 SCORE VIEW All Rights Reserved\.<\/div><\/footer>$/,
    )
  })

  test('視覚的に隠した h1 は大会名とページの内容。英語名が空なら日本語名', () => {
    const html = renderHtml(
      <SiteFrame variant={{ kind: 'tournament', frame, lang: 'jp' }} heading="チーム一覧">
        {null}
      </SiteFrame>,
    )
    assert.match(html, /<h1 class="sv_sr" data-scoreview-addition="">[^<]+ チーム一覧<\/h1>/)
    const noEnglish = { ...frame, name: { ja: '架空大会', en: '' } }
    const en = renderHtml(
      <SiteFrame variant={{ kind: 'tournament', frame: noEnglish, lang: 'en' }} heading="Team List">
        {null}
      </SiteFrame>,
    )
    assert.match(en, />架空大会 Team List<\/h1>/)
    const list = renderHtml(<SiteFrame variant={{ kind: 'list' }}>{null}</SiteFrame>)
    assert.match(list, /<h1 class="sv_sr" data-scoreview-addition="">SCORE VIEW大会一覧<\/h1>/)
    const info = renderHtml(
      <SiteFrame variant={{ kind: 'list' }} heading="見方ガイド">
        {null}
      </SiteFrame>,
    )
    assert.match(info, /<h1 class="sv_sr" data-scoreview-addition="">見方ガイド<\/h1>/)
  })

  test('topMenu=null なら空の #score_topmenu を出す', () => {
    assert.match(renderTournament(null), /<div id="score_topmenu"><\/div><div id="score_view">/)
  })

  test('大会一覧: 黒背景、ナビとサイドバーなし', () => {
    const html = renderHtml(
      <SiteFrame variant={{ kind: 'list' }} year={2026}>
        <p>一覧</p>
      </SiteFrame>,
    )
    assert.match(html, /^<style>body \{background: var\(--brand-list-body-bg\);\}<\/style>/)
    assert.doesNotMatch(html, /score_navi|id="sub"/)
  })

  test('ヘッダーはブランド設定の値だけを使う', () => {
    const html = renderTournament()
    assert.match(
      html,
      /<img src="\/brand\/logo.svg" width="168" height="80" alt="SCORE VIEW スコア速報" class="logo"\/>/,
    )
    assert.match(html, /<td id="header_bar">\|　<a href="\/about#operator">運営者情報<\/a>　\|　<a/)
    assert.match(html, /<li id="navi04"><a href="\/sctnmt">Events<\/a><\/li>/)
    assert.doesNotMatch(html, /clubjr|CLUB Jr/i)
  })

  test('ヘッダーのリンクは全部サイトの中のページに行き先がある(仮の外部 URL や mailto を残さない)', () => {
    const links = [...brand.headerLinks, ...brand.navigation].map((link) => link.href)
    for (const href of links) {
      assert.match(href, /^\/[a-z]+(#[a-z]+)?$/, href)
      const page = new URL(
        `../src/app${new URL(href, 'http://x').pathname}/page.tsx`,
        import.meta.url,
      )
      assert.ok(existsSync(page), `${href} のページ(${page.pathname})が無い`)
    }
  })

  test('サイドバーのバナーは <br /> 2 つで区切り、リンク先がなければ画像だけ', () => {
    const html = renderTournament()
    assert.match(
      html,
      /<div class="sub_banners"><a href="https:\/\/example.com\/"><img src="\/brand\/banner-sample.svg" alt="example.com" width="160" height="60" loading="lazy"\/><\/a><br\/><br\/><img src="\/brand\/banner-sample.svg" alt="" width="160" height="60" loading="lazy"\/><\/div>/,
    )
  })

  test('参照サイトのバナー画像は(REFERENCE_ASSET_MODE が既定の block なら)プレースホルダにする', () => {
    const reference = {
      ...frame,
      banners: [{ imageUrl: 'https://score.clubjr.com/zfiles/b.jpeg', linkUrl: null }],
    }
    const html = renderHtml(
      <SiteFrame variant={{ kind: 'tournament', frame: reference, lang: 'jp' }} year={2026}>
        {null}
      </SiteFrame>,
    )
    assert.match(html, /<img src="\/placeholder\/reference-image.svg"/)
    assert.doesNotMatch(html, /clubjr/)
  })

  test('asOf を渡すと本文の上に「HH:MM 時点」(日本時間)を出す。渡さなければ出さない', () => {
    // 2026-10-21 15:04 UTC は日本時間の 10-22 00:04
    const at = new Date('2026-10-21T15:04:59Z')
    const jp = renderHtml(
      <SiteFrame variant={{ kind: 'tournament', frame, lang: 'jp' }} asOf={at} year={2026}>
        <p>本文</p>
      </SiteFrame>,
    )
    assert.match(
      jp,
      /<div id="score_view"><p class="sv_as_of" data-scoreview-addition="">00:04 時点<\/p><p>本文<\/p><\/div>/,
    )
    const en = renderHtml(
      <SiteFrame variant={{ kind: 'tournament', frame, lang: 'en' }} asOf={at} year={2026}>
        {null}
      </SiteFrame>,
    )
    assert.match(en, />As of 00:04 JST<\/p>/)
    assert.doesNotMatch(renderTournament(), /sv_as_of/)
  })

  test('バナー 0 枚でも枠は出る', () => {
    const empty = { ...frame, banners: [] }
    const html = renderHtml(
      <SiteFrame variant={{ kind: 'tournament', frame: empty, lang: 'en' }} year={2026}>
        {null}
      </SiteFrame>,
    )
    assert.match(html, /<div class="sub_banners"><\/div>/)
    assert.match(html, /sub_frame_top/)
  })
})
