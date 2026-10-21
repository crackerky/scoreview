import { Fragment } from 'react'
import type { TournamentFrame } from '@/api/common.types'
import { divisionHref, groupHref } from '@/i18n/href'
import type { Lang } from '@/i18n/lang'
import { pickText } from '@/i18n/lang'
import { groupLinkLabel } from './navi.utils'

// clone-spec A 1.5: 大会ページの div#sub。バナーが 0 枚でも枠だけの白い箱を出す。
// バナーは <br /><br /> 区切りで縦に並べ、リンク先があれば <a> で囲む。
// 参照サイトの枠画像(09_menu_outline_*.jpg)は legacy.css の罫線で置き換える。
//
// ディビジョン成績、グループ、チーム、試合、選手、ギャラリーのページでは、バナーの上に
// 「SCORE」メニュー(ディビジョン成績とグループへのリンク一覧)が出る。参照サイトの生 HTML
// 271 ページで確認した(大会トップ、チーム一覧、スケジュールには出ない)。
// スケジュールのないディビジョン(ディスクゴルフ、フリースタイル)は載らない。

// 参照サイトでは観測した全ディビジョンの名前の後ろに "(Ult)" が付く(日英とも)。競技の種別を
// 表す接尾辞だと推測するが、Ult 以外の値は 348 大会で観測されていないので固定値にする
const divisionSuffix = '(Ult)'

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
  const divisions = frame.divisions.filter((division) => division.scheduleEnabled)
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
          {divisions.map((division) => (
            <li key={division.id}>
              <a href={divisionHref(division.id, lang)}>
                {`${pickText(lang, division.name)}${divisionSuffix}`}
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

function Banners({ banners }: { banners: TournamentFrame['banners'] }) {
  return (
    <div className="sub_banners">
      {banners.map((banner, index) => {
        const image = <img src={banner.imageUrl} alt="" />
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
