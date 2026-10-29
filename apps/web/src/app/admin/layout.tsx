import { Barlow_Condensed, LINE_Seed_JP } from 'next/font/google'
import type { ReactNode } from 'react'
import { DeviceSync } from '@/components/admin/DeviceSync'
import { SubmitFeedback } from '@/components/admin/SubmitFeedback'
import '@/styles/admin.css'

// 管理画面は検索エンジンに載せない
export const metadata = { robots: { index: false, follow: false } }

// 管理画面の文字(Claude Design「Score View Admin」)。本文は LINE Seed JP、点数と時刻は Barlow Condensed。
// 日本語のフォントは大きいので先読みしない(表示は代わりのフォントから入れ替わる)
const bodyFont = LINE_Seed_JP({
  weight: ['400', '700', '800'],
  display: 'swap',
  preload: false,
  variable: '--adm-font-body',
})
const numberFont = Barlow_Condensed({
  weight: ['600', '700'],
  subsets: ['latin'],
  display: 'swap',
  variable: '--adm-font-number',
})

// DeviceSync は記録係の未送信の記録を送り、記録画面を通信なしでも開けるようにする(どの管理画面からでも)。
// SubmitFeedback はフォームを送ってから応答が来るまでボタンに「処理中」を出す
export default function AdminLayout({ children }: { children: ReactNode }) {
  return (
    <div className={`adm-fonts ${bodyFont.variable} ${numberFont.variable}`}>
      <DeviceSync />
      <SubmitFeedback />
      {children}
    </div>
  )
}
