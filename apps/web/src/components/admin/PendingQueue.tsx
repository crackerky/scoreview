'use client'

import { useEffect } from 'react'
import { precacheScorePages, usePendingCounts } from './DeviceSync'

// 担当試合の一覧に出す、この端末の未送信の記録(レビュー C3: 未送信がその試合を開くまで送られず、
// 一覧にも出なかった)。送るのは DeviceSync が行い、ここは件数を出すだけ。

/** 一覧の上に出す、端末全体の未送信の記録の案内。listed は一覧に出ている試合 ID */
export function PendingSummary({ listed }: { listed: number[] }) {
  const counts = usePendingCounts()
  if (counts.size === 0) {
    return null
  }
  const total = [...counts.values()].reduce((sum, count) => sum + count, 0)
  // 担当から外れた試合など、一覧に無い試合の未送信も見落とさないように出す
  const hidden = [...counts.keys()].filter((id) => !listed.includes(id))
  return (
    <p className="adm-pending" role="status">
      この端末に未送信の記録が {total} 件あります。つながると自動で送ります(ブラウザを閉じたり、
      ログアウトしたりしないでください)。
      {hidden.map((id) => (
        <a key={id} className="adm-pending-link" href={`/admin/matches/${id}/score`}>
          試合 {id} の記録画面
        </a>
      ))}
    </p>
  )
}

/** 一覧の試合の行に出す「この端末に未送信 n 件」 */
export function PendingBadge({ matchId }: { matchId: number }) {
  const count = usePendingCounts().get(matchId) ?? 0
  if (count === 0) {
    return null
  }
  return <span className="adm-pending-badge">この端末に未送信 {count} 件</span>
}

/** 担当試合の記録画面を、通信のあるうちに端末へ保存させる */
export function PrecacheScorePages({ paths }: { paths: string[] }) {
  const key = paths.join(' ')
  useEffect(() => {
    if (key !== '') {
      precacheScorePages(['/admin', ...key.split(' ')])
    }
  }, [key])
  return null
}
