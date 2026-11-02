// 色だけで示している情報(試合中、勝ち負けなど)を、画面には出さずに読み上げさせる補足。見た目を
// 隠す class(sv_sr)は src/styles/bracket-table.css にある。

/** "(試合中)" のように括弧で囲んだ文字を、視覚的に隠して出す。text が null か空なら何も出さない。 */
export function ScreenReaderNote({ text }: { text: string | null }) {
  if (text === null || text === '') {
    return null
  }
  return <span className="sv_sr">{`(${text})`}</span>
}
