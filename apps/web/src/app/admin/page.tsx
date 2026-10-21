import { tournamentStatusOptions } from '@/admin/labels.utils'
import { loadAdmin, requireUser } from '@/admin/session'
import { tournaments } from '@/api/admin'
import { createTournamentAction } from '@/app/admin/_actions/tournament'
import type { SearchProps } from '@/app/admin/_lib/page'
import { AdminShell, Field, Hidden, readNotice } from '@/components/admin/AdminParts'

// 管理画面のトップ。大会の一覧(記録係は担当大会だけ)と、管理者なら大会の作成。
export default async function AdminTopPage({ searchParams }: SearchProps) {
  const notice = readNotice(await searchParams)
  const { user, token } = await requireUser('any')
  const list = await loadAdmin(() => tournaments.list(token))
  const statusLabel = (value: string) =>
    tournamentStatusOptions.find((option) => option.value === value)?.label ?? value
  return (
    <AdminShell user={user} title="大会一覧" notice={notice}>
      {list.tournaments.length === 0 ? (
        <p className="adm-muted">
          {user.role === 'admin' ? '大会がありません。' : '担当の大会がありません。'}
        </p>
      ) : (
        <ul className="adm-list">
          {list.tournaments.map((tournament) => (
            <li key={tournament.id}>
              <a href={`/admin/tournaments/${tournament.id}`}>
                {tournament.name.ja}
                <span className="adm-muted">
                  ({statusLabel(tournament.status)}
                  {tournament.startDate ? ` ${tournament.startDate}` : ''}
                  {tournament.listed ? '' : ' 一覧に非掲載'})
                </span>
              </a>
            </li>
          ))}
        </ul>
      )}
      {user.role === 'admin' ? (
        <>
          <h2>大会を作る</h2>
          <form action={createTournamentAction}>
            <Hidden
              values={{
                back: '/admin',
                status: 'entry_open',
                teamListLabel: 'teams',
                groupLabel: 'groups',
                listed: 'on',
                showSchedule: 'on',
              }}
            />
            <div className="adm-grid">
              <Field label="大会名(日本語)" name="nameJa" required />
              <Field label="大会名(英語)" name="nameEn" />
              <Field label="開催日" name="startDate" type="date" />
            </div>
            <button type="submit">作成</button>
          </form>
        </>
      ) : null}
    </AdminShell>
  )
}
