import type { ReactNode } from 'react'
import '@/styles/admin.css'

// 管理画面は検索エンジンに載せない
export const metadata = { robots: { index: false, follow: false } }

export default function AdminLayout({ children }: { children: ReactNode }) {
  return children
}
