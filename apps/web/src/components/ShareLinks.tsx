'use client'

import { usePathname, useSearchParams } from 'next/navigation'
import { useEffect, useState } from 'react'
import { shareTargets, shareUrl } from './share.utils'

// ヘッダー右上の共有ボタン(X、LINE、リンクのコピー)。share.utils.ts のコメント参照。
// サイトの URL はブラウザでしか分からない(プレビューと本番で違う)ので、表示の後に組み立てる。
// それまでは href の無い <a>(押せない)にする。

const icons = { x: '/brand/sns-x.svg', line: '/brand/sns-line.svg' } as const

// 「コピーしました」を出しておく長さ(ミリ秒)。読める長さで、次の操作の邪魔にならない程度
const copiedMessageMs = 2000

export function ShareLinks() {
  const pathname = usePathname()
  const query = useSearchParams().toString()
  const [page, setPage] = useState<{ url: string; title: string } | null>(null)
  const [copied, setCopied] = useState(false)
  useEffect(() => {
    setPage({ url: shareUrl(window.location.origin, pathname, query), title: document.title })
  }, [pathname, query])
  useEffect(() => {
    if (!copied) {
      return
    }
    const timer = window.setTimeout(() => setCopied(false), copiedMessageMs)
    return () => window.clearTimeout(timer)
  }, [copied])
  const targets = page === null ? null : shareTargets(page.url, page.title)
  const copy = async () => {
    if (page === null) {
      return
    }
    try {
      await navigator.clipboard.writeText(page.url)
      setCopied(true)
    } catch (error) {
      // 権限が無いブラウザではコピーできない。アドレスバーから写せるので画面には何も出さない
      console.warn('copy link failed', error)
    }
  }
  return (
    <>
      {(['x', 'line'] as const).map((kind) => {
        const target = targets?.find((item) => item.kind === kind)
        return (
          <a key={kind} href={target?.href} target="_blank" rel="noopener">
            <img src={icons[kind]} width={32} height={32} alt={target?.label ?? ''} />
          </a>
        )
      })}
      <button type="button" className="sv_share_copy" onClick={copy} disabled={page === null}>
        <img
          src="/brand/sns-link.svg"
          width={32}
          height={32}
          alt="このページのリンクをコピーする"
        />
      </button>
      <span className="sv_share_copied" role="status">
        {copied ? 'コピーしました' : ''}
      </span>
    </>
  )
}
