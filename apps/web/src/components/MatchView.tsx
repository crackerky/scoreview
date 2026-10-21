import type { PlayerRef, Side, TeamRef } from '@/api/common.types'
import type { MatchResponse } from '@/api/public.types'
import {
  formatClockJst,
  formatElapsed,
  formatPercent,
  formatTime,
  formatYmdSlash,
} from '@/format/format.utils'
import { pscoreHref, teamHref } from '@/i18n/href'
import { type Lang, pickText } from '@/i18n/lang'
import { type MessageKey, msg } from '@/i18n/messages'
import { MatchRefreshButton } from './MatchRefreshButton'
import { eventsNewestFirst } from './matchEvents.utils'
import { PlayerLink } from './PlayerLink'
import { Tbl, type TblColumn, TblHeaderRow } from './Tbl'

// clone-spec D 2.2〜2.5: 試合ページ /pscore/scv_match/open/{id} の #score_view の中身。
// スコアボード → 得点経過 → 出場選手 → 統計 の順。前半/後半の区切り、SOTG、MVP はない。

const nbsp = ' '

type Match = MatchResponse['match']
type MatchEvent = MatchResponse['events'][number]
type LineupRow = MatchResponse['lineups']['left'][number]
type TeamStats = MatchResponse['stats']['left']

function teamName(team: TeamRef | null, lang: Lang): string {
  return team === null ? '' : pickText(lang, team.name)
}

/** スコアボードのチーム名(短縮名)。推測: チーム未確定の試合ページは未観測なので空にする */
function ScoreboardTeam({ team, lang }: { team: TeamRef | null; lang: Lang }) {
  return (
    <td className="CM TG_VSTITLE" width="41%">
      <span className="teamimg_big_match">
        {team !== null && <a href={teamHref(team.id, lang)}>{pickText(lang, team.name)}</a>}
      </span>
    </td>
  )
}

/** clone-spec D 2.2: "YYYY/MM/DD&nbsp;HH:MM&nbsp; @Field:&nbsp;{コート}" */
function scheduleText(match: Match, lang: Lang): string {
  const date = formatYmdSlash(match.date)
  const time = formatTime(match.time)
  return `${date}${nbsp}${time}${nbsp} ${msg(lang, 'fieldPrefix')}${nbsp}${match.field ?? ''}`
}

function Scoreboard({ match, now, lang }: { match: Match; now: Date; lang: Lang }) {
  return (
    <div className="TBL">
      <table>
        <tbody>
          <tr>
            <td className="LM TG_MD1" width="16%">
              {match.code}
            </td>
            <td className="LM TG_MD1" width="48%">
              {match.groupName ?? ''}
            </td>
            <td className="CM TG_MD1" width="36%">
              {scheduleText(match, lang)}
            </td>
          </tr>
        </tbody>
      </table>
      <table>
        <tbody>
          <tr style={{ height: '50px' }}>
            <ScoreboardTeam team={match.left.team} lang={lang} />
            <td className="CM TG_POINT" width="7%">
              {match.left.score ?? ''}
            </td>
            <td className="CM TG_VSTITLE" width="6%">
              {msg(lang, 'colVs')}
            </td>
            <td className="CM TG_POINT" width="7%">
              {match.right.score ?? ''}
            </td>
            <ScoreboardTeam team={match.right.team} lang={lang} />
          </tr>
        </tbody>
      </table>
      <table>
        <tbody>
          <tr>
            <td className="CM TG_MD1" width="16%">
              {msg(lang, 'matchResultBadge')}
            </td>
            <td className="LM TG_MD1" width="68%" />
            {/* clone-spec D 2.2: 試合の時刻ではなく、ページを表示した時点の時刻 */}
            <td className="CM TG_MD1" width="16%">
              {formatClockJst(now)}
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  )
}

function eventColumns(lang: Lang): TblColumn[] {
  return [
    { label: msg(lang, 'colElapsed'), width: '6%' },
    { label: msg(lang, 'colAssistPlayer'), width: '17%' },
    { label: msg(lang, 'colGoalPlayer'), width: '17%' },
    { label: msg(lang, 'colPoint'), width: '3%' },
    { label: msg(lang, 'colEvent'), width: '14%' },
    { label: msg(lang, 'colPoint'), width: '3%' },
    { label: msg(lang, 'colGoalPlayer'), width: '17%' },
    { label: msg(lang, 'colAssistPlayer'), width: '17%' },
  ]
}

