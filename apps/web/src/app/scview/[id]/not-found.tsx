import { NotFoundFromRequest } from '@/app/_lib/notFound'

// clone-spec A 7: /scview/{id} のエラー画面だけ「メニュー画面より…」と戻るリンクが付く
export default function ScviewNotFound() {
  return <NotFoundFromRequest withMenuLink />
}
