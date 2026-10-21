import type { TeamResponse } from '@/api/public.types'
import { formatWinRate } from '@/format/format.utils'
import { type Lang, pickText } from '@/i18n/lang'
import { angleHeading, msg } from '@/i18n/messages'
import { MatchListRows } from './MatchRowLine'
import { PlayerLink } from './PlayerLink'
import { ScvMidaM } from './ScvMidaM'
import { Tbl, type TblColumn, TblHeaderRow } from './Tbl'

// clone-spec D 1: チームページ /pscore/scv_team/open/{id} の #score_view の中身。
// SOTG 表(D 1.1 の 6)は api にデータがないので出さない(フェーズ 2)。

// clone-spec D 1.1: 得点・失点・得失差・順位・トータルのセルは値の後ろに半角空白 2 つ
const trailing = '  '

type Team = TeamResponse['team']

function TeamHeading({ team, lang }: { team: Team; lang: Lang }) {
  const comment = pickText(lang, team.comment)
  return (
    <>
      <ScvMidaM>
        {pickText(lang, team.fullName)}
        <br />
      </ScvMidaM>
      {comment !== '' && <div className="SCV_TEAM_COMM">{comment}</div>}
    </>
  )
}

function recordColumns(lang: Lang): TblColumn[] {
  const keys = [
    'colGames',
    'colWins',
    'colDraws',
    'colLosses',
    'colGoalsFor',
    'colGoalsAgainst',
    'colGoalsDiff',
    'colWinRate',
    'colRank',
  ] as const
  return keys.map((key) => ({ label: msg(lang, key), width: '9.444%' }))
}

/** 成績表(9 列)。順位はディビジョン内の通算順位で、未確定なら空白だけのセル。 */
function RecordTable({ record, lang }: { record: TeamResponse['record']; lang: Lang }) {
  const cells = [
    String(record.games),
    String(record.wins),
    String(record.draws),
    String(record.losses),
    `${record.goalsFor}${trailing}`,
    `${record.goalsAgainst}${trailing}`,
    `${record.goalsFor - record.goalsAgainst}${trailing}`,
    formatWinRate(record.wins, record.games),
    `${record.rank ?? ''}${trailing}`,
  ]
  return (
    <Tbl>
      <TblHeaderRow columns={recordColumns(lang)} />
      <tr>
        {cells.map((text, index) => (
          // biome-ignore lint/suspicious/noArrayIndexKey: 列の並びは固定
          <td key={index} className="CM">
            {text}
          </td>
        ))}
      </tr>
    </Tbl>
  )
}

function playerColumns(lang: Lang): TblColumn[] {
  return [
    { label: msg(lang, 'colNumber'), width: '5%' },
    { label: msg(lang, 'colPlayer'), width: '35%' },
    { label: msg(lang, 'colGames'), width: '10%' },
    { label: msg(lang, 'colAssist'), width: '10%' },
    { label: msg(lang, 'colGoal'), width: '10%' },
    { label: msg(lang, 'colTotal'), width: '10%' },
    { label: msg(lang, 'colTeamBlock'), width: '10%' },
    { label: msg(lang, 'colEmpty'), width: '10%' },
  ]
}

/** 選手一覧。並びは api の順(得点+アシスト降順、得点降順、背番号昇順)のまま。 */
function PlayersTable({ players, lang }: { players: TeamResponse['players']; lang: Lang }) {
  return (
    <Tbl tableStyle={{ width: '100%' }}>
      <TblHeaderRow columns={playerColumns(lang)} />
      {players.map(({ player, games, assists, goals, blocks }) => (
        <tr key={player.id}>
          <td className="CM">{player.number}</td>
          <td className="LM">
            <PlayerLink id={player.id} gender={player.gender} lang={lang}>
              {pickText(lang, player.name)}
            </PlayerLink>
          </td>
          <td className="CM">{games}</td>
          <td className="CM">{assists}</td>
          <td className="CM">{goals}</td>
          <td className="CM">{`${assists + goals}${trailing}`}</td>
          {/* clone-spec D 1.1: ブロック 0 は数字を出さない(空白 2 つだけのセル) */}
          <td className="CM">{`${blocks === 0 ? '' : blocks}${trailing}`}</td>
          <td className="CM" />
        </tr>
      ))}
    </Tbl>
  )
}

function StaffTable({ staff, lang }: { staff: TeamResponse['staff']; lang: Lang }) {
  return (
    <Tbl tableStyle={{ width: '70%' }}>
      <TblHeaderRow
        columns={[
          { label: msg(lang, 'colRole'), width: '30%' },
          { label: msg(lang, 'colStaff'), width: '70%' },
        ]}
      />
      {staff.map((member) => (
        <tr key={member.id}>
          <td className="CM">{pickText(lang, member.role)}</td>
          <td className="LM">
            <PlayerLink id={member.id} gender={member.gender} lang={lang}>
              {pickText(lang, member.name)}
            </PlayerLink>
          </td>
        </tr>
      ))}
    </Tbl>
  )
}

/** clone-spec D 1.2: ディスクゴルフ団体戦は背番号と名前だけの選手一覧(成績表・対戦一覧なし)。 */
function RosterOnlyView({ data, lang }: { data: TeamResponse; lang: Lang }) {
  return (
    <>
      <TeamHeading team={data.team} lang={lang} />
      <ScvMidaM>{angleHeading(lang, msg(lang, 'playersList'))}</ScvMidaM>
      <Tbl tableStyle={{ width: '70%' }}>
        <TblHeaderRow
          columns={[
            { label: msg(lang, 'colNumber'), width: '10%' },
            { label: msg(lang, 'colPlayer'), width: '90%' },
          ]}
        />
        {data.players.map(({ player }) => (
          <tr key={player.id}>
            <td className="CM">{player.number}</td>
            <td className="LM">
              <PlayerLink id={player.id} gender={player.gender} lang={lang}>
                {pickText(lang, player.name)}
              </PlayerLink>
            </td>
          </tr>
        ))}
      </Tbl>
      <br />
      <br />
      <br />
    </>
  )
}

export function TeamView({ data, lang }: { data: TeamResponse; lang: Lang }) {
  if (data.team.layout === 'roster_only') {
    return <RosterOnlyView data={data} lang={lang} />
  }
  return (
    <>
      <TeamHeading team={data.team} lang={lang} />
      <RecordTable record={data.record} lang={lang} />
      <hr />
      <ScvMidaM>{angleHeading(lang, msg(lang, 'teamGameList'))}</ScvMidaM>
      <br />
      <Tbl>
        <MatchListRows rows={data.matches} lang={lang} />
      </Tbl>
      <hr />
      <ScvMidaM>{angleHeading(lang, msg(lang, 'playersList'))}</ScvMidaM>
      <PlayersTable players={data.players} lang={lang} />
      <hr />
      {data.staff.length > 0 && (
        <>
          <ScvMidaM>{angleHeading(lang, msg(lang, 'staffList'))}</ScvMidaM>
          <StaffTable staff={data.staff} lang={lang} />
          <hr />
        </>
      )}
      <br />
      <br />
      <br />
    </>
  )
}
