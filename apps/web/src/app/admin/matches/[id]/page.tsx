import { sideChoiceValue, sideName, statusLabel } from '@/admin/labels.utils'
import { loadAdmin, requireUser } from '@/admin/session'
import { groups, matches, teams } from '@/api/admin'
import type { IdPageProps } from '@/app/_lib/pageProps'
import { deleteMatchAction, updateMatchAction } from '@/app/admin/_actions/match'
import { readAdminPage } from '@/app/admin/_lib/page'
import { AdminShell, Check, DeleteForm, Field, Hidden, Select } from '@/components/admin/AdminParts'
import { SideChoice } from '@/components/admin/SideChoice'

// 試合の登録内容(日時、コート、CD、対戦チーム)の編集。スコアと状態はスコア入力画面で変える。
export default async function AdminMatchPage(props: IdPageProps) {
  const { id, notice } = await readAdminPage(props)
  const { user, token } = await requireUser('admin')
  const detail = await loadAdmin(() => matches.detail(token, id))
  const match = detail.match
  const [teamList, groupList, matchList] = await loadAdmin(() =>
    Promise.all([
      teams.list(token, match.divisionId),
      groups.list(token, match.divisionId),
      matches.list(token, match.divisionId),
    ]),
  )
  const back = `/admin/matches/${id}`
  const others = matchList.matches.filter((other) => other.id !== id)
  const title = `${match.code || '試合'} ${sideName(match.left)} - ${sideName(match.right)}`
  const crumbs = [{ href: `/admin/divisions/${match.divisionId}`, label: 'ディビジョン' }]
  return (
    <AdminShell user={user} title={title} notice={notice} crumbs={crumbs}>
      <p className="adm-actions">
        <span className={`adm-badge ${match.status}`}>{statusLabel(match.status)}</span>
        <a href={`/admin/matches/${id}/score`}>スコア入力</a>
        <a href={`/pscore/scv_match/open/${id}`}>閲覧ページを開く</a>
      </p>
      <form action={updateMatchAction}>
        <Hidden
          values={{
            back,
            id,
            leftOriginal: sideChoiceValue(match.left),
            rightOriginal: sideChoiceValue(match.right),
          }}
        />
        <div className="adm-grid">
          <Field label="試合 CD" name="code" defaultValue={match.code} />
          <Field label="日付" name="date" type="date" defaultValue={match.date} />
          <Field label="時刻" name="time" type="time" defaultValue={match.time} />
          <Field label="コート" name="field" defaultValue={match.field} />
          <Select
            label="グループ"
            name="groupId"
            defaultValue={match.groupId === null ? '' : String(match.groupId)}
            options={[
              { value: '', label: 'なし' },
              ...groupList.groups.map((group) => ({ value: String(group.id), label: group.name })),
            ]}
          />
          <SideChoice
            label={`左(表の上)のチーム(現在: ${sideName(match.left)})`}
            name="left"
            teams={teamList.teams}
            matches={others}
            defaultValue={sideChoiceValue(match.left)}
          />
          <SideChoice
            label={`右のチーム(現在: ${sideName(match.right)})`}
            name="right"
            teams={teamList.teams}
            matches={others}
            defaultValue={sideChoiceValue(match.right)}
          />
        </div>
        <Check
          label="試合ページを出す(外すと日程の状況列は「終了」だけになる)"
          name="hasDetail"
          defaultChecked={match.hasDetail}
        />
        <button type="submit">保存</button>
      </form>
      <DeleteForm
        action={deleteMatchAction}
        values={{ back, id, divisionId: match.divisionId }}
        label="この試合"
      />
    </AdminShell>
  )
}
