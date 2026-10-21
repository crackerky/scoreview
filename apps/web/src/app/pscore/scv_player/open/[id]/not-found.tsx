import { NotFoundFromRequest } from '@/app/_lib/notFound'

// clone-spec D 0: "ENTRY_PLAYER_ID [{id}]での操作に失敗しました。" でメニューへのリンクはない
export default function PlayerNotFound() {
  return <NotFoundFromRequest withMenuLink={false} />
}
