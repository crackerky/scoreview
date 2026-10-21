import type { ScheduleResponse } from '@/api/public.types'
import { formatYmdCompact, formatYmdSlash } from '@/format/format.utils'
import { type Lang, pickText } from '@/i18n/lang'
import { type MessageKey, msg } from '@/i18n/messages'
import { AutoSubmitSelect } from './AutoSubmitSelect'
import { MatchRowLine } from './MatchRowLine'
import { RefreshButtons } from './RefreshButtons'
import {
  nextSort,
  type ScheduleSort,
  type ScheduleSortKey,
  scheduleHref,
  sortFromResponse,
} from './schedule.utils'
import { Tbl, type TblColumn, TblHeaderRow } from './Tbl'

// clone-spec B 5: スケジュール(scv_search_tn / scv_search_div)の #score_view の中身。
// 検索フォーム(div.ITBL)と 11 列の一覧表(div.TBL)。2 つのページはテンプレートが共通。

type Target = { controller: 'scv_search_tn' | 'scv_search_div'; id: number }

function targetOf(data: ScheduleResponse): Target {
  if (data.scope === 'division' && data.division !== null) {
    return { controller: 'scv_search_div', id: data.division.id }
  }
  return { controller: 'scv_search_tn', id: data.frame.id }
}

type HeaderSpec = {
  label: MessageKey
  width: string
  align?: 'LM' | 'RM'
  /** 並べ替えのリンクを付ける列 */
  sortKey?: ScheduleSortKey
  /** clone-spec B 5: 左チームの見出しだけリンクの後ろに半角空白がある */
  trailingSpace?: boolean
}

// clone-spec B 5: 列幅の合計は 100%
const headers: HeaderSpec[] = [
  { label: 'colDivision', width: '13%', sortKey: 'division' },
  { label: 'colDate', width: '7%', sortKey: 'date' },
  { label: 'colTime', width: '7%', sortKey: 'time' },
  { label: 'colField', width: '7%', sortKey: 'field' },
  { label: 'colCode', width: '7%', sortKey: 'code' },
  { label: 'colTeam', width: '17%', align: 'RM', sortKey: 'leftTeam', trailingSpace: true },
  { label: 'colDash', width: '4%' },
  { label: 'colVs', width: '3%' },
  { label: 'colDash', width: '4%' },
  { label: 'colTeam', width: '17%', align: 'LM', sortKey: 'rightTeam' },
  { label: 'colStatus', width: '14%' },
]

/** clone-spec B 2: 並べ替え中の列は MD1_ASC + ▼ / MD1_DSC + ▲。 */
function sortLinkLook(key: ScheduleSortKey, current: ScheduleSort | null) {
  if (current?.key !== key) {
    return { className: 'MD1', arrow: '' }
  }
  return current.dir === 'asc'
    ? { className: 'MD1_ASC', arrow: '▼' }
    : { className: 'MD1_DSC', arrow: '▲' }
}

function headerColumns(data: ScheduleResponse, lang: Lang, target: Target): TblColumn[] {
  const current = sortFromResponse(data.sort)
  return headers.map((header) => {
    const text = msg(lang, header.label)
    if (header.sortKey === undefined) {
      return { label: text, width: header.width, align: header.align }
    }
    const look = sortLinkLook(header.sortKey, current)
    const href = scheduleHref(
      target.controller,
      target.id,
      lang,
      data.filters.selected,
      nextSort(header.sortKey, current),
    )
    // key は Biome の useJsxKeyInIterable のため(見出しの label として 1 つずつ置くだけで並べ替えない)
    const link = (
      <a key={header.sortKey} className={look.className} href={href}>
        {`${text}${look.arrow}`}
      </a>
    )
    return {
      label: header.trailingSpace ? <>{link} </> : link,
      width: header.width,
      align: header.align,
    }
  })
}

/**
 * 検索フォーム。参照サイトは POST だが本サイトは GET(requirements「意図的な差分」)。
 * 本サイト独自: 見出しの文字を select の label にする(支援技術向け。見た目は同じ)。
 * 並べ替え(sortkey1)はフォームに含めないので、select を変えると並べ替えが外れる(clone-spec B 2)。
 */
function SearchForm({
  data,
  lang,
  target,
}: {
  data: ScheduleResponse
  lang: Lang
  target: Target
}) {
  const { filters } = data
  return (
    <div className="ITBL">
      <form acceptCharset="utf-8" method="get">
        <input type="hidden" name="lang" value={lang} />
        <table>
          <tbody>
            <tr>
              <td className="NINI CM BOLD" width="40px">
                <label htmlFor="ymd">{msg(lang, 'colDate')}</label>
              </td>
              <td className="" width="120px">
                <AutoSubmitSelect
                  name="ymd"
                  className="NWAKU  WMAX"
                  id="ymd"
                  defaultValue={filters.selected.ymd}
                >
                  <option value="" />
                  {filters.dates.map((date) => (
                    <option key={date} value={formatYmdCompact(date)}>
                      {formatYmdSlash(date)}
                    </option>
                  ))}
                </AutoSubmitSelect>
              </td>
              <td className="NINI" width="10px" />
              <td className="NINI CM BOLD" width="60px">
                <label htmlFor="team">{msg(lang, 'colTeam')}</label>
              </td>
              <td className="" width="180px">
                <AutoSubmitSelect
                  name="team"
                  className="NWAKU  WMAX"
                  id="team"
                  defaultValue={filters.selected.team}
                >
                  <option value="" />
                  {filters.teams.map((team) => (
                    <option key={team.value} value={team.value}>
                      {pickText(lang, team.name)}
                    </option>
                  ))}
                </AutoSubmitSelect>
              </td>
              <td className="NINI" width="10px" />
              <td className="NINI CM BOLD" width="60px">
                <label htmlFor="field">{msg(lang, 'colField')}</label>
              </td>
              <td className="" width="100px">
                <AutoSubmitSelect
                  name="field"
                  className="NWAKU  WMAX"
                  id="field"
                  defaultValue={filters.selected.field}
                >
                  <option value="" />
                  {filters.fields.map((field) => (
                    <option key={field} value={field}>
                      {field}
                    </option>
                  ))}
                </AutoSubmitSelect>
              </td>
              <td className="NINI" width="10px" />
              <td className="RM" width="120px" style={{ height: '33px' }}>
                <RefreshButtons
                  reloadHref={scheduleHref(target.controller, target.id, lang, filters.selected)}
                  waitMessage={msg(lang, 'refreshWait')}
                  okAlt={msg(lang, 'refreshOkAlt')}
                  ngAlt={msg(lang, 'refreshNgAlt')}
                />
                <noscript>
                  <input type="submit" value={msg(lang, 'scheduleSearchSubmit')} />
                </noscript>
              </td>
            </tr>
          </tbody>
        </table>
      </form>
    </div>
  )
}

export function ScheduleView({ data, lang }: { data: ScheduleResponse; lang: Lang }) {
  const target = targetOf(data)
  return (
    <>
      <SearchForm data={data} lang={lang} target={target} />
      <Tbl>
        <TblHeaderRow columns={headerColumns(data, lang, target)} />
        {data.rows.map((row) => (
          <MatchRowLine
            key={row.matchId}
            row={row}
            lang={lang}
            variant={{ kind: 'schedule', division: row.division }}
          />
        ))}
      </Tbl>
    </>
  )
}
