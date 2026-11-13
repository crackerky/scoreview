'use client'

import { type ReactNode, useEffect, useState } from 'react'
import { jstToday } from '@/admin/tournamentList.utils'
import { initialDayIndex } from '@/score/score.utils'
import { LiveDot } from './LiveDot'

// 日ごとの時間割の切り替え。中身(panels)はサーバーで描いたものを、選んだ日だけ見せる。
// 最初はサーバーが決めた日(試合中の日)を出し、開いたあとで見ている人の今日に合わせる
// (サーバーの描画は使い回すので、今日はブラウザで決める)。

export type DayTab = { date: string | null; label: string; live: boolean }

export function DaySwitch({ days, panels }: { days: DayTab[]; panels: ReactNode[] }) {
  // 見ている人が選ぶまでは今日(無ければ試合中の日)に合わせる。描き直しても選んだ日は保つ
  const [chosen, setChosen] = useState<number | null>(null)
  const [today, setToday] = useState('')
  useEffect(() => {
    setToday(jstToday(new Date()))
  }, [])
  const selected = chosen ?? initialDayIndex(days, today)
  return (
    <div className="sc-days">
      {days.length > 1 ? (
        <div className="sc-day-tabs" role="tablist" aria-label="日付">
          {days.map((day, index) => (
            <button
              key={day.date ?? 'undated'}
              type="button"
              role="tab"
              aria-selected={index === selected}
              onClick={() => setChosen(index)}
            >
              {day.label}
              {day.live ? <LiveDot /> : null}
            </button>
          ))}
        </div>
      ) : null}
      {panels.map((panel, index) => (
        <div key={days[index]?.date ?? 'undated'} role="tabpanel" hidden={index !== selected}>
          {panel}
        </div>
      ))}
    </div>
  )
}