function eventLabel(event: MatchEvent): MessageKey {
  switch (event.kind) {
    case 'goal':
      // import-cache の試合 45322: キャラハンは得点行の「イベント」列の文言だけが変わる
      return event.callahan ? 'eventCallahan' : 'eventGoal'
    case 'pull':
      return 'eventPull'
    case 'timeout':
      return 'eventTimeout'
    case 'end':
      return 'eventEnd'
  }
}

/** 得点経過の選手セル "#{背番号} {名前}"(半角空白区切り。clone-spec D 2.3) */
function EventPlayer({ player, lang }: { player: PlayerRef | null; lang: Lang }) {
  if (player === null) {
    return null
  }
  return (
    <PlayerLink id={player.id} gender={player.gender} lang={lang}>
      {`#${player.number} ${pickText(lang, player.name)}`}
    </PlayerLink>
  )
}

/** 片側 3 セル(アシスト選手・ゴール選手・点)の中身。左は外側から、右は中心から並べ替えて使う */
type SideCells = { assist: PlayerRef | null; scorer: PlayerRef | null; point: string }

const emptySide: SideCells = { assist: null, scorer: null, point: '' }

function sideCells(event: MatchEvent, side: Side, lang: Lang): SideCells {
  if (event.side !== side) {
    return emptySide
  }
  if (event.kind === 'goal') {
    // 点のセルはその側の累計
    const point = side === 'left' ? event.leftScore : event.rightScore
    return { assist: event.assist, scorer: event.scorer, point: String(point) }
  }
  // clone-spec D 2.3: ★ はプルした側 / タイムアウトを取った側の「点」列に出る
  if (event.kind === 'pull' || event.kind === 'timeout') {
    return { ...emptySide, point: msg(lang, 'eventMark') }
  }
  return emptySide
}

/**
 * 得点経過の 1 行。得点行は左右ミラー(左 = アシスト→ゴール→点、右 = 点→ゴール→アシスト)。
 * イベント行(試合開始ｽﾛｰｵﾌ / タイムアウト / 試合終了)は全セルに EVT が付く。
 */
function EventRow({ event, lang }: { event: MatchEvent; lang: Lang }) {
  const suffix = event.kind === 'goal' ? '' : ' EVT'
  const cell = (align: 'LM' | 'CM') => `TG_EVMOJI ${align}${suffix}`
  const left = sideCells(event, 'left', lang)
  const right = sideCells(event, 'right', lang)
  return (
    <tr>
      <td className={cell('CM')}>{formatElapsed(event.elapsedSec)}</td>
      <td className={cell('LM')}>
        <EventPlayer player={left.assist} lang={lang} />
      </td>
      <td className={cell('LM')}>
        <EventPlayer player={left.scorer} lang={lang} />
      </td>
      <td className={cell('CM')}>{left.point}</td>
      <td className={cell('CM')}>{msg(lang, eventLabel(event))}</td>
      <td className={cell('CM')}>{right.point}</td>
      <td className={cell('LM')}>
        <EventPlayer player={right.scorer} lang={lang} />
      </td>
      <td className={cell('LM')}>
        <EventPlayer player={right.assist} lang={lang} />
      </td>
    </tr>
  )
}

function EventsTable({ events, lang }: { events: MatchEvent[]; lang: Lang }) {
  return (
    <Tbl className="TG_EventList" tableId="events">
      <TblHeaderRow columns={eventColumns(lang)} />
      {eventsNewestFirst(events).map((event, index) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: イベントに ID がなく、並びは描画ごとに固定
        <EventRow key={index} event={event} lang={lang} />
      ))}
    </Tbl>
  )
}

// clone-spec D 2.4: 出場選手表の見出しは jp でも英語
const lineupColumns: TblColumn[] = [
  { label: msg('en', 'lineupNumber'), width: '10%' },
  { label: msg('en', 'lineupPlayer'), width: '41%' },
  { label: msg('en', 'lineupAssist'), width: '13%' },
  { label: msg('en', 'lineupGoal'), width: '13%' },
  { label: msg('en', 'lineupBlock'), width: '13%' },
]

