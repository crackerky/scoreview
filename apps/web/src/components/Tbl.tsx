import { type CSSProperties, Fragment, type ReactNode } from 'react'

// clone-spec A 2.2 / C 0: div.TBL > table の表。見出し行は td.MD1(橙地に白の太字)。
// 参照サイトは <table> 直下に <tr> を置くが、ブラウザが補う <tbody> を明示する。

type TblProps = {
  children: ReactNode
  /** div.TBL に付ける追加のクラス(TG_EventList など) */
  className?: string
  /** div.TBL のインライン style(大会トップの margin-left: 80px など) */
  style?: CSSProperties
  /** table のインライン style(width: 70% など) */
  tableStyle?: CSSProperties
  /** table の id(試合ページの #events など) */
  tableId?: string
}

export function Tbl({ children, className, style, tableStyle, tableId }: TblProps) {
  return (
    <div className={className ? `TBL ${className}` : 'TBL'} style={style}>
      <table id={tableId} style={tableStyle}>
        <tbody>{children}</tbody>
      </table>
    </div>
  )
}

/** "\n" を <br /> にして出す(英語の表見出し "Goals\nfor" など)。 */
export function MultilineText({ text }: { text: string }) {
  const lines = text.split('\n')
  return (
    <>
      {lines.map((line, index) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: 行の並びは固定
        <Fragment key={index}>
          {index > 0 && <br />}
          {line}
        </Fragment>
      ))}
    </>
  )
}

export type TblColumn = {
  label: ReactNode
  /** td の width 属性("13%" など) */
  width?: string
  /** 揃え(LM / CM / RM)。既定は CM */
  align?: 'LM' | 'CM' | 'RM'
}

/** 見出し行 <tr> と td.MD1 の列。label が文字列なら "\n" を改行にする。 */
export function TblHeaderRow({
  columns,
  cellClass = 'MD1',
}: {
  columns: TblColumn[]
  /** 見出しセルのクラス(リーグの対戦表は RM_MD1) */
  cellClass?: string
}) {
  return (
    <tr>
      {columns.map((column, index) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: 列の並びは固定で、見出しが重複する("-" など)
        <td key={index} className={`${cellClass} ${column.align ?? 'CM'}`} width={column.width}>
          {typeof column.label === 'string' ? <MultilineText text={column.label} /> : column.label}
        </td>
      ))}
    </tr>
  )
}
