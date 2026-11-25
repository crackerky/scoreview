import { loadAdmin, requireUser } from '@/admin/session'
import { tournaments } from '@/api/admin'
import { entries } from '@/api/entry'
import type { IdPageProps } from '@/app/_lib/pageProps'
import { decideEntryAction } from '@/app/admin/_actions/entry'
import { readAdminPage } from '@/app/admin/_lib/page'
import { AdminShell, Chip, Hidden } from '@/components/admin/AdminParts'
import { entryPaths } from '@/entry/entry.paths'
import {
  answerText,
  channelLabels,
  dateTimeText,
  registeredText,
  statusLabels,
  statusTones,
} from '@/entry/entry.utils'

// 申込 1 件。中身(チーム、代表、メンバー、質問への答え)を確かめて、承認、差し戻し、取消をする。
// 承認すると部門の最後にチームができ、メンバーが選手になる。承認と差し戻しは申込中だけ、取消は承認の
// 前だけ(承認したチームはチームの棄権で外す。api が決める)。
export default async function EntryPage(props: IdPageProps) {
  const { id, notice } = await readAdminPage(props)
  const { user, token } = await requireUser('admin')
  const detail = await loadAdmin(() => entries.get(token, id))
  const [tournament, questionList] = await loadAdmin(() =>
    Promise.all([
      tournaments.get(token, detail.tournamentId),
      entries.questions(token, detail.tournamentId),
    ]),
  )
  const back = entryPaths.detail(id)
  const questions = new Map(questionList.questions.map((question) => [question.id, question]))
  // メンバー向けの質問は、メンバーの表の列にする(本人が LINE で登録したときに答える)
  const memberQuestions = questionList.questions.filter((question) => question.scope === 'member')
  const pending = detail.status === 'pending'
  const cancellable = pending || detail.status === 'returned'
  return (
    <AdminShell
      user={user}
      title={detail.fullName}
      kicker={`${detail.division.name.ja} ・ ${dateTimeText(detail.submittedAt)} ${channelLabels[detail.channel]}`}
      notice={notice}
      crumbs={[
        { href: `/admin/tournaments/${detail.tournamentId}`, label: tournament.name.ja },
        { href: entryPaths.list(detail.tournamentId), label: '申込' },
      ]}
      primary={pending ? { label: '承認する', form: 'entry-approve' } : undefined}
    >
      <p>
        <Chip tone={statusTones[detail.status]}>{statusLabels[detail.status]}</Chip>{' '}
        {detail.status === 'approved' && detail.teamId !== null ? (
          <a href={`/admin/teams/${detail.teamId}`}>部門のチームを開く</a>
        ) : null}
      </p>
      {detail.status === 'returned' ? (
        <p className="adm-notice error">差し戻しの理由: {detail.returnNote}</p>
      ) : null}
      {detail.channel === 'line' && cancellable ? (
        <p className="adm-muted">
          LINE からの申込です。承認、差し戻し、取消は、公式 LINE
          で代表に知らせます(差し戻しの理由もそのまま届きます)。
        </p>
      ) : null}

      <h2 className="adm-section-title">チームと代表</h2>
      <dl className="adm-summary">
        <dt>チーム名(正式)</dt>
        <dd>{detail.fullName}</dd>
        <dt>チーム名(短縮)</dt>
        <dd>{detail.shortName}</dd>
        <dt>代表</dt>
        <dd>{detail.repName}</dd>
        <dt>電話番号</dt>
        <dd>{detail.repPhone || '(なし)'}</dd>
        <dt>メールアドレス</dt>
        <dd>{detail.repEmail || '(なし)'}</dd>
      </dl>

      <h2 className="adm-section-title">
        メンバー({registeredText(detail.members, detail.registered)})
      </h2>
      {detail.inviteUrl && detail.status !== 'cancelled' ? (
        <p className="adm-muted">
          メンバーの招待リンク(代表に渡すと、メンバーが各自の LINE から登録できます):
          <br />
          <a href={detail.inviteUrl}>{detail.inviteUrl}</a>
        </p>
      ) : null}
      {detail.memberList.length === 0 ? (
        <p className="adm-muted">メンバーはまだ書かれていません。</p>
      ) : (
        <table className="adm-table">
          <thead>
            <tr>
              <th>背番号</th>
              <th>名前</th>
              <th>本人登録</th>
              {memberQuestions.map((question) => (
                <th key={question.id}>{question.label}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {detail.memberList.map((member) => {
              const own = new Map(member.answers.map((answer) => [answer.questionId, answer.value]))
              return (
                <tr key={member.id}>
                  <td className="adm-num">{member.number}</td>
                  <td>{member.name}</td>
                  <td>{member.registered ? '済み' : 'まだ'}</td>
                  {memberQuestions.map((question) => {
                    const value = own.get(question.id)
                    return (
                      <td key={question.id}>
                        {value === undefined ? '' : answerText(question, value)}
                      </td>
                    )
                  })}
                </tr>
              )
            })}
          </tbody>
        </table>
      )}

      {detail.answers.length > 0 ? (
        <>
          <h2 className="adm-section-title">質問への答え</h2>
          <table className="adm-table">
            <tbody>
              {detail.answers.map((answer) => {
                const question = questions.get(answer.questionId)
                return question ? (
                  <tr key={answer.questionId}>
                    <th scope="row">{question.label}</th>
                    <td>{answerText(question, answer.value)}</td>
                  </tr>
                ) : null
              })}
            </tbody>
          </table>
        </>
      ) : null}

      {pending ? (
        <>
          <form id="entry-approve" action={decideEntryAction}>
            <Hidden values={{ back, id, action: 'approve', note: '' }} />
          </form>
          <h2 className="adm-section-title">差し戻す</h2>
          <form action={decideEntryAction} className="adm-stack">
            <Hidden values={{ back, id, action: 'return' }} />
            <label className="adm-field">
              <span>代表に直してほしいこと</span>
              <textarea name="note" rows={3} required />
            </label>
            <button type="submit" className="secondary">
              差し戻す
            </button>
          </form>
        </>
      ) : null}
      {cancellable ? (
        <form action={decideEntryAction} className="adm-actions">
          <Hidden values={{ back, id, action: 'cancel', note: '' }} />
          <label className="adm-check">
            <input type="checkbox" name="confirm" required />
            <span>この申込を取り消す</span>
          </label>
          <button type="submit" className="danger">
            取り消す
          </button>
        </form>
      ) : null}
    </AdminShell>
  )
}
