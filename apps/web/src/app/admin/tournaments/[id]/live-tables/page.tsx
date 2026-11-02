import { loadAdmin, requireUser } from '@/admin/session'
import '@/styles/live-tables.css'
import { tournaments } from '@/api/admin'
import type { IdPageProps } from '@/app/_lib/pageProps'
import { saveLiveTablesAction } from '@/app/admin/_actions/tournament'
import { readAdminPage } from '@/app/admin/_lib/page'
import { AdminShell } from '@/components/admin/AdminParts'
import { LiveTablesEditor } from '@/components/admin/LiveTablesEditor'
import { loadLiveData } from '@/live/liveData'

// 観戦アプリの表(Claude Design「Score View Admin」の 2a)。管理者だけ。左に観戦アプリの「表で見る」の
// 見本を実物のデータで出し、押した部分の見せ方を右で選ぶ。保存すると観戦アプリに出る。
export default async function LiveTablesPage(props: IdPageProps) {
  const { id, notice } = await readAdminPage(props)
  const { user, token } = await requireUser('admin')
  const [tournament, config] = await loadAdmin(() =>
    Promise.all([tournaments.get(token, id), tournaments.liveTables(token, id)]),
  )
  const data = await loadLiveData(id, tournament.startDate)
  const formId = 'live-tables-form'
  return (
    <AdminShell
      user={user}
      title="観戦アプリの表"
      kicker="表を押して、見え方を変えます"
      notice={notice}
      crumbs={[{ href: `/admin/tournaments/${id}`, label: tournament.name.ja }]}
      primary={{ label: '保存する', form: formId }}
    >
      <form id={formId} action={saveLiveTablesAction}>
        <input type="hidden" name="back" value={`/admin/tournaments/${id}/live-tables`} />
        <input type="hidden" name="id" value={id} />
        <LiveTablesEditor data={data} saved={config} />
      </form>
    </AdminShell>
  )
}
