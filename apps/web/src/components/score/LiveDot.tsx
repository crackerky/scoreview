// 試合中の印。点は飾りで、読み上げには「試合中」と伝える。

export function LiveDot() {
  return (
    <>
      <span className="sc-dot" aria-hidden="true" />
      <span className="sc-sr">試合中</span>
    </>
  )
}
