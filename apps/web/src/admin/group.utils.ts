// 管理画面の組(組を手で足す、組の設定)の計算。

type GroupOrder = { sortOrder: number }
type GroupVisibility = { published: boolean; linked: boolean }

/** 手で足す組の並び順の初めの値。今ある組の一番後ろ(組が無ければ 0) */
export function nextGroupSortOrder(groups: readonly GroupOrder[]): number {
  return groups.length === 0 ? 0 : Math.max(...groups.map((group) => group.sortOrder)) + 1
}

/**
 * 組がスコアサイトに出ているか。今のスコアサイトは、公開していない組も、リンクにしない組(linked が偽)も
 * 出さない。
 */
export function groupShown(group: GroupVisibility): boolean {
  return group.published && group.linked
}
