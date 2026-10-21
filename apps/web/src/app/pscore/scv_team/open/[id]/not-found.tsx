import { NotFoundFromRequest } from '@/app/_lib/notFound'

// clone-spec D 0: "ENTRY_TEAM_ID [{id}]での操作に失敗しました。" でメニューへのリンクはない
export default function TeamNotFound() {
  return <NotFoundFromRequest withMenuLink={false} />
}
