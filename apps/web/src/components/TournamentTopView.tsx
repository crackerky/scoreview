import type { TournamentTopResponse } from '@/api/public.types'
import { brand } from '@/brand/brand.config'
import { divisionHref, divisionScheduleHref } from '@/i18n/href'
import { type Lang, pickText } from '@/i18n/lang'
import { msg } from '@/i18n/messages'
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

export function TournamentTopView({ data, lang }: { data: TournamentTopResponse; lang: Lang }) {
  return (
    <>
      {/* clone-spec A 1.7: 見出しの大会名は lang=en でも日本語名 */}
      <div className="text_title">{data.frame.name.ja}</div>
      <div style={{ textAlign: 'center', margin: '10px 10px 15px 10px' }}>
        <img src={data.logoUrl ?? brand.defaultTournamentLogo} alt={msg(lang, 'logoAlt')} />
      </div>
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
      <br />
      <hr />
      <br />
      {/* 自由記述は言語別のフィールド。英語が空なら空の div(clone-spec A 4) */}
      <SanitizedHtml
        id="SCV_TopPage"
        style={{ marginLeft: '10px' }}
        html={pickText(lang, data.topHtml)}
      />
    </>
  )
}
