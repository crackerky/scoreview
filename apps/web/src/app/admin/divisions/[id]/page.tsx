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
import type { AdminMatch } from '@/api/admin.types'
import type { IdPageProps } from '@/app/_lib/pageProps'
import {
  createGroupAction,
  createMatchAction,
  createTeamAction,
  deleteDivisionAction,
  putFinalRanksAction,
  updateDivisionAction,
} from '@/app/admin/_actions/division'
import { readAdminPage } from '@/app/admin/_lib/page'
import { AdminShell, Check, DeleteForm, Field, Hidden, Select } from '@/components/admin/AdminParts'
import { SideChoice } from '@/components/admin/SideChoice'

export default async function AdminDivisionPage(props: IdPageProps) {
  const { id, notice } = await readAdminPage(props)
  const { user, token } = await requireUser('any')
  const division = await loadAdmin(() => divisions.get(token, id))
  const [tournament, matchList] = await loadAdmin(() =>
    Promise.all([tournaments.get(token, division.tournamentId), matches.list(token, id)]),
  )
  const crumbs = [{ href: `/admin/tournaments/${tournament.id}`, label: tournament.name.ja }]
  return (
    <AdminShell user={user} title={division.name.ja} notice={notice} crumbs={crumbs}>
      <h2>試合</h2>
      <MatchTable matches={matchList.matches} isAdmin={user.role === 'admin'} />
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

function MatchTable({ matches: list, isAdmin }: { matches: AdminMatch[]; isAdmin: boolean }) {
  if (list.length === 0) {
    return <p className="adm-muted">試合がありません。</p>
  }
  return (
    <div className="adm-scroll">
      <table className="adm-table">
        <thead>
          <tr>
            <th>CD</th>
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
                {isAdmin ? <a href={`/admin/matches/${match.id}`}>編集</a> : null}
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
          <Field label="試合 CD(例 A1)" name="code" />
          <Field label="日付" name="date" type="date" />
          <Field label="時刻" name="time" type="time" />
          <Field label="コート" name="field" />
          <Select
            label="グループ"
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

      <h2>グループ</h2>
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
          <Field label="グループ名" name="name" required />
          <Select label="種類" name="kind" options={groupKindOptions} defaultValue="pool" />
          <Field label="並び順" name="sortOrder" type="number" defaultValue={0} />
          <Field label="画像 URL(画像のみの型)" name="imageUrl" />
        </div>
        <button type="submit">グループを作る</button>
      </form>

      <h2>チーム</h2>
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
      <form action={createTeamAction}>
        <Hidden values={{ back, divisionId: id }} />
        <div className="adm-grid">
          <Field label="正式名(日本語)" name="fullNameJa" required />
          <Field label="短縮名(日本語)" name="shortNameJa" required />
          <Field label="正式名(英語)" name="fullNameEn" />
          <Field label="短縮名(英語)" name="shortNameEn" />
          <Select label="色帯" name="barStyle" options={barStyleOptions} defaultValue="default" />
          <Field label="並び順" name="sortOrder" type="number" defaultValue={0} />
        </div>
        <button type="submit">チームを作る</button>
      </form>

      <h2>最終順位</h2>
      <form action={putFinalRanksAction}>
        <Hidden values={{ back, id }} />
        <div className="adm-grid">
          {teamList.teams.map((team) => (
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

      <h2>ディビジョンの設定</h2>
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
          <Field label="並び順" name="sortOrder" type="number" defaultValue={division.sortOrder} />
          <Select
            label="参加単位"
            name="entryKind"
            options={entryKindOptions}
            defaultValue={division.entryKind}
          />
          <Select
            label="成績表の型"
            name="statsKind"
            options={statsKindOptions}
            defaultValue={division.statsKind}
          />
        </div>
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
        label="このディビジョン"
      />
    </>
  )
}
