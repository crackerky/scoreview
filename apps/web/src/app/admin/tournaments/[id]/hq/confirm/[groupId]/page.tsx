import { notFound } from 'next/navigation'
import { confirmEffects, hqPath } from '@/admin/hqBoard.utils'
import { loadBoards } from '@/admin/hqData'
import { loadAdmin, requireUser } from '@/admin/session'
import { tournaments } from '@/api/admin'
import { confirmGroupFromHqAction } from '@/app/admin/_actions/hqBoard'
import { AdminShell, Hidden, readNotice } from '@/components/admin/AdminParts'
import { StandingsCard } from '@/components/admin/Standings'

// 組の確定(本部の知らせ「確定待ち」から)。順位と、確定すると次の組み合わせにどう入るかを見せてから
// 確定する。同順位が規程で決まらない組は、部門の進行画面で順位を決めてから確定する。

type Props = {
  params: Promise<{ id: string; groupId: string }>
  searchParams: Promise<Record<string, string | string[] | undefined>>
}

export default async function ConfirmGroupPage(props: Props) {
  const [params, query] = await Promise.all([props.params, props.searchParams])
  if (!/^[0-9]+$/.test(params.id) || !/^[0-9]+$/.test(params.groupId)) {
    notFound()
  }
  const id = Number(params.id)
  const groupId = Number(params.groupId)
  const { user, token } = await requireUser('hq')
  const [tournament, board] = await loadAdmin(() =>
    Promise.all([tournaments.get(token, id), loadBoards(token, id)]),
  )
  const owner = board.boards.find((item) =>
    item.progress.groups.some((group) => group.id === groupId),
  )
  const group = owner?.progress.groups.find((item) => item.id === groupId)
  if (!owner || !group) {
    notFound()
  }
  const effects = confirmEffects(owner, group)
  const ready = group.state === 'ready'
  const crumbs = [
    { href: `/admin/tournaments/${id}`, label: tournament.name.ja },
    { href: hqPath(id), label: '本部' },
  ]
  return (
    <AdminShell
      user={user}
      title={`${group.label}の確定`}
      kicker="全試合が終わった組の順位を確定します"
      notice={readNotice(query)}
      crumbs={crumbs}
      primary={ready ? { label: `${group.label}を確定する`, form: 'confirm' } : undefined}
    >
      <div className="adm-narrow adm-stack-lg">
        <StandingsCard group={group} title={group.label} />
        {ready ? null : (
          <div className="adm-card">
            <strong>この組はまだ確定できません</strong>
            <span className="adm-muted">
              {group.state === 'tied'
                ? '規程で決まらない同順位があります。進行画面で順位を決めてください。'
                : group.state === 'playing'
                  ? `残り ${group.remaining} 試合が終わると確定できます。`
                  : 'この組は確定済みか、前の段を待っています。'}
            </span>
            <a className="adm-go" href={`/admin/divisions/${owner.divisionId}/progress`}>
              進行画面を開く
            </a>
          </div>
        )}
        {effects.length > 0 ? (
          <section>
            <h2 className="adm-block-title">確定すると、こう入ります</h2>
            <ul className="adm-rows">
              {effects.map((effect) => (
                <li key={effect.match.id} className="adm-effect-row">
                  <span className="adm-muted">
                    {[
                      effect.match.groupLabel,
                      effect.match.time,
                      effect.match.field ? `コート ${effect.match.field}` : '',
                    ]
                      .filter(Boolean)
                      .join(' ・ ')}
                  </span>
                  <span>
                    {effect.left} <span className="adm-faint">対</span> {effect.right}
                  </span>
                </li>
              ))}
            </ul>
            <p className="adm-muted">閲覧ページのトーナメント表にもすぐ出ます。</p>
          </section>
        ) : null}
        <form id="confirm" action={confirmGroupFromHqAction}>
          <Hidden
            values={{
              back: `${hqPath(id)}/confirm/${groupId}`,
              tournamentId: id,
              groupId,
              groupLabel: group.label,
            }}
          />
        </form>
      </div>
    </AdminShell>
  )
}
