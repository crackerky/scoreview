import { sideName } from '@/admin/labels.utils'
import { courtLabel } from '@/admin/scoreKeeper.utils'
import { loadAdmin, requireUser } from '@/admin/session'
import { matches } from '@/api/admin'
import type { AssignedMatch } from '@/api/admin.types'
import type { IdPageProps } from '@/app/_lib/pageProps'
import { readAdminPage } from '@/app/admin/_lib/page'
import { ScoreKeeper } from '@/components/admin/ScoreKeeper'
import type { NextMatch } from '@/components/admin/ScoreSheets'
import '@/styles/scorekeeper.css'

// 記録係のスコア入力画面(docs/operations/operator-experience.md「記録係」)。スマートフォン縦持ちの
// 1 画面で、得点、取り消し、試合終了を中心に置く。入力は端末に溜めて /admin/api 経由で api へ送る。
// JavaScript が動かない端末のために、簡易の入力画面(score/basic)への案内を noscript で出す。
export default async function AdminScorePage(props: IdPageProps) {
  const { id } = await readAdminPage(props)
  const { user, token } = await requireUser('any', `/admin/matches/${id}/score`)
  const [detail, mine] = await loadAdmin(() =>
    Promise.all([matches.detail(token, id), matches.mine(token)]),
  )
  // 記録係は自分の割り当てが覆う試合だけを記録できる(api が最終的に判定する。ここは表示のため)
  const canScore = user.role === 'admin' || mine.matches.some((match) => match.id === id)
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
        isAdmin={user.role === 'admin'}
        canScore={canScore}
        userId={user.id}
        nextMatch={nextAssigned(mine.matches, id)}
      />
    </>
  )
}

/** 試合を終えた後に案内する次の担当試合(まだ終わっていない担当試合のうち、時刻順で最初のもの) */
function nextAssigned(list: AssignedMatch[], current: number): NextMatch | null {
  const next = list.find((match) => match.id !== current && match.status !== 'finished')
  if (next === undefined) {
    return null
  }
  const label = [next.time, courtLabel(next.field), next.code]
    .filter((part) => part !== null && part !== '')
    .join(' ')
  return { id: next.id, label: `${label} ${sideName(next.left)} - ${sideName(next.right)}` }
}