/** 出場選手(固定高 150px でスクロール)。並びは api の順のまま、名前だけをリンクにする */
function LineupTable({ id, rows, lang }: { id: string; rows: LineupRow[]; lang: Lang }) {
  return (
    <Tbl className="TG_PlayersList_Ult" tableId={id}>
      <TblHeaderRow columns={lineupColumns} />
      {rows.map(({ player, assists, goals, blocks }) => (
        <tr key={player.id}>
          <td className="CM">{player.number}</td>
          <td className="LM">
            <PlayerLink id={player.id} gender={player.gender} lang={lang}>
              {pickText(lang, player.name)}
            </PlayerLink>
          </td>
          <td className="CM">{assists}</td>
          <td className="CM">{goals}</td>
          {/* ブロック 0 は空セル(アシスト・ゴールの 0 は出す) */}
          <td className="CM">{blocks === 0 ? '' : blocks}</td>
        </tr>
      ))}
    </Tbl>
  )
}

function Lineups({ lineups, lang }: { lineups: MatchResponse['lineups']; lang: Lang }) {
  return (
    <table style={{ width: '100%' }}>
      <tbody>
        <tr>
          <td width="50%">
            <LineupTable id="lplayers" rows={lineups.left} lang={lang} />
          </td>
          <td width="50%">
            <LineupTable id="rplayers" rows={lineups.right} lang={lang} />
          </td>
        </tr>
      </tbody>
    </table>
  )
}

/** clone-spec D 2.5: "5&nbsp;&nbsp;&nbsp;[71.43%]"。割合の分母はその側の得点数 */
function withShare(count: number, goals: number): string {
  return `${count}${nbsp.repeat(3)}[${formatPercent(count, goals)}]`
}

function StatsRow({ label, left, right }: { label: string; left: string; right: string }) {
  return (
    <tr>
      <td className="LM">{label}</td>
      <td className="LM">{left}</td>
      <td className="LM">{right}</td>
    </tr>
  )
}

// 参照サイトは得点数の次に 4 行(オフェンス時間累計など)を HTML コメントで隠している。
// React は HTML コメントを出せず、api にもその値がないので出さない。
function StatsTable({
  match,
  stats,
  lang,
}: {
  match: Match
  stats: MatchResponse['stats']
  lang: Lang
}) {
  const row = (label: MessageKey, format: (side: TeamStats) => string) => (
    <StatsRow label={msg(lang, label)} left={format(stats.left)} right={format(stats.right)} />
  )
  return (
    <Tbl tableStyle={{ width: '80%' }}>
      <TblHeaderRow
        columns={[
          { label: msg(lang, 'gameStatistics'), width: '40%' },
          { label: teamName(match.left.team, lang), width: '30%' },
          { label: teamName(match.right.team, lang), width: '30%' },
        ]}
      />
      {row('statGoals', (side) => String(side.goals))}
      {row('statFromOffence', (side) => withShare(side.fromOffense, side.goals))}
      {row('statFromDefence', (side) => withShare(side.fromDefense, side.goals))}
      {row('statFromTurnover', (side) => String(side.fromTurnover))}
      {row('statTimeouts', (side) => String(side.timeouts))}
    </Tbl>
  )
}

/**
 * #score_topmenu の中身(clone-spec D 2.1)。参照サイトはここに大会・ディビジョン・グループへの
 * パンくずを HTML コメントで残しているが、React はコメントを出せないので省く。
 */
export function MatchTopMenu({
  matchId,
  lang,
  scrollToNavi,
}: {
  matchId: number
  lang: Lang
  scrollToNavi: boolean
}) {
  return (
    <table style={{ width: '100%' }}>
      <tbody>
        <tr>
          <td className="RM" width="140px" style={{ height: '33px' }}>
            <MatchRefreshButton
              reloadHref={pscoreHref('scv_match', matchId, lang, { refpos: '1' })}
              waitMessage={msg(lang, 'refreshWait')}
              okAlt={msg(lang, 'refreshOkAlt')}
              ngAlt={msg(lang, 'refreshNgAlt')}
              scrollToNavi={scrollToNavi}
            />
            {nbsp}
          </td>
        </tr>
      </tbody>
    </table>
  )
}

/** now はスコアボード右下に出す表示時刻(テストで固定できるよう外から渡す)。 */
export function MatchView({ data, now, lang }: { data: MatchResponse; now: Date; lang: Lang }) {
  return (
    <>
      <Scoreboard match={data.match} now={now} lang={lang} />
      <hr />
      <EventsTable events={data.events} lang={lang} />
      <hr />
      <Lineups lineups={data.lineups} lang={lang} />
      <hr />
      <StatsTable match={data.match} stats={data.stats} lang={lang} />
    </>
  )
}
