import { NotFoundFromRequest } from '@/app/_lib/notFound'

// clone-spec D 0: 存在しない試合は "MATCH_ID [{id}]" の画面(メニューへのリンクなし)
export default function MatchNotFound() {
  return <NotFoundFromRequest withMenuLink={false} />
}
