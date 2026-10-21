// ヘッダーの共有ボタン(ShareLinks)の行き先。参照サイトの同じ位置には運営会社の SNS アカウントへの
// リンクが並ぶが、本サイトは自前のアカウントを持たないので、今のページを共有する入口にする
// (docs/requirements.md「意図的な差分」)。会場で結果を仲間に送る使い方を想定する。

export type ShareTarget = { kind: 'x' | 'line'; href: string; label: string }

/** 今のページ(url と title)を共有する X と LINE の URL。どちらも公式に案内されている共有用の URL */
export function shareTargets(url: string, title: string): ShareTarget[] {
  const x = new URL('https://x.com/intent/post')
  x.searchParams.set('url', url)
  x.searchParams.set('text', title)
  const line = new URL('https://social-plugins.line.me/lineit/share')
  line.searchParams.set('url', url)
  return [
    { kind: 'x', href: x.href, label: 'X でこのページを共有する' },
    { kind: 'line', href: line.href, label: 'LINE でこのページを送る' },
  ]
}

/** 共有する URL。ページの位置(#...)は付けない */
export function shareUrl(origin: string, pathname: string, query: string): string {
  return query === '' ? `${origin}${pathname}` : `${origin}${pathname}?${query}`
}
