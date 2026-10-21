import type { Gender } from '@/api/common.types'
import type { StandingsResponse } from '@/api/public.types'
import { formatWinRate } from '@/format/format.utils'
import { playerHref, teamHref } from '@/i18n/href'
import { type Lang, pickText } from '@/i18n/lang'
import { angleHeading, msg } from '@/i18n/messages'
import { ScvMidaM } from './ScvMidaM'
import { Tbl, type TblColumn, TblHeaderRow } from './Tbl'

// clone-spec C 1: ディビジョン成績 /pscore/scv_division/open/{id} の #score_view の中身。
// 順位表、スコアボードリーダー、ブロックの 3 節を、データがある節だけ上から順に出す。各節の後に <hr />。
// 得点・失点・得失差・順位・合計のセルは数値の後ろに半角空白が付く(参照サイトの原文ママ)。

/** clone-spec D 1: 選手リンクの色は性別で変わる。性別のない旧データは playerLink。 */
function playerLinkClass(gender: Gender | null): string {
  if (gender === null) {
    return 'playerLink'
  }
  return gender === 'F' ? 'playerLinkF' : 'playerLinkM'
}

/** 数値の後ろに半角空白を 1 つ付ける(clone-spec C 1.2)。 */
function spaced(value: number): string {
  return `${value} `
}

// clone-spec C 1.2: 1 列目 24%、残り 9 列は 8.444%
const standingsWidth = '8.444%'

function standingsColumns(lang: Lang): TblColumn[] {
  const rest = [
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
  return [
    { label: msg(lang, 'colTeams'), width: '24%' },
    ...rest.map((key) => ({ label: msg(lang, key), width: standingsWidth })),
  ]
}

function StandingsTable({ data, lang }: { data: StandingsResponse; lang: Lang }) {
  return (
    <Tbl>
      <TblHeaderRow columns={standingsColumns(lang)} />
      {data.standings.map((line) => (
        <tr key={line.team.id}>
          <td className="MD1">
            <span className="teamimg_rnk">
              <a href={teamHref(line.team.id, lang)}>{pickText(lang, line.team.name)}</a>
            </span>{' '}
          </td>
          <td className="CM HL1">{line.games}</td>
          <td className="CM">{line.wins}</td>
          <td className="CM">{line.draws}</td>
          <td className="CM">{line.losses}</td>
          <td className="CM">{spaced(line.goalsFor)}</td>
          <td className="CM">{spaced(line.goalsAgainst)}</td>
          <td className="CM HL1">{spaced(line.goalsFor - line.goalsAgainst)}</td>
          <td className="CM HL1">{formatWinRate(line.wins, line.games)}</td>
          {/* clone-spec C 1.4: 順位決定戦が終わるまで順位は空 */}
          <td className="CM HL2">{line.rank === null ? '' : spaced(line.rank)}</td>
        </tr>
      ))}
    </Tbl>
  )
}

/** clone-spec C 1.5: リーダー表の先頭 3 列(チーム名はリンクなし、# の幅は原文ママで ' 5%')。 */
function leaderLeadColumns(lang: Lang): TblColumn[] {
  return [
    { label: msg(lang, 'colTeam'), width: '25%' },
    { label: msg(lang, 'colNumber'), width: ' 5%' },
    { label: msg(lang, 'colPlayer'), width: '30%' },
    { label: msg(lang, 'colGames'), width: '10%' },
  ]
}

/** 得点とブロックのリーダー表に共通する部分。 */
type LeaderBase = Pick<StandingsResponse['leaders'][number], 'team' | 'player' | 'games'>

function LeaderLeadCells({ line, lang }: { line: LeaderBase; lang: Lang }) {
  return (
    <>
      <td className="LM">
        <span className="teamimg">{pickText(lang, line.team.name)}</span>{' '}
      </td>
      <td className="CM">{line.player.number}</td>
      <td className="LM">
        <a className={playerLinkClass(line.player.gender)} href={playerHref(line.player.id, lang)}>
          {pickText(lang, line.player.name)}
        </a>
      </td>
      <td className="CM HL1">{line.games}</td>
    </>
  )
}

function LeadersTable({ data, lang }: { data: StandingsResponse; lang: Lang }) {
  const columns: TblColumn[] = [
    ...leaderLeadColumns(lang),
    { label: msg(lang, 'colAssist'), width: '10%' },
    { label: msg(lang, 'colGoal'), width: '10%' },
    { label: msg(lang, 'colTotal'), width: '10%' },
  ]
  return (
    <Tbl>
      <TblHeaderRow columns={columns} />
      {data.leaders.map((line) => (
        <tr key={line.player.id}>
          <LeaderLeadCells line={line} lang={lang} />
          <td className="CM">{line.assists}</td>
          <td className="CM">{line.goals}</td>
          <td className="CM HL1">{spaced(line.assists + line.goals)}</td>
        </tr>
      ))}
    </Tbl>
  )
}

function BlockLeadersTable({ data, lang }: { data: StandingsResponse; lang: Lang }) {
  const columns: TblColumn[] = [
    ...leaderLeadColumns(lang),
    { label: msg(lang, 'colBlock'), width: '10%' },
    // clone-spec C 1.5: 最終列は見出しが "--" で、行は空
    { label: msg(lang, 'colEmpty'), width: '20%' },
  ]
  return (
    <Tbl>
      <TblHeaderRow columns={columns} />
      {data.blockLeaders.map((line) => (
        <tr key={line.player.id}>
          <LeaderLeadCells line={line} lang={lang} />
          <td className="CM HL1">{spaced(line.blocks)}</td>
          <td className="CM HL1" />
        </tr>
      ))}
    </Tbl>
  )
}

export function DivisionStandingsView({ data, lang }: { data: StandingsResponse; lang: Lang }) {
  // clone-spec C 1.1: jp "< ビーチミックス 順位表 >"、en "<Beach Mixed Standings>"
  const standingsHeading = `${pickText(lang, data.division.name)} ${msg(lang, 'standings')}`
  return (
    <>
      {data.standings.length > 0 && (
        <>
          <ScvMidaM>{angleHeading(lang, standingsHeading)}</ScvMidaM>
          <StandingsTable data={data} lang={lang} />
          <hr />
        </>
      )}
      {data.leaders.length > 0 && (
        <>
          <ScvMidaM>{angleHeading(lang, msg(lang, 'scoreboardLeaders'))}</ScvMidaM>
          <LeadersTable data={data} lang={lang} />
          <hr />
        </>
      )}
      {data.blockLeaders.length > 0 && (
        <>
          <ScvMidaM>{angleHeading(lang, msg(lang, 'scoreboardLeadersBlock'))}</ScvMidaM>
          <BlockLeadersTable data={data} lang={lang} />
          <hr />
        </>
      )}
    </>
  )
}
