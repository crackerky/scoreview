import { auditUser, describeAudit, formatAuditTime } from '@/admin/audit.utils'
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
  const [teamList, groupList, matchList, audit] = await loadAdmin(() =>
    Promise.all([
      teams.list(token, match.divisionId),
      groups.list(token, match.divisionId),
      matches.list(token, match.divisionId),
      matches.audit(token, id),
    ]),
  )
  const names = { left: sideName(match.left), right: sideName(match.right) }
  const back = `/admin/matches/${id}`
  // 勝者・敗者の候補は同じ組の前の試合に絞る(部門の全試合だと 100 件を超える。運営者レビュー L5)。
  // 組の無い試合と、今入っている参照の試合は残す
  const current = new Set([match.left.placeholder?.code, match.right.placeholder?.code])
  const others = matchList.matches.filter(
    (other) =>
      other.id !== id &&
      (match.groupId === null || other.groupId === match.groupId || current.has(other.code)),
  )
  const title = `${match.code || '試合'} ${sideName(match.left)} - ${sideName(match.right)}`
  const crumbs = [{ href: `/admin/divisions/${match.divisionId}`, label: '部門' }]
  return (
    <AdminShell user={user} title={title} notice={notice} crumbs={crumbs}>
      <p className="adm-actions">
        <span className={`adm-badge ${match.status}`}>{statusLabel(match.status)}</span>
        <a href={`/admin/matches/${id}/score`}>スコア入力</a>
        <a href={`/admin/matches/${id}/sheet`}>スコアシートを印刷</a>
        <a href={`/admin/matches/${id}/score/basic`}>簡易の入力画面</a>
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
      <h2>変更の履歴</h2>
      <p className="adm-muted">
        スコア、得点経過、試合の進行、結果の種類を、誰がいつ変えたかの記録です(新しい順)。
      </p>
      {audit.entries.length === 0 ? (
        <p className="adm-muted">まだ記録がありません。</p>
      ) : (
        <div className="adm-scroll">
          <table className="adm-table adm-audit">
            <thead>
              <tr>
                <th>時刻</th>
                <th>誰が</th>
                <th>何を</th>
                <th>変更前</th>
                <th>変更後</th>
              </tr>
            </thead>
            <tbody>
              {audit.entries.map((entry) => {
                const line = describeAudit(entry, names)
                return (
                  <tr key={entry.id}>
                    <td>{formatAuditTime(entry.at)}</td>
                    <td>{auditUser(entry)}</td>
                    <td>
                      {line.what}
                      {line.target ? <span className="adm-muted"> {line.target}</span> : null}
                    </td>
                    <td>{line.before}</td>
                    <td>{line.after}</td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}
      <DeleteForm
        action={deleteMatchAction}
        values={{ back, id, divisionId: match.divisionId }}
        label="この試合"
      />
    </AdminShell>
  )
}
