import { runsDayOf } from '@/admin/access.utils'
import { operatorErrorText } from '@/admin/errorText.utils'
import { decisionReasons, groupStateLabels, roundUpToFive } from '@/admin/format.utils'
import { describeSide } from '@/admin/formatWizard.utils'
import {
  hqTabOf,
  hqTabs,
  sharedRanks,
  shiftSentence,
  suspendedMinutes,
  updatedAtText,
} from '@/admin/hq.utils'
import { loadAdmin, requireUser } from '@/admin/session'
import { divisions, tournaments } from '@/api/admin'
import { ApiError } from '@/api/client'
import type { Suspension } from '@/api/common.types'
import { formats } from '@/api/format'
import type { GroupProgress, Progress, ProgressMatch, Todo } from '@/api/format.types'
import type { IdPageProps } from '@/app/_lib/pageProps'
import {
  alignTeamsAction,
  confirmGroupAction,
  confirmStageAction,
  decideRankAction,
} from '@/app/admin/_actions/format'
import { resumeAction, shiftScheduleAction, suspendAction } from '@/app/admin/_actions/hq'
import { readAdminPage } from '@/app/admin/_lib/page'
import { AdminShell, Hidden } from '@/components/admin/AdminParts'
import { AutoRefresh } from '@/components/admin/AutoRefresh'
import { suspensionText } from '@/components/suspension.utils'
import '@/styles/format-admin.css'

// 本部の進行画面(docs/operations/operator-experience.md「当日」1)。「いまやること」を緊急度の順に
// 出し、試合中と次の時間帯、プールとトーナメント、遅れの調整を続ける。スマートフォンではタブで
// 分け(レビュー M7)、30 秒ごとに自動で取り直す(M3)。中断(H2)はどのタブでも上に出す。

type Query = Record<string, string | string[] | undefined>

function pick(query: Query, key: string): string {
  const value = query[key]
  return typeof value === 'string' ? value : ''
}

