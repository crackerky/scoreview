import type { DivisionKind, MatchRow, MatchSide, MatchStatus } from '@/api/common.types'
import {
  formatMonthDay,
  formatTimeCell,
  scheduleDivisionWidth,
  scheduleTeamWidth,
  trimToWidth,
} from '@/format/format.utils'
import { matchHref, teamHref } from '@/i18n/href'
import { type Lang, pickText } from '@/i18n/lang'
import { type MessageKey, msg, placeholderLabel } from '@/i18n/messages'
import { type TblColumn, TblHeaderRow } from './Tbl'

// 試合 1 行(tr.H24)。2 種類の列構成がある。
// - schedule: スケジュール(clone-spec B 5)。先頭にディビジョン列、名前を表示幅で切り詰め、区切りは "x"
// - list: グループの結果一覧とチーム対戦一覧(clone-spec C 2.1(c)、D 1.1)。区切りは全角 "Ｘ"
// 行の全セルに状態クラス(MC_MAE / MC_NOW / MC_END)が付き、ディビジョン列だけに種別クラスが付く。

const statusClass: Record<MatchStatus, string> = {
  scheduled: 'MC_MAE',
  // 推測: 試合中の行は参照サイトで未観測。CSS の定義(赤字)だけがある
  live: 'MC_NOW',
  finished: 'MC_END',
}

// clone-spec B 6: ディビジョンの区分で決まる背景。オープン等(none)はクラスなし
const divisionKindClass: Record<DivisionKind, string | null> = {
  men: 'MC_MEN',
  women: 'MC_WOM',
  mixed: 'MC_MIX',
  none: null,
}

export type MatchRowVariant =
  | { kind: 'list' }
  | { kind: 'schedule'; division: { name: { ja: string; en: string }; kind: DivisionKind } }

function TeamCell({
  side,
  lang,
  variant,
  className,
}: {
  side: MatchSide
  lang: Lang
  variant: MatchRowVariant
  className: string
}) {
  if (side.team !== null) {
    // clone-spec B 8: 英語版で英語名のないチームは、名前の代わりに決まり方の表記("[A1-Win]")を出す
    const name =
      lang === 'en' && side.team.name.en === '' ? side.labelEn : pickText(lang, side.team.name)
    return (
      <td className={className}>
        <span className="teamLink">
          <a href={teamHref(side.team.id, lang)}>
            {variant.kind === 'schedule' ? trimToWidth(name, scheduleTeamWidth) : name}
          </a>
        </span>
      </td>
    )
  }
  // clone-spec B 8: 未確定チームはリンクなしの素のテキスト
  return (
    <td className={className}>
      {side.placeholder === null
        ? ''
        : placeholderLabel(lang, side.placeholder.code, side.placeholder.outcome)}
    </td>
  )
}

function StatusContent({ row, lang }: { row: MatchRow; lang: Lang }) {
  if (row.status === 'scheduled') {
    return msg(lang, 'statusWaiting')
  }
  // 推測: 試合中は試合ページがあればリンクにし、文言は statusLive とする(参照サイトで未観測)
  const label: MessageKey = row.status === 'live' ? 'statusLive' : 'statusResult'
  if (!row.hasDetail) {
    return msg(lang, row.status === 'live' ? 'statusLive' : 'statusEnd')
  }
  return (
    <a className="Select_Link" href={matchHref(row.matchId, lang)}>
      {msg(lang, label)}
    </a>
  )
}

/** 試合 1 行。<tbody> の中に置く。 */
export function MatchRowLine({
  row,
  lang,
  variant,
}: {
  row: MatchRow
  lang: Lang
  variant: MatchRowVariant
}) {
  const state = statusClass[row.status]
  const cell = (align: 'LM' | 'CM' | 'RM') => `${state} ${align}`
  return (
    <tr className="H24">
      {variant.kind === 'schedule' && (
        // clone-spec B 6: 種別クラスが無いときも区切りの空白は残る('MC_END  CM' と空白 2 つ)
        <td className={`${state} ${divisionKindClass[variant.division.kind] ?? ''} CM`}>
          {trimToWidth(pickText(lang, variant.division.name), scheduleDivisionWidth)}
        </td>
      )}
      {variant.kind === 'list' && <td className={cell('LM')}>{row.code}</td>}
      <td className={cell('CM')}>{formatMonthDay(row.date)}</td>
      <td className={cell('CM')}>{formatTimeCell(row.time)}</td>
      <td className={cell('CM')}>{row.field ?? ''}</td>
      {variant.kind === 'schedule' && <td className={cell('CM')}>{row.code}</td>}
      <TeamCell side={row.left} lang={lang} variant={variant} className={cell('RM')} />
      <td className={cell('CM')}>{row.left.score ?? ''}</td>
      <td className={cell('CM')}>
        {msg(lang, variant.kind === 'schedule' ? 'scheduleScoreSeparator' : 'listScoreSeparator')}
      </td>
      <td className={cell('CM')}>{row.right.score ?? ''}</td>
      <TeamCell side={row.right} lang={lang} variant={variant} className={cell('LM')} />
      <td className={cell('CM')}>
        <StatusContent row={row} lang={lang} />
      </td>
    </tr>
  )
}

/** 結果一覧・チーム対戦一覧の見出しの列(clone-spec C 2.1(c)、D 1.1)。 */
export function matchListColumns(lang: Lang): TblColumn[] {
  return [
    { label: msg(lang, 'colCode'), width: '10%' },
    { label: msg(lang, 'colDate'), width: '7%' },
    { label: msg(lang, 'colTime'), width: '7%' },
    { label: msg(lang, 'colField'), width: '7%' },
    { label: msg(lang, 'colTeam'), width: '20%' },
    { label: msg(lang, 'colDash'), width: '4%' },
    { label: msg(lang, 'colVs'), width: '3%' },
    { label: msg(lang, 'colDash'), width: '4%' },
    { label: msg(lang, 'colTeam'), width: '20%' },
    { label: msg(lang, 'colStatus'), width: '18%' },
  ]
}

/** 結果一覧の表(見出し行 + 試合行)。div.TBL の中に置く <tr> 群を返す。 */
export function MatchListRows({ rows, lang }: { rows: MatchRow[]; lang: Lang }) {
  return (
    <>
      <TblHeaderRow columns={matchListColumns(lang)} />
      {rows.map((row) => (
        <MatchRowLine key={row.matchId} row={row} lang={lang} variant={{ kind: 'list' }} />
      ))}
    </>
  )
}
