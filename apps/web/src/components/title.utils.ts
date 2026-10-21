import type { TournamentFrame } from '@/api/common.types'
import { brand } from '@/brand/brand.config'
import { type Lang, pickText } from '@/i18n/lang'

// clone-spec A 1.7 / B 4 / C 0 / D 0: <title> は "{接頭辞} {大会名} {接尾辞}"。
// 大会名は lang に応じて選び、英語名が空なら空のまま(例 "[CJS]  -All Schedule-")。
// 接尾辞がないページ(大会トップ、チーム一覧、ギャラリー)は末尾に半角空白が 1 つ残る。

/** 大会に属するページの <title>。suffix は "-全スケジュール-" や "-{グループ名}-" など。 */
export function tournamentPageTitle(frame: TournamentFrame, lang: Lang, suffix = ''): string {
  return `${brand.titlePrefix} ${pickText(lang, frame.name)} ${suffix}`
}

/** 大会一覧の <title>。lang=en でも同じ(clone-spec A 1.6)。 */
export function tournamentListTitle(): string {
  return `${brand.titlePrefix} ${brand.tournamentListTitle}`
}
