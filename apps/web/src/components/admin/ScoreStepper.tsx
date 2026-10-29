'use client'

import { useState } from 'react'

// 本部の結果の入力の点数(Claude Design「Score View Admin」の結果の入力)。＋と−で点を合わせ、値は
// 同じ名前の hidden 欄でフォームに載せる。数字を打つより押し間違いが少なく、片手で操作できる。

/** 1 試合の点の上限(api の検証と同じ 0〜999) */
const maxScore = 999

export function ScoreStepper({
  name,
  team,
  initial,
  side,
}: {
  name: string
  team: string
  initial: number
  side: 'left' | 'right'
}) {
  const [value, setValue] = useState(initial)
  return (
    <div className={`adm-stepper ${side}`}>
      <span className="adm-stepper-bar" aria-hidden="true" />
      <span className="adm-stepper-team">{team}</span>
      <div className="adm-stepper-controls">
        <button
          type="button"
          className="adm-stepper-button"
          aria-label={`${team}の点を 1 減らす`}
          onClick={() => setValue((current) => Math.max(0, current - 1))}
        >
          −
        </button>
        <output className="adm-num adm-stepper-value" aria-live="polite">
          {value}
        </output>
        <button
          type="button"
          className="adm-stepper-button"
          aria-label={`${team}の点を 1 増やす`}
          onClick={() => setValue((current) => Math.min(maxScore, current + 1))}
        >
          ＋
        </button>
      </div>
      <input type="hidden" name={name} value={value} />
    </div>
  )
}
