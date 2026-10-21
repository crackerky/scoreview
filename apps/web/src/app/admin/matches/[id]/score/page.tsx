import { loadAdmin, requireUser } from '@/admin/session'
import { matches } from '@/api/admin'
import type { IdPageProps } from '@/app/_lib/pageProps'
import { readAdminPage } from '@/app/admin/_lib/page'
import { ScoreKeeper } from '@/components/admin/ScoreKeeper'
import '@/styles/scorekeeper.css'

// 記録係のスコア入力画面(docs/operations/operator-experience.md「記録係」)。スマートフォン縦持ちの
// 1 画面で、得点、取り消し、試合終了を中心に置く。入力は端末に溜めて /admin/api 経由で api へ送る。
// JavaScript が動かない端末のために、簡易の入力画面(score/basic)への案内を noscript で出す。
export default async function AdminScorePage(props: IdPageProps) {
  const { id } = await readAdminPage(props)
  const { user, token } = await requireUser('any')
  const detail = await loadAdmin(() => matches.detail(token, id))
  // 記録係は自分の割り当てが覆う試合だけを記録できる(api が最終的に判定する。ここは表示のため)
  const canScore =
    user.role === 'admin' ||
    (await loadAdmin(() => matches.mine(token))).matches.some((match) => match.id === id)
  return (
    <>
      <title>{`${detail.match.code || '試合'} 記録 - 管理画面`}</title>
      <noscript>
        <p className="sk-banner">
          この端末では記録画面が動きません。
          <a href={`/admin/matches/${id}/score/basic`}>簡易の入力画面</a>を使ってください。
        </p>
      </noscript>
      <ScoreKeeper
        initial={detail}
        userId={user.id}
        isAdmin={user.role === 'admin'}
        canScore={canScore}
      />
    </>
  )
}
