import type { MatchRow, MatchSide, MatchStatus } from '@/api/common.types'
import { type Lang, pickText } from '@/i18n/lang'
import { msg, placeholderLabel } from '@/i18n/messages'

// 試合の片側の表記と、通常以外の結果の文言(本サイト独自。docs/requirements.md「意図的な差分」)。

/**
 * 試合の片側の名前。英語版で英語名のないチームは決まり方の表記("[A1-Win]")、未確定なら
 * "[A1-勝ち]"(clone-spec B 8)。どちらも無ければ空。
 */
export function sideLabel(side: MatchSide, lang: Lang): string {
  if (side.team !== null) {
    return lang === 'en' && side.team.name.en === '' ? side.labelEn : pickText(lang, side.team.name)
  }
  return side.placeholder === null
    ? ''
    : placeholderLabel(lang, side.placeholder.code, side.placeholder.outcome)
}

type ResultRow = Pick<MatchRow, 'resultType' | 'winnerSide' | 'left' | 'right'>

/**
 * 通常以外の結果の文言。不戦勝と不成立は勝ったチームの名前を入れる(「BREEZE の不戦勝」
 * 「Forfeit win: BREEZE」)。通常の結果は null。nameOf は勝ったチームの表記(表の幅で切り詰めるなど)。
 */
export function resultLabel(
  row: ResultRow,
  lang: Lang,
  nameOf: (side: MatchSide) => string = (side) => sideLabel(side, lang),
): string | null {
  const winner =
    row.winnerSide === null ? '' : nameOf(row.winnerSide === 'left' ? row.left : row.right)
  switch (row.resultType) {
    case 'normal':
      return null
    case 'forfeit':
      if (winner === '') {
        return msg(lang, 'resultForfeit')
      }
      return lang === 'en' ? `Forfeit win: ${winner}` : `${winner} の不戦勝`
    case 'no_contest':
      if (winner === '') {
        return msg(lang, 'resultNoContest')
      }
      return lang === 'en' ? `No contest (${winner} wins)` : `不成立(${winner} の勝ち)`
    case 'double_forfeit':
      return msg(lang, 'resultDoubleForfeit')
    case 'abandoned':
      return msg(lang, 'resultAbandoned')
    case 'unplayed':
      return msg(lang, 'resultUnplayed')
  }
}

/**
 * 対戦表とブラケットの試合セルの補足(title と読み上げ用の文字)。試合中(セルの状態)は「試合中」、
 * 通常以外の結果は resultLabel(row は結果一覧の行)。どちらでもなければ null(色とスコアで足りる)。
 */
export function matchCellNote(
  status: MatchStatus | null,
  row: MatchRow | undefined,
  lang: Lang,
): string | null {
  if (status === 'live') {
    return msg(lang, 'statusLive')
  }
  return row === undefined ? null : resultLabel(row, lang)
}
