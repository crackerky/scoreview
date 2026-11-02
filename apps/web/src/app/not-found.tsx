import { SiteShell } from '@/components/site/SiteShell'

// どのルートにも合わない URL と、存在しない大会の観戦アプリ
export default function RootNotFound() {
  return (
    <SiteShell>
      <title>ページが見つかりません - SCORE VIEW</title>
      <h1 className="site-title">ページが見つかりません</h1>
      <p className="site-empty">
        URL が違うか、大会が公開されていません。<a href="/">大会の一覧</a>から選び直してください。
      </p>
    </SiteShell>
  )
}
