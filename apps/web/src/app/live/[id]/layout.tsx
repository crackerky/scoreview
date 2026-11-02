import type { ReactNode } from 'react'
import { liveFontClass } from '@/live/fonts'
import '@/styles/live.css'
import '@/styles/live-tables.css'

export const viewport = { themeColor: '#0a0d14' }

export default function LiveLayout({ children }: { children: ReactNode }) {
  return <div className={liveFontClass}>{children}</div>
}
