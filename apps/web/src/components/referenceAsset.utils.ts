// 取込データの画像 URL のうち、参照サイト(score.clubjr.com と運営会社の www.clubjr.com)を指すものを、
// 表示のときに自作のプレースホルダ画像へ差し替える。開発環境で参照サイトに画像を取りに行かないため
// (観戦者レビュー H4。1 ページで 16〜30 枚の画像が参照サイトを指していた)。
//
// 差し替えは web の表示時に行う。api の応答(無害化済みの自由記述 HTML を含む)はデータそのままに保ち、
// 表示の方針を環境変数 REFERENCE_ASSET_MODE で web 側だけで切り替えるため。

/** block(既定)は参照サイトの画像を差し替える。allow はそのまま出す(参照サイトとの見比べ用)。 */
export type ReferenceAssetMode = 'block' | 'allow'

/** 環境変数の値が block / allow 以外のとき。設定の誤りを黙って既定値で隠さない。 */
export class ReferenceAssetModeError extends Error {
  constructor(value: string) {
    super(`REFERENCE_ASSET_MODE must be "block" or "allow": ${value}`)
    this.name = 'ReferenceAssetModeError'
  }
}

/** 差し替え先(apps/web/public の自作画像。160x60 の灰色の枠) */
export const referencePlaceholderSrc = '/placeholder/reference-image.svg'

// 参照サイトと運営会社のサイト。サブドメインを含む
const referenceHost = /(^|\.)clubjr\.com$/i
// 自由記述の画像は参照サイトのルート相対パス(/zfiles/ScoreViewTop/...)で書かれている。本サイトには
// このディレクトリが無いので、同じく参照サイトの画像として扱う
const referenceRootPath = '/zfiles/'
// 相対 URL を解決するための仮のホスト(.invalid は実在しない)
const relativeBase = 'https://scoreview.invalid'

/** 未設定は block。block / allow 以外は ReferenceAssetModeError。 */
export function parseReferenceAssetMode(value: string | undefined): ReferenceAssetMode {
  if (value === undefined || value === '' || value === 'block') {
    return 'block'
  }
  if (value === 'allow') {
    return 'allow'
  }
  throw new ReferenceAssetModeError(value)
}

/** サーバーの環境変数 REFERENCE_ASSET_MODE を読む(Server Component から呼ぶ)。 */
export function referenceAssetMode(): ReferenceAssetMode {
  return parseReferenceAssetMode(process.env.REFERENCE_ASSET_MODE)
}

/** 参照サイトの画像を指す URL か。解釈できない文字列は参照サイトではないとみなす。 */
export function isReferenceAssetUrl(url: string): boolean {
  if (!URL.canParse(url, relativeBase)) {
    return false
  }
  const parsed = new URL(url, relativeBase)
  if (parsed.origin === relativeBase) {
    return parsed.pathname.startsWith(referenceRootPath)
  }
  return referenceHost.test(parsed.hostname)
}

/** mode が block で参照サイトの画像なら、プレースホルダの URL を返す。それ以外はそのまま。 */
export function assetSrc(url: string, mode: ReferenceAssetMode): string {
  return mode === 'block' && isReferenceAssetUrl(url) ? referencePlaceholderSrc : url
}

// api の無害化(bluemonday)を通った HTML では、属性値は二重引用符で囲まれ、& などは実体参照になる
const imgTag = /<img\b[^>]*>/gi
const srcAttribute = /\ssrc="([^"]*)"/i
const loadingAttribute = /\sloading=/i

const namedEntities: Record<string, string> = {
  amp: '&',
  quot: '"',
  apos: "'",
  lt: '<',
  gt: '>',
  colon: ':',
  sol: '/',
}
const entity = /&(#x[0-9a-f]+|#[0-9]+|[a-z]+);/gi

/** 属性値の実体参照を戻す(ブラウザが読む URL で判定するため。数値参照も含む)。 */
function decodeAttribute(value: string): string {
  return value.replace(entity, (whole, body: string) => {
    if (body.startsWith('#')) {
      const hex = body[1] === 'x' || body[1] === 'X'
      const code = Number.parseInt(body.slice(hex ? 2 : 1), hex ? 16 : 10)
      return Number.isNaN(code) || code > 0x10ffff ? whole : String.fromCodePoint(code)
    }
    return namedEntities[body.toLowerCase()] ?? whole
  })
}

/**
 * 無害化済みの自由記述 HTML の img を書き換える。参照サイトの画像は(block のとき)プレースホルダに
 * 差し替え、loading="lazy" を足す(自由記述の画像は大会ロゴとディビジョン表の下にあり、最初の画面に
 * 入らない)。差し替える src は固定のパスなので、無害化済みであることは崩れない。
 */
export function rewriteHtmlImages(html: string, mode: ReferenceAssetMode): string {
  return html.replace(imgTag, (tag) => {
    let out = tag.replace(srcAttribute, (whole, raw: string) =>
      assetSrc(decodeAttribute(raw), mode) === referencePlaceholderSrc
        ? ` src="${referencePlaceholderSrc}"`
        : whole,
    )
    if (!loadingAttribute.test(out)) {
      out = out.replace(/^<img\b/i, '<img loading="lazy"')
    }
    return out
  })
}
