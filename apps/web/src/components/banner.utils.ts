import type { TournamentFrame } from '@/api/common.types'
import type { Lang } from '@/i18n/lang'

// サイドバーのバナーの代替テキストと、会場図のバナーの判定(本サイト独自。docs/requirements.md
// 「意図的な差分」)。バナーには名前のデータが無く、参照サイトも alt を空で出していた。取込データの
// 画像のファイル名は「スコアサイト用リンクバナー(Boon).jpeg」の形で、括弧の中が協賛者などの名前になっている。

type Banner = TournamentFrame['banners'][number]

// 全角と半角の括弧の中身(最後の組)
const bracketed = /[(（]([^()（）]+)[)）][^()（）]*$/
// 括弧の中の末尾の番号(「主催２」の「２」)
const trailingNumber = /[0-9０-９]+$/
// 会場図の画像のファイル名(「Field Map3」「会場図」)
const venueMapName = /field\s*map|会場/i

/** URL の最後のパスの部分を、パーセント符号化を戻して返す。解釈できなければ空。 */
function fileName(url: string): string {
  if (!URL.canParse(url, 'https://scoreview.invalid')) {
    return ''
  }
  const last = new URL(url, 'https://scoreview.invalid').pathname.split('/').pop() ?? ''
  try {
    return decodeURIComponent(last)
  } catch (error) {
    // 不正なパーセント符号化(%E3 で途切れるなど)は名前として読めないので、名前なしとして扱う
    if (error instanceof URIError) {
      return ''
    }
    throw error
  }
}

function bannerName(banner: Banner): string {
  const match = bracketed.exec(fileName(banner.imageUrl))
  return match?.[1] === undefined ? '' : match[1].trim().replace(trailingNumber, '').trim()
}

/** リンク先が PDF で、画像のファイル名が会場図を示すバナー(参照サイトの「Field Map」バナー)。 */
export function isVenueMapBanner(banner: Banner): boolean {
  if (banner.linkUrl === null) {
    return false
  }
  const pdf = fileName(banner.linkUrl).toLowerCase().endsWith('.pdf')
  return pdf && venueMapName.test(fileName(banner.imageUrl))
}

/** 会場図のリンクの文字。 */
export function venueMapLabel(lang: Lang): string {
  return lang === 'en' ? 'Field map (PDF)' : '会場図(PDF)'
}

/**
 * バナー画像の代替テキスト。会場図は「会場図(PDF)」、ファイル名の括弧に名前があればその名前、
 * なければリンク先のホスト名。リンクの無いバナーで名前も無ければ空(飾りとして読み飛ばす)。
 */
export function bannerAlt(banner: Banner, lang: Lang): string {
  if (isVenueMapBanner(banner)) {
    return venueMapLabel(lang)
  }
  const name = bannerName(banner)
  if (name !== '' || banner.linkUrl === null) {
    return name
  }
  // mailto: などホスト名の無いリンクは URL そのもの
  const host = URL.canParse(banner.linkUrl) ? new URL(banner.linkUrl).hostname : ''
  return host === '' ? banner.linkUrl : host
}
