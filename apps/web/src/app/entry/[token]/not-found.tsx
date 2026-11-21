import { SiteShell } from '@/components/site/SiteShell'
import { linkUnusableText } from '@/entry/entryLink.utils'
import '@/styles/entry.css'

// 本人用のリンクが無いか、期限が切れたとき(entryLink.load が notFound() を呼ぶ)。どちらも公式 LINE に
// 送り直せば新しいリンクが届くので、分けずに同じ案内を出す。
export default function EntryLinkNotFound() {
  return (
    <SiteShell>
      <title>リンクが使えません - SCORE VIEW</title>
      <h1 className="site-title">リンクが使えません</h1>
      <p>{linkUnusableText}</p>
    </SiteShell>
  )
}
