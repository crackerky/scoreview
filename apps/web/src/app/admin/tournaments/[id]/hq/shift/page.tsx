import {
  courtCards,
  divisionsOnCourt,
  hqPath,
  shiftPreview,
  sideName,
  suggestedShift,
} from '@/admin/hqBoard.utils'
import { loadBoards } from '@/admin/hqData'
import { loadAdmin, requireUser } from '@/admin/session'
import { jstToday } from '@/admin/tournamentList.utils'
import { tournaments } from '@/api/admin'
import type { IdPageProps } from '@/app/_lib/pageProps'
import { shiftCourtAction } from '@/app/admin/_actions/hqBoard'
import { readAdminPage } from '@/app/admin/_lib/page'
import { AdminShell, Hidden } from '@/components/admin/AdminParts'

// 時間をずらす(本部の知らせ「遅れ」から)。遅れたコートの、まだ始まっていない試合をまとめてずらす。
// コートとずらす時間は選んだ状態で開き、変わる時刻を先に見せる。選び直しはリンク(画面を開き直す)。

const minuteChoices = [5, 10, 15, 20] as const

function pick(value: string | string[] | undefined): string {
  return typeof value === 'string' ? value : ''
}

export default async function ShiftPage(props: IdPageProps) {
  const { id, notice } = await readAdminPage(props)
  const query = await props.searchParams
  const { user, token } = await requireUser('admin')
  const [tournament, board] = await loadAdmin(() =>
    Promise.all([tournaments.get(token, id), loadBoards(token, id)]),
  )
  const courts = courtCards(board.boards, [], jstToday(new Date()))
  const court = courts.find((card) => card.court === pick(query.court)) ?? courts[0]
  const asked = Number(pick(query.minutes))
  const minutes =
    minuteChoices.find((choice) => choice === asked) ?? suggestedShift(court?.delay ?? null)
  const lines = court ? shiftPreview(board.boards, court.court, minutes) : []
  const self = `/admin/tournaments/${id}/hq/shift`
  const link = (next: { court?: string; minutes?: number }) =>
    `${self}?${new URLSearchParams({
      court: next.court ?? court?.court ?? '',
      minutes: String(next.minutes ?? minutes),
    }).toString()}`
  const crumbs = [
    { href: `/admin/tournaments/${id}`, label: tournament.name.ja },
    { href: hqPath(id), label: '本部' },
  ]
  return (
    <AdminShell
      user={user}
      title="時間をずらす"
      kicker="遅れたコートの、このあとの試合をまとめてずらします"
      notice={notice}
      crumbs={crumbs}
      primary={{ label: `${minutes}分ずらす`, form: 'shift', disabled: lines.length === 0 }}
    >
      <div className="adm-narrow adm-stack-lg">
        <section>
          <h2 className="adm-block-title">コート</h2>
          <div className="adm-choices">
            {courts.map((card) => (
              <a
                key={card.court}
                className="adm-choice"
                aria-current={card.court === court?.court ? 'true' : undefined}
                href={link({ court: card.court, minutes: suggestedShift(card.delay) })}
              >
                {card.name}
                {card.delay !== null ? (
                  <span className="adm-choice-sub">{card.delay}分遅れ</span>
                ) : null}
              </a>
            ))}
          </div>
        </section>
        <section>
          <h2 className="adm-block-title">ずらす時間</h2>
          <div className="adm-choices">
            {minuteChoices.map((choice) => (
              <a
                key={choice}
                className="adm-choice"
                aria-current={choice === minutes ? 'true' : undefined}
                href={link({ minutes: choice })}
              >
                {choice}分
              </a>
            ))}
          </div>
        </section>
        <section>
          <h2 className="adm-block-title">変わる時刻</h2>
          {lines.length === 0 ? (
            <div className="adm-card adm-muted">このコートにこれからの試合はありません。</div>
          ) : (
            <>
              <ul className="adm-rows">
                {lines.map((line) => (
                  <li key={line.match.id} className="adm-shift-row">
                    <span className="adm-num adm-shift-before">{line.before}</span>
                    <span className="adm-faint">→</span>
                    <span className="adm-num adm-shift-after">{line.after}</span>
                    <span className="adm-shift-what">
                      {line.match.groupLabel}　{sideName(line.match.left)} 対{' '}
                      {sideName(line.match.right)}
                    </span>
                  </li>
                ))}
              </ul>
              <p className="adm-muted">
                このコートのまだ始まっていない試合を全部ずらします。閲覧ページの時刻もすぐ変わります。
              </p>
            </>
          )}
        </section>
        {court ? (
          <form id="shift" action={shiftCourtAction}>
            <Hidden
              values={{
                back: link({}),
                tournamentId: id,
                court: court.court,
                minutes,
                divisionIds: divisionsOnCourt(board.boards, court.court).join(','),
              }}
            />
          </form>
        ) : null}
      </div>
    </AdminShell>
  )
}
