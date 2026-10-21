'use client'

import {
  type CSSProperties,
  type FocusEvent,
  type KeyboardEvent,
  type ReactNode,
  useState,
} from 'react'

// 大会ナビのドロップダウン 1 つ(見出しの p と、その下の ul)。開閉は legacy.css の :hover と
// :focus-within で行い、JavaScript がなくてもタップ(フォーカス)とキーボードで開く。
// このコンポーネントは支援技術向けに aria-expanded を開閉の状態(li の中にフォーカスがあるか)に
// 合わせるだけで、見た目は変えない(本サイト独自。docs/requirements.md「意図的な差分」)。

type NaviMenuItemProps = {
  label: string
  /** 見出し p のクラス(第 1 階層は "ar"、2 階層目はなし) */
  headClassName?: string
  style?: CSSProperties
  /** 開いたときに出す ul */
  children: ReactNode
}

export function NaviMenuItem({ label, headClassName, style, children }: NaviMenuItemProps) {
  const [open, setOpen] = useState(false)
  const onBlur = (event: FocusEvent<HTMLLIElement>) => {
    // フォーカスが li の中の別の要素(サブメニューのリンク)へ移っただけなら開いたままにする
    if (!event.currentTarget.contains(event.relatedTarget)) {
      setOpen(false)
    }
  }
  const onKeyDown = (event: KeyboardEvent<HTMLLIElement>) => {
    if (event.key !== 'Escape') {
      return
    }
    // :focus-within で開いているので、フォーカスを外して閉じる
    const target = event.target
    if (target instanceof HTMLElement) {
      target.blur()
    }
  }
  return (
    <li style={style} onFocus={() => setOpen(true)} onBlur={onBlur} onKeyDown={onKeyDown}>
      {/* 参照サイトの p(legacy.css の見た目と比較ツールの対象)を保ったまま、開閉できる見出しとして読ませる */}
      {/* biome-ignore lint/a11y/useSemanticElements: 上のコメントの理由 */}
      <p
        className={headClassName}
        tabIndex={0}
        // biome-ignore lint/a11y/noNoninteractiveElementToInteractiveRole: 上のコメントの理由
        role="button"
        aria-haspopup="true"
        aria-expanded={open}
      >
        {label}
      </p>
      {children}
    </li>
  )
}
