import type { ReactNode } from 'react'
import type { AdminUser } from '@/api/admin.types'
import { LogoutButton } from './LogoutButton'

// 管理画面の共通部品(外枠、通知、フォームの欄)。見た目は src/styles/admin.css。

export type Notice = { ok?: string; error?: string }

/** ページの searchParams から ?ok= と ?error= を読む */
export function readNotice(query: Record<string, string | string[] | undefined>): Notice {
  const pick = (value: string | string[] | undefined) =>
    typeof value === 'string' ? value : undefined
  return { ok: pick(query.ok), error: pick(query.error) }
}

export type Crumb = { href: string; label: string }

/** 画面の主な操作(オレンジのボタン 1 つ)。リンクか、ページ内のフォームの送信(form に id) */
export type PrimaryAction =
  | { label: string; href: string }
  | { label: string; form: string; disabled?: boolean }

export type ShellTab = { href: string; label: string; current: boolean; count?: number }

type ShellProps = {
  user: AdminUser
  title: string
  /** 見出しの上の小さい文字(日付、会場、公開の状態など) */
  kicker?: string
  notice?: Notice
  /** 大会一覧と今の画面の間の階層(大会一覧と今の画面は自動で足す) */
  crumbs?: Crumb[]
  primary?: PrimaryAction
  tabs?: ShellTab[]
  children: ReactNode
}

const roleLabel = { admin: '管理者', staff: '記録係' } as const

// 管理画面の外枠(Claude Design「Score View Admin」)。上から、黒い帯(サイト名と役割とメニュー)、
// 今いる場所(大会 › 大会名 › …)、見出しと主な操作、タブ、本文。主な操作は画面に 1 つだけで、
// スマートフォンの幅では画面の下に固定して片手で押せるようにする(admin.css)。
export function AdminShell({
  user,
  title,
  kicker,
  notice,
  crumbs = [],
  primary,
  tabs,
  children,
}: ShellProps) {
  const trail: Crumb[] = [{ href: '/admin', label: '大会' }, ...crumbs]
  return (
    <div className={`adm-app${primary ? ' has-primary' : ''}`}>
      <title>{`${title} - 管理画面`}</title>
      <header className="adm-top">
        <a className="adm-brand" href="/admin">
          <span className="adm-logo" aria-hidden="true" />
          <span>
            SCORE <span className="adm-brand-accent">VIEW</span>
          </span>
          <span className="adm-brand-sub">管理</span>
        </a>
        <span className="adm-role">{roleLabel[user.role]}</span>
        <details className="adm-menu">
          <summary>メニュー</summary>
          <div className="adm-menu-panel">
            <span className="adm-menu-user">{user.displayName || user.email}</span>
            <a href="/admin">大会一覧</a>
            {user.role === 'admin' ? <a href="/admin/users">ユーザー</a> : null}
            <a href="/admin/password">パスワード変更</a>
            <LogoutButton />
          </div>
        </details>
      </header>
      <nav className="adm-crumbbar" aria-label="今いる場所">
        <ol className="adm-wrap">
          {trail.map((crumb) => (
            <li key={crumb.href}>
              <a href={crumb.href}>{crumb.label}</a>
            </li>
          ))}
          {title === '大会' && crumbs.length === 0 ? null : <li aria-current="page">{title}</li>}
        </ol>
      </nav>
      <main className="adm adm-wrap">
        <div className="adm-head">
          <div className="adm-head-text">
            {kicker ? <span className="adm-kicker">{kicker}</span> : null}
            <h1>{title}</h1>
          </div>
          {primary ? <PrimaryButton action={primary} /> : null}
        </div>
        {tabs && tabs.length > 0 ? <ShellTabs tabs={tabs} /> : null}
        <NoticeBox notice={notice} />
        {children}
      </main>
    </div>
  )
}

function PrimaryButton({ action }: { action: PrimaryAction }) {
  if ('href' in action) {
    return (
      <a className="adm-primary" href={action.href}>
        {action.label}
      </a>
    )
  }
  return (
    <button type="submit" form={action.form} className="adm-primary" disabled={action.disabled}>
      {action.label}
    </button>
  )
}

function ShellTabs({ tabs }: { tabs: ShellTab[] }) {
  return (
    <nav className="adm-tabs" aria-label="表示の切り替え">
      {tabs.map((tab) => (
        <a key={tab.href} href={tab.href} aria-current={tab.current ? 'page' : undefined}>
          {tab.label}
          {tab.count ? <span className="adm-tab-count">{tab.count}</span> : null}
        </a>
      ))}
    </nav>
  )
}

/** 入口のカード(大会ホームなど)。上の画面は要約と入口だけにし、操作は入口の先の画面で行う */
export function EntryCard({
  href,
  label,
  sub,
  badge,
  tone,
  highlight,
}: {
  href: string
  label: string
  sub: string
  badge?: string
  tone?: ChipTone
  highlight?: boolean
}) {
  return (
    <a className={`adm-entry${highlight ? ' highlight' : ''}`} href={href}>
      <span className="adm-entry-head">
        <span className="adm-entry-label">{label}</span>
        {badge ? <Chip tone={tone ?? 'warn'}>{badge}</Chip> : null}
        <span className="adm-chevron" aria-hidden="true">
          ›
        </span>
      </span>
      <span className="adm-entry-sub">{sub}</span>
    </a>
  )
}

export type ChipTone = 'live' | 'done' | 'sched' | 'warn' | 'ok'

/** 状態の小さい札(試合中、終了、確定待ちなど) */
export function Chip({ tone, children }: { tone: ChipTone; children: ReactNode }) {
  return <span className={`adm-chip ${tone}`}>{children}</span>
}

/** 数字を大きく出す要約(大会ホームの「いま試合中」など) */
export function Stat({ label, value, tone }: { label: string; value: string; tone?: ChipTone }) {
  return (
    <div className="adm-stat">
      <span className="adm-stat-label">{label}</span>
      <span className={`adm-stat-value${tone ? ` ${tone}` : ''}`}>{value}</span>
    </div>
  )
}

/** 削除などの危険な操作。設定画面のいちばん下に置く */
export function DangerZone({ children }: { children: ReactNode }) {
  return (
    <section className="adm-danger-zone">
      <h2>削除</h2>
      {children}
    </section>
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
