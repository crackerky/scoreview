import { randomUUID } from 'node:crypto'
import { DEV_SITE_URL } from '@/admin/relay.utils'
import { loadAdmin, requireUser } from '@/admin/session'
import { divisions, tournaments } from '@/api/admin'
import { notifications } from '@/api/notify'
import type { IdPageProps } from '@/app/_lib/pageProps'
import { readAdminPage } from '@/app/admin/_lib/page'
import { AdminShell, Chip, type PrimaryAction, Stat } from '@/components/admin/AdminParts'
import { NotificationComposer } from '@/components/admin/NotificationComposer'
import { notifyPaths } from '@/notify/notify.paths'
import {
  audienceFromQuery,
  draftStorageKey,
  localTimeToIso,
  parsePreset,
  presetBody,
  presetTime,
  quotaState,
  type ReminderPreset,
  reminderPresets,
  scheduleText,
  scorerRequestBody,
  scorerRequestButtons,
  timingFromQuery,
} from '@/notify/notify.utils'

// 公式 LINE で送る。上で宛先の条件と送る時刻を選んで人数を確かめ、下で文を書いて送る(すぐか、日時を
// 決めて予約)。送る画面を開くたびに作る requestKey で、押し直しや再送でも 1 回しか送らない(api が弾く)。
// 前日と当日の案内は、大会の日から送る時刻を決め、文のひな形を入れた状態で開く(preset)。
export default async function NewNotificationPage(props: IdPageProps) {
  const { id, notice } = await readAdminPage(props)
  const query = await props.searchParams
  const audience = audienceFromQuery(query)
  const timing = timingFromQuery(query)
  const preset = parsePreset(query.preset)
  const { user, token } = await requireUser('admin')
  const [tournament, divisionList, preview] = await loadAdmin(() =>
    Promise.all([
      tournaments.get(token, id),
      divisions.list(token, id),
      notifications.preview(token, id, audience),
    ]),
  )
  const quota = quotaState(preview.quota, preview.reachable)
  const sendAt = timing.kind === 'later' ? localTimeToIso(timing.at) : null
  const formId = 'notification-form'
  const siteUrl = (process.env.PUBLIC_SITE_URL || DEV_SITE_URL).replace(/\/$/, '')
  const presetLink = (name: ReminderPreset) =>
    tournament.startDate
      ? notifyPaths.createFor(
          id,
          { kind: 'participants', status: '', divisionId: null },
          { kind: 'later', at: presetTime(tournament.startDate, name) },
          name,
        )
      : null
  return (
    <AdminShell
      user={user}
      title="公式 LINE で送る"
      notice={notice}
      crumbs={[
        { href: `/admin/tournaments/${id}`, label: tournament.name.ja },
        { href: notifyPaths.list(id), label: '送った記録' },
      ]}
      primary={primaryAction(formId, preview, quota.enough, timing.kind === 'later', sendAt)}
    >
      <p className="adm-links">
        <a href={notifyPaths.scorerRequest(id)}>記録係を選んでもらう</a>
        {' ・ '}
        {(['eve', 'day'] as const).map((name, index) => {
          const href = presetLink(name)
          return (
            <span key={name}>
              {index > 0 ? ' ・ ' : ''}
              {href ? (
                <a href={href}>{reminderPresets[name].label}を予約する</a>
              ) : (
                <span className="adm-muted">{reminderPresets[name].label}を予約する</span>
              )}
            </span>
          )
        })}
      </p>
      {tournament.startDate ? null : (
        <p className="adm-muted">
          前日と当日の案内は、大会の設定で開催日を決めると、日時を入れた状態で開けます。
        </p>
      )}
      <NotificationComposer
        checked={audience}
        checkedTime={timing}
        divisions={divisionList.divisions.map((division) => ({
          value: String(division.id),
          label: division.name.ja,
        }))}
        heading={preview.heading}
        formId={formId}
        hidden={{ back: notifyPaths.create(id), tournamentId: id, requestKey: randomUUID() }}
        draftKey={draftStorageKey(id)}
        initialBody={
          preset
            ? presetBody(preset, `${siteUrl}/tournaments/${id}`)
            : query.template === 'scorer'
              ? scorerRequestBody
              : ''
        }
        initialButtons={query.template === 'scorer' ? scorerRequestButtons : undefined}
        variables={preview.variables}
        sampleName={preview.sampleName}
      >
        {preview.everyone ? (
          <EveryoneStats reachable={preview.reachable} />
        ) : (
          <div className="adm-stats">
            <Stat
              label={timing.kind === 'later' ? '今の宛先の人数' : '送る人'}
              value={String(preview.reachable)}
            />
            <Stat
              label="ブロック中で送らない人"
              value={String(preview.unreachable)}
              tone={preview.unreachable > 0 ? 'warn' : undefined}
            />
            <Stat label="今回使う配信数" value={`${preview.reachable} 通`} />
          </div>
        )}
        {timing.kind === 'later' ? (
          <p className="adm-muted">
            {quota.text}
            。予約は、送る時刻にそのときの宛先の人数と配信数を確かめて送ります(足りなければ送りません)。
          </p>
        ) : (
          <p className={quota.enough ? 'adm-muted' : 'adm-notice error'}>
            {quota.text}
            {quota.enough ? '' : '。今回の分が足りないので送れません。宛先を絞ってください'}
          </p>
        )}
        {timing.kind === 'later' && sendAt === null ? (
          <p className="adm-notice error">
            送る日時を入れて、「宛先と時刻を確かめる」を押してください。
          </p>
        ) : null}
        {preview.canSend ? null : (
          <p className="adm-notice error">
            公式 LINE の設定が無いので、宛先の確認だけできます。送るには api に LINE
            の設定が要ります。
          </p>
        )}
        {preview.everyone ? (
          <label className="adm-check adm-audience">
            <input type="checkbox" name="confirmAll" required />
            <span>
              公式 LINE の友だち全員
              {preview.reachable > 0 ? `(約 ${preview.reachable} 人)` : ''}
              に送ることを確かめました
              <small className="adm-muted">
                大会に申し込んでいない人にも届きます。送ったあとは取り消せません
              </small>
            </span>
          </label>
        ) : preview.recipients.length === 0 ? (
          <p className="adm-muted">今、この条件に当てはまる人はいません。</p>
        ) : (
          <details className="adm-recipients">
            <summary>宛先 {preview.recipients.length} 人を見る</summary>
            <ul className="adm-rows">
              {preview.recipients.map((recipient, index) => (
                // 同じ名前の人がいるので、並び(api が返す順)を key に使う
                // biome-ignore lint/suspicious/noArrayIndexKey: 宛先の並びはこの描画の間は変わらない
                <li key={index} className="adm-row">
                  <span className="adm-row-main">
                    <span className="adm-row-title">{recipient.name || '名前なし'}</span>
                    <span className="adm-row-meta">{recipient.teams.join('、')}</span>
                  </span>
                  {recipient.reachable ? null : <Chip tone="done">ブロック中</Chip>}
                </li>
              ))}
            </ul>
          </details>
        )}
      </NotificationComposer>
    </AdminShell>
  )
}

