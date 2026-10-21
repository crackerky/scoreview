import { notFound } from 'next/navigation'
import type { ReactNode } from 'react'
import { loadAdmin, requireUser } from '@/admin/session'
import { divisions, tournaments } from '@/api/admin'
import { formats } from '@/api/format'
import { AdminShell } from '@/components/admin/AdminParts'

// 本部の進行画面の手前。形を決める前に開くと api が 409 を返し、画面が英語の 500 になっていた
// (運営者レビュー H12)。形が無ければ進行画面(page.tsx)を描かずに、形を決める画面へ案内する。

export default async function ProgressLayout({
  children,
  params,
}: {
  children: ReactNode
  params: Promise<{ id: string }>
}) {
  const { id: raw } = await params
  if (!/^[0-9]+$/.test(raw)) {
    notFound()
  }
  const id = Number(raw)
  const { user, token } = await requireUser('any')
  // 形の有無は、記録係も読める試合一覧(形が無ければ null)で確かめる
  const formatted = await loadAdmin(() => formats.matchesIfFormatted(token, id))
  if (formatted !== null) {
    return children
  }
  const division = await loadAdmin(() => divisions.get(token, id))
  const tournament = await loadAdmin(() => tournaments.get(token, division.tournamentId))
  const crumbs = [
    { href: `/admin/tournaments/${tournament.id}`, label: tournament.name.ja },
    { href: `/admin/divisions/${id}`, label: division.name.ja },
  ]
  return (
    <AdminShell user={user} title={`${division.name.ja}: 本部`} crumbs={crumbs}>
      <h2>まず大会の形を決めましょう</h2>
      <p>
        本部の進行画面は、組と順位決定戦と時間割(大会の形)を作った後に使います。
        チームを登録して、形の候補から選ぶと、ここに試合の進み具合が出ます。
      </p>
      {user.role === 'admin' ? (
        <p className="adm-actions">
          <a className="adm-primary-link" href={`/admin/divisions/${id}/format`}>
            大会の形を決める
          </a>
        </p>
      ) : (
        <p className="adm-muted">大会の形は本部(管理者)が決めます。</p>
      )}
    </AdminShell>
  )
}
