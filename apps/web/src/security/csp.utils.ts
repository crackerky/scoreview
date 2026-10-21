// Content-Security-Policy の組み立て(セキュリティ点検 L-4)。Next.js に依存しない純粋な関数にして
// テストする。送るのは src/proxy.ts。
//
// - 管理画面(/admin)は nonce の CSP。Next.js が要求ヘッダーの CSP から nonce を読み、自分の
//   スクリプトに付ける(Next.js 16 の Content Security Policy のガイド)。nonce はページを動的に
//   描画するときにしか付けられないが、管理画面はセッションの Cookie を読むので常に動的。
// - 閲覧ページは nonce を使わない。スクリプトの制限は 'self' とインラインだけにし(Next.js が
//   ページに埋めるインラインのスクリプトがあるため)、画像の読み込み先を参照サイトの画像の置き場に
//   限る。どちらのページもフレームへの埋め込み、<base>、他サイトへのフォーム送信を禁じる。

/**
 * 閲覧ページと管理画面が読む外部の画像の置き場。取込データ(大会トップの自由記述、バナー、写真、
 * グループの画像)の画像は参照サイト(score.clubjr.com)と運営会社のサイト(www.clubjr.com)にある
 * (開発用 DB の画像の URL で、サイト内のパス以外はこの 2 つと開発用シードの placehold.co だけだった。
 * http の画像は upgrade-insecure-requests で https で読む)。
 * REFERENCE_ASSET_MODE=allow(参照サイトの画像をそのまま出す。src/components/referenceAsset.utils.ts)の
 * ときだけ許す。既定の block では自作のプレースホルダに差し替えるので、外部の画像は読まない。
 * 他の置き場の画像を管理画面で登録するときは、ここに足す。
 */
export const IMAGE_HOSTS = ['https://score.clubjr.com', 'https://www.clubjr.com'] as const

/** 開発用のシード(cmd/seed)が使う仮の画像の置き場。開発中だけ許す */
export const DEV_IMAGE_HOSTS = ['https://placehold.co'] as const

/**
 * CSP の組み立ての条件。dev は next dev(React の開発用の評価に 'unsafe-eval' が要り、http の
 * localhost を https に上げない)。referenceImages は参照サイトの画像をそのまま出すか
 */
export type CspOptions = { dev: boolean; referenceImages: boolean }

/** CSP の指令を 1 行にする */
function serialize(directives: Record<string, string[]>): string {
  return Object.entries(directives)
    .map(([name, values]) => [name, ...values].join(' ').trim())
    .join('; ')
}

function imageSources({ dev, referenceImages }: CspOptions): string[] {
  return [
    "'self'",
    'data:',
    'blob:',
    ...(referenceImages ? IMAGE_HOSTS : []),
    ...(dev ? DEV_IMAGE_HOSTS : []),
  ]
}

/** どのページにも掛ける指令 */
function commonDirectives(options: CspOptions): Record<string, string[]> {
  return {
    'default-src': ["'self'"],
    // インラインの style 属性を使う(参照サイトの再現と自由記述の HTML)。スクリプトは実行できない
    'style-src': ["'self'", "'unsafe-inline'"],
    'img-src': imageSources(options),
    'font-src': ["'self'"],
    'object-src': ["'none'"],
    'base-uri': ["'none'"],
    'form-action': ["'self'"],
    'frame-ancestors': ["'none'"],
    // 手元(http://localhost)では https に上げると読めなくなるので付けない
    ...(options.dev ? {} : { 'upgrade-insecure-requests': [] }),
  }
}

/** 管理画面の CSP */
export function adminCsp(nonce: string, options: CspOptions): string {
  return serialize({
    ...commonDirectives(options),
    'script-src': [
      "'self'",
      `'nonce-${nonce}'`,
      "'strict-dynamic'",
      ...(options.dev ? ["'unsafe-eval'"] : []),
    ],
    // 'strict-dynamic' のもとでは 'self' の許可が効かないので、Service Worker の置き場を明示する
    'worker-src': ["'self'"],
    'connect-src': ["'self'"],
  })
}

/** 閲覧ページの CSP */
export function browseCsp(options: CspOptions): string {
  return serialize({
    ...commonDirectives(options),
    'script-src': ["'self'", "'unsafe-inline'", ...(options.dev ? ["'unsafe-eval'"] : [])],
    'connect-src': ["'self'"],
  })
}

/** 管理画面のパスかどうか(/admin と /admin/ 以下。/administrator などは含めない) */
export function isAdminPath(pathname: string): boolean {
  return pathname === '/admin' || pathname.startsWith('/admin/')
}
