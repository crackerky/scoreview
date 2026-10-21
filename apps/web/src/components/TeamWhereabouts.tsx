import type { MatchRow, MatchSide } from '@/api/common.types'
import type { TeamResponse } from '@/api/public.types'
import { formatMonthDay, formatTime } from '@/format/format.utils'
import { groupHref } from '@/i18n/href'
import { type Lang, pickText } from '@/i18n/lang'
import { msg, placeholderLabel } from '@/i18n/messages'
import { groupLinkLabel } from './navi.utils'

// 本サイト独自の追記(docs/requirements.md「意図的な差分」)。チーム名の下に、次の試合(日時、コート、
// 相手、試合コード)と所属グループへのリンクを小さく出す。参照サイトのチームページは対戦一覧から
// 次の試合を探し、ナビからグループを探す必要がある(観戦者レビュー H2)。
// data-scoreview-addition は比較ツール(apps/api/internal/parity)が比べない印。

/** 相手の表記。英語版で英語名が空なら決まり方の表記、未確定なら "[A1-勝ち]"(clone-spec B 8 と同じ)。 */
function sideLabel(side: MatchSide, lang: Lang): string {
  if (side.team !== null) {
    return lang === 'en' && side.team.name.en === '' ? side.labelEn : pickText(lang, side.team.name)
  }
  return side.placeholder === null
    ? ''
    : placeholderLabel(lang, side.placeholder.code, side.placeholder.outcome)
}

/** "03/16 13:20 コート 13 vs Blooms(F7)"。日時やコートが未定なら、その部分を省く。 */
export function nextMatchText(row: MatchRow, teamId: number, lang: Lang): string {
  const opponent = sideLabel(row.left.team?.id === teamId ? row.right : row.left, lang)
  const parts = [
    formatMonthDay(row.date),
    formatTime(row.time),
    row.field === null ? '' : `${msg(lang, 'noteField')} ${row.field}`,
    opponent === '' ? '' : `vs ${opponent}`,
  ].filter((part) => part !== '')
  const code = row.code === '' ? '' : lang === 'en' ? ` (${row.code})` : `(${row.code})`
  return `${parts.join(' ')}${code}`
}

export function TeamWhereabouts({ data, lang }: { data: TeamResponse; lang: Lang }) {
  const next = data.matches.find((row) => row.matchId === data.nextMatchId) ?? null
  const group = data.currentGroup
  if (next === null && group === null) {
    return null
  }
  return (
    <div className="sv_team_notes" data-scoreview-addition="">
      {next !== null && (
        <p>
          {`${msg(lang, next.status === 'live' ? 'noteLiveMatch' : 'noteNextMatch')}: `}
          {nextMatchText(next, data.team.id, lang)}
        </p>
      )}
      {group !== null && (
        <p>
          {`${msg(lang, 'noteGroup')}: `}
          {group.linked ? (
            <a href={groupHref(group.id, lang)}>
              {groupLinkLabel(pickText(lang, group.name), groupHref(group.id, lang))}
            </a>
          ) : (
            pickText(lang, group.name)
          )}
        </p>
      )}
    </div>
  )
}
