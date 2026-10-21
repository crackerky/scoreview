// 色だけで示している情報(試合中、勝ち負け、性別など)を、画面には出さずに読み上げさせる補足
// (本サイト独自。docs/requirements.md「意図的な差分」)。見た目は変えないので、参照サイトとの比較から
// 外す印 data-scoreview-addition を付ける(apps/api/internal/parity が要素ごと除く)。

/** "(試合中)" のように括弧で囲んだ文字を、視覚的に隠して出す。text が null か空なら何も出さない。 */
export function ScreenReaderNote({ text }: { text: string | null }) {
  if (text === null || text === '') {
    return null
  }
  return <span className="sv_sr" data-scoreview-addition="">{`(${text})`}</span>
}
