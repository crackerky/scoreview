import type { ScheduleQuery } from '@/api/public'
import type { ScheduleResponse } from '@/api/public.types'
import { type PscoreController, pscoreHref } from '@/i18n/href'
import type { Lang } from '@/i18n/lang'

// スケジュールの絞り込みと並べ替えを URL のクエリで表す。
// 参照サイトは POST(ymd / team / field と sortkey1)で送るが、本サイトは同じ名前の GET クエリで送る
// (requirements「意図的な差分」)。sortkey1 の値は参照サイトと同じ文字列にする(clone-spec B 2)。

/** api の sort の値(docs/api.md)と、参照サイトの sortkey1 の列名の対応。 */
const sortColumns = {
  division: 'mdv.DIVISION_SORTKEY',
  date: 'mat.MATCH_DATE',
  time: 'mat.START_TIME',
  field: 'mat.FIELD_NO',
  code: 'mat.MATCH_CD',
  leftTeam: 'lt_name',
  rightTeam: 'rt_name',
} as const

export type ScheduleSortKey = keyof typeof sortColumns
export type SortDir = 'asc' | 'desc'
export type ScheduleSort = { key: ScheduleSortKey; dir: SortDir }

function isSortKey(value: string): value is ScheduleSortKey {
  return Object.hasOwn(sortColumns, value)
}

/** sortkey1 の値。clone-spec B 2: 降順の接頭辞は DESC_ ではなく DSC_。 */
export function toSortkey1(sort: ScheduleSort): string {
  return `${sort.dir === 'asc' ? 'ASC' : 'DSC'}_${sortColumns[sort.key]}`
}

/** sortkey1 を api の sort / dir にする。知らない値は null(既定の並びで出す)。 */
export function parseSortkey1(value: string): ScheduleSort | null {
  const match = /^(ASC|DSC)_(.+)$/.exec(value)
  if (match === null) {
    return null
  }
  const [, prefix, column] = match
  for (const [key, name] of Object.entries(sortColumns)) {
    if (name === column && isSortKey(key)) {
      return { key, dir: prefix === 'ASC' ? 'asc' : 'desc' }
    }
  }
  return null
}

/** api の応答の sort を画面の並べ替え状態にする。知らない key は null。 */
export function sortFromResponse(sort: ScheduleResponse['sort']): ScheduleSort | null {
  if (sort === null || !isSortKey(sort.key)) {
    return null
  }
  return { key: sort.key, dir: sort.dir }
}

/**
 * 見出しのリンクが次に送る並べ替え。clone-spec B 2: 並べ替え中の列は向きを反転し、
 * それ以外の列は昇順から始める。
 */
export function nextSort(key: ScheduleSortKey, current: ScheduleSort | null): ScheduleSort {
  if (current?.key === key && current.dir === 'asc') {
    return { key, dir: 'desc' }
  }
  return { key, dir: 'asc' }
}

type SearchParams = Record<string, string | string[] | undefined>

function first(value: string | string[] | undefined): string | undefined {
  return Array.isArray(value) ? value[0] : value
}

/**
 * URL のクエリを api のクエリにする。ymd は「キーなし」と「空」で意味が違う(キーなしは
 * 大会全体なら最終日、ディビジョン別なら全日程。空は全日程)ので、キーの有無をそのまま渡す。
 */
export function scheduleQueryFromSearchParams(params: SearchParams): ScheduleQuery {
  const query: ScheduleQuery = {
    ymd: first(params.ymd),
    team: first(params.team),
    field: first(params.field),
  }
  const sortkey1 = first(params.sortkey1)
  const sort = sortkey1 === undefined ? null : parseSortkey1(sortkey1)
  if (sort !== null) {
    query.sort = sort.key
    query.dir = sort.dir
  }
  return query
}

/** 絞り込みの現在値(api が実際に適用した値)。 */
export type ScheduleSelected = ScheduleResponse['filters']['selected']

/**
 * 同じページを絞り込みの現在値で開き直す URL(更新ボタン)。sort を渡すと並べ替えも付ける(見出しのリンク)。
 * clone-spec B 1: 参照サイトの search / sort は select の現在値を毎回すべて送る。
 */
export function scheduleHref(
  controller: Extract<PscoreController, 'scv_search_tn' | 'scv_search_div'>,
  id: number,
  lang: Lang,
  selected: ScheduleSelected,
  sort: ScheduleSort | null = null,
): string {
  const extra: Record<string, string> = {
    ymd: selected.ymd,
    team: selected.team,
    field: selected.field,
  }
  if (sort !== null) {
    extra.sortkey1 = toSortkey1(sort)
  }
  return pscoreHref(controller, id, lang, extra)
}
