// ギャラリーのライトボックスの純粋な計算。参照サイトの colorbox(v1.3.20.1)の既定値に合わせる。

/** 前後へ移動した位置。colorbox の既定 loop: true と同じく端で反対側へ回る。 */
export function stepIndex(current: number, delta: -1 | 1, total: number): number {
  return (current + delta + total) % total
}

/** colorbox の既定文言 "image {current} of {total}"(lang=jp でも英語のまま。clone-spec A 6)。 */
export function currentLabel(index: number, total: number): string {
  return `image ${index + 1} of ${total}`
}
