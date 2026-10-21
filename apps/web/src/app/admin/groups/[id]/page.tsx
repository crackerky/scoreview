import { groupKindOptions } from '@/admin/labels.utils'
import { loadAdmin, requireUser } from '@/admin/session'
import { divisions, groups, matches, teams } from '@/api/admin'
import type { IdPageProps } from '@/app/_lib/pageProps'
import {
  deleteGroupAction,
  generateBracketAction,
  putGroupTeamsAction,
  updateGroupAction,
} from '@/app/admin/_actions/division'
import { readAdminPage } from '@/app/admin/_lib/page'
import { AdminShell, Check, DeleteForm, Field, Hidden, Select } from '@/components/admin/AdminParts'
import { SideChoice } from '@/components/admin/SideChoice'

/** トーナメント表の枠の欄(表の上から順)。欄の名前は bracketFormSchema が読む seed-{n} */
function seedSlots(count: number): Array<{ name: string; label: string }> {
  return Array.from({ length: count }, (_, index) => ({
    name: `seed-${index}`,
    label: `${index + 1} 番目`,
  }))
}

export default async function AdminGroupPage(props: IdPageProps) {
  const { id, notice } = await readAdminPage(props)
  const { user, token } = await requireUser('admin')
  const group = await loadAdmin(() => groups.get(token, id))
  const [division, teamList, matchList, templates] = await loadAdmin(() =>
    Promise.all([
      divisions.get(token, group.divisionId),
      teams.list(token, group.divisionId),
      matches.list(token, group.divisionId),
      groups.templates(token),
    ]),
  )
  const back = `/admin/groups/${id}`
  const order = new Map(group.teams.map((member, index) => [member.teamId, { index, member }]))
  const groupMatches = matchList.matches.filter((match) => match.groupId === id)
  const crumbs = [
    { href: `/admin/tournaments/${division.tournamentId}`, label: '大会' },
    { href: `/admin/divisions/${division.id}`, label: division.name.ja },
  ]
  return (
    <AdminShell user={user} title={group.name} notice={notice} crumbs={crumbs}>
      <p>
        <a href={`/pscore/scv_group/open/${id}`}>閲覧ページを開く</a>
      </p>
      <h2>グループの設定</h2>
      <form action={updateGroupAction}>
        <Hidden values={{ back, id }} />
        <div className="adm-grid">
          <Field label="グループ名" name="name" defaultValue={group.name} required />
          <Select label="種類" name="kind" options={groupKindOptions} defaultValue={group.kind} />
          <Field label="並び順" name="sortOrder" type="number" defaultValue={group.sortOrder} />
          <Field label="画像 URL(画像のみの型)" name="imageUrl" defaultValue={group.image?.url} />
          <Field
            label="画像の幅"
            name="imageWidth"
            type="number"
            defaultValue={group.image?.width}
          />
          <Field
            label="画像の高さ"
            name="imageHeight"
            type="number"
            defaultValue={group.image?.height}
          />
        </div>
        <Check label="ナビに出す(公開)" name="published" defaultChecked={group.published} />
        <Check label="ナビでリンクにする" name="linked" defaultChecked={group.linked} />
        <button type="submit">保存</button>
      </form>

      {group.kind === 'bracket' ? (
        <>
          <h2>トーナメント表</h2>
          {group.bracket ? (
            <p>
              生成済み: {group.bracket.pattern}({group.bracket.ptnName})、試合 {groupMatches.length}{' '}
              件。 作り直すときは、このグループの試合を消してからにしてください。
            </p>
          ) : (
            templates.templates.map((template) => (
              <form key={template.id} action={generateBracketAction}>
                <fieldset>
                  <legend>{template.name.ja}</legend>
                  <Hidden values={{ back, id, template: template.id, seedCount: template.teams }} />
                  <Field label="試合 CD の接頭辞(例 B → B1, B2, …)" name="codePrefix" required />
                  <p className="adm-muted">
                    表の上から順に枠を選びます(前の試合の勝者も選べます)。
                  </p>
                  <div className="adm-grid">
                    {seedSlots(template.teams).map((slot) => (
                      <SideChoice
                        key={slot.name}
                        label={slot.label}
                        name={slot.name}
                        teams={teamList.teams}
                        matches={matchList.matches}
                        allowEmpty={false}
                      />
                    ))}
                  </div>
                  <button type="submit">生成</button>
                </fieldset>
              </form>
            ))
          )}
        </>
      ) : null}

      <h2>チーム</h2>
      <p className="adm-muted">
        グループに入れるチームの「順」に数字を入れます(対戦表と順位表はこの順)。空欄のチームは入れません。
        「順位の上書き」は計算した順位と違う順位を出すとき(0 は空欄で表示)。
      </p>
      <form action={putGroupTeamsAction}>
        <Hidden values={{ back, id }} />
        <div className="adm-scroll">
          <table className="adm-table">
            <thead>
              <tr>
                <th>チーム</th>
                <th>順</th>
                <th>順位の上書き</th>
              </tr>
            </thead>
            <tbody>
              {teamList.teams.map((team) => {
                const current = order.get(team.id)
                return (
                  <tr key={team.id}>
                    <td>{team.shortName.ja}</td>
                    <td>
                      <input
                        type="number"
                        name={`team-${team.id}`}
                        defaultValue={current ? current.index + 1 : ''}
                        aria-label={`${team.shortName.ja} の順`}
                      />
                    </td>
                    <td>
                      <input
                        type="number"
                        name={`rank-${team.id}`}
                        defaultValue={current?.member.rankOverride ?? ''}
                        aria-label={`${team.shortName.ja} の順位の上書き`}
                      />
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
        <button type="submit">チームを保存</button>
      </form>
      <DeleteForm
        action={deleteGroupAction}
        values={{ back, id, divisionId: division.id }}
        label="このグループ(試合は残ります)"
      />
    </AdminShell>
  )
}
