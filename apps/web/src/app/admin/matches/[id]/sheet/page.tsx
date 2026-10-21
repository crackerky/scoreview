import { loadAdmin, requireUser } from '@/admin/session'
import { matches } from '@/api/admin'
import type { IdPageProps } from '@/app/_lib/pageProps'
import { readAdminPage } from '@/app/admin/_lib/page'
import { PrintButton } from '@/components/admin/PrintButton'
import { ScoreSheet } from '@/components/admin/ScoreSheet'
import '@/styles/print-sheet.css'

// 試合 1 つのスコアシート(A4 縦)。印刷は無料の機能として誰でも使える(有料の印刷への不満。S1 B12)。
export default async function MatchSheetPage(props: IdPageProps) {
  const { id } = await readAdminPage(props)
  const { token } = await requireUser('any')
  const detail = await loadAdmin(() => matches.detail(token, id))
  return (
    <div className="pr">
      <title>{`${detail.match.code || '試合'} スコアシート`}</title>
      <div className="pr-toolbar">
        <a href={`/admin/matches/${id}`}>← 試合に戻る</a>
        <PrintButton />
        <span>A4 縦で印刷します。</span>
      </div>
      <ScoreSheet detail={detail} />
    </div>
  )
}
