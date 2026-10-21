import { loginAction } from '@/app/admin/_actions/auth'
import type { SearchProps } from '@/app/admin/_lib/page'
import { Field, NoticeBox, readNotice } from '@/components/admin/AdminParts'

export default async function AdminLoginPage({ searchParams }: SearchProps) {
  const notice = readNotice(await searchParams)
  return (
    <div className="adm">
      <title>ログイン - 管理画面</title>
      <h1>管理画面ログイン</h1>
      <NoticeBox notice={{ error: notice.error }} />
      <form action={loginAction}>
        <Field label="メールアドレス" name="email" type="email" required autoComplete="username" />
        <Field
          label="パスワード"
          name="password"
          type="password"
          required
          autoComplete="current-password"
        />
        <button type="submit">ログイン</button>
      </form>
    </div>
  )
}
