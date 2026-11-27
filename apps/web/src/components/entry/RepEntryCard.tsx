import type { EntryDetail, EntryLink } from '@/api/entry.types'
import { registeredText, statusLabels } from '@/entry/entry.utils'
import { type LinkFormState, resubmitValues, shareToLineUrl } from '@/entry/entryLink.utils'
import { CopyField } from './CopyField'
import { RemoveMember } from './RemoveMember'
import { ScorerForm } from './ScorerForm'
import { TeamEntryForm } from './TeamEntryForm'

// 代表が出した申込 1 件。状態と、メンバーが LINE で登録した進み具合(まだの人の名前)、メンバーへの招待の
// ボタンを出す。参加が決まったら、記録係を選ぶ欄を出す。直し待ちの申込には、運営からのお願いのすぐ下に、直して送り直すフォームを開いて出す
// (畳むと、下の別のチームの申込の欄に書いて、同じチームの申込を作ってしまいやすいため。同じ申込を直す
// ので、招待とメンバーの登録はそのまま残る)。

type Team = NonNullable<EntryLink['team']>

type Props = {
  token: string
  entry: EntryDetail
  team: Team
  resubmit: (state: LinkFormState, formData: FormData) => Promise<LinkFormState>
  chooseScorer: (state: LinkFormState, formData: FormData) => Promise<LinkFormState>
  removeMember: (state: LinkFormState, formData: FormData) => Promise<LinkFormState>
}

export function RepEntryCard({ token, entry, team, resubmit, chooseScorer, removeMember }: Props) {
  const waiting = entry.memberList.filter((member) => !member.registered)
  const uninvited = entry.memberList.filter((member) => member.registered && member.withoutInvite)
  return (
    <section className="entry-card">
      <p className="entry-card-head">
        <span className="entry-card-name">{entry.fullName}</span>
        <span className={`entry-status ${entry.status}`}>{statusLabels[entry.status]}</span>
      </p>
      <p className="entry-muted">
        {entry.division.name.ja} ・ {registeredText(entry.members, entry.registered)}
      </p>
      {entry.status !== 'cancelled' && waiting.length > 0 ? (
        <p className="entry-muted">
          まだ LINE で登録していない人：{waiting.map((member) => member.name).join('、')}
        </p>
      ) : null}
      {entry.status !== 'cancelled' && uninvited.length > 0 ? (
        <UninvitedMembers
          token={token}
          entryId={entry.id}
          members={uninvited}
          removable={entry.status === 'pending' || entry.status === 'returned'}
          action={removeMember}
        />
      ) : null}
      {entry.status === 'returned' ? (
        <>
          <p className="entry-notice error">運営からのお願い：{entry.returnNote}</p>
          <h3 className="entry-subheading">直して送り直す</h3>
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
        </>
      ) : null}
      {entry.status === 'approved' ? (
        <ScorerForm token={token} entry={entry} action={chooseScorer} />
      ) : null}
      {entry.inviteUrl && entry.status !== 'cancelled' ? (
        <>
          <a
            className={`entry-button${entry.status === 'returned' ? ' secondary' : ''}`}
            href={shareToLineUrl(entry.fullName, entry.inviteUrl)}
          >
            LINE でメンバーに登録のお願いを送る
          </a>
          <p className="entry-muted">
            チームの LINE グループに送ると、メンバーが各自の LINE から登録できます。
          </p>
          <CopyField
            label="招待リンク(コピーして、ほかの場所にも貼り付けられます)"
            value={entry.inviteUrl}
          />
        </>
      ) : null}
    </section>
  )
}

// 招待リンクを使わずに(公式 LINE から大会を選んでチームを探して)登録した人。違うチームの人なら、参加が
// 決まる前は代表が外せる(押し間違えないよう、開いてから外す)。参加が決まった後は選手ができているので、
// 運営に頼んでもらう。
function UninvitedMembers({
  token,
  entryId,
  members,
  removable,
  action,
}: {
  token: string
  entryId: number
  members: EntryDetail['memberList']
  removable: boolean
  action: (state: LinkFormState, formData: FormData) => Promise<LinkFormState>
}) {
  return (
    <div className="entry-uninvited">
      <p className="entry-muted">
        招待リンクを使わずに登録した人です。チームの人でなければ
        {removable ? '外してください。' : '、運営に連絡してください。'}
      </p>
      <ul className="entry-list">
        {members.map((member) => (
          <li key={member.id}>
            {member.number ? `${member.number} ` : ''}
            {member.name}
            {removable ? (
              <RemoveMember token={token} entryId={entryId} member={member} action={action} />
            ) : null}
          </li>
        ))}
      </ul>
    </div>
  )
}
