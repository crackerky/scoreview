import type { TeamResponse } from '@/api/public.types'
import { formatWinRate } from '@/format/format.utils'
import { type Lang, pickText } from '@/i18n/lang'
import { angleHeading, msg } from '@/i18n/messages'
import { MatchListRows, statusClass } from './MatchRowLine'
import { PlayerLink } from './PlayerLink'
import { ScvMidaM } from './ScvMidaM'
import { Tbl, type TblColumn, TblHeaderRow } from './Tbl'

// clone-spec D 1: チームページ /pscore/scv_team/open/{id} の #score_view の中身。
// SOTG 表(D 1.1 の 6)は参照サイトにも値がなく、表の形だけを出す(入力はフェーズ 2)。

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
      {players.map(({ player, games, assists, goals, blocks, absent }) => {
        // clone-spec D 1.1: 欠場の選手は行の全セルに ABS(灰色の背景)を付ける
        const cm = absent ? 'CM ABS' : 'CM'
        return (
          <tr key={player.id}>
            <td className={cm}>{player.number}</td>
            <td className={absent ? 'LM ABS' : 'LM'}>
              <PlayerLink id={player.id} gender={player.gender} lang={lang}>
                {pickText(lang, player.name)}
              </PlayerLink>
            </td>
            <td className={cm}>{games}</td>
            <td className={cm}>{assists}</td>
            <td className={cm}>{goals}</td>
            <td className={cm}>{`${assists + goals}${trailing}`}</td>
            {/* clone-spec D 1.1: ブロック 0 は数字を出さない(空白 2 つだけのセル) */}
            <td className={cm}>{`${blocks === 0 ? '' : blocks}${trailing}`}</td>
            <td className={cm} />
          </tr>
        )
      })}
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

/**
 * clone-spec D 1.1 の 6: SOTG スコア一覧。見出しと凡例の表、試合ごとの行(CD と相手チームの短縮名、
 * 値は空)、平均の行。参照サイトでも値は空。
 */
function SotgTables({ data, lang }: { data: TeamResponse; lang: Lang }) {
  const [line1, line2] = msg(lang, 'sotgLegend').split('\n')
  const items = [
    'colSotg1',
    'colSotg2',
    'colSotg3',
    'colSotg4',
    'colSotg5',
    'colSotgTotal',
  ] as const
  const columns: TblColumn[] = [
    { label: 'CD', width: '10%' },
    { label: msg(lang, 'colOpponent'), width: '20%' },
    ...items.map((key) => ({ label: msg(lang, key), width: '11.6%' })),
  ]
  const emptyCells = (className: string) =>
    ['1', '2', '3', '4', '5', 'total'].map((key) => <td key={key} className={className} />)
  return (
    <>
      <div className="">
        <table className="WMAX">
          <tbody>
            <tr>
              <td className="LM" width="26%">
                <ScvMidaM>{angleHeading(lang, msg(lang, 'sotgScoreList'))}</ScvMidaM>
              </td>
              <td className="LM" width="74%">
                <span className="SCV_SOTG_COMM">
                  {line1}
                  <br />
                  {/* 参照サイトは <br /> の後に改行と字下げがあり、画面では空白 1 つ */}
                  {` ${line2}`}
                </span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <Tbl>
        <TblHeaderRow columns={columns} />
        {data.matches.map((row) => {
          const state = statusClass[row.status]
          const opponent = row.left.team?.id === data.team.id ? row.right.team : row.left.team
          return (
            <tr key={row.matchId}>
              <td className={`${state} LM`}>{row.code}</td>
              <td className={`${state} LM`}>
                {opponent === null ? '' : pickText(lang, opponent.name)}
              </td>
              {emptyCells(`${state} CM`)}
            </tr>
          )
        })}
        <tr>
          <td className="MD1 CM" colSpan={2}>
            {msg(lang, 'sotgAverage')}
          </td>
          {emptyCells('MD1 CM')}
        </tr>
      </Tbl>
      <hr />
    </>
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
      {/* 試合が 1 つもないチーム(237 の中止大会)は対戦一覧を見出しごと出さない */}
      {data.matches.length > 0 && (
        <>
          <ScvMidaM>{angleHeading(lang, msg(lang, 'teamGameList'))}</ScvMidaM>
          <br />
          <Tbl>
            <MatchListRows rows={data.matches} lang={lang} />
          </Tbl>
          <hr />
        </>
      )}
      {data.showSotg && <SotgTables data={data} lang={lang} />}
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
