import { type FormEvent, startTransition } from 'react'

/**
 * 公開側のフォームの onSubmit。Server Action を form の action だけで送ると、React は送った後に欄を最初の
 * 値へ戻す。エラーで戻ったときに、スマートフォンで書いた長いフォーム(選んだ部門や名簿の行も)を書き直させ
 * ないよう、送信は onSubmit から自分で行い、欄をそのまま残す(onSubmit が送信を止めると React は action を
 * 使わない)。action も form に渡しておくのは、JavaScript が読み込まれる前の送信を、電話番号などが URL に
 * 載る GET にせず Server Action の POST にするため。二重送信の止め方(SubmitFeedback)は document の
 * submit を見ているので、そのまま効く。
 */
export function submitLinkForm(dispatch: (formData: FormData) => void) {
  return (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const formData = new FormData(event.currentTarget)
    startTransition(() => dispatch(formData))
  }
}
