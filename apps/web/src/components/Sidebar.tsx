import { Fragment } from 'react'
import type { TournamentFrame } from '@/api/common.types'

// clone-spec A 1.5: 大会ページの div#sub。バナーが 0 枚でも枠だけの白い箱を出す。
// バナーは <br /><br /> 区切りで縦に並べ、リンク先があれば <a> で囲む。
// 参照サイトの枠画像(09_menu_outline_*.jpg)は legacy.css の罫線で置き換える。

export function Sidebar({ banners }: { banners: TournamentFrame['banners'] }) {
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
