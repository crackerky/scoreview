import type { Suspension } from '@/api/common.types'
import type { Lang } from '@/i18n/lang'

// 大会の中断の帯の文(本サイト独自の追加。docs/requirements.md「意図的な差分」)。閲覧ページの帯と
// 本部の進行画面で同じ文を使う。

const reasonText: Record<Suspension['reason'], { ja: string; en: string }> = {
  thunder: { ja: '雷', en: 'lightning' },
  rain: { ja: '雨', en: 'rain' },
  heat: { ja: '暑さ', en: 'heat' },
  other: { ja: 'その他の理由', en: 'other reasons' },
}

/** 「雷のため中断中(再開予定 13:30)」。その他は本部の説明を理由にする */
export function suspensionText(suspension: Suspension, lang: Lang): string {
  const note = suspension.note.trim()
  const reason =
    suspension.reason === 'other' && note !== ''
      ? note
      : reasonText[suspension.reason][lang === 'en' ? 'en' : 'ja']
  if (lang === 'en') {
    const resume = suspension.resumeAt ? ` (resuming at ${suspension.resumeAt})` : ''
    return `Suspended due to ${reason}${resume}`
  }
  const resume = suspension.resumeAt ? `(再開予定 ${suspension.resumeAt})` : '(再開時刻は未定)'
  return `${reason}のため中断中${resume}`
}
