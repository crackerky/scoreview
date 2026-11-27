import type { Metadata } from 'next'
import type { EntryLink } from '@/api/entry.types'
import {
  chooseScorerAction,
  joinTeamAction,
  resubmitTeamEntryAction,
  scorerSignInAction,
  submitTeamEntryAction,
} from '@/app/entry/[token]/actions'
import { SubmitFeedback } from '@/components/admin/SubmitFeedback'
import { MemberJoinForm } from '@/components/entry/MemberJoinForm'
import { RepEntryCard } from '@/components/entry/RepEntryCard'
import { ScorerSignIn } from '@/components/entry/ScorerSignIn'
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
    : view.scorer
      ? `${view.scorer.teamName} の記録係`
      : `${view.member?.team.fullName ?? ''} のメンバー登録`
  return { title: `${title} - SCORE VIEW`, robots: { index: false, follow: false } }
}

/** 送った後の知らせ(?done=) */
const doneTexts: Readonly<Record<string, string>> = {
  team: '申込を受け付けました。運営が確かめるまでお待ちください。次に、下のボタンからメンバーに招待リンクを送ってください。メンバーは各自の LINE から自分の名前を入れて登録します。代表の方も選手として出るなら、同じ招待リンクから登録してください。',
  resubmit: '直した申込を受け付けました。運営が確かめるまでお待ちください。',
  member: '登録しました。大会のお知らせは、この公式 LINE に届きます。',
  scorer:
    '記録係を決めました。当日は、その人が公式 LINE で「記録」と送ると、記録の画面が開きます。',
  'scorer-cleared': '記録係を外しました。',
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
      {view.scorer ? (
        <ScorerSection token={token} scorer={view.scorer} match={matchOfQuery(query.match)} />
      ) : null}
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
          chooseScorer={chooseScorerAction}
        />
      ))}
      {team.open ? (
        team.mine.length > 0 ? (
          // 出した申込があるなら、新しい申込の欄は畳む(申込の直後や直し待ちのときに、大きな空の欄が
          // 続くと、もう一度書く必要があるように見え、同じチームの申込を作りやすいため)
          <details className="entry-details">
            <summary>別のチームも申し込む</summary>
            <TeamEntryForm
              token={token}
              divisions={team.divisions}
              questions={team.questions}
              action={submitTeamEntryAction}
              submitLabel="申し込む"
              initial={profileValues(team.profile)}
              pastTeams={team.pastTeams}
            />
          </details>
        ) : (
          <>
            <p className="entry-lead">
              代表の方の申込です。チームと代表の方のことを入れてください。メンバーは、申し込んだあとに届く招待リンクから、各自で登録します。
            </p>
            <h2 className="site-heading">申込の内容</h2>
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
        )
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
      {member.mine && !done && member.team.status !== 'cancelled' ? (
        <p className="entry-notice ok">
          このチームに登録済みです。直すときは、書き換えてから登録を直すを押してください。
        </p>
      ) : member.mine ? null : (
        <p className="entry-lead">メンバーの方の登録です。自分の名前を入れて、登録してください。</p>
      )}
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

/** 公式 LINE の試合のボタンから開いたときの試合(?match=。数でなければ null) */
function matchOfQuery(value: string | string[] | undefined): number | null {
  const id = typeof value === 'string' && /^[1-9][0-9]{0,15}$/.test(value) ? Number(value) : null
  return id
}

// 記録係のリンクで開く画面。公式 LINE の試合のボタン(?match=)から開いたら、その試合のスコアの画面に、
// 一覧のボタンから開いたら担当の試合の一覧に、そのまま入る(ScorerSignIn が開いたときに送る)。
function ScorerSection({
  token,
  scorer,
  match,
}: {
  token: string
  scorer: NonNullable<EntryLink['scorer']>
  match: number | null
}) {
  return (
    <>
      <p className="entry-kicker">{scorer.tournamentName}</p>
      <h1 className="site-title">{scorer.teamName} の記録係</h1>
      <p>
        {scorer.memberName} さん、
        {match === null ? '担当の試合の一覧' : '試合のスコアの画面'}
        を開いています。開かないときは、下のボタンを押してください。
      </p>
      <ScorerSignIn token={token} match={match} action={scorerSignInAction} />
      <p className="entry-muted">
        このリンクはあなた専用です。{dateTimeText(scorer.expiresAt)} まで使えます。切れたら、公式
        LINE のメニューの「スコアを入れる」を押してください。
      </p>
    </>
  )
}
