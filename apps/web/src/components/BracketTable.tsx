import type { BracketCell, MatchRow } from '@/api/common.types'
import type { GroupResponse } from '@/api/public.types'
import { matchHref, teamHref } from '@/i18n/href'
import { type Lang, pickText } from '@/i18n/lang'
import { bracketCellClass } from './group.utils'
import { matchCellNote } from './matchResult.utils'
import { ScreenReaderNote } from './ScreenReaderNote'

// clone-spec C 2.3: トーナメント表は 1 枚の table のセル罫線で描く(div.TN_TMPL)。
// 参照サイトは直前に <!-- GeneratePtn:T8A --> と <!-- PtnName :... --> の HTML コメントを出すが、
// React はコメントを出力できないので再現しない。

const nbsp = ' '

type Bracket = NonNullable<GroupResponse['bracket']>

// 本サイト独自: 勝ち上がり(win、赤)と敗退(lose、青)のセルは色だけで区別しているので、読み上げ用に補う
const outcomeNotes: Record<Lang, { win: string; lose: string }> = {
  jp: { win: '勝ち', lose: '負け' },
  en: { win: 'won', lose: 'lost' },
}

function CellContent({
  cell,
  note,
  lang,
}: {
  cell: BracketCell
  /** 読み上げ用の補足(試合セルの試合中や不戦勝、勝ち上がり・敗退のセルの勝ち負け) */
  note: string | null
  lang: Lang
}) {
  if (cell.team !== null) {
    return (
      <span className="teamimg_tn">
        <a href={teamHref(cell.team.id, lang)}>
          {pickText(lang, cell.team.name)}
          <ScreenReaderNote text={note} />
        </a>
      </span>
    )
  }
  // 推測: 未試合の試合セルは参照サイトで未観測。スコアがまだ無いのでリンクにしない
  if (cell.kind === 'game' && cell.matchId !== null && cell.status !== 'scheduled') {
    return (
      <a
        className={cell.status === 'live' ? 'tnmt_play_now' : 'tnmt_game_link'}
        href={matchHref(cell.matchId, lang)}
      >
        {cell.text}
        <ScreenReaderNote text={note} />
      </a>
    )
  }
  // 空セルは &nbsp;(参照サイトの原文ママ)
  return (
    <>
      {cell.text === '' ? nbsp : cell.text}
      <ScreenReaderNote text={note} />
    </>
  )
}

export function BracketTable({
  bracket,
  matches,
  lang,
}: {
  bracket: Bracket
  /** グループの結果一覧の行(試合セルの補足を引く)。管理画面のプレビューなど試合が無ければ省く */
  matches?: MatchRow[]
  lang: Lang
}) {
  const rows = new Map((matches ?? []).map((match) => [match.matchId, match]))
  // 本サイト独自: 試合中と不戦勝などは色とスコアだけでは分からないので、title と読み上げ用の文字で補う
  const noteOf = (cell: BracketCell) =>
    cell.kind === 'game' && cell.matchId !== null
      ? matchCellNote(cell.status, rows.get(cell.matchId), lang)
      : null
  return (
    <div className="TN_TMPL">
      <table>
        <tbody>
          {bracket.rows.map((row, rowIndex) => (
            // biome-ignore lint/suspicious/noArrayIndexKey: テンプレートの固定グリッドで、行は並べ替えない
            <tr key={rowIndex}>
              {row.map((cell, cellIndex) => {
                const note = noteOf(cell)
                const outcome =
                  cell.kind === 'win' || cell.kind === 'lose' ? outcomeNotes[lang][cell.kind] : null
                return (
                  <td
                    // biome-ignore lint/suspicious/noArrayIndexKey: 同上(列も固定)
                    key={cellIndex}
                    className={bracketCellClass(cell, bracket.columns)}
                    title={note ?? undefined}
                  >
                    <CellContent cell={cell} note={note ?? outcome} lang={lang} />
                  </td>
                )
              })}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
