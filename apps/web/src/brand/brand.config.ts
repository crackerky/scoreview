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
  // 行き先はサイトの中の案内ページ(src/app/about などの InfoPage)。参照サイトは運営会社のサイトへ出る
  headerLinks: [
    { label: '運営者情報', href: '/about#operator' },
    { label: '個人情報保護方針', href: '/privacy' },
    { label: 'お問い合わせ', href: '/contact' },
  ],
  navigation: [
    { id: 'navi01', label: 'About', href: '/about' },
    { id: 'navi02', label: 'Organizers', href: '/about#organizers' },
    { id: 'navi03', label: 'Guide', href: '/guide' },
    { id: 'navi04', label: 'Events', href: '/sctnmt' },
  ],
  copyrightHolder: 'SCORE VIEW',
  defaultTournamentLogo: '/brand/logo-default-tournament.svg',
}
