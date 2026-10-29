import { loginErrorMessage } from '@/admin/login.utils'
import { safeNext } from '@/admin/session.utils'
import { loginAction } from '@/app/admin/_actions/auth'
import type { SearchProps } from '@/app/admin/_lib/page'
import { Field, Hidden, NoticeBox, readNotice } from '@/components/admin/AdminParts'

export default async function AdminLoginPage({ searchParams }: SearchProps) {
  const query = await searchParams
  const notice = readNotice(query)
  const next = safeNext(query.next)
  return (
    <div className="adm-app">
      <title>ログイン - 管理画面</title>
      <header className="adm-top">
        <span className="adm-brand">
          <span className="adm-logo" aria-hidden="true" />
          <span>
            SCORE <span className="adm-brand-accent">VIEW</span>
          </span>
          <span className="adm-brand-sub">管理</span>
        </span>
      </header>
      <main className="adm adm-wrap adm-login">
        <h1>管理画面ログイン</h1>
        {/* ?error= はコードだけを受け取り、文面はここで決める(任意の文を出させない) */}
        <NoticeBox notice={{ error: loginErrorMessage(notice.error) }} />
        <form action={loginAction}>
          <Hidden values={{ next }} />
          <Field
            label="メールアドレス"
            name="email"
            type="email"
            required
            autoComplete="username"
          />
          <Field
            label="パスワード"
            name="password"
            type="password"
            required
            autoComplete="current-password"
          />
          <button type="submit" className="adm-primary">
            ログイン
          </button>
        </form>
      </main>
    </div>
  )
}
