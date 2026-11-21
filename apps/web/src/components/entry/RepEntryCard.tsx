import type { EntryDetail, EntryLink } from '@/api/entry.types'
import { registeredText, statusLabels } from '@/entry/entry.utils'
import { type LinkFormState, resubmitValues, shareToLineUrl } from '@/entry/entryLink.utils'
import { TeamEntryForm } from './TeamEntryForm'

// 代表が出した申込 1 件。状態と本人登録の進み具合、メンバーへの招待のボタンを出す。差し戻された申込には、
// 運営からのお願いと、直して送り直すフォームを出す(同じ申込を直すので、招待と本人登録はそのまま残る)。

type Team = NonNullable<EntryLink['team']>

type Props = {
  token: string
  entry: EntryDetail
  team: Team
  resubmit: (state: LinkFormState, formData: FormData) => Promise<LinkFormState>
}

export function RepEntryCard({ token, entry, team, resubmit }: Props) {
  return (
    <section className="entry-card">
      <p className="entry-card-head">
        <span className="entry-card-name">{entry.fullName}</span>
        <span className={`entry-status ${entry.status}`}>{statusLabels[entry.status]}</span>
      </p>
      <p className="entry-muted">
        {entry.division.name.ja} ・ {registeredText(entry.members, entry.registered)}
      </p>
      {entry.status === 'returned' ? (
        <>
          <p className="entry-notice error">運営からのお願い：{entry.returnNote}</p>
          <details className="entry-details">
            <summary>直して送り直す</summary>
            <TeamEntryForm
              token={token}
              divisions={team.divisions}
              questions={team.questions}
              action={resubmit}
              submitLabel="直して送り直す"
              initial={resubmitValues(entry)}
              entryId={entry.id}
              registered={entry.memberList.filter((member) => member.registered)}
            />
          </details>
        </>
      ) : null}
      {entry.inviteUrl && entry.status !== 'cancelled' ? (
        <>
          <a className="entry-button" href={shareToLineUrl(entry.fullName, entry.inviteUrl)}>
            LINE でメンバーに登録のお願いを送る
          </a>
          <p className="entry-muted">
            チームの LINE グループに送ると、メンバーが各自の LINE から登録できます。
          </p>
        </>
      ) : null}
    </section>
  )
}