export default async function ProgressPage(props: IdPageProps) {
  const { id, notice } = await readAdminPage(props)
  const query = await props.searchParams
  const { user, token } = await requireUser('any')
  const division = await loadAdmin(() => divisions.get(token, id))
  const [tournament, progress] = await loadAdmin(() =>
    Promise.all([tournaments.get(token, division.tournamentId), formats.progress(token, id)]),
  )
  const back = `/admin/divisions/${id}/progress`
  // 本部の操作(知らせの作業、確定、締め)は本部スタッフもできる。大会の形のリンクは管理者だけ
  const isAdmin = runsDayOf(user)
  const canEditFormat = user.role === 'admin'
  const tab = hqTabOf(pick(query, 'tab'))
  const now = new Date(progress.now)
  const sharedFinal = sharedRanks(progress.final)
  // 締めた部門は締めの取り消しが無いので、締めと中断の操作を出さない(中断中なら再開は出す)
  const closed = progress.closedAt !== null
  const crumbs = [
    { href: `/admin/tournaments/${tournament.id}`, label: tournament.name.ja },
    { href: `/admin/divisions/${id}`, label: division.name.ja },
  ]
  return (
    <AdminShell user={user} title={`${division.name.ja}: 本部`} notice={notice} crumbs={crumbs}>
      <p className="adm-actions hq-head">
        <AutoRefresh updatedAt={updatedAtText(now)} />
        <a href={`${back}?tab=${tab}`}>今すぐ更新</a>
        <a href={`/pscore/scv_division/open/${id}`}>閲覧ページの成績</a>
        {canEditFormat ? <a href={`/admin/divisions/${id}/format`}>大会の形</a> : null}
      </p>
      <p className="adm-muted">{progress.title}</p>

      <SuspensionPanel
        suspension={progress.suspension}
        tournamentId={tournament.id}
        back={back}
        now={now}
        isAdmin={isAdmin}
        closed={closed}
      />

      <nav className="hq-tabs" aria-label="本部画面の切り替え">
        {hqTabs.map((item) => (
          <a
            key={item.id}
            href={`${back}?tab=${item.id}`}
            aria-current={item.id === tab ? 'page' : undefined}
          >
            {item.label}
            {item.id === 'todo' && progress.todo.length > 0 ? `(${progress.todo.length})` : ''}
          </a>
        ))}
      </nav>

      <div className={`hq-sections hq-active-${tab}`}>
        <section className="hq-section" data-tab="todo">
          <h2>いまやること</h2>
          <TodoList progress={progress} back={back} divisionId={id} isAdmin={isAdmin} />
          {isAdmin ? <ClosingLink progress={progress} divisionId={id} /> : null}
        </section>

        <section className="hq-section" data-tab="matches">
          <h2>試合中</h2>
          <MatchList matches={progress.live} empty="試合中の試合はありません" isAdmin={isAdmin} />
          <h2>次の時間帯</h2>
          <MatchList
            matches={progress.next}
            empty="これから始まる試合はありません"
            isAdmin={isAdmin}
          />
        </section>

        <section className="hq-section" data-tab="groups">
          <h2>プールとトーナメント</h2>
          <div className="hq-groups">
            {progress.groups.map((group) => (
              <GroupCard
                key={group.id}
                group={group}
                back={back}
                isAdmin={isAdmin}
                withdrawn={progress.withdrawn}
              />
            ))}
          </div>
          {progress.final.length > 0 ? (
            <>
              <h2>最終順位(決まった分)</h2>
              <ol className="adm-list">
                {progress.final.map((row) => (
                  <li key={row.team.id}>
                    {row.rank} 位{sharedFinal.has(row.rank) ? '(同順位)' : ''} {row.team.name.ja}
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
        </section>

        <section className="hq-section" data-tab="delay">
          <h2>遅れの調整</h2>
          {isAdmin && progress.courts.length > 0 ? (
            <DelayPanel
              progress={progress}
              query={query}
              back={back}
              divisionId={id}
              token={token}
            />
          ) : (
            <p className="adm-muted">時間割の設定が無いか、管理者でないので調整できません。</p>
          )}
        </section>
      </div>
    </AdminShell>
  )
}

function SuspensionPanel({
  suspension,
  tournamentId,
  back,
  now,
  isAdmin,
  closed,
}: {
  suspension: Suspension | null
  tournamentId: number
  back: string
  now: Date
  isAdmin: boolean
  closed: boolean
}) {
  if (suspension !== null) {
    const since = new Date(suspension.since)
    const sinceText = new Intl.DateTimeFormat('ja-JP', {
      timeZone: 'Asia/Tokyo',
      hour: '2-digit',
      minute: '2-digit',
      hourCycle: 'h23',
    }).format(since)
    return (
      <section className="hq-suspended" role="alert">
        <strong>{suspensionText(suspension, 'jp')}</strong>
        <span>({sinceText} から。閲覧ページの上部にも同じ帯を出しています)</span>
        {isAdmin ? (
          <form action={resumeAction} className="hq-inline">
            <Hidden values={{ back, tournamentId }} />
            <label className="adm-field">
              <span>まだ始まっていない試合を遅らせる分数(全部門)</span>
              <input
                type="number"
                name="minutes"
                min={0}
                step={5}
                defaultValue={suspendedMinutes(suspension.since, now)}
              />
            </label>
            <button type="submit">再開する</button>
          </form>
        ) : null}
      </section>
    )
  }
  if (!isAdmin || closed) {
    return null
  }
  return (
    <details className="hq-suspend">
      <summary>雷・雨・暑さで大会を中断する</summary>
      <form action={suspendAction}>
        <Hidden values={{ back, tournamentId }} />
        <fieldset className="adm-field">
          <legend>理由</legend>
          {(
            [
              ['thunder', '雷'],
              ['rain', '雨'],
              ['heat', '暑さ'],
              ['other', 'その他'],
            ] as const
          ).map(([value, label]) => (
            <label key={value} className="adm-check">
              <input type="radio" name="reason" value={value} required />
              <span>{label}</span>
            </label>
          ))}
        </fieldset>
        <div className="hq-inline">
          <label className="adm-field">
            <span>その他の理由(閲覧ページに出ます)</span>
            <input type="text" name="note" maxLength={100} />
          </label>
          <label className="adm-field">
            <span>再開予定(分からなければ空)</span>
            <input type="time" name="resumeAt" />
          </label>
        </div>
        <p className="adm-muted">
          大会の全部門が対象です。再開するときに、中断した分だけまだ始まっていない試合をずらします。
        </p>
        <button type="submit" className="danger">
          中断する
        </button>
      </form>
    </details>
  )
}

function ClosingLink({ progress, divisionId }: { progress: Progress; divisionId: number }) {
  return (
    <section className="hq-box">
      <h2>大会の締め</h2>
      {progress.closedAt ? (
        <p>
          締めました。最終順位は <a href={`/admin/divisions/${divisionId}/close`}>締めの画面</a>{' '}
          で確かめられます。
        </p>
      ) : (
        <p className="adm-muted">
          時間切れで残りの試合をまとめて未消化にし、決まらない順位を同順位にして、最終順位を公開します。
        </p>
      )}
      {progress.closedAt ? null : (
        <p className="adm-actions">
          <a href={`/admin/divisions/${divisionId}/close`} className="hq-button">
            大会を締める
          </a>
        </p>
      )}
    </section>
  )
}

/** 遅れの調整。分数を入れると動く試合の数を出して確認し(M5)、ずらした直後は「元に戻す」を出す */
async function DelayPanel({
  progress,
  query,
  back,
  divisionId,
  token,
}: {
  progress: Progress
  query: Query
  back: string
  divisionId: number
  token: string
}) {
  const court = pick(query, 'shiftCourt')
  const minutes = Number(pick(query, 'shiftMinutes'))
  let confirm: { sentence: string } | { error: string } | null = null
  if (Number.isInteger(minutes) && minutes !== 0) {
    try {
      const dry = await formats.shift(token, divisionId, court === '' ? null : court, minutes, true)
      confirm = { sentence: shiftSentence(court === '' ? null : court, dry.moved, minutes) }
    } catch (error) {
      if (!(error instanceof ApiError) || error.status >= 500) {
        throw error
      }
      confirm = { error: operatorErrorText(error.code, error.message) }
    }
  }
  const undoCourt = pick(query, 'undoCourt')
  const undoMinutes = Number(pick(query, 'undoMinutes'))
  return (
    <>
      {undoCourt !== '' && Number.isInteger(undoMinutes) && undoMinutes !== 0 ? (
        <form action={shiftScheduleAction} className="hq-box">
          <Hidden
            values={{
              back,
              divisionId,
              court: undoCourt === '*' ? '' : undoCourt,
              minutes: undoMinutes,
              undo: '1',
            }}
          />
          <p>直前の調整を取り消せます。</p>
          <button type="submit" className="secondary">
            元に戻す
          </button>
        </form>
      ) : null}
      {confirm !== null && 'sentence' in confirm ? (
        <form action={shiftScheduleAction} className="hq-box danger">
          <Hidden values={{ back, divisionId, court, minutes }} />
          <p>
            <strong>{confirm.sentence}。</strong>試合の順序とコートは変えません。
          </p>
          <div className="adm-actions">
            <button type="submit">ずらす</button>
            <a href={`${back}?tab=delay`}>やめる</a>
          </div>
        </form>
      ) : null}
      {confirm !== null && 'error' in confirm ? (
        <p className="adm-notice error">{confirm.error}</p>
      ) : null}
      <form method="get" action={back} className="hq-inline">
        <input type="hidden" name="tab" value="delay" />
        <label className="adm-field">
          <span>どのコート</span>
          <select name="shiftCourt" defaultValue={court}>
            <option value="">全体</option>
            {progress.courts.map((name) => (
              <option key={name} value={name}>
                コート {name}
              </option>
            ))}
          </select>
        </label>
        <label className="adm-field">
          <span>遅らせる分数(早めるときは -)</span>
          <input
            type="number"
            name="shiftMinutes"
            defaultValue={Number.isInteger(minutes) && minutes !== 0 ? minutes : 10}
            step={5}
          />
        </label>
        <button type="submit">まだ始まっていない試合をずらす</button>
      </form>
    </>
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
    <ol className="hq-todo">
      {progress.todo.map((todo) => (
        <li
          key={`${todo.kind}-${todo.groupId}-${todo.stageId}-${todo.matchId}-${todo.court}-${todo.teamId}`}
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
    </ol>
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
  const resultLink = `/admin/matches/${todo.matchId ?? 0}/result?next=${encodeURIComponent(back)}`
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
          <a href={resultLink}>結果を入れる</a>
          <a href={`/admin/matches/${todo.matchId}/score`}>スコア入力画面</a>
        </p>
      )
    case 'court_delay':
      return (
        <p className="adm-actions">
          <a
            href={`/admin/divisions/${divisionId}/progress?${new URLSearchParams({
              tab: 'delay',
              shiftCourt: todo.court ?? '',
              shiftMinutes: String(roundUpToFive(todo.minutes)),
            }).toString()}`}
          >
            このコートを {roundUpToFive(todo.minutes)} 分遅らせる
          </a>
        </p>
      )
    case 'withdrawn':
      return (
        <p className="adm-actions">
          <a href={`/admin/teams/${todo.teamId ?? 0}/withdraw?next=${encodeURIComponent(back)}`}>
            相手の不戦勝にする
          </a>
        </p>
      )
    case 'overlap':
      return (
        <p className="adm-actions">
          <a href={`/admin/matches/${todo.matchId ?? 0}`}>試合のチームと時刻を直す</a>
        </p>
      )
    case 'mismatch':
      return (
        <form action={alignTeamsAction}>
          <Hidden values={{ back, matchId: todo.matchId ?? 0 }} />
          <button type="submit" className="danger">
            順位どおりのチームに差し替える(結果はそのまま)
          </button>{' '}
          <a href={resultLink}>結果を直す</a>
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
          <span>決め方(閲覧ページの順位の根拠に出ます)</span>
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

function RankTable({
  group,
  withdrawn = [],
  isAdmin = false,
  back,
}: {
  group: GroupProgress
  withdrawn?: number[]
  isAdmin?: boolean
  back?: string
}) {
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
          <th className="num">勝-負-分</th>
          <th className="num">得失</th>
        </tr>
      </thead>
      <tbody>
        {group.rows.map((row) => (
          <tr key={row.team.id} className={row.unresolved ? 'tied' : ''}>
            <td>{row.unresolved ? `${row.rank}(未決)` : row.rank}</td>
            <td>
              {row.team.name.ja}
              {withdrawn.includes(row.team.id) ? <span className="hq-state tied">棄権</span> : null}
              {row.reason ? <div className="hq-reason">{row.reason}</div> : null}
              {isAdmin && back && !withdrawn.includes(row.team.id) ? (
                <a
                  className="hq-withdraw"
                  href={`/admin/teams/${row.team.id}/withdraw?next=${encodeURIComponent(back)}`}
                >
                  棄権にする
                </a>
              ) : null}
            </td>
            <td className="num">{row.points}</td>
            <td className="num">
              {row.wins}-{row.losses}-{row.draws}
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
  withdrawn,
}: {
  group: GroupProgress
  back: string
  isAdmin: boolean
  withdrawn: number[]
}) {
  const timeUp =
    isAdmin &&
    group.remaining > 0 &&
    ((group.kind === 'pool' && group.state === 'playing') ||
      (group.kind === 'bracket' && group.state === 'playing'))
  return (
    <section className="hq-group">
      <h3>
        <a href={`/pscore/scv_group/open/${group.id}`}>{group.label}</a>
        <span className={`hq-state ${group.state}`}>{groupStateLabels[group.state]}</span>
      </h3>
      {group.kind === 'pool' ? (
        <RankTable group={group} withdrawn={withdrawn} isAdmin={isAdmin} back={back} />
      ) : (
        <p className="adm-muted">
          {group.slots.map((slot) => slot.team?.name.ja ?? describeSide(slot.label)).join('、')}
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
      {timeUp ? (
        <details>
          <summary>時間切れで未消化のまま確定する</summary>
          <form action={confirmGroupAction}>
            <Hidden values={{ back, groupId: group.id }} />
            <label className="adm-check">
              <input type="checkbox" name="unplayed" required />
              <span>
                {group.kind === 'pool'
                  ? `残りの ${group.remaining} 試合を行わずに、今の順位で確定する`
                  : `残りの ${group.remaining} 試合(相手が未定の試合も)を未消化にする。決まらない順位は「大会を締める」で同順位にします`}
              </span>
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
              {match.left.team?.name.ja ?? describeSide(match.left.label)} {match.left.score ?? ''}{' '}
              - {match.right.score ?? ''}{' '}
              {match.right.team?.name.ja ?? describeSide(match.right.label)}
            </span>
            <span className="sub">{match.groupLabel}</span>
          </a>
          {isAdmin ? (
            <a href={`/admin/matches/${match.id}/result`} className="hq-sublink">
              結果を入れる・直す
            </a>
          ) : null}
        </li>
      ))}
    </ul>
  )
}
