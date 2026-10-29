import { runsDayOf } from '@/admin/access.utils'
import {
  barStyleOptions,
  divisionKindOptions,
  entryKindOptions,
  groupKindOptions,
  sideName,
  statsKindOptions,
  statusLabel,
} from '@/admin/labels.utils'
import { loadAdmin, requireUser } from '@/admin/session'
import { divisions, groups, matches, teams, tournaments } from '@/api/admin'
import type { AdminMatch, AdminTeam } from '@/api/admin.types'
import { formats } from '@/api/format'
import type { IdPageProps } from '@/app/_lib/pageProps'
import {
  createGroupAction,
  createMatchAction,
  createTeamAction,
  deleteDivisionAction,
  putFinalRanksAction,
  updateDivisionAction,
} from '@/app/admin/_actions/division'
import { fillEnglishNamesAction } from '@/app/admin/_actions/team'
import { readAdminPage } from '@/app/admin/_lib/page'
import { AdminShell, Check, DeleteForm, Field, Hidden, Select } from '@/components/admin/AdminParts'
import { SideChoice } from '@/components/admin/SideChoice'

export default async function AdminDivisionPage(props: IdPageProps) {
  const { id, notice } = await readAdminPage(props)
  const { user, token } = await requireUser('any')
  const division = await loadAdmin(() => divisions.get(token, id))
  const isAdmin = user.role === 'admin'
  const [tournament, matchList, saved] = await loadAdmin(() =>
    Promise.all([
      tournaments.get(token, division.tournamentId),
      matches.list(token, id),
      // 形の有無は管理者だけが読める(記録係には形を決めるリンクを出さない)
      isAdmin ? formats.get(token, id) : null,
    ]),
  )
  const crumbs = [{ href: `/admin/tournaments/${tournament.id}`, label: tournament.name.ja }]
  // 準備中(形がまだ無い)は「形を決める」を先頭に。本部の進行画面は形を決めた後に使う
  const preparing = isAdmin && saved?.format === null
  return (
    <AdminShell user={user} title={division.name.ja} notice={notice} crumbs={crumbs}>
      <h2>試合</h2>
      {preparing ? (
        <p className="adm-actions">
          <a className="adm-primary-link" href={`/admin/divisions/${id}/format`}>
            まず大会の形を決める(チームの登録、組、順位決定戦、時間割)
          </a>
          <a href={`/admin/tournaments/${tournament.id}/timetable`}>時間割を印刷</a>
        </p>
      ) : (
        <p className="adm-actions">
          <a href={`/admin/divisions/${id}/progress`}>本部の進行画面</a>
          {isAdmin ? <a href={`/admin/divisions/${id}/format`}>大会の形(組と順位決定戦)</a> : null}
          <a href={`/admin/divisions/${id}/sheets`}>スコアシートをまとめて印刷</a>
          <a href={`/admin/tournaments/${tournament.id}/timetable`}>時間割を印刷</a>
        </p>
      )}
      <MatchTable
        matches={matchList.matches}
        isAdmin={user.role === 'admin'}
        runsDayOf={runsDayOf(user)}
      />
      {user.role === 'admin' ? (
        <AdminSections
          token={token}
          division={division}
          matchList={matchList.matches}
          back={`/admin/divisions/${id}`}
        />
      ) : null}
    </AdminShell>
  )
}

/** 最終順位の入力欄の並び: 順位の順、同順位は登録順、順位の無いチームは最後(運営者レビュー M10) */
function byFinalRank(list: AdminTeam[]): AdminTeam[] {
  return [...list].sort((a, b) => {
    if (a.finalRank === null || b.finalRank === null) {
      return a.finalRank === b.finalRank ? 0 : a.finalRank === null ? 1 : -1
    }
    return a.finalRank - b.finalRank
  })
}

/** 英語の正式名か短縮名が空のチームの数 */
function missingEnglish(list: AdminTeam[]): number {
  return list.filter((team) => team.fullName.en === '' || team.shortName.en === '').length
}

