'use client'

// 印刷用のページの「印刷する」ボタン。ブラウザの印刷(PDF への保存を含む)を開く。
export function PrintButton() {
  return (
    <button type="button" onClick={() => window.print()}>
      印刷する
    </button>
  )
}
