import { BracketTree } from '@/components/tables/BracketTree'
import { requireScoreBoard } from '@/score/score.load'
import { scoreDivisions } from '@/score/score.utils'

// トーナメント。部門ごとに、大会の形から描いた勝ち上がりの木を出す。

export default async function BracketPage({ params }: { params: Promise<{ id: string }> }) {
  const { board, data } = await requireScoreBoard((await params).id)
  const divisions = scoreDivisions(board, data).filter((division) => division.brackets.length > 0)
  return (
    <>
      <title>{`${board.tournament.name.ja} - トーナメント`}</title>
      {divisions.length === 0 ? (
        <p className="sc-empty">トーナメントはまだ決まっていません。</p>
      ) : null}
      {divisions.map((division) => (
        <section key={division.id} id={`division-${division.id}`} className="sc-section">
          <h2 className="sc-heading">{division.name}</h2>
          {division.brackets.map((view) => (
            <div key={view.name} className="sc-bracket">
              <h3>{view.name}</h3>
              {view.trees.map((tree) => (
                <div key={tree.root.match.id} className="sc-paper">
                  <BracketTree tree={tree} />
                </div>
              ))}
            </div>
          ))}
        </section>
      ))}
      {divisions.length > 0 ? (
        <p className="sc-legend">
          赤い線が勝ち上がり。線の上の数字はその試合の得点です。まだ決まっていない枠は点線で出しています。
        </p>
      ) : null}
    </>
  )
}
