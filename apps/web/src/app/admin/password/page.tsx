import { requireUser } from '@/admin/session'
import { changePasswordAction } from '@/app/admin/_actions/auth'
import type { SearchProps } from '@/app/admin/_lib/page'
import { AdminShell, Field, Hidden, readNotice } from '@/components/admin/AdminParts'

// 本人のパスワードの変更(管理者も記録係も)。管理者に初期パスワードを決めてもらった記録係が、
// 自分しか知らないパスワードに変えるため(セキュリティ点検 L-8)。
export default async function AdminPasswordPage({ searchParams }: SearchProps) {
  const notice = readNotice(await searchParams)
  const { user } = await requireUser('any')
  return (
    <AdminShell user={user} title="パスワード変更" notice={notice}>
      <p className="adm-muted">
        12 文字以上で、同じ文字の繰り返しやよく使われるパスワードは使えません。
        変更すると、この端末以外ではログインし直しになります。
      </p>
      <form action={changePasswordAction}>
        <Hidden values={{ back: '/admin/password' }} />
        {/* パスワードマネージャーがアカウントを特定できるように、メールアドレスを隠しで置く */}
        <input type="hidden" name="username" autoComplete="username" value={user.email} readOnly />
        <Field
          label="今のパスワード"
          name="currentPassword"
          type="password"
          required
          autoComplete="current-password"
        />
        <Field
          label="新しいパスワード"
          name="newPassword"
          type="password"
          required
          autoComplete="new-password"
        />
        <Field
          label="新しいパスワード(確認)"
          name="newPasswordConfirm"
          type="password"
          required
          autoComplete="new-password"
        />
        <button type="submit">変更する</button>
      </form>
    </AdminShell>
  )
}
