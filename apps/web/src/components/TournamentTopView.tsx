import { Fragment } from 'react'
import type { TournamentTopResponse } from '@/api/public.types'
import { brand } from '@/brand/brand.config'
import { divisionHref, divisionScheduleHref } from '@/i18n/href'
import { type Lang, pickText } from '@/i18n/lang'
import { msg } from '@/i18n/messages'
import { isVenueMapBanner, venueMapLabel } from './banner.utils'
import { assetSrc, referenceAssetMode, rewriteHtmlImages } from './referenceAsset.utils'
import { SanitizedHtml } from './SanitizedHtml'
import { Tbl, TblHeaderRow } from './Tbl'

// clone-spec A 4: 大会トップ(/scview/{id} と /pscore/scv_top/open/{id})の #score_view の中身。

const nbsp3 = '   '

/** リンクがあれば a.Select_Link、なければ灰色の span.Dummy_Link(押せない文字)。 */
function DivisionLink({ href, label }: { href: string | null; label: string }) {
  return href === null ? (
    <span className="Dummy_Link">{label}</span>
  ) : (
    <a className="Select_Link" href={href}>
      {label}
    </a>
  )
}

/**
 * 本サイト独自の追記(docs/requirements.md「意図的な差分」): サイドバーの会場図のバナー(リンク先が PDF)
 * への文字のリンク。スマートフォンではサイドバーが画面の外にあり、会場図に辿り着けないため。
 * 該当するバナーが無ければ何も出さない。
 */
function VenueMapLinks({
  banners,
  lang,
}: {
  banners: TournamentTopResponse['frame']['banners']
  lang: Lang
}) {
  const maps = banners.filter(isVenueMapBanner)
  if (maps.length === 0) {
    return null
  }
  return (
    <p className="sv_venue_map" data-scoreview-addition="">
      {maps.map((banner, index) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: 同じリンク先の会場図が並ぶことがある
        <Fragment key={index}>
          {index > 0 && ' '}
          <a href={banner.linkUrl ?? undefined}>{venueMapLabel(lang)}</a>
        </Fragment>
      ))}
    </p>
  )
}

export function TournamentTopView({ data, lang }: { data: TournamentTopResponse; lang: Lang }) {
  const mode = referenceAssetMode()
  return (
    <>
      {/* clone-spec A 1.7: 見出しの大会名は lang=en でも日本語名 */}
      <div className="text_title">{data.frame.name.ja}</div>
      <div style={{ textAlign: 'center', margin: '10px 10px 15px 10px' }}>
        {/* 最初の画面に入るので遅延読み込みにしない。寸法は大会ごとに違い api に無い */}
        <img
          src={assetSrc(data.logoUrl ?? brand.defaultTournamentLogo, mode)}
          alt={msg(lang, 'logoAlt')}
        />
      </div>
      {/* 本サイト独自: スマートフォン向けの観戦アプリ(/live/{id})への入口 */}
      <p className="sv_live_link" data-scoreview-addition="">
        <a href={`/live/${data.frame.id}`}>{msg(lang, 'liveAppLink')}</a>
      </p>
      <Tbl style={{ marginLeft: '80px', marginRight: '100px' }} tableStyle={{ width: '100%' }}>
        {/* 列幅は 40% + 40%。table-layout: fixed なので右の 20% は空く(参照サイトのまま) */}
        <TblHeaderRow
          columns={[
            { label: msg(lang, 'topDivision'), width: '40%' },
            { label: msg(lang, 'topInformation'), width: '40%' },
          ]}
        />
        {data.divisions.map((division) => (
          <tr key={division.id}>
            <td>
              {' '}
              {pickText(lang, division.name)}
            </td>
            <td className="CM">
              <DivisionLink
                href={division.scheduleLinked ? divisionScheduleHref(division.id, lang) : null}
                label={msg(lang, 'topScheduleLink')}
              />
              {nbsp3}
              <DivisionLink
                href={division.standingsLinked ? divisionHref(division.id, lang) : null}
                label={msg(lang, 'topStandingsLink')}
              />
            </td>
          </tr>
        ))}
      </Tbl>
      <VenueMapLinks banners={data.frame.banners} lang={lang} />
      <br />
      <hr />
      <br />
      {/* 自由記述は言語別のフィールド。英語が空なら空の div(clone-spec A 4) */}
      <SanitizedHtml
        id="SCV_TopPage"
        style={{ marginLeft: '10px' }}
        html={rewriteHtmlImages(pickText(lang, data.topHtml), mode)}
      />
    </>
  )
}
