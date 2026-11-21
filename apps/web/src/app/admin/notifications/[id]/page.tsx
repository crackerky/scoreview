import { loadAdmin, requireUser } from '@/admin/session'
import { divisions, tournaments } from '@/api/admin'
import { notifications } from '@/api/notify'
import type { IdPageProps } from '@/app/_lib/pageProps'
import { readAdminPage } from '@/app/admin/_lib/page'
import { AdminShell, Chip, Stat } from '@/components/admin/AdminParts'
import { dateTimeText } from '@/entry/entry.utils'
import { notifyPaths } from '@/notify/notify.paths'
import { audienceText, jobStatusLabels, jobStatusTones } from '@/notify/notify.utils'

// 公式 LINE で送った 1 回。届けた文と、宛先ごとの結果(送った、ブロック中で送らなかった、送れなかった)。
// 名前とチームは送った時点のもの。
export default async function NotificationPage(props: IdPageProps) {
  const { id, notice } = await readAdminPage(props)
  const { user, token } = await requireUser('admin')
  const detail = await loadAdmin(() => notifications.get(token, id))
  const [tournament, divisionList] = await loadAdmin(() =>
    Promise.all([
      tournaments.get(token, detail.tournamentId),
      divisions.list(token, detail.tournamentId),
    ]),
  )
  const divisionNames = new Map(
    divisionList.divisions.map((division) => [division.id, division.name.ja]),
  )
  const { counts } = detail
  return (
    <AdminShell
      user={user}
      title="送った文"
      kicker={[dateTimeText(detail.createdAt), detail.sentBy]
        .filter((part) => part !== '')
        .join(' ・ ')}
      notice={notice}
      crumbs={[
        { href: `/admin/tournaments/${detail.tournamentId}`, label: tournament.name.ja },
        { href: notifyPaths.list(detail.tournamentId), label: '送った記録' },
      ]}
    >
      <p>宛先: {audienceText(detail.audience, (division) => divisionNames.get(division))}</p>
      <figure className="adm-bubble-preview">
        <figcaption>届けた文</figcaption>
        <p className="adm-bubble">{detail.body}</p>
      </figure>
      <div className="adm-stats">
        <Stat label="送った" value={String(counts.sent)} />
        <Stat label="ブロック中で送らなかった" value={String(counts.skipped)} />
        <Stat
          label="送れなかった"
          value={String(counts.failed + counts.pending)}
          tone={counts.failed + counts.pending > 0 ? 'live' : undefined}
        />
      </div>
      {counts.failed > 0 ? (
        <p className="adm-muted">
          送れなかった人には、条件を選び直してもう一度送ってください。LINE
          の混み合いなどで一時的に受け付けられなかったことがあります。
        </p>
      ) : null}
      <ul className="adm-rows">
        {detail.jobs.map((job, index) => (
          // biome-ignore lint/suspicious/noArrayIndexKey: 宛先の並びは記録した順で変わらない
          <li key={index} className="adm-row">
            <span className="adm-row-main">
              <span className="adm-row-title">{job.name || '名前なし'}</span>
              <span className="adm-row-meta">
                {[job.teams, job.sentAt ? dateTimeText(job.sentAt) : '']
                  .filter((part) => part !== '')
                  .join(' ・ ')}
              </span>
            </span>
            <Chip tone={jobStatusTones[job.status]}>{jobStatusLabels[job.status]}</Chip>
          </li>
        ))}
      </ul>
    </AdminShell>
  )
}
