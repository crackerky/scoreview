import { Fragment, Suspense } from 'react'
import { brand } from '@/brand/brand.config'
import { LangSwitch } from './LangSwitch'
import { ShareLinks } from './ShareLinks'

// clone-spec A 1.2: div#outer1 > div#header(id が入れ子で重複する)> table。
// ロゴ、SNS アイコン、文字リンク列、4 項目のナビ。lang に関係なく同じものを出す。
// 参照サイトの画像と会社リンクはブランド設定に置き換える(requirements「意図的な差分」)。
// SNS アイコンの位置には今のページの共有ボタンを置く(ShareLinks)。

const fullWidthSpace = '　'

function HeaderBar() {
  // 先頭が "|　"、以降は "　|　" で区切る(末尾には付けない)。参照サイトで "English" があった右端には、
  // 本サイト独自の言語切替を置く(LangSwitch)。クエリを読むので、静的に描くページ(404)では
  // Suspense の fallback(何も出さない)になる
  return (
    <td id="header_bar">
      {`|${fullWidthSpace}`}
      {brand.headerLinks.map((link) => (
        <Fragment key={link.href}>
          <a href={link.href}>{link.label}</a>
          {`${fullWidthSpace}|${fullWidthSpace}`}
        </Fragment>
      ))}
      <Suspense fallback={null}>
        <LangSwitch />
      </Suspense>
    </td>
  )
}

export function SiteHeader() {
  return (
    <div id="outer1">
      <div id="header">
        <table cellPadding={0} cellSpacing={0}>
          <tbody>
            <tr>
              <td width="22%" rowSpan={2}>
                <a href={brand.logo.href}>
                  <img
                    src={brand.logo.src}
                    width={brand.logo.width}
                    height={brand.logo.height}
                    alt={brand.logo.alt}
                    className="logo"
                  />
                </a>
              </td>
              <td width="78%" style={{ textAlign: 'right', verticalAlign: 'top' }}>
                <table cellPadding={0} cellSpacing={0}>
                  <tbody>
                    <tr>
                      <td>
                        <div id="header_social">
                          <Suspense fallback={null}>
                            <ShareLinks />
                          </Suspense>
                        </div>
                      </td>
                      <td style={{ textAlign: 'right', verticalAlign: 'top' }}>
                        <table cellPadding={0} cellSpacing={0}>
                          <tbody>
                            <tr>
                              <td id="header_bar_spacer" />
                            </tr>
                            <tr>
                              <HeaderBar />
                            </tr>
                          </tbody>
                        </table>
                      </td>
                    </tr>
                  </tbody>
                </table>
              </td>
            </tr>
            <tr>
              <td style={{ textAlign: 'right', verticalAlign: 'bottom' }}>
                <div id="navigation">
                  <ul>
                    {brand.navigation.map((item) => (
                      <li key={item.id} id={item.id}>
                        <a href={item.href}>{item.label}</a>
                      </li>
                    ))}
                  </ul>
                </div>
              </td>
            </tr>
            <tr>
              <td colSpan={4}>
                <div id="header_spacer" />
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  )
}
