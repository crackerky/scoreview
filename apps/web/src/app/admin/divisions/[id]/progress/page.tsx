import { decisionReasons, groupStateLabels, roundUpToFive } from '@/admin/format.utils'
import { loadAdmin, requireUser } from '@/admin/session'
import { divisions, tournaments } from '@/api/admin'
import { formats } from '@/api/format'
import type { GroupProgress, Progress, ProgressMatch, Todo } from '@/api/format.types'
import type { IdPageProps } from '@/app/_lib/pageProps'
import {
  alignTeamsAction,
  confirmGroupAction,
  confirmStageAction,
  decideRankAction,
  shiftScheduleAction,
} from '@/app/admin/_actions/format'
import { readAdminPage } from '@/app/admin/_lib/page'
import { AdminShell, Hidden } from '@/components/admin/AdminParts'
import '@/styles/format-admin.css'

// 本部の進行画面(docs/operations/operator-experience.md「当日」1)。上から「いまやること」、
// 試合中、次の時間帯、遅れの調整、プールの順位と勝ち上がり条件、最終順位の順に並べる。

export default async function ProgressPage(props: IdPageProps) {
  const { id, notice } = await readAdminPage(props)
  const { user, token } = await requireUser('any')
  const division = await loadAdmin(() => divisions.get(token, id))
  const [tournament, progress] = await loadAdmin(() =>
    Promise.all([tournaments.get(token, division.tournamentId), formats.progress(token, id)]),
  )
  const back = `/admin/divisions/${id}/progress`
  const isAdmin = user.role === 'admin'
  const crumbs = [
    { href: `/admin/tournaments/${tournament.id}`, label: tournament.name.ja },
    { href: `/admin/divisions/${id}`, label: division.name.ja },
  ]
  return (
    <AdminShell user={user} title={`${division.name.ja}: 本部`} notice={notice} crumbs={crumbs}>
      <p className="adm-actions">
        <a href={back}>最新にする</a>
        <a href={`/pscore/scv_division/open/${id}`}>閲覧ページの成績</a>
        {isAdmin ? <a href={`/admin/divisions/${id}/format`}>大会の形</a> : null}
      </p>
      <p className="adm-muted">{progress.title}</p>

      <h2>いまやること</h2>
      <TodoList progress={progress} back={back} divisionId={id} isAdmin={isAdmin} />

      <h2>試合中</h2>
      <MatchList matches={progress.live} empty="試合中の試合はありません" isAdmin={isAdmin} />
      <h2>次の時間帯</h2>
      <MatchList matches={progress.next} empty="これから始まる試合はありません" isAdmin={isAdmin} />

      {isAdmin && progress.courts.length > 0 ? (
        <>
          <h2>遅れの調整</h2>
          <form action={shiftScheduleAction} className="hq-inline">
            <Hidden values={{ back, divisionId: id }} />
            <label className="adm-field">
              <span>どのコート</span>
              <select name="court" defaultValue="">
                <option value="">全体</option>
                {progress.courts.map((court) => (
                  <option key={court} value={court}>
                    コート {court}
                  </option>
                ))}
              </select>
            </label>
            <label className="adm-field">
              <span>遅らせる分数(早めるときは -)</span>
              <input type="number" name="minutes" defaultValue={10} step={5} />
            </label>
            <button type="submit">まだ始まっていない試合をずらす</button>
          </form>
        </>
      ) : null}

      <h2>プールとトーナメント</h2>
      <div className="hq-groups">
        {progress.groups.map((group) => (
          <GroupCard key={group.id} group={group} back={back} isAdmin={isAdmin} />
        ))}
      </div>

      {progress.final.length > 0 ? (
        <>
          <h2>最終順位(決まった分)</h2>
          <ol className="adm-list">
            {progress.final.map((row) => (
              <li key={row.team.id}>
                {row.rank} 位{row.tied ? '(同順位)' : ''} {row.team.name.ja}
              </li>
            ))}
          </ol>
        </>
      ) : null}

      <h2>順位の決め方と勝ち上がり</h2>
      <div className="fmt-rules">
        {[...progress.ruleText, ...progress.advancement].map((line) => (
          <p key={line}>{line}</p>
        ))}
      </div>
    </AdminShell>
  )
}

