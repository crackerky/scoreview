import type { MatchRow } from '@/api/common.types'
import type { GroupResponse } from '@/api/public.types'
import { type Lang, pickText } from '@/i18n/lang'
import { angleHeading, msg } from '@/i18n/messages'
import { BracketTable } from './BracketTable'
import { MatchListRows } from './MatchRowLine'
import { PoolNotes } from './PoolNotes'
import { PoolMatrix, PoolResults } from './PoolTables'
import { ScvMidaM } from './ScvMidaM'
import { Tbl } from './Tbl'

// clone-spec C 2: グループ成績 /pscore/scv_group/open/{id} の #score_view の中身。
// kind に応じて、総当たり(マトリクス + リーグ結果 + 結果一覧)、トーナメント(ブラケット + 結果一覧)、
// 画像のみ(clone-spec E 4)の 3 種を出し分ける。

/** api の応答で kind と非 null の節が食い違っているとき(docs/api.md の契約違反)に投げる。 */
export class GroupShapeError extends Error {
  constructor(groupId: number, kind: string) {
    super(`group ${groupId}: kind=${kind} but its section is null`)
    this.name = 'GroupShapeError'
  }
}

/** 結果一覧の節と、参照サイトの末尾にある空の div.TBL(clone-spec C 2.1、C 2.3)。 */
function GameList({ matches, lang }: { matches: MatchRow[]; lang: Lang }) {
  return (
    <>
      <ScvMidaM>{angleHeading(lang, msg(lang, 'groupGameList'))}</ScvMidaM>
      <br />
      <Tbl>
        <MatchListRows rows={matches} lang={lang} />
      </Tbl>
      <hr />
      <div className="TBL"> </div>
    </>
  )
}

function GroupBody({ data, lang }: { data: GroupResponse; lang: Lang }) {
  const { group } = data
  switch (group.kind) {
    case 'pool': {
      if (data.pool === null) {
        throw new GroupShapeError(group.id, group.kind)
      }
      return (
        <>
          <PoolMatrix pool={data.pool} matches={data.matches} lang={lang} />
          <hr />
          <ScvMidaM>{angleHeading(lang, msg(lang, 'leagueResult'))}</ScvMidaM>
          <br />
          <PoolResults pool={data.pool} lang={lang} />
          <PoolNotes notes={data.pool.notes} lang={lang} />
          <hr />
          <GameList matches={data.matches} lang={lang} />
        </>
      )
    }
    case 'bracket': {
      if (data.bracket === null) {
        throw new GroupShapeError(group.id, group.kind)
      }
      return (
        <>
          <BracketTable bracket={data.bracket} matches={data.matches} lang={lang} />
          <br />
          <GameList matches={data.matches} lang={lang} />
        </>
      )
    }
    case 'image': {
      const { image } = data
      if (image === null) {
        throw new GroupShapeError(group.id, group.kind)
      }
      // clone-spec E 4: div.TBL の中に img だけ。大きさは style の px で指定する
      return (
        <div className="TBL">
          <img
            src={image.url}
            alt=""
            style={{
              width: image.width === null ? undefined : `${image.width}px`,
              height: image.height === null ? undefined : `${image.height}px`,
            }}
          />
        </div>
      )
    }
  }
}

export function GroupView({ data, lang }: { data: GroupResponse; lang: Lang }) {
  // clone-spec C 2: "< {ディビジョン名} - {グループ名} >"(en は "<Beach Mixed - Tounament A>")
  const heading = `${pickText(lang, data.group.division.name)} - ${data.group.name}`
  return (
    <>
      <ScvMidaM>{angleHeading(lang, heading)}</ScvMidaM>
      <br />
      <GroupBody data={data} lang={lang} />
    </>
  )
}
