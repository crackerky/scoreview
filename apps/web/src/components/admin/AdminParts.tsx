import type { ReactNode } from 'react'
import type { AdminUser } from '@/api/admin.types'
import { logoutAction } from '@/app/admin/_actions/auth'

// 管理画面の共通部品(外枠、通知、フォームの欄)。見た目は src/styles/admin.css。

export type Notice = { ok?: string; error?: string }

/** ページの searchParams から ?ok= と ?error= を読む */
export function readNotice(query: Record<string, string | string[] | undefined>): Notice {
  const pick = (value: string | string[] | undefined) =>
    typeof value === 'string' ? value : undefined
  return { ok: pick(query.ok), error: pick(query.error) }
}

type ShellProps = {
  user: AdminUser
  title: string
  notice?: Notice
  crumbs?: Array<{ href: string; label: string }>
  children: ReactNode
}

export function AdminShell({ user, title, notice, crumbs, children }: ShellProps) {
  return (
    <div className="adm">
      <title>{`${title} - 管理画面`}</title>
      <header className="adm-header">
        <nav>
          <a href="/admin">大会一覧</a>
          {user.role === 'admin' ? <a href="/admin/users">ユーザー</a> : null}
        </nav>
        <span className="adm-user">
          {user.displayName || user.email}({user.role === 'admin' ? '管理者' : '記録係'})
        </span>
        <form action={logoutAction}>
          <button type="submit" className="secondary">
            ログアウト
          </button>
        </form>
      </header>
      {crumbs && crumbs.length > 0 ? (
        <p className="adm-crumbs">
          {crumbs.map((crumb, index) => (
            <span key={crumb.href}>
              {index > 0 ? ' / ' : ''}
              <a href={crumb.href}>{crumb.label}</a>
            </span>
          ))}
        </p>
      ) : null}
      <h1>{title}</h1>
      <NoticeBox notice={notice} />
      {children}
    </div>
  )
}

export function NoticeBox({ notice }: { notice?: Notice }) {
  if (notice?.error) {
    return (
      <p className="adm-notice error" role="alert">
        {notice.error}
      </p>
    )
  }
  if (notice?.ok) {
    return (
      <p className="adm-notice ok" role="status">
        {notice.ok}
      </p>
    )
  }
  return null
}

type FieldProps = {
  label: string
  name: string
  defaultValue?: string | number | null
  type?: 'text' | 'email' | 'password' | 'number' | 'date' | 'time'
  required?: boolean
  placeholder?: string
  autoComplete?: string
}

export function Field({ label, name, defaultValue, type = 'text', ...rest }: FieldProps) {
  return (
    <label className="adm-field">
      <span>{label}</span>
      <input type={type} name={name} defaultValue={defaultValue ?? ''} {...rest} />
    </label>
  )
}

export function TextArea({
  label,
  name,
  defaultValue,
  rows = 5,
}: {
  label: string
  name: string
  defaultValue?: string
  rows?: number
}) {
  return (
    <label className="adm-field">
      <span>{label}</span>
      <textarea name={name} defaultValue={defaultValue ?? ''} rows={rows} />
    </label>
  )
}

type Option = { value: string; label: string }

export function Select({
  label,
  name,
  options,
  defaultValue,
}: {
  label: string
  name: string
  options: readonly Option[]
  defaultValue?: string
}) {
  return (
    <label className="adm-field">
      <span>{label}</span>
      <select name={name} defaultValue={defaultValue}>
        {options.map((option) => (
          <option key={option.value} value={option.value}>
            {option.label}
          </option>
        ))}
      </select>
    </label>
  )
}

export function Check({
  label,
  name,
  defaultChecked,
  value,
}: {
  label: string
  name: string
  defaultChecked?: boolean
  value?: string
}) {
  return (
    <label className="adm-check">
      <input type="checkbox" name={name} defaultChecked={defaultChecked} value={value} />
      <span>{label}</span>
    </label>
  )
}

/** フォームに hidden 欄を並べる(id、親の id、戻り先の back) */
export function Hidden({ values }: { values: Record<string, string | number> }) {
  return (
    <>
      {Object.entries(values).map(([name, value]) => (
        <input key={name} type="hidden" name={name} value={String(value)} />
      ))}
    </>
  )
}

/**
 * 削除のフォーム。誤タップで消さないよう「削除する」のチェックを必須にする(JavaScript の確認ダイアログを使わない)。
 */
export function DeleteForm({
  action,
  values,
  label,
}: {
  action: (formData: FormData) => Promise<never>
  values: Record<string, string | number>
  label: string
}) {
  return (
    <form action={action} className="adm-actions">
      <Hidden values={values} />
      <label className="adm-check">
        <input type="checkbox" name="confirm" required />
        <span>{label}を削除する</span>
      </label>
      <button type="submit" className="danger">
        削除
      </button>
    </form>
  )
}