function TodoList({
  progress,
  back,
  divisionId,
  isAdmin,
}: {
  progress: Progress
  back: string
  divisionId: number
  isAdmin: boolean
}) {
  if (progress.todo.length === 0) {
    return <p className="fmt-ok">いまやることはありません。</p>
  }
  const groupById = new Map(progress.groups.map((group) => [group.id, group]))
  return (
    <ul className="hq-todo">
      {progress.todo.map((todo) => (
        <li
          key={`${todo.kind}-${todo.groupId}-${todo.stageId}-${todo.matchId}-${todo.court}`}
          className={todo.kind}
        >
          <strong>{todo.text}</strong>
          {isAdmin ? (
            <TodoAction
              todo={todo}
              group={todo.groupId === null ? undefined : groupById.get(todo.groupId)}
              back={back}
              divisionId={divisionId}
            />
          ) : null}
        </li>
      ))}
    </ul>
  )
}

function TodoAction({
  todo,
  group,
  back,
  divisionId,
}: {
  todo: Todo
  group: GroupProgress | undefined
  back: string
  divisionId: number
}) {
  switch (todo.kind) {
    case 'tie':
      return group ? <TieForm group={group} back={back} /> : null
    case 'confirm_group':
      return group ? (
        <>
          <RankTable group={group} />
          <form action={confirmGroupAction}>
            <Hidden values={{ back, groupId: group.id }} />
            <button type="submit">この順位で確定する</button>
          </form>
        </>
      ) : null
    case 'confirm_stage':
      return (
        <form action={confirmStageAction}>
          <Hidden values={{ back, stageId: todo.stageId ?? 0 }} />
          <button type="submit">段を確定する</button>
        </form>
      )
    case 'overdue':
      return (
        <p className="adm-actions">
          <a href={`/admin/matches/${todo.matchId}/result?next=${encodeURIComponent(back)}`}>
            結果を入れる
          </a>
          <a href={`/admin/matches/${todo.matchId}/score`}>スコア入力画面</a>
        </p>
      )
    case 'court_delay':
      return (
        <form action={shiftScheduleAction} className="hq-inline">
          <Hidden values={{ back, divisionId, court: todo.court ?? '' }} />
          <label className="adm-field">
            <span>遅らせる分数</span>
            <input
              type="number"
              name="minutes"
              defaultValue={roundUpToFive(todo.minutes)}
              step={5}
            />
          </label>
          <button type="submit">このコートを遅らせる</button>
        </form>
      )
    case 'mismatch':
      return (
        <form action={alignTeamsAction}>
          <Hidden values={{ back, matchId: todo.matchId ?? 0 }} />
          <button type="submit" className="danger">
            順位どおりのチームに差し替える(結果はそのまま)
          </button>
        </form>
      )
  }
}

/** 未決の同順位を、上から順に選んで決めるフォーム */
function TieForm({ group, back }: { group: GroupProgress; back: string }) {
  const tie = group.ties[0]
  if (!tie) {
    return null
  }
  return (
    <form action={decideRankAction}>
      <Hidden values={{ back, groupId: group.id, size: tie.length }} />
      <RankTable group={group} />
      <p className="adm-muted">
        規程の基準をすべて比べても並んでいます。ブリックマーク投げなどの結果を、上の順位から選んでください。
      </p>
      <div className="hq-inline">
        {tie.map((team, index) => (
          <label key={team.id} className="adm-field">
            <span>{index + 1} 番目</span>
            <select name={`order-${index + 1}`} defaultValue={team.id}>
              {tie.map((option) => (
                <option key={option.id} value={option.id}>
                  {option.name.ja}
                </option>
              ))}
            </select>
          </label>
        ))}
        <label className="adm-field">
          <span>決め方</span>
          <select name="reason" defaultValue={decisionReasons[0]}>
            {decisionReasons.map((reason) => (
              <option key={reason} value={reason}>
                {reason}
              </option>
            ))}
          </select>
        </label>
        <button type="submit">この順で決める</button>
      </div>
    </form>
  )
}

