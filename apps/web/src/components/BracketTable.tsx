import type { BracketCell } from '@/api/common.types'
import type { GroupResponse } from '@/api/public.types'
import { matchHref, teamHref } from '@/i18n/href'
import { type Lang, pickText } from '@/i18n/lang'
import { bracketCellClass } from './group.utils'

// clone-spec C 2.3: トーナメント表は 1 枚の table のセル罫線で描く(div.TN_TMPL)。
// 参照サイトは直前に <!-- GeneratePtn:T8A --> と <!-- PtnName :... --> の HTML コメントを出すが、
// React はコメントを出力できないので再現しない。

const nbsp = ' '

type Bracket = NonNullable<GroupResponse['bracket']>

function CellContent({ cell, lang }: { cell: BracketCell; lang: Lang }) {
  if (cell.team !== null) {
    return (
      <span className="teamimg_tn">
        <a href={teamHref(cell.team.id, lang)}>{pickText(lang, cell.team.name)}</a>
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
      </a>
    )
  }
  // 空セルは &nbsp;(参照サイトの原文ママ)
  return cell.text === '' ? nbsp : cell.text
}

export function BracketTable({ bracket, lang }: { bracket: Bracket; lang: Lang }) {
  return (
    <div className="TN_TMPL">
      <table>
        <tbody>
          {bracket.rows.map((row, rowIndex) => (
            // biome-ignore lint/suspicious/noArrayIndexKey: テンプレートの固定グリッドで、行は並べ替えない
            <tr key={rowIndex}>
              {row.map((cell, cellIndex) => (
                // biome-ignore lint/suspicious/noArrayIndexKey: 同上(列も固定)
                <td key={cellIndex} className={bracketCellClass(cell, bracket.columns)}>
                  <CellContent cell={cell} lang={lang} />
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
