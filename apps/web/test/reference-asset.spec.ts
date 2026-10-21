import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import {
  assetSrc,
  isReferenceAssetUrl,
  parseReferenceAssetMode,
  ReferenceAssetModeError,
  referencePlaceholderSrc,
  rewriteHtmlImages,
} from '../src/components/referenceAsset.utils.ts'

describe('参照サイトの画像の差し替え(REFERENCE_ASSET_MODE)', () => {
  test('未設定と block は block、allow は allow、それ以外は設定の誤り', () => {
    assert.equal(parseReferenceAssetMode(undefined), 'block')
    assert.equal(parseReferenceAssetMode(''), 'block')
    assert.equal(parseReferenceAssetMode('block'), 'block')
    assert.equal(parseReferenceAssetMode('allow'), 'allow')
    assert.throws(() => parseReferenceAssetMode('off'), ReferenceAssetModeError)
  })

  test('参照サイトと運営会社のホスト、参照サイトの /zfiles/ を参照サイトの画像とみなす', () => {
    assert.ok(isReferenceAssetUrl('https://score.clubjr.com/zfiles/ScoreViewTop/a.jpeg'))
    assert.ok(isReferenceAssetUrl('http://www.clubjr.com/images/2015ung-banner.jpg'))
    assert.ok(isReferenceAssetUrl('/zfiles/ScoreViewTop/event_2026atami_photo_B_1st.jpg'))
    assert.ok(!isReferenceAssetUrl('https://placehold.co/160x64?text=Banner'))
    assert.ok(!isReferenceAssetUrl('/brand/logo-default-tournament.svg'))
    assert.ok(!isReferenceAssetUrl('https://notclubjr.com/a.png'))
    assert.ok(!isReferenceAssetUrl('http://[bad'))
  })

  test('block なら参照サイトの画像だけ差し替え、allow ならそのまま', () => {
    const reference = 'https://score.clubjr.com/zfiles/a.jpeg'
    assert.equal(assetSrc(reference, 'block'), referencePlaceholderSrc)
    assert.equal(assetSrc(reference, 'allow'), reference)
    assert.equal(assetSrc('https://placehold.co/1x1', 'block'), 'https://placehold.co/1x1')
  })

  test('自由記述の img: 参照サイトの src を差し替え、loading="lazy" を足す', () => {
    const html =
      '<p><img alt="" src="/zfiles/ScoreViewTop/a.jpg" style="width: 500px; height: 350px"/>' +
      '<img src="https://placehold.co/600x400?text=1st&amp;x=1"/></p>'
    assert.equal(
      rewriteHtmlImages(html, 'block'),
      `<p><img loading="lazy" alt="" src="${referencePlaceholderSrc}" style="width: 500px; height: 350px"/>` +
        '<img loading="lazy" src="https://placehold.co/600x400?text=1st&amp;x=1"/></p>',
    )
    assert.equal(
      rewriteHtmlImages(html, 'allow'),
      '<p><img loading="lazy" alt="" src="/zfiles/ScoreViewTop/a.jpg" style="width: 500px; height: 350px"/>' +
        '<img loading="lazy" src="https://placehold.co/600x400?text=1st&amp;x=1"/></p>',
    )
  })

  test('実体参照で書かれたホストも判定し、img 以外と loading 済みの img の loading は変えない', () => {
    const html =
      '<a href="https://score.clubjr.com/x">x</a>' +
      '<img loading="eager" src="http&#58;//www.clubjr.com/a.jpg"/>'
    assert.equal(
      rewriteHtmlImages(html, 'block'),
      '<a href="https://score.clubjr.com/x">x</a>' +
        `<img loading="eager" src="${referencePlaceholderSrc}"/>`,
    )
    const quoted = '<img src="https://score.clubjr.com/a.jpg?a=1&amp;b=2"/>'
    assert.equal(
      rewriteHtmlImages(quoted, 'block'),
      `<img loading="lazy" src="${referencePlaceholderSrc}"/>`,
    )
  })
})
