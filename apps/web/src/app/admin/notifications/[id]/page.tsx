import { loadAdmin, requireUser } from '@/admin/session'
import { divisions, tournaments } from '@/api/admin'
import { notifications } from '@/api/notify'
import type { IdPageProps } from '@/app/_lib/pageProps'
import { cancelNotificationAction, resendNotificationAction } from '@/app/admin/_actions/notify'
import { readAdminPage } from '@/app/admin/_lib/page'
import { AdminShell, Chip, Hidden, Stat } from '@/components/admin/AdminParts'
import { ForgetDraft } from '@/components/admin/MessageDraft'
import { dateTimeText } from '@/entry/entry.utils'
import { notifyPaths } from '@/notify/notify.paths'
import {
  audienceText,
  draftStorageKey,
  jobStatusLabels,
  jobStatusTones,
  scheduleText,
  skipReasonLabels,
} from '@/notify/notify.utils'

// 公式 LINE で送った 1 回。届けた文と、宛先ごとの結果(送った、ブロック中で送らなかった、送れなかった)。
// 名前とチームは送った時点のもの。予約なら、送る日時と取りやめのボタン(宛先は送る時刻に決まる)。
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
  const resendForm = 'notification-resend'
  const cancelForm = 'notification-cancel'
  const scheduled = detail.state === 'scheduled'
  const primary = scheduled
    ? { label: '予約を取りやめる', form: cancelForm }
    : detail.canResend
      ? { label: '送れなかった人に送り直す', form: resendForm }
      : undefined
  return (
    <AdminShell
      user={user}
      title={scheduled ? '予約した文' : '送った文'}
      kicker={[dateTimeText(detail.createdAt), detail.sentBy]
        .filter((part) => part !== '')
        .join(' ・ ')}
      notice={notice}
      crumbs={[
        { href: `/admin/tournaments/${detail.tournamentId}`, label: tournament.name.ja },
        { href: notifyPaths.list(detail.tournamentId), label: '送った記録' },
      ]}
      primary={primary}
    >
      <ForgetDraft draftKey={draftStorageKey(detail.tournamentId)} />
      <p>宛先: {audienceText(detail.audience, (division) => divisionNames.get(division))}</p>
      {detail.sendAt ? <p>送る日時: {scheduleText(detail.sendAt)}(予約)</p> : null}
      {scheduled ? (
        <form id={cancelForm} action={cancelNotificationAction} className="adm-stack">
          <Hidden values={{ back: notifyPaths.detail(id), id }} />
          <p className="adm-muted">
            送る時刻に、そのときの条件で宛先を数え直して送ります(15
            分ほど遅れることがあります)。今月の配信数が足りなければ送りません。
          </p>
        </form>
      ) : null}
      {detail.state === 'cancelled' ? (
        <p className="adm-notice error">この予約は取りやめたので、送っていません。</p>
      ) : null}
      {detail.state === 'skipped' && detail.note !== '' ? (
        <p className="adm-notice error">{skipReasonLabels[detail.note]}。</p>
      ) : null}
      <figure className="adm-bubble-preview">
        <figcaption>{detail.state === 'sent' ? '届けた文' : '送る文'}</figcaption>
        <p className="adm-bubble">{detail.body}</p>
      </figure>
      {detail.state !== 'sent' ? null : (
        <SentResults detail={detail} resendForm={resendForm} resendBack={notifyPaths.detail(id)} />
      )}
    </AdminShell>
  )
}

function SentResults({
  detail,
  resendForm,
  resendBack,
}: {
  detail: Awaited<ReturnType<typeof notifications.get>>
  resendForm: string
  resendBack: string
}) {
  const { counts } = detail
  return (
    <>
      <div className="adm-stats">
        <Stat label="送った" value={String(counts.sent)} />
        <Stat label="ブロック中で送らなかった" value={String(counts.skipped)} />
        <Stat
          label="送れなかった"
          value={String(counts.failed)}
          tone={counts.failed > 0 ? 'live' : undefined}
        />
        {counts.pending > 0 ? (
          <Stat label="送っている途中" value={String(counts.pending)} tone="warn" />
        ) : null}
      </div>
      {detail.canResend ? (
        <form id={resendForm} action={resendNotificationAction} className="adm-stack">
          <Hidden values={{ back: resendBack, id: detail.id }} />
          <p className="adm-muted">
            送れなかった人にだけ、同じ文を送り直せます(送ってから 24
            時間のあいだ)。前の送信が実は届いていた人には、2 回は届きません。
          </p>
        </form>
      ) : counts.pending > 0 ? (
        <p className="adm-muted">
          送っている途中の人がいます。途中で止まったままなら、15
          分ほどで送り直せるようになります。少しあとで、この画面を開き直してください。
        </p>
      ) : counts.failed > 0 ? (
        <p className="adm-muted">
          送ってから 24
          時間を過ぎたので、ここからは送り直せません。送れなかった人は下の一覧で確かめられます。
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
    </>
  )
}
