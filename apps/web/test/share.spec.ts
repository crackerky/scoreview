import assert from 'node:assert/strict'
import { test } from 'node:test'
import { shareTargets, shareUrl } from '../src/components/share.utils.ts'

// ヘッダーの共有ボタン(src/components/ShareLinks.tsx)の行き先

test('共有する URL はページの位置(#)を含めず、クエリ(言語や絞り込み)は保つ', () => {
  assert.equal(shareUrl('https://s.example', '/scview/7', ''), 'https://s.example/scview/7')
  assert.equal(
    shareUrl('https://s.example', '/pscore/scv_team/open/3', 'lang=en'),
    'https://s.example/pscore/scv_team/open/3?lang=en',
  )
})

test('X と LINE の共有 URL に、ページの URL と題名を符号化して載せる', () => {
  const [x, line] = shareTargets('https://s.example/scview/7?lang=jp&a=1', '[SCV] 決勝 & 順位')
  assert.equal(x?.kind, 'x')
  const xUrl = new URL(x?.href ?? '')
  assert.equal(xUrl.origin + xUrl.pathname, 'https://x.com/intent/post')
  assert.equal(xUrl.searchParams.get('url'), 'https://s.example/scview/7?lang=jp&a=1')
  assert.equal(xUrl.searchParams.get('text'), '[SCV] 決勝 & 順位')
  assert.equal(line?.kind, 'line')
  const lineUrl = new URL(line?.href ?? '')
  assert.equal(lineUrl.hostname, 'social-plugins.line.me')
  assert.equal(lineUrl.searchParams.get('url'), 'https://s.example/scview/7?lang=jp&a=1')
})