function RankTable({ group }: { group: GroupProgress }) {
  if (group.rows.length === 0) {
    return null
  }
  return (
    <table className="hq-rank">
      <thead>
        <tr>
          <th>順位</th>
          <th>チーム</th>
          <th className="num">勝点</th>
          <th className="num">勝-分-負</th>
          <th className="num">得失</th>
        </tr>
      </thead>
      <tbody>
        {group.rows.map((row) => (
          <tr key={row.team.id} className={row.unresolved ? 'tied' : ''}>
            <td>{row.unresolved ? `${row.rank}(未決)` : row.rank}</td>
            <td>
              {row.team.name.ja}
              {row.reason ? <div className="hq-reason">{row.reason}</div> : null}
            </td>
            <td className="num">{row.points}</td>
            <td className="num">
              {row.wins}-{row.draws}-{row.losses}
            </td>
            <td className="num">{row.goalsFor - row.goalsAgainst}</td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}

function GroupCard({
  group,
  back,
  isAdmin,
}: {
  group: GroupProgress
  back: string
  isAdmin: boolean
}) {
  return (
    <section className="hq-group">
      <h3>
        <a href={`/pscore/scv_group/open/${group.id}`}>{group.name}</a>
        <span className={`hq-state ${group.state}`}>{groupStateLabels[group.state]}</span>
      </h3>
      {group.kind === 'pool' ? (
        <RankTable group={group} />
      ) : (
        <p className="adm-muted">
          {group.slots.map((slot) => slot.team?.name.ja ?? slot.label).join('、')}
        </p>
      )}
      {group.conditions.length > 0 ? (
        <ul className="hq-cond">
          {group.conditions.map((condition) => (
            <li key={condition.team.id}>
              {condition.team.name.ja}: {condition.text}
            </li>
          ))}
        </ul>
      ) : null}
      {group.remaining > 0 && group.state !== 'waiting' ? (
        <p className="adm-muted">残り {group.remaining} 試合</p>
      ) : null}
      {isAdmin && group.kind === 'pool' && group.state === 'ready' ? (
        <form action={confirmGroupAction}>
          <Hidden values={{ back, groupId: group.id }} />
          <button type="submit">この順位で確定する</button>
        </form>
      ) : null}
      {isAdmin && group.kind === 'pool' && group.state === 'playing' ? (
        <details>
          <summary>時間切れで未消化のまま確定する</summary>
          <form action={confirmGroupAction}>
            <Hidden values={{ back, groupId: group.id }} />
            <label className="adm-check">
              <input type="checkbox" name="unplayed" required />
              <span>残りの {group.remaining} 試合を行わずに、今の順位で確定する</span>
            </label>
            <button type="submit" className="danger">
              未消化のまま確定
            </button>
          </form>
        </details>
      ) : null}
      {group.confirmedUnplayed ? <p className="adm-muted">未消化のまま確定しました</p> : null}
    </section>
  )
}

function MatchList({
  matches,
  empty,
  isAdmin,
}: {
  matches: ProgressMatch[]
  empty: string
  isAdmin: boolean
}) {
  if (matches.length === 0) {
    return <p className="adm-muted">{empty}</p>
  }
  return (
    <ul className="adm-matches">
      {matches.map((match) => (
        <li key={match.id} className={match.status}>
          <a href={`/admin/matches/${match.id}/score`}>
            <span className="when">
              {match.time ?? '--:--'} {match.field ? `コート ${match.field}` : ''} {match.code}
            </span>
            <span className="teams">
              {match.left.team?.name.ja ?? match.left.label} {match.left.score ?? ''} -{' '}
              {match.right.score ?? ''} {match.right.team?.name.ja ?? match.right.label}
            </span>
            <span className="sub">{match.groupName}</span>
          </a>
          {isAdmin ? (
            <a href={`/admin/matches/${match.id}/result`} className="adm-muted">
              結果を入れる・直す
            </a>
          ) : null}
        </li>
      ))}
    </ul>
  )
}
