import { roleOptions } from '@/admin/labels.utils'
import { loadAdmin, requireUser } from '@/admin/session'
import { tournaments, users } from '@/api/admin'
import type { IdPageProps } from '@/app/_lib/pageProps'
import {
  deleteUserAction,
  putUserTournamentsAction,
  updateUserAction,
} from '@/app/admin/_actions/user'
import { readAdminPage } from '@/app/admin/_lib/page'
import { AdminShell, Check, DeleteForm, Field, Hidden, Select } from '@/components/admin/AdminParts'

export default async function AdminUserPage(props: IdPageProps) {
  const { id, notice } = await readAdminPage(props)
  const { user, token } = await requireUser('admin')
  const [target, tournamentList] = await loadAdmin(() =>
    Promise.all([users.get(token, id), tournaments.list(token)]),
  )
  const back = `/admin/users/${id}`
  const assigned = new Set(target.tournamentIds)
  return (
    <AdminShell
      user={user}
      title={target.displayName || target.email}
      notice={notice}
      crumbs={[{ href: '/admin/users', label: 'ユーザー' }]}
    >
      <form action={updateUserAction}>
        <Hidden values={{ back, id }} />
        <div className="adm-grid">
          <Field
            label="メールアドレス"
            name="email"
            type="email"
            defaultValue={target.email}
            required
          />
          <Field label="表示名" name="displayName" defaultValue={target.displayName} />
          <Select label="役割" name="role" options={roleOptions} defaultValue={target.role} />
          <Field
            label="新しいパスワード(変えるときだけ。12 文字以上)"
            name="password"
            type="password"
            autoComplete="new-password"
          />
        </div>
        <button type="submit">保存</button>
      </form>

      <h2>担当大会(記録係はスコアを入力でき、本部スタッフは当日の進行ができる大会)</h2>
      <form action={putUserTournamentsAction}>
        <Hidden values={{ back, id }} />
        {tournamentList.tournaments.map((tournament) => (
          <Check
            key={tournament.id}
            label={tournament.name.ja}
            name="tournamentIds"
            value={String(tournament.id)}
            defaultChecked={assigned.has(tournament.id)}
          />
        ))}
        <button type="submit">担当大会を保存</button>
      </form>

      {target.id === user.id ? null : (
        <DeleteForm action={deleteUserAction} values={{ back, id }} label="このユーザー" />
      )}
    </AdminShell>
  )
}
