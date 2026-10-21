import { Fragment } from 'react'
import type { TournamentFrame } from '@/api/common.types'
import { divisionHref, groupHref } from '@/i18n/href'
import type { Lang } from '@/i18n/lang'
import { pickText } from '@/i18n/lang'
import { groupLinkLabel } from './navi.utils'
import { assetSrc, referenceAssetMode } from './referenceAsset.utils'

// clone-spec A 1.5: 大会ページの div#sub。バナーが 0 枚でも枠だけの白い箱を出す。
// バナーは <br /><br /> 区切りで縦に並べ、リンク先があれば <a> で囲む。
// 参照サイトの枠画像(09_menu_outline_*.jpg)は legacy.css の罫線で置き換える。
//
// ディビジョン成績、グループ、チーム、試合、選手、ギャラリーのページでは、バナーの上に
// 「SCORE」メニュー(ディビジョン成績とグループへのリンク一覧)が出る。参照サイトの生 HTML
// 271 ページで確認した(大会トップ、チーム一覧、スケジュールには出ない)。
// 大会の全ディビジョンが載り、名前の後ろに競技の略号 "(Ult)" "(Free)" "(DiscGolfT)" が付く
// (日英とも)。日程のないディビジョン(237 のフリースタイル、536 のディスクゴルフ団体戦)は、
// 参照サイトでは alert("ページは存在しません") を出すだけのリンクなので、href のない <a> にする。

type SidebarProps = {
  frame: TournamentFrame
  lang: Lang
  /** SCORE メニューを出すか */
  scoreMenu: boolean
}

export function Sidebar({ frame, lang, scoreMenu }: SidebarProps) {
  return (
    <div id="sub">
      <table cellPadding={0} cellSpacing={0}>
        <tbody>
          <tr>
            <td colSpan={3} className="sub_frame_top" />
          </tr>
          <tr>
            <td width={4} className="sub_frame_side" />
            <td width={202} style={{ textAlign: 'center' }}>
              <table cellPadding={0} cellSpacing={0}>
                <tbody>
                  {scoreMenu && (
                    <tr>
                      <td style={{ textAlign: 'left' }}>
                        <ScoreMenu frame={frame} lang={lang} />
                      </td>
                    </tr>
                  )}
                  <tr>
                    <td>
                      <Banners banners={frame.banners} />
                    </td>
                  </tr>
                </tbody>
              </table>
            </td>
            <td width={4} className="sub_frame_side" />
          </tr>
          <tr>
            <td colSpan={3} className="sub_frame_bottom" />
          </tr>
        </tbody>
      </table>
    </div>
  )
}

function ScoreMenu({ frame, lang }: { frame: TournamentFrame; lang: Lang }) {
  return (
    <>
      <div className="CM">
        <div className="sub_score_title">SCORE</div>
      </div>
      <div style={{ height: '7px' }} />
      <div className="CM">
        <div className="sub_brake_line" />
      </div>
      <div className="menu_list">
        <ul>
          {frame.divisions.map((division) => (
            <li key={division.id}>
              <a href={division.scheduleEnabled ? divisionHref(division.id, lang) : undefined}>
                {`${pickText(lang, division.name)}(${division.menuSuffix})`}
              </a>
              <ul>
                {division.groups.map((group) => (
                  <li key={group.id}>
                    {group.linked ? (
                      <a href={groupHref(group.id, lang)}>
                        {groupLinkLabel(pickText(lang, group.name), groupHref(group.id, lang))}
                      </a>
                    ) : (
                      <p>{pickText(lang, group.name)}</p>
                    )}
                  </li>
                ))}
              </ul>
            </li>
          ))}
        </ul>
        <div className="CM">
          <div className="sub_brake_line" />
        </div>
      </div>
    </>
  )
}

// 参照サイトのバナーは幅 160px で高さ 54〜64px(legacy.css の #sub .sub_banners img)。読み込み前の
// 枠の比率に使う代表値で、実際の比率は画像を読んだ後に CSS の height: auto で決まる
const bannerWidthPx = 160
const bannerHeightPx = 60

function Banners({ banners }: { banners: TournamentFrame['banners'] }) {
  const mode = referenceAssetMode()
  return (
    <div className="sub_banners">
      {banners.map((banner, index) => {
        const image = (
          <img
            src={assetSrc(banner.imageUrl, mode)}
            alt=""
            width={bannerWidthPx}
            height={bannerHeightPx}
            loading="lazy"
          />
        )
        return (
          // biome-ignore lint/suspicious/noArrayIndexKey: 同じ画像のバナーが並ぶことがある
          <Fragment key={index}>
            {index > 0 && (
              <>
                <br />
                <br />
              </>
            )}
            {banner.linkUrl === null ? image : <a href={banner.linkUrl}>{image}</a>}
          </Fragment>
        )
      })}
    </div>
  )
}
