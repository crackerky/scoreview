import type { MatchRow, TeamRef } from '@/api/common.types'
import type { GroupResponse, PoolMatrixCell } from '@/api/public.types'
import { trimToWidth } from '@/format/format.utils'
import { matchHref, teamHref } from '@/i18n/href'
import { type Lang, pickText } from '@/i18n/lang'
import { msg } from '@/i18n/messages'
import { matrixColumnWidth } from './group.utils'
import { matchCellNote } from './matchResult.utils'
import { ScreenReaderNote } from './ScreenReaderNote'
import { Tbl, type TblColumn, TblHeaderRow } from './Tbl'

// clone-spec C 2.1: 総当たり(プール)型のグループの対戦マトリクスとリーグ結果。

type Pool = NonNullable<GroupResponse['pool']>

// clone-spec C 2.1(a): 列見出しを切り詰めるチーム数と、切り詰める表示幅(半角 1、全角 2)
const narrowMatrixTeams = 5
const narrowHeaderWidth = 4

function TeamName({ team, lang }: { team: TeamRef; lang: Lang }) {
  return (
    <span className="teamimg_rnk">
      <a href={teamHref(team.id, lang)}>{pickText(lang, team.name)}</a>
    </span>
  )
}

/**
 * マトリクスの 1 セル。clone-spec C 2.1(a): 対角 RM_AIDA、下三角 RM_NASI、上三角の試合は RM_ARI
 * (class は 'RM_ARI  CM' と空白 2 つ)。試合詳細のない試合はスコアだけでリンクにしない。
 */
function MatrixCell({
  cell,
  rowOf,
  lang,
}: {
  cell: PoolMatrixCell
  /** 結果一覧の行。結果一覧に無い試合は undefined */
  rowOf: (matchId: number) => MatchRow | undefined
  lang: Lang
}) {
  switch (cell.kind) {
    case 'self':
      return <td className="RM_AIDA CM" />
    case 'lower':
      return <td className="RM_NASI CM" />
    case 'empty':
      // 推測: 上三角で試合が無いセルは参照サイトで未観測。上三角と同じ RM_ARI の空セルにする
      return <td className="RM_ARI  CM" />
    case 'match': {
      // 推測: 試合中(RM_NOW / RM_NOW_Link)は CSS の定義だけがあり、HTML では未観測
      const live = cell.status === 'live'
      const cellClass = live ? 'RM_NOW  CM' : 'RM_ARI  CM'
      const row = rowOf(cell.matchId)
      // 本サイト独自: 試合中と不戦勝などは色とスコアだけでは分からないので、title と読み上げ用の文字で補う
      const note = matchCellNote(cell.status, row, lang)
      const title = note ?? undefined
      // 推測: 未試合の試合はスコアが無いのでリンクにしない(未観測)
      // 結果一覧に無い試合は詳細ありとみなす(リンクを出す)
      if (cell.status === 'scheduled' || !(row?.hasDetail ?? true)) {
        return (
          <td className={cellClass} title={title}>
            {cell.text}
            <ScreenReaderNote text={note} />
          </td>
        )
      }
      return (
        <td className={cellClass} title={title}>
          <a className={live ? 'RM_NOW_Link' : 'RM_ARI_Link'} href={matchHref(cell.matchId, lang)}>
            {cell.text}
            <ScreenReaderNote text={note} />
          </a>
        </td>
      )
    }
  }
}

export function PoolMatrix({
  pool,
  matches,
  lang,
}: {
  pool: Pool
  matches: MatchRow[]
  lang: Lang
}) {
  const rows = new Map(matches.map((match) => [match.matchId, match]))
  const rowOf = (matchId: number) => rows.get(matchId)
  // 5 チーム以上の対戦表は列見出しが狭いので、名前を表示幅 4 で切り詰めてリンクにしない
  // (2182、2188、3272 の 5 チーム、6631 の 6 チーム。4 チーム以下は短縮名のリンク)
  const narrow = pool.teams.length >= narrowMatrixTeams
  const columns: TblColumn[] = [
    { label: msg(lang, 'matrixCorner'), width: '22%' },
    ...pool.teams.map((team) => ({
      label: narrow ? (
        trimToWidth(pickText(lang, team.name), narrowHeaderWidth)
      ) : (
        <TeamName team={team} lang={lang} />
      ),
      width: matrixColumnWidth(pool.teams.length),
    })),
  ]
  return (
    <Tbl>
      <TblHeaderRow columns={columns} cellClass="RM_MD1" />
      {pool.teams.map((team, rowIndex) => (
        <tr key={team.id}>
          <td className="RM_MD1">
            <TeamName team={team} lang={lang} />
          </td>
          {(pool.matrix[rowIndex] ?? []).map((cell, cellIndex) => (
            <MatrixCell
              // biome-ignore lint/suspicious/noArrayIndexKey: マトリクスの列はチームの並びで固定
              key={cellIndex}
              cell={cell}
              rowOf={rowOf}
              lang={lang}
            />
          ))}
        </tr>
      ))}
    </Tbl>
  )
}

// clone-spec C 2.1(b): 1 列目 22%、残り 8 列は 8.667%。勝ち点の列は参照サイトでは HTML コメントの中に
// あって表示されない。React はコメントを出力できないので、勝ち点の列は出さない。
const resultWidth = '8.667%'

function resultColumns(lang: Lang): TblColumn[] {
  const rest = [
    'colGames',
    'colWins',
    'colDraws',
    'colLosses',
    'colGoalsFor',
    'colGoalsAgainst',
    'colGoalsDiff',
    'colRank',
  ] as const
  return [
    { label: msg(lang, 'resultCorner'), width: '22%' },
    ...rest.map((key) => ({ label: msg(lang, key), width: resultWidth })),
  ]
}

/** リーグ結果。行はチームの登録順(順位順ではない。clone-spec C 2.1(b))。 */
export function PoolResults({ pool, lang }: { pool: Pool; lang: Lang }) {
  return (
    <Tbl>
      <TblHeaderRow columns={resultColumns(lang)} />
      {pool.results.map((line) => (
        <tr key={line.team.id}>
          <td className="MD1 LM">
            <TeamName team={line.team} lang={lang} />
          </td>
          <td className="CM HL1">{line.games}</td>
          <td className="CM">{line.wins}</td>
          <td className="CM">{line.draws}</td>
          <td className="CM">{line.losses}</td>
          <td className="CM">{`${line.goalsFor} `}</td>
          <td className="CM">{`${line.goalsAgainst} `}</td>
          <td className="CM HL1">{`${line.goalsFor - line.goalsAgainst} `}</td>
          {/* clone-spec C 2.2: 順位が空になるプールがある */}
          <td className="CM HL2">{line.rank === null ? '' : `${line.rank} `}</td>
        </tr>
      ))}
    </Tbl>
  )
}
