// clone-spec D 2.3: 得点経過は新しい順(最上行が試合終了、最下行が試合開始ｽﾛｰｵﾌ)。
// api は古い順で返すので web で並べ替える(scoreview-design 5)。
// 経過時間は分刻みの入力が多く同時刻が並ぶ。同時刻は累計点の大きい方(= 後の得点)を上にし、
// それでも並ぶとき(得点とタイムアウト、最終得点と試合終了)は api の順の逆にする。

type TimedEvent = { elapsedSec: number; leftScore: number; rightScore: number }

/** events(古い順)を新しい順に並べ替えた新しい配列を返す。 */
export function eventsNewestFirst<T extends TimedEvent>(events: readonly T[]): T[] {
  return events
    .map((event, index) => ({ event, index }))
    .sort(
      (a, b) =>
        b.event.elapsedSec - a.event.elapsedSec ||
        b.event.leftScore + b.event.rightScore - (a.event.leftScore + a.event.rightScore) ||
        b.index - a.index,
    )
    .map(({ event }) => event)
}
