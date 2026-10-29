import { divisionKindOptions } from '@/admin/labels.utils'
import { loadAdmin, requireUser } from '@/admin/session'
import { tournaments } from '@/api/admin'
import type { IdPageProps } from '@/app/_lib/pageProps'
import { createDivisionAction } from '@/app/admin/_actions/tournament'
import { readAdminPage } from '@/app/admin/_lib/page'
import { AdminShell, Field, Hidden, Select } from '@/components/admin/AdminParts'

// 部門を作る(大会ホームの入口から)。作ったら部門の画面へ進む
export default async function NewDivisionPage(props: IdPageProps) {
  const { id, notice } = await readAdminPage(props)
  const { user, token } = await requireUser('admin')
  const tournament = await loadAdmin(() => tournaments.get(token, id))
  const crumbs = [{ href: `/admin/tournaments/${id}`, label: tournament.name.ja }]
  return (
    <AdminShell
      user={user}
      title="部門を作る"
      notice={notice}
      crumbs={crumbs}
      primary={{ label: '作成', form: 'new-division' }}
    >
      <form id="new-division" action={createDivisionAction}>
        <Hidden
          values={{
            back: `/admin/tournaments/${id}/divisions/new`,
            tournamentId: id,
            scheduleEnabled: 'on',
            entryKind: 'team',
            statsKind: 'ultimate',
          }}
        />
        <div className="adm-grid">
          <Field label="名前(日本語)" name="nameJa" required />
          <Field label="名前(英語)" name="nameEn" />
          <Select label="日程の色" name="kind" options={divisionKindOptions} defaultValue="mixed" />
        </div>
        <details>
          <summary>詳しい設定</summary>
          <div className="adm-grid">
            <Field label="並び順" name="sortOrder" type="number" defaultValue={0} />
          </div>
        </details>
      </form>
    </AdminShell>
  )
}
