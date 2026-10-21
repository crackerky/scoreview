import type { PlayerRef } from '@/api/common.types'
import type { PlayerResponse } from '@/api/public.types'
import { formatAverage, formatElapsed, formatMonthDay } from '@/format/format.utils'
import { teamHref } from '@/i18n/href'
import { type Lang, pickText } from '@/i18n/lang'
import { angleHeading, type MessageKey, msg } from '@/i18n/messages'
import { PlayerLink } from './PlayerLink'
import { ScvMidaM } from './ScvMidaM'
import { Tbl, type TblColumn, TblHeaderRow } from './Tbl'

// clone-spec D 3: 選手ページ /pscore/scv_player/open/{id} の #score_view の中身。
// 前後ナビ → <hr /> → 見出し → 成績表 → <hr /> → イベント(あるときだけ)。
// スタッフは見出しで終わる。写真はない。

const nbsp = ' '

type PersonRef = PlayerResponse['player']
type Stats = PlayerResponse['stats']

/** "#{背番号} : {名前}"。スタッフは "{役割} : {名前}"(推測: 前後ナビのスタッフ表記は未観測) */
function personLabel(person: PersonRef, lang: Lang): string {
  const name = pickText(lang, person.name)
  return person.role === null
    ? `#${person.number} : ${name}`
    : `${pickText(lang, person.role)} : ${name}`
}

/** 前後ナビ。前後は同じチーム内の ID 順で循環する(api が決める)。 */
function NeighborNav({ data, lang }: { data: PlayerResponse; lang: Lang }) {
  const { prev, next, team } = data
  return (
    <table className="WMAX">
      <tbody>
        <tr>
          <td width="30%" className="LM">
            <PlayerLink id={prev.id} gender={prev.gender} lang={lang}>
              {`<=[${personLabel(prev, lang)}]`}
            </PlayerLink>
          </td>
          <td width="40%" className="CM">
            <a className="playerLink" href={teamHref(team.id, lang)}>
              {`[${pickText(lang, team.name)}]`}
            </a>
          </td>
          <td width="30%" className="RM">
            <PlayerLink id={next.id} gender={next.gender} lang={lang}>
              {`[${personLabel(next, lang)}]=>`}
            </PlayerLink>
          </td>
        </tr>
      </tbody>
    </table>
  )
}

/**
 * 成績表の "値{空白}&nbsp;&nbsp;(avg:平均)"。値の後ろの半角空白の数は列ごとに参照サイトの
 * 出力どおり(アシスト 1、ゴール 3、トータル 2。clone-spec D 3.1 の実例)。
 */
function withAverage(total: number, games: number, spaces: number): string {
  return `${total}${' '.repeat(spaces)}${nbsp}${nbsp}(avg:${formatAverage(total, games)})`
}

function columns(lang: Lang, firstWidth: string, restWidth: string, keys: MessageKey[]) {
  return keys.map(
    (key, index): TblColumn => ({
      label: msg(lang, key),
      width: index === 0 ? firstWidth : restWidth,
    }),
  )
}

function StatsRow({ cells }: { cells: string[] }) {
  return (
    <tr>
      {cells.map((text, index) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: 列の並びは固定
        <td key={index} className="RM CM">
          {text}
        </td>
      ))}
    </tr>
  )
}

function UltimateStats({ stats, lang }: { stats: Stats; lang: Lang }) {
  const { games, assists, goals, callahans, greatests, blocks } = stats
  const keys: MessageKey[] = [
    'colGames',
    'colAssist',
    'colGoal',
    'colTotal',
    'colCallahan',
    'colGreatest',
    'colBlock',
  ]
  return (
    <Tbl tableStyle={{ width: '100%' }}>
      <TblHeaderRow columns={columns(lang, '10%', '15%', keys)} />
      <StatsRow
        cells={[
          String(games),
          withAverage(assists, games, 1),
          withAverage(goals, games, 3),
          withAverage(assists + goals, games, 2),
          String(callahans),
          String(greatests),
          // ブロック 0 は空セル(キャラハン・グレイテストの 0 は出す)
          blocks === 0 ? '' : String(blocks),
        ]}
      />
    </Tbl>
  )
}

/**
 * clone-spec D 3.1 の 6: ディスクゴルフの成績表。api にディスクゴルフの集計値がないので
 * 試合数以外は 0 を出す(参照サイトの実例も全て 0)。空白の数は参照サイトの出力どおり。
 */
