import { sideName } from '@/admin/labels.utils'
import type { AdminMatch, AdminUser, Assignment } from '@/api/admin.types'
import { createAssignmentAction, deleteAssignmentAction } from '@/app/admin/_actions/assignment'
import { Field, Hidden, Select } from './AdminParts'

// 記録係の割り当て表(docs/operations/operator-experience.md「人手が足りない現場を前提にする」2)。
// 記録係ごとに、担当するコートと時間帯(または試合)を並べる。割り当てた試合だけが記録係の
// 「自分の担当試合」に出て、記録できる。

type Props = {
  tournamentId: number
  back: string
  assignments: Assignment[]
  staff: AdminUser[]
  matches: AdminMatch[]
}

function describe(assignment: Assignment): string {
  if (assignment.matchId !== null) {
    return `試合 ${assignment.matchCode || assignment.matchId}`
  }
  const parts = [
    assignment.date ?? '全日程',
    assignment.field === null ? '全コート' : `コート ${assignment.field}`,
    assignment.from === null && assignment.to === null
      ? '終日'
      : `${assignment.from ?? ''}〜${assignment.to ?? ''}`,
  ]
  return parts.join(' / ')
}

export function AssignmentSection({ tournamentId, back, assignments, staff, matches }: Props) {
  const fields = [
    ...new Set(matches.flatMap((match) => (match.field === null ? [] : [match.field]))),
  ]
  const dates = [...new Set(matches.flatMap((match) => (match.date === null ? [] : [match.date])))]
  const staffOptions = staff.map((user) => ({
    value: String(user.id),
    label: user.displayName || user.email,
  }))
  return (
    <>
      <h2>記録係の割り当て</h2>
      <p className="adm-muted">
        記録係は、割り当てたコートと時間帯(または試合)の試合だけを記録できます。全試合を任せるときは、コートと時間を空欄にします。記録係のユーザーは「ユーザー」で作ります。
      </p>
      {assignments.length === 0 ? (
        <p className="adm-muted">まだ割り当てがありません。</p>
      ) : (
        <table className="adm-table">
          <thead>
            <tr>
              <th>記録係</th>
              <th>担当</th>
              <th />
            </tr>
          </thead>
          <tbody>
            {assignments.map((assignment) => (
              <tr key={assignment.id}>
                <td>{assignment.userName}</td>
                <td>{describe(assignment)}</td>
                <td>
                  <form action={deleteAssignmentAction}>
                    <Hidden values={{ back, id: assignment.id }} />
                    <button type="submit" className="secondary">
                      外す
                    </button>
                  </form>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
      {staffOptions.length === 0 ? (
        <p className="adm-muted">記録係のユーザーがまだいません。</p>
      ) : (
        <>
          <h3>コートと時間帯で割り当てる</h3>
          <form action={createAssignmentAction}>
            <Hidden values={{ back, tournamentId }} />
            <div className="adm-grid">
              <Select label="記録係" name="userId" options={staffOptions} />
              <Select
                label="コート"
                name="field"
                options={[
                  { value: '', label: '全コート' },
                  ...fields.map((field) => ({ value: field, label: `コート ${field}` })),
                ]}
              />
              <Select
                label="日付"
                name="date"
                options={[
                  { value: '', label: '全日程' },
                  ...dates.map((date) => ({ value: date, label: date })),
                ]}
              />
              <Field label="開始(この時刻以降に始まる試合)" name="from" type="time" />
              <Field label="終了(この時刻より前に始まる試合)" name="to" type="time" />
            </div>
            <button type="submit">割り当てる</button>
          </form>
          <h3>試合で割り当てる</h3>
          <form action={createAssignmentAction}>
            <Hidden values={{ back, tournamentId }} />
            <div className="adm-grid">
              <Select label="記録係" name="userId" options={staffOptions} />
              <Select
                label="試合"
                name="matchId"
                options={matches.map((match) => ({
                  value: String(match.id),
                  label: `${match.code} ${match.time ?? ''} ${match.field ? `コート ${match.field}` : ''} ${sideName(match.left)}-${sideName(match.right)}`,
                }))}
              />
            </div>
            <button type="submit">割り当てる</button>
          </form>
        </>
      )}
    </>
  )
}
