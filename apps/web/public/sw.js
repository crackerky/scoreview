// 記録係の画面を通信の無いところでも開けるようにする Service Worker
// (docs/operations/operator-experience.md「紙を上回ることが前提」2)。
// 登録は管理画面の DeviceSync.tsx が scope '/admin' で行う。閲覧ページは scope の外なので、
// このファイルの処理は閲覧ページの読み込みに一切かからない。
//
// - 記録画面と担当試合の一覧のページ(HTML): ネットワーク優先。届かなければ端末に保存した最後のページ
// - /_next/static/ の JS と CSS(ファイル名にハッシュが入り中身が変わらない): 端末の保存を優先
// - それ以外(/admin/api、Server Actions、RSC の取得): 保存しない
// 担当試合の一覧を開いたとき、まだ開いていない担当試合の記録画面も前もって保存する(precache)。

const VERSION = 'v1'
const PAGES = `sv-admin-pages-${VERSION}`
const STATIC = `sv-admin-static-${VERSION}`
// ネットワークを待つ上限(ミリ秒)。会場の回線が細いとき、保存したページへ早めに切り替える
const NETWORK_TIMEOUT_MS = 4000
// 保存するページ。記録画面と担当試合の一覧だけ(他の管理画面は通信がある前提)
const CACHEABLE_PAGE = /^\/admin(\/matches\/[0-9]+\/score)?\/?$/
const STATIC_ASSET = /\/_next\/static\/[^"'\s)\\]+/g

self.addEventListener('install', () => {
  self.skipWaiting()
})

self.addEventListener('activate', (event) => {
  event.waitUntil(
    (async () => {
      const keep = new Set([PAGES, STATIC])
      for (const name of await caches.keys()) {
        if (name.startsWith('sv-admin-') && !keep.has(name)) {
          await caches.delete(name)
        }
      }
      await self.clients.claim()
    })(),
  )
})

function isRscRequest(request, url) {
  return request.headers.get('RSC') === '1' || url.searchParams.has('_rsc')
}

/** ページを保存してよい応答か(ログイン画面へのリダイレクトやエラーを保存しない) */
function storable(response) {
  return response.ok && !response.redirected && response.type === 'basic'
}

async function withTimeout(promise, ms) {
  let timer
  const timeout = new Promise((_, reject) => {
    timer = setTimeout(() => reject(new TypeError('network timeout')), ms)
  })
  try {
    return await Promise.race([promise, timeout])
  } finally {
    clearTimeout(timer)
  }
}

/** ページの HTML が参照する /_next/static/ のファイルも保存する(通信が無くても画面が動くように) */
async function storeAssetsOf(html) {
  const cache = await caches.open(STATIC)
  const urls = new Set(html.match(STATIC_ASSET) ?? [])
  await Promise.all(
    [...urls].map(async (path) => {
      if (await cache.match(path)) {
        return
      }
      try {
        const response = await fetch(path)
        if (response.ok) {
          await cache.put(path, response)
        }
      } catch (error) {
        // 1 つ取れなくても他は保存する(次にページを開いたときに取り直す)
        if (!(error instanceof TypeError)) {
          throw error
        }
      }
    }),
  )
}

async function storePage(path, response) {
  const html = await response.clone().text()
  const cache = await caches.open(PAGES)
  await cache.put(path, response)
  await storeAssetsOf(html)
}

async function pageNetworkFirst(event, path) {
  try {
    const response = await withTimeout(fetch(event.request), NETWORK_TIMEOUT_MS)
    if (storable(response)) {
      event.waitUntil(storePage(path, response.clone()))
    }
    return response
  } catch (error) {
    if (!(error instanceof TypeError)) {
      throw error
    }
    const saved = await caches.match(path, { cacheName: PAGES })
    if (saved) {
      return saved
    }
    throw error
  }
}

async function staticCacheFirst(request) {
  const saved = await caches.match(request, { cacheName: STATIC })
  if (saved) {
    return saved
  }
  const response = await fetch(request)
  if (response.ok) {
    const cache = await caches.open(STATIC)
    await cache.put(request, response.clone())
  }
  return response
}

self.addEventListener('fetch', (event) => {
  const { request } = event
  if (request.method !== 'GET') {
    return
  }
  const url = new URL(request.url)
  if (url.origin !== self.location.origin) {
    return
  }
  if (url.pathname.startsWith('/_next/static/')) {
    event.respondWith(staticCacheFirst(request))
    return
  }
  if (
    request.mode === 'navigate' &&
    CACHEABLE_PAGE.test(url.pathname) &&
    !isRscRequest(request, url)
  ) {
    event.respondWith(pageNetworkFirst(event, url.pathname.replace(/\/$/, '') || '/admin'))
  }
})

async function precache(paths) {
  for (const path of paths) {
    if (!CACHEABLE_PAGE.test(path)) {
      continue
    }
    try {
      const response = await fetch(path, { credentials: 'same-origin' })
      if (storable(response)) {
        await storePage(path, response)
      }
    } catch (error) {
      // 圏外になった。保存できたところまでで止める
      if (error instanceof TypeError) {
        return
      }
      throw error
    }
  }
}

self.addEventListener('message', (event) => {
  const data = event.data
  if (data?.type === 'precache' && Array.isArray(data.paths)) {
    event.waitUntil(precache(data.paths.filter((path) => typeof path === 'string')))
  }
  if (data?.type === 'clear') {
    // ログアウトしたら、前の利用者のページを端末に残さない
    event.waitUntil(Promise.all([caches.delete(PAGES), caches.delete(STATIC)]))
  }
})