function DiscGolfStats({ stats, lang }: { stats: Stats; lang: Lang }) {
  const keys: MessageKey[] = [
    'colGames',
    'dgPoints',
    'dgCatch',
    'dgThrowMiss',
    'colTotal',
    'dgCleanCatch',
    'dgFollowCatch',
    'dgFootFault',
  ]
  const { games } = stats
  return (
    <Tbl tableStyle={{ width: '100%' }}>
      <TblHeaderRow columns={columns(lang, '10%', '12.8%', keys)} />
      <StatsRow
        cells={[
          String(games),
          withAverage(0, games, 4),
          withAverage(0, games, 3),
          withAverage(0, games, 4),
          withAverage(0, games, 3),
          '0  ',
          '0  ',
          '0  ',
        ]}
      />
    </Tbl>
  )
}

function eventColumns(lang: Lang): TblColumn[] {
  return [
    { label: msg(lang, 'colCode'), width: '10%' },
    { label: msg(lang, 'colDate'), width: '7%' },
    { label: msg(lang, 'colVsTeam'), width: '20%' },
    { label: msg(lang, 'colElapsed'), width: '7%' },
    { label: msg(lang, 'colEvent'), width: '16%' },
    { label: msg(lang, 'colGoalPlayer'), width: '20%' },
    { label: msg(lang, 'colAssistPlayer'), width: '20%' },
  ]
}

/** イベント表の選手セル "#{背番号}&nbsp;{名前}"(試合ページの半角空白とは違う。clone-spec D 3.1) */
function EventPlayer({ player, lang }: { player: PlayerRef | null; lang: Lang }) {
  if (player === null) {
    return null
  }
  return (
    <PlayerLink id={player.id} gender={player.gender} lang={lang}>
      {`#${player.number}${nbsp}${pickText(lang, player.name)}`}
    </PlayerLink>
  )
}

/**
 * 本人が得点かアシストをした得点の一覧(行のクラスは TG_EVMOJ。EVMOJI ではない)。
 * api はキャラハンかどうかを返さないので、イベント名は常に「得点」。
 */
function EventsTable({ events, lang }: { events: PlayerResponse['events']; lang: Lang }) {
  return (
    <Tbl tableStyle={{ width: '100%' }}>
      <TblHeaderRow columns={eventColumns(lang)} />
      {events.map((event, index) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: イベントに ID がなく、並びは api の順で固定
        <tr key={index}>
          <td className="TG_EVMOJ LM">{event.code}</td>
          <td className="TG_EVMOJ CM">{formatMonthDay(event.date)}</td>
          <td className="TG_EVMOJ LM">
            {event.opponent !== null && (
              <span className="teamLink">
                <a href={teamHref(event.opponent.id, lang)}>
                  {pickText(lang, event.opponent.name)}
                </a>
              </span>
            )}
          </td>
          <td className="TG_EVMOJ CM">{formatElapsed(event.elapsedSec)}</td>
          <td className="TG_EVMOJ CM">{msg(lang, 'eventGoal')}</td>
          <td className="TG_EVMOJ LM">
            <EventPlayer player={event.scorer} lang={lang} />
          </td>
          <td className="TG_EVMOJ LM">
            <EventPlayer player={event.assist} lang={lang} />
          </td>
        </tr>
      ))}
    </Tbl>
  )
}

export function PlayerView({ data, lang }: { data: PlayerResponse; lang: Lang }) {
  const { player, team } = data
  const heading = `${pickText(lang, team.name)} - ${personLabel(player, lang)}`
  return (
    <>
      <NeighborNav data={data} lang={lang} />
      <hr />
      <ScvMidaM>{heading}</ScvMidaM>
      {player.role === null && (
        <>
          {data.statsKind === 'discgolf' ? (
            <DiscGolfStats stats={data.stats} lang={lang} />
          ) : (
            <UltimateStats stats={data.stats} lang={lang} />
          )}
          <hr />
          {data.events.length > 0 && (
            <>
              <ScvMidaM>{angleHeading(lang, msg(lang, 'gameEvents'))}</ScvMidaM>
              <EventsTable events={data.events} lang={lang} />
            </>
          )}
        </>
      )}
    </>
  )
}