function MatchTable({
  matches: list,
  isAdmin,
  runsDayOf: canFix,
}: {
  matches: AdminMatch[]
  isAdmin: boolean
  runsDayOf: boolean
}) {
  if (list.length === 0) {
    return <p className="adm-muted">試合がありません。</p>
  }
  return (
    <div className="adm-scroll">
      <table className="adm-table">
        <thead>
          <tr>
            <th>試合番号</th>
            <th>日時・コート</th>
            <th>対戦</th>
            <th>状態</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          {list.map((match) => (
            <tr key={match.id}>
              <td>{match.code}</td>
              <td>
                {match.date ?? '-'} {match.time ?? ''} {match.field ? `コート${match.field}` : ''}
              </td>
              <td>
                {sideName(match.left)} {match.left.score ?? ''} - {match.right.score ?? ''}{' '}
                {sideName(match.right)}
              </td>
              <td>
                <span className={`adm-badge ${match.status}`}>{statusLabel(match.status)}</span>
              </td>
              <td className="adm-actions">
                <a href={`/admin/matches/${match.id}/score`}>スコア入力</a>
                <a href={`/admin/matches/${match.id}/sheet`}>印刷</a>
                {isAdmin ? <a href={`/admin/matches/${match.id}`}>編集</a> : null}
                {canFix && match.status === 'finished' ? (
                  // 確定後の訂正は影響の表示と下流の扱い(keep / reset)のある画面で行う
                  <a
                    href={`/admin/matches/${match.id}/result?next=${encodeURIComponent(
                      `/admin/divisions/${match.divisionId}`,
                    )}`}
                  >
                    結果を直す
                  </a>
                ) : null}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

async function AdminSections({
  token,
  division,
  matchList,
  back,
}: {
  token: string
  division: Awaited<ReturnType<typeof divisions.get>>
  matchList: AdminMatch[]
  back: string
}) {
  const id = division.id
  const [groupList, teamList] = await loadAdmin(() =>
    Promise.all([groups.list(token, id), teams.list(token, id)]),
  )
  return (
    <>
      <h3>試合を作る</h3>
      <form action={createMatchAction}>
        <Hidden values={{ back, divisionId: id, hasDetail: 'on' }} />
        <div className="adm-grid">
          <Field label="試合番号(例 A1)" name="code" />
          <Field label="日付" name="date" type="date" />
          <Field label="時刻" name="time" type="time" />
          <Field label="コート" name="field" />
          <Select
            label="組"
            name="groupId"
            options={[
              { value: '', label: 'なし' },
              ...groupList.groups.map((group) => ({ value: String(group.id), label: group.name })),
            ]}
          />
          <SideChoice
            label="左(表の上)のチーム"
            name="left"
            teams={teamList.teams}
            matches={matchList}
          />
          <SideChoice label="右のチーム" name="right" teams={teamList.teams} matches={matchList} />
        </div>
        <button type="submit">試合を作る</button>
      </form>

      <h2>組(プールとトーナメント)</h2>
      <ul className="adm-list">
        {groupList.groups.map((group) => (
          <li key={group.id}>
            <a href={`/admin/groups/${group.id}`}>
              {group.name}
              <span className="adm-muted">
                ({groupKindOptions.find((option) => option.value === group.kind)?.label}
                {group.published ? '' : ' 非公開'})
              </span>
            </a>
          </li>
        ))}
      </ul>
      <form action={createGroupAction}>
        <Hidden values={{ back, divisionId: id, linked: 'on', published: 'on' }} />
        <div className="adm-grid">
          <Field label="組の名前" name="name" required />
          <Select label="種類" name="kind" options={groupKindOptions} defaultValue="pool" />
        </div>
        <details>
          <summary>詳しい設定</summary>
          <div className="adm-grid">
            <Field label="並び順" name="sortOrder" type="number" defaultValue={0} />
            <Field label="画像 URL(組の表を画像で出すとき)" name="imageUrl" />
          </div>
        </details>
        <button type="submit">組を作る</button>
      </form>

      <h2>チーム</h2>
      <p className="adm-muted">
        名簿をまとめて貼り付けるときは{' '}
        <a href={`/admin/divisions/${id}/format`}>大会の形を決める</a>{' '}
        の画面から(貼る前に確かめられます)。
      </p>
      <ul className="adm-list">
        {teamList.teams.map((team) => (
          <li key={team.id}>
            <a href={`/admin/teams/${team.id}`}>
              {team.fullName.ja}
              {team.shortName.ja !== team.fullName.ja ? (
                <span className="adm-muted">(短縮名 {team.shortName.ja})</span>
              ) : null}
            </a>
          </li>
        ))}
      </ul>
      {teamList.teams.length > 0 ? (
        <form action={fillEnglishNamesAction}>
          <Hidden values={{ back, divisionId: id }} />
          <p className="adm-muted">
            英語名が空のチーム {missingEnglish(teamList.teams)}{' '}
            件。英語の閲覧ページでは英語名が空だと
            名前が出ません。選手と組の英語名も同じく入れます(入っている英語名は変えません)。
          </p>
          <button type="submit">英語名が空のチームに日本語名を入れる</button>
        </form>
      ) : null}
      <form action={createTeamAction}>
        <Hidden values={{ back, divisionId: id }} />
        <div className="adm-grid">
          <Field label="正式名(日本語)" name="fullNameJa" required />
          <Field label="短縮名(日本語)" name="shortNameJa" required />
        </div>
        <details>
          <summary>詳しい設定(英語の名前、チーム一覧の色、並び順)</summary>
          <div className="adm-grid">
            <Field label="正式名(英語)" name="fullNameEn" />
            <Field label="短縮名(英語)" name="shortNameEn" />
            <Select
              label="チーム一覧の色"
              name="barStyle"
              options={barStyleOptions}
              defaultValue="default"
            />
            <Field label="並び順" name="sortOrder" type="number" defaultValue={0} />
          </div>
        </details>
        <button type="submit">チームを作る</button>
      </form>

      <h2>最終順位</h2>
      <form action={putFinalRanksAction}>
        <Hidden values={{ back, id }} />
        <p className="adm-muted">順位の順に並べ、順位が未決のチームは最後に出します。</p>
        <div className="adm-grid">
          {byFinalRank(teamList.teams).map((team) => (
            <Field
              key={team.id}
              label={team.shortName.ja}
              name={`rank-${team.id}`}
              type="number"
              defaultValue={team.finalRank}
            />
          ))}
        </div>
        <Check
          label="大会トップの [最終順位] を公開する"
          name="standingsPublished"
          defaultChecked={division.standingsPublished}
        />
        <button type="submit">順位を保存</button>
      </form>

      <h2>部門の設定</h2>
      <form action={updateDivisionAction}>
        <Hidden values={{ back, id }} />
        <div className="adm-grid">
          <Field label="名前(日本語)" name="nameJa" defaultValue={division.name.ja} required />
          <Field label="名前(英語)" name="nameEn" defaultValue={division.name.en} />
          <Select
            label="区分"
            name="kind"
            options={divisionKindOptions}
            defaultValue={division.kind}
          />
        </div>
        <details>
          <summary>詳しい設定(並び順、参加単位、競技の種類)</summary>
          <div className="adm-grid">
            <Field
              label="並び順"
              name="sortOrder"
              type="number"
              defaultValue={division.sortOrder}
            />
            <Select
              label="参加単位"
              name="entryKind"
              options={entryKindOptions}
              defaultValue={division.entryKind}
            />
            <Select
              label="成績の数え方(競技)"
              name="statsKind"
              options={statsKindOptions}
              defaultValue={division.statsKind}
            />
          </div>
        </details>
        <Check
          label="大会トップの [スケジュール] をリンクにする"
          name="scheduleEnabled"
          defaultChecked={division.scheduleEnabled}
        />
        <Check
          label="[最終順位] を公開する"
          name="standingsPublished"
          defaultChecked={division.standingsPublished}
        />
        <button type="submit">保存</button>
      </form>
      <DeleteForm
        action={deleteDivisionAction}
        values={{ back, id, tournamentId: division.tournamentId }}
        label="この部門"
      />
    </>
  )
}
