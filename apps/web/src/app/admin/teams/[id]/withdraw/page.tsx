import { safeAdminPath } from '@/admin/form.utils'
import { forfeitScoreText, withdrawalSentence } from '@/admin/hq.utils'
import { loadAdmin, requireUser } from '@/admin/session'
import { formats } from '@/api/format'
import type { WithdrawalMatch } from '@/api/format.types'
import type { IdPageProps } from '@/app/_lib/pageProps'
import { withdrawTeamAction } from '@/app/admin/_actions/hq'
import { readAdminPage } from '@/app/admin/_lib/page'
import { AdminShell, Hidden } from '@/components/admin/AdminParts'
import '@/styles/format-admin.css'

// チームの棄権の確認(レビュー H3)。「残り 2 試合(C3、G3)を相手の不戦勝にします」を出し、押すと
// 一括で記録する。表に出した試合だけを送り、この間に変わっていれば api が断る。

function MatchRows({ matches }: { matches: WithdrawalMatch[] }) {
  return (
    <ul className="adm-list">
      {matches.map((match) => (
        <li key={match.id}>
          {match.code} {match.time ?? '--:--'} {match.field ? `コート ${match.field}` : ''} 相手:{' '}
          {match.opponent?.name.ja ?? '未定'}
          {match.result === 'double_forfeit' ? '(相手も棄権。両チーム不戦敗)' : ''}
          {match.status === 'live' ? '(試合中)' : ''}
        </li>
      ))}
    </ul>
  )
}

export default async function WithdrawPage(props: IdPageProps) {
  const { id, notice } = await readAdminPage(props)
  const query = await props.searchParams
  const { user, token } = await requireUser('hq')
  const withdrawal = await loadAdmin(() => formats.withdrawal(token, id))
  const progress = `/admin/divisions/${withdrawal.divisionId}/progress`
  const nextRaw = typeof query.next === 'string' ? query.next : ''
  const next = safeAdminPath(nextRaw || progress)
  const team = withdrawal.team.name.ja
  return (
    <AdminShell
      user={user}
      title={`${team} を棄権にする`}
      notice={notice}
      crumbs={[{ href: progress, label: '本部の進行画面' }]}
    >
      {withdrawal.withdrawnAt ? (
        <p className="adm-notice ok">{team} は棄権済みです。後から入った試合を記録できます。</p>
      ) : null}
      <section className="hq-box danger">
        <h2>{withdrawalSentence(withdrawal)}</h2>
        {withdrawal.targets.length > 0 ? <MatchRows matches={withdrawal.targets} /> : null}
        <p>
          {forfeitScoreText(withdrawal.forfeitScore)}。変更の履歴には「棄権の一括処理」と残ります。
        </p>
        <form action={withdrawTeamAction}>
          <Hidden values={{ back: `/admin/teams/${id}/withdraw`, next, teamId: id }} />
          {withdrawal.targets.map((match) => (
            <input key={match.id} type="hidden" name="matchIds" value={match.id} />
          ))}
          <div className="adm-actions">
            <button type="submit" className="danger">
              {withdrawal.targets.length > 0
                ? `${team} を棄権にして ${withdrawal.targets.length} 試合を記録する`
                : `${team} を棄権にする`}
            </button>
            <a href={next}>やめる</a>
          </div>
        </form>
      </section>
      {withdrawal.waiting.length > 0 ? (
        <section className="hq-box">
          <h2>今は記録しない試合</h2>
          <p className="adm-muted">
            相手が決まっていない試合と試合中の試合です。相手が決まると本部の「いまやること」に出るので、そこから記録します。
          </p>
          <MatchRows matches={withdrawal.waiting} />
        </section>
      ) : null}
    </AdminShell>
  )
}
