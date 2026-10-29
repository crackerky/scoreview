import type { GroupProgress } from '@/api/format.types'
import { Chip, type ChipTone } from './AdminParts'

// 本部の組の順位表(大会単位の本部の「組」タブと、組の確定の画面)。

const groupChip: Record<GroupProgress['state'], { label: string; tone: ChipTone }> = {
  waiting: { label: '組み合わせ待ち', tone: 'done' },
  playing: { label: '途中経過', tone: 'done' },
  tied: { label: '同順位', tone: 'warn' },
  ready: { label: '確定待ち', tone: 'warn' },
  confirmed: { label: '確定', tone: 'ok' },
  finished: { label: '確定', tone: 'ok' },
}

/** 組の順位表。勝敗は「○勝○敗○分」と言う順(docs/requirements.md「意図的な差分」) */
export function StandingsCard({
  group,
  title,
  confirmHref,
}: {
  group: GroupProgress
  title: string
  confirmHref?: string
}) {
  const chip = groupChip[group.state]
  const reasons = group.rows.filter((row) => row.reason !== '')
  return (
    <div className="adm-standings">
      <div className="adm-alert-head">
        <strong>{title}</strong>
        <Chip tone={chip.tone}>{chip.label}</Chip>
      </div>
      <table>
        <thead>
          <tr>
            <th>順位</th>
            <th>チーム</th>
            <th>勝</th>
            <th>負</th>
            <th>分</th>
            <th>得失</th>
            <th>勝点</th>
          </tr>
        </thead>
        <tbody>
          {group.rows.map((row) => {
            const diff = row.goalsFor - row.goalsAgainst
            return (
              <tr key={row.team.id}>
                <td className="adm-num">{row.rank}</td>
                <td className="adm-standings-team">{row.team.name.ja}</td>
                <td className="adm-num">{row.wins}</td>
                <td className="adm-num">{row.losses}</td>
                <td className="adm-num">{row.draws}</td>
                <td className="adm-num">{diff > 0 ? `+${diff}` : diff}</td>
                <td className="adm-num adm-points">{row.points}</td>
              </tr>
            )
          })}
        </tbody>
      </table>
      {reasons.length > 0 ? (
        <p className="adm-muted">
          {reasons.map((row) => `${row.team.name.ja}: ${row.reason}`).join(' / ')}
        </p>
      ) : null}
      {group.state === 'ready' && confirmHref ? (
        <a className="adm-go" href={confirmHref}>
          {group.label}を確定する
        </a>
      ) : null}
    </div>
  )
}
