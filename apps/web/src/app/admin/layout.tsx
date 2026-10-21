import type { ReactNode } from 'react'
import { DeviceSync } from '@/components/admin/DeviceSync'
import '@/styles/admin.css'

// 管理画面は検索エンジンに載せない
export const metadata = { robots: { index: false, follow: false } }

// DeviceSync は記録係の未送信の記録を送り、記録画面を通信なしでも開けるようにする(どの管理画面からでも)
export default function AdminLayout({ children }: { children: ReactNode }) {
  return (
    <>
      <DeviceSync />
      {children}
    </>
  )
}
