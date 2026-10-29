import type { AdminMatch } from '@/api/admin.types'
import type { MatchSide } from '@/api/common.types'

// 管理画面に出す選択肢の文言と、値の表示の整形。

export const tournamentStatusOptions = [
  { value: 'entry_open', label: 'エントリー受付中' },
  { value: 'entry_closed', label: 'エントリー締切り' },
  { value: 'finished', label: '大会終了' },
] as const

export const divisionKindOptions = [
  { value: 'mixed', label: 'ミックス' },
  { value: 'men', label: 'メン' },
  { value: 'women', label: 'ウィメン' },
  { value: 'none', label: 'その他(色なし)' },
] as const

export const entryKindOptions = [
  { value: 'team', label: 'チーム戦' },
  { value: 'individual', label: '個人戦' },
] as const

export const statsKindOptions = [
  { value: 'ultimate', label: 'アルティメット' },
  { value: 'discgolf', label: 'ディスクゴルフ' },
] as const

export const groupKindOptions = [
  { value: 'pool', label: '総当たり' },
  { value: 'bracket', label: 'トーナメント' },
  { value: 'image', label: '画像(組の表を画像で出す)' },
] as const

export const barStyleOptions = [
  { value: 'default', label: '標準' },
  { value: 'mixed', label: 'ミックス' },
  { value: 'women', label: 'ウィメン' },
  { value: 'univ', label: '大学' },
  { value: 'waku1', label: '枠 1' },
  { value: 'waku2', label: '枠 2' },
] as const

export const matchStatusOptions = [
  { value: 'scheduled', label: '未試合' },
  { value: 'live', label: '試合中' },
  { value: 'finished', label: '終了' },
] as const

export const roleOptions = [
  { value: 'staff', label: '記録係' },
  { value: 'hq', label: '本部スタッフ' },
  { value: 'admin', label: '管理者' },
] as const

export function statusLabel(status: AdminMatch['status']): string {
  return matchStatusOptions.find((option) => option.value === status)?.label ?? status
}

/** 経過秒を「分:秒」にする(得点経過の時刻の表記) */
export function formatElapsed(seconds: number): string {
  const minutes = Math.floor(seconds / 60)
  return `${minutes}:${String(seconds % 60).padStart(2, '0')}`
}

/** 試合の片側の表示名。チームが未確定なら "[A1-勝ち]"(clone-spec B 8 と同じ表記) */
export function sideName(side: MatchSide): string {
  if (side.team) {
    return side.team.name.ja
  }
  if (side.placeholder) {
    return `[${side.placeholder.code}-${side.placeholder.outcome === 'win' ? '勝ち' : '負け'}]`
  }
  return '未定'
}

/**
 * 試合の片側のフォームの初期値(チーム ID か "A1-win")。sideChoiceSchema と対になる。
 * 未確定の参照があればそちらを出す(参照が解決してチームが入っていても、参照のほうが登録した値のため)。
 */
export function sideChoiceValue(side: MatchSide): string {
  if (side.placeholder) {
    return `${side.placeholder.code}-${side.placeholder.outcome}`
  }
  if (side.team) {
    return String(side.team.id)
  }
  return ''
}
