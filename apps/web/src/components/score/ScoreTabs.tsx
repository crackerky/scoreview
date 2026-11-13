'use client'

import { usePathname } from 'next/navigation'
import { scoreTabs, tabOf } from '@/score/score.paths'

// 3 つの入口。いまの画面がどの入口の中かは URL から決める(組と試合の画面はスコアの中)。

export function ScoreTabs({ tournamentId }: { tournamentId: number }) {
  const current = tabOf(usePathname(), tournamentId)
  return (
    <nav className="sc-tabs" aria-label="大会の中の画面">
      {scoreTabs.map((tab) => (
        <a
          key={tab.key}
          href={tab.path(tournamentId)}
          aria-current={tab.key === current ? 'page' : undefined}
        >
          {tab.label}
        </a>
      ))}
    </nav>
  )
}
