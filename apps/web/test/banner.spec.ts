import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { bannerAlt, isVenueMapBanner } from '../src/components/banner.utils.ts'

// 取込データと同じ形のファイル名(「スコアサイト用リンクバナー(名前).jpeg」をパーセント符号化したもの)
const encoded = (name: string) =>
  `https://score.clubjr.com/zfiles/ScoreViewTop/${encodeURIComponent(`スコアサイト用リンクバナー${name}.jpeg`)}`

describe('サイドバーのバナーの代替テキストと会場図', () => {
  test('ファイル名の括弧の中を名前にし、末尾の番号は外す', () => {
    assert.equal(
      bannerAlt({ imageUrl: encoded('（Boon）'), linkUrl: 'https://example.com/' }, 'jp'),
      'Boon',
    )
    assert.equal(bannerAlt({ imageUrl: encoded('(主催２)'), linkUrl: null }, 'jp'), '主催')
  })

  test('名前が無ければリンク先のホスト名、リンクも無ければ空', () => {
    assert.equal(
      bannerAlt({ imageUrl: '/brand/banner-sample.svg', linkUrl: 'https://example.com/x' }, 'en'),
      'example.com',
    )
    assert.equal(bannerAlt({ imageUrl: '/brand/banner-sample.svg', linkUrl: null }, 'jp'), '')
    assert.equal(
      bannerAlt({ imageUrl: '/b.png', linkUrl: 'mailto:info@example.com' }, 'jp'),
      'mailto:info@example.com',
    )
  })

  test('会場図: リンク先が PDF で画像名に Field Map / 会場を含む', () => {
    const map = {
      imageUrl: encoded('（Field Map3）'),
      linkUrl: 'http://www.clubjr.com/event_2026atami_4.Map.pdf',
    }
    assert.ok(isVenueMapBanner(map))
    assert.equal(bannerAlt(map, 'jp'), '会場図(PDF)')
    assert.equal(bannerAlt(map, 'en'), 'Field map (PDF)')
    assert.ok(!isVenueMapBanner({ ...map, linkUrl: 'https://example.com/' }))
    assert.ok(!isVenueMapBanner({ ...map, linkUrl: null }))
    assert.ok(!isVenueMapBanner({ imageUrl: encoded('（Boon）'), linkUrl: 'https://x/a.pdf' }))
    assert.ok(isVenueMapBanner({ imageUrl: '/会場図.png', linkUrl: '/map.PDF' }))
  })

  test('壊れたパーセント符号化は名前なしとして扱う', () => {
    assert.equal(bannerAlt({ imageUrl: '/zfiles/%E3%81.jpeg', linkUrl: null }, 'jp'), '')
  })
})
