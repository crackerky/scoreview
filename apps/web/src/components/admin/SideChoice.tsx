import type { AdminMatch, AdminTeam } from '@/api/admin.types'

/**
 * 試合の片側やトーナメント表の枠の選択。チーム(値はチーム ID)か、同じディビジョンの試合の勝者・敗者
 * (値は "A1-win"。表示は参照サイトと同じ "[A1-勝ち]")を選ぶ。値の読み取りは sideChoiceSchema。
 */
export function SideChoice({
  label,
  name,
  teams,
  matches,
  defaultValue,
  allowEmpty = true,
}: {
  label: string
  name: string
  teams: AdminTeam[]
  matches: AdminMatch[]
  defaultValue?: string
  allowEmpty?: boolean
}) {
  const codes = [...new Set(matches.map((match) => match.code).filter((code) => code !== ''))]
  return (
    <label className="adm-field">
      <span>{label}</span>
      <select name={name} defaultValue={defaultValue ?? ''}>
        {allowEmpty ? <option value="">未定</option> : <option value="">選んでください</option>}
        <optgroup label="チーム">
          {teams.map((team) => (
            <option key={team.id} value={team.id}>
              {team.shortName.ja}
            </option>
          ))}
        </optgroup>
        {codes.length > 0 ? (
          <optgroup label="試合の勝者・敗者">
            {codes.flatMap((code) => [
              <option key={`${code}-win`} value={`${code}-win`}>
                [{code}-勝ち]
              </option>,
              <option key={`${code}-lose`} value={`${code}-lose`}>
                [{code}-負け]
              </option>,
            ])}
          </optgroup>
        ) : null}
      </select>
    </label>
  )
}
