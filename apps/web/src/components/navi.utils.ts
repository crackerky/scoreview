import type { TournamentFrame } from '@/api/common.types'
import {
  divisionHref,
  divisionScheduleHref,
  galleryHref,
  groupHref,
  scvTopHref,
  teamListHref,
  tournamentScheduleHref,
} from '@/i18n/href'
import { type Lang, pickText } from '@/i18n/lang'
import { msg } from '@/i18n/messages'

// 大会ナビ(ul#score_navi)の項目を TournamentFrame から組み立てる。clone-spec E 3 の 7 パターンは
// 「ラベル種別」「スケジュールの有無」「ギャラリーの有無」の組み合わせで決まる。

export type NaviLeaf =
  | { kind: 'link'; label: string; href: string }
  | { kind: 'text'; label: string }

/** ドロップダウン内の 1 項目。submenu は右へ開く 2 階層目(グループ成績のディビジョン)。 */
export type NaviEntry = NaviLeaf | { kind: 'submenu'; label: string; entries: NaviLeaf[] }

export type NaviItem =
  | { kind: 'link'; label: string; href: string }
  | { kind: 'menu'; label: string; widthPx: number | null; entries: NaviEntry[] }

export type NaviModel = {
  items: NaviItem[]
  /** 末尾の幅調整用の空 li の幅(px)。null なら出さない */
  fillerWidthPx: number | null
}

// clone-spec A 1.3 / E 3: ギャラリーの li は幅 172px。末尾の空 li の幅は項目数で決まる
// (1 項目 = 160px + 罫線 1px で、#score_navi 全体を約 980px に揃える)。
const galleryWidthPx = 172
const fillerWidthByItems: Record<
  'three' | 'five',
  { withGallery: number | null; without: number }
> = {
  three: { withGallery: 321, without: 495 },
  five: { withGallery: null, without: 172 },
}

/**
 * グループへのリンクの文字。名前(英語版では英語名)が空なら、参照サイト(CakePHP の link())と同じく
 * リンク先を文字にする(clone-spec A 1.3)。参照サイトは絶対 URL、本サイトはルート相対パス。
 */
export function groupLinkLabel(name: string, href: string): string {
  return name === '' ? href : name
}

export function buildNavi(frame: TournamentFrame, lang: Lang): NaviModel {
  const items: NaviItem[] = [
    { kind: 'link', label: msg(lang, 'navTournamentTop'), href: scvTopHref(frame.id, lang) },
    {
      kind: 'link',
      label: msg(lang, frame.labels.teamList === 'entries' ? 'navEntryList' : 'navTeamList'),
      href: teamListHref(frame.id, lang),
    },
  ]

  if (frame.showSchedule) {
    // 日程のないディビジョン(237 のフリースタイル)はスケジュールとディビジョン成績に出さない。
    // グループ成績には全ディビジョンが出る(clone-spec E 3)
    const scheduled = frame.divisions.filter((division) => division.scheduleEnabled)
    items.push({
      kind: 'menu',
      label: msg(lang, 'navSchedule'),
      widthPx: null,
      entries: [
        {
          kind: 'link',
          label: msg(lang, 'navAllDivision'),
          href: tournamentScheduleHref(frame.id, lang),
        },
        ...scheduled.map(
          (division): NaviEntry => ({
            kind: 'link',
            label: pickText(lang, division.name),
            href: divisionScheduleHref(division.id, lang),
          }),
        ),
      ],
    })
    items.push({
      kind: 'menu',
      label: msg(lang, 'navDivisionScore'),
      widthPx: null,
      entries: scheduled.map((division) => ({
        kind: 'link',
        label: pickText(lang, division.name),
        href: divisionHref(division.id, lang),
      })),
    })
  }

  items.push({
    kind: 'menu',
    label: msg(lang, frame.labels.groups === 'results' ? 'navResult' : 'navGroupScore'),
    widthPx: null,
    // グループが 0 個のディビジョンも名前だけは出る(clone-spec E 3)
    entries: frame.divisions.map((division) => ({
      kind: 'submenu',
      label: pickText(lang, division.name),
      entries: division.groups.map((group): NaviLeaf => {
        if (!group.linked) {
          // 推測: リンクなしのグループは名前だけの <p> で出す(参照サイトで名前つきの実例は未観測)
          return { kind: 'text', label: pickText(lang, group.name) }
        }
        const href = groupHref(group.id, lang)
        return { kind: 'link', label: groupLinkLabel(pickText(lang, group.name), href), href }
      }),
    })),
  })

  const hasGallery = frame.galleries.length > 0
  if (hasGallery) {
    items.push({
      kind: 'menu',
      label: msg(lang, 'navGallery'),
      widthPx: galleryWidthPx,
      entries: frame.galleries.map((gallery) => ({
        kind: 'link',
        label: pickText(lang, gallery.name),
        href: galleryHref(gallery.id, lang),
      })),
    })
  }

  const filler = fillerWidthByItems[frame.showSchedule ? 'five' : 'three']
  return { items, fillerWidthPx: hasGallery ? filler.withGallery : filler.without }
}
