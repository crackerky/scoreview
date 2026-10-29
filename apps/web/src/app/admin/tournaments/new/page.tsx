import { loadAdmin, requireUser } from '@/admin/session'
import { japaneseDate, jstToday } from '@/admin/tournamentList.utils'
import { tournaments } from '@/api/admin'
import type { SearchProps } from '@/app/admin/_lib/page'
import { AdminShell, readNotice } from '@/components/admin/AdminParts'
import { NewTournamentWizard, type WizardSource } from '@/components/admin/NewTournamentWizard'

// 大会を作る(大会一覧の「大会を作る」から)。前回の大会をコピーするのが基本の 3 ステップ
export default async function NewTournamentPage({ searchParams }: SearchProps) {
  const notice = readNotice(await searchParams)
  const { user, token } = await requireUser('admin')
  const list = await loadAdmin(() => tournaments.list(token))
  // コピー元は開催日の新しい順(前回の大会が先頭)
  const sources: WizardSource[] = [...list.tournaments]
    .sort((a, b) => (b.startDate ?? '').localeCompare(a.startDate ?? '') || b.id - a.id)
    .map((tournament) => ({
      id: tournament.id,
      name: tournament.name.ja,
      meta: tournament.startDate ? japaneseDate(tournament.startDate) : '開催日未定',
    }))
  return (
    <AdminShell user={user} title="大会を作る" notice={notice}>
      <NewTournamentWizard sources={sources} defaultDate={jstToday(new Date())} />
    </AdminShell>
  )
}
