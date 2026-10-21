// サイトのブランド設定。参照サイト運営会社の名前・ロゴ・URL は使わない(requirements「意図的な差分」)。
// 初期値は仮の自社ブランド。差し替えるときはこのファイルと brand.css、public/brand/ の画像だけを変える。
// 色は brand.css の CSS 変数で持つ。

type ImageRef = { src: string; alt: string; width: number; height: number }
type TextLink = { label: string; href: string }

export type BrandConfig = {
  /** <title> の先頭に付ける接頭辞(参照サイトの "[CJS]" に当たる) */
  titlePrefix: string
  /** 大会一覧ページの <title> の本文(参照サイトの "ClubJr大会一覧") */
  tournamentListTitle: string
  /** ヘッダー左のロゴ。参照サイトは 168x80 */
  logo: ImageRef & { href: string }
  /** ヘッダー右上の文字リンク列(全角空白と | で区切って出す) */
  headerLinks: TextLink[]
  /** ヘッダー右上の SNS アイコン列(32x32、別タブで開く) */
  socialLinks: Array<{ href: string; icon: string; alt: string }>
  /** ヘッダー下段の 4 項目(画面上は左から 04, 03, 02, 01 の順に並ぶ。clone-spec A 1.2) */
  navigation: Array<{ id: 'navi01' | 'navi02' | 'navi03' | 'navi04'; label: string; href: string }>
  /** フッターの著作権者名。年は表示時の西暦を入れる */
  copyrightHolder: string
  /** ロゴ未登録の大会に出す既定ロゴ(参照サイトは 701x96) */
  defaultTournamentLogo: string
}

export const brand: BrandConfig = {
  titlePrefix: '[SCV]',
  tournamentListTitle: 'SCORE VIEW大会一覧',
  logo: {
    src: '/brand/logo.svg',
    alt: 'SCORE VIEW スコア速報',
    width: 168,
    height: 80,
    href: '/sctnmt',
  },
  headerLinks: [
    { label: '会社案内', href: 'https://example.com/company' },
    { label: '個人情報保護方針', href: 'https://example.com/privacy' },
    { label: 'お問い合わせ', href: 'mailto:info@example.com' },
  ],
  socialLinks: [
    { href: 'https://example.com/sns/x', icon: '/brand/sns-x.svg', alt: 'X icon' },
    { href: 'https://example.com/sns/video', icon: '/brand/sns-video.svg', alt: 'Video icon' },
    { href: 'https://example.com/sns/photo', icon: '/brand/sns-photo.svg', alt: 'Photo icon' },
  ],
  navigation: [
    { id: 'navi01', label: 'Event', href: 'https://example.com/events' },
    { id: 'navi02', label: 'Results', href: 'https://example.com/results' },
    { id: 'navi03', label: 'Gear', href: 'https://example.com/gear' },
    { id: 'navi04', label: 'About', href: 'https://example.com/about' },
  ],
  copyrightHolder: 'SCORE VIEW',
  defaultTournamentLogo: '/brand/logo-default-tournament.svg',
}
