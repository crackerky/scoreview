import type { BracketCell } from '@/api/common.types'

// グループ成績(clone-spec C 2)の表のクラス名と幅を、参照サイトの出力と同じ文字列で作る。

/** PHP が float を echo したときの有効桁数(precision の既定値 14)。 */
const phpFloatPrecision = 14

/**
 * 対戦マトリクスのチーム列の幅 "{78/N}%"(clone-spec C 2.1(a): 3 チーム 26%、4 チーム 19.5%、
 * 5 チーム 15.6%)。割り切れない N は PHP の float 表示(有効 14 桁)に合わせる(推測: 6 チーム以上は未観測)。
 */
export function matrixColumnWidth(teamCount: number): string {
  const percent = Number((78 / teamCount).toPrecision(phpFloatPrecision))
  return `${percent}%`
}

// clone-spec C 2.3: 罫線クラスは Left, Right, Top, Btm の順に並ぶ
const borderClasses: ReadonlyArray<readonly [keyof BracketCell['borders'], string]> = [
  ['left', 'BdLeft'],
  ['right', 'BdRight'],
  ['top', 'BdTop'],
  ['bottom', 'BdBtm'],
]

// clone-spec C 2.3: セル種別の後ろの空白の数は種別ごとに決まっている(参照サイトの全 HTML で確認。
// 'win  '、'txt_and   ' など)
const kindClass: Record<Exclude<BracketCell['kind'], ''>, string> = {
  team: 'team ',
  win: 'win  ',
  lose: 'lose ',
  game: 'game ',
  txt_yen: 'txt_yen   ',
  txt_sharp: 'txt_sharp ',
  txt_and: 'txt_and   ',
  // rank は旧い大会(2017〜2018)の格子で観測(空白 1 つ)。txt_left / txt_right は CSS 定義のみで
  // 実例がないので、空白 1 つは推測
  rank: 'rank ',
  txt_left: 'txt_left ',
  txt_right: 'txt_right ',
}

/** ブラケットの td の class。例 "WDiv4 BdRight BdTop game "、空セルは "WDiv4 "。 */
export function bracketCellClass(cell: BracketCell, columns: number): string {
  const borders = borderClasses
    .filter(([side]) => cell.borders[side])
    .map(([, name]) => `${name} `)
    .join('')
  const kind = cell.kind === '' ? '' : kindClass[cell.kind]
  return `WDiv${columns} ${borders}${kind}`
}
