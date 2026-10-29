import { roleOptions } from '@/admin/labels.utils'
import { loadAdmin, requireUser } from '@/admin/session'
import { users } from '@/api/admin'
import { createUserAction } from '@/app/admin/_actions/user'
import type { SearchProps } from '@/app/admin/_lib/page'
import { AdminShell, Field, Hidden, readNotice, Select } from '@/components/admin/AdminParts'

export default async function AdminUsersPage({ searchParams }: SearchProps) {
  const notice = readNotice(await searchParams)
  const { user, token } = await requireUser('admin')
  const list = await loadAdmin(() => users.list(token))
  return (
    <AdminShell user={user} title="ユーザー" notice={notice}>
      <ul className="adm-list">
        {list.users.map((item) => (
          <li key={item.id}>
            <a href={`/admin/users/${item.id}`}>
              {item.displayName || item.email}
              <span className="adm-muted">
                ({item.email}、
                {item.role === 'admin'
                  ? '管理者'
                  : `${item.role === 'hq' ? '本部スタッフ' : '記録係'} 担当 ${item.tournamentIds.length} 大会`}
                )
              </span>
            </a>
          </li>
        ))}
      </ul>
      <h2>ユーザーを作る</h2>
      <form action={createUserAction}>
        <Hidden values={{ back: '/admin/users' }} />
        <div className="adm-grid">
          <Field label="メールアドレス" name="email" type="email" required autoComplete="off" />
          <Field label="表示名" name="displayName" />
          <Select label="役割" name="role" options={roleOptions} defaultValue="staff" />
          <Field
            label="初期パスワード(12 文字以上)"
            name="password"
            type="password"
            required
            autoComplete="new-password"
          />
        </div>
        <button type="submit">作成</button>
      </form>
    </AdminShell>
  )
}