/**
 * 画面の上の主な操作。予約なら送る日時を、すぐ送るなら人数をボタンに書く(押すと何が起きるかを
 * ボタンの言葉で分かるように)。送れない状態なら押せなくする。
 */
function primaryAction(
  formId: string,
  preview: { canSend: boolean; reachable: number; everyone: boolean },
  quotaEnough: boolean,
  later: boolean,
  sendAt: string | null,
): PrimaryAction {
  if (later) {
    return preview.canSend && sendAt !== null
      ? { label: `${scheduleText(sendAt)} に送る予約をする`, form: formId }
      : { label: '予約する', form: formId, disabled: true }
  }
  // 友だち全員は、LINE がまだ数えていなくても(見込みが 0)送れる
  if (preview.everyone) {
    return preview.canSend && quotaEnough
      ? {
          label: `友だち全員${preview.reachable > 0 ? `(約 ${preview.reachable} 人)` : ''}に送る`,
          form: formId,
        }
      : { label: '送る', form: formId, disabled: true }
  }
  return preview.canSend && preview.reachable > 0 && quotaEnough
    ? { label: `${preview.reachable} 人に送る`, form: formId }
    : { label: '送る', form: formId, disabled: true }
}

/**
 * 友だち全員の人数の見込み。宛先の人は LINE の一斉送信で決まるので一覧は無く、人数は LINE が数えた
 * 前日までの友だちの数から、ブロックしている人を除いたもの(まだ数えていなければ分からない)。
 */
function EveryoneStats({ reachable }: { reachable: number }) {
  return (
    <>
      <div className="adm-stats">
        <Stat label="送る人(見込み)" value={reachable > 0 ? `約 ${reachable}` : '不明'} />
        <Stat
          label="今回使う配信数(見込み)"
          value={reachable > 0 ? `約 ${reachable} 通` : '不明'}
        />
      </div>
      <p className="adm-muted">
        {reachable > 0
          ? '人数は、LINE が数えた前日までの友だちの数から、ブロックしている人を除いた見込みです。'
          : 'LINE がまだ友だちの数を数えていないので、人数と配信数は送るまで分かりません。'}
      </p>
    </>
  )
}
