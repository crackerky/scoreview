import type { Metadata } from 'next'
import type { EntryLink } from '@/api/entry.types'
import {
  joinTeamAction,
  resubmitTeamEntryAction,
  submitTeamEntryAction,
} from '@/app/entry/[token]/actions'
import { SubmitFeedback } from '@/components/admin/SubmitFeedback'
import { MemberJoinForm } from '@/components/entry/MemberJoinForm'
import { RepEntryCard } from '@/components/entry/RepEntryCard'
import { TeamEntryForm } from '@/components/entry/TeamEntryForm'
import { SiteShell } from '@/components/site/SiteShell'
import { dateTimeText } from '@/entry/entry.utils'
import { loadEntryLink } from '@/entry/entryLink.load'
import { profileValues } from '@/entry/entryLink.utils'
import '@/styles/entry.css'

// 本人用のリンクで開く画面(docs/line-integration.md「本人の確かめ方」)。公式 LINE に「エントリー」と
// 送った代表には大会への申込と出した申込を、招待リンクで「参加」を送ったメンバーにはチームへの登録を
// 出す。ログインは無く、URL の合言葉だけで本人を確かめるので、検索に載せない。

type Props = {
  params: Promise<{ token: string }>
  searchParams: Promise<Record<string, string | string[] | undefined>>
}

export async function generateMetadata(props: Props): Promise<Metadata> {
  const { token } = await props.params
  const view = await loadEntryLink(token)
  const title = view.team
    ? `${view.team.tournament.name} の申込`
    : `${view.member?.team.fullName ?? ''} のメンバー登録`
  return { title: `${title} - SCORE VIEW`, robots: { index: false, follow: false } }
}

/** 送った後の知らせ(?done=) */
const doneTexts: Readonly<Record<string, string>> = {
  team: '申込を受け付けました。運営が確かめるまでお待ちください。下のボタンから、メンバーに登録のお願いを送れます。',
  resubmit: '直した申込を受け付けました。運営が確かめるまでお待ちください。',
  member: '登録しました。大会のお知らせは、この公式 LINE に届きます。',
}

export default async function EntryLinkPage(props: Props) {
  const [{ token }, query] = await Promise.all([props.params, props.searchParams])
  const view = await loadEntryLink(token)
  const done = typeof query.done === 'string' ? (doneTexts[query.done] ?? null) : null
  return (
    <SiteShell>
      <SubmitFeedback />
      {view.team ? <TeamSection token={token} team={view.team} done={done} /> : null}
      {view.member ? <MemberSection token={token} member={view.member} done={done} /> : null}
    </SiteShell>
  )
}

function Done({ text }: { text: string | null }) {
  return text ? (
    <p className="entry-notice ok" role="status">
      {text}
    </p>
  ) : null
}

function TeamSection({
  token,
  team,
  done,
}: {
  token: string
  team: NonNullable<EntryLink['team']>
  done: string | null
}) {
  return (
    <>
      <p className="entry-kicker">{team.tournament.name}</p>
      <h1 className="site-title">チームの申込</h1>
      <Done text={done} />
      {team.mine.map((entry) => (
        <RepEntryCard
          key={entry.id}
          token={token}
          entry={entry}
          team={team}
          resubmit={resubmitTeamEntryAction}
        />
      ))}
      {team.open ? (
        <>
          <h2 className="site-heading">
            {team.mine.length > 0 ? '別のチームを申し込む' : '申込の内容'}
          </h2>
          <TeamEntryForm
            token={token}
            divisions={team.divisions}
            questions={team.questions}
            action={submitTeamEntryAction}
            submitLabel="申し込む"
            initial={profileValues(team.profile)}
            pastTeams={team.pastTeams}
          />
        </>
      ) : (
        <p className="entry-notice error">この大会の新しい申込は締め切りました。</p>
      )}
      <p className="entry-muted">
        このリンクはあなた専用です。{dateTimeText(team.expiresAt)} まで使えます。
      </p>
    </>
  )
}

function MemberSection({
  token,
  member,
  done,
}: {
  token: string
  member: NonNullable<EntryLink['member']>
  done: string | null
}) {
  return (
    <>
      <p className="entry-kicker">
        {member.tournamentName} ・ {member.team.division.name.ja}
      </p>
      <h1 className="site-title">{member.team.fullName} のメンバー登録</h1>
      <Done text={done} />
      {member.team.status === 'cancelled' ? (
        <p className="entry-notice error">このチームの申込は取り消されています。</p>
      ) : (
        <MemberJoinForm token={token} member={member} action={joinTeamAction} />
      )}
      <p className="entry-muted">
        このリンクはあなた専用です。{dateTimeText(member.expiresAt)} まで使えます。
      </p>
    </>
  )
}
