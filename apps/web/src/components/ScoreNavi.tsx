import type { TournamentFrame } from '@/api/common.types'
import type { Lang } from '@/i18n/lang'
import { buildNavi, type NaviEntry, type NaviLeaf } from './navi.utils'

// clone-spec A 1.3: ul#score_navi。第 1 階層のドロップダウン見出しは p.ar、2 階層目は p。
// 開閉は legacy.css の :hover で行う(参照サイトは jQuery の dropdown。JavaScript は使わない)。

function Leaf({ leaf }: { leaf: NaviLeaf }) {
  return (
    <li>{leaf.kind === 'link' ? <a href={leaf.href}>{leaf.label}</a> : <p>{leaf.label}</p>}</li>
  )
}

function Entry({ entry }: { entry: NaviEntry }) {
  if (entry.kind !== 'submenu') {
    return <Leaf leaf={entry} />
  }
  return (
    <li>
      <p>{entry.label}</p>
      <ul>
        {entry.entries.map((leaf, index) => (
          // 同名のグループが並ぶことがあるので位置で識別する
          // biome-ignore lint/suspicious/noArrayIndexKey: 並びは描画ごとに変わらない
          <Leaf key={index} leaf={leaf} />
        ))}
      </ul>
    </li>
  )
}

export function ScoreNavi({ frame, lang }: { frame: TournamentFrame; lang: Lang }) {
  const navi = buildNavi(frame, lang)
  return (
    <ul id="score_navi">
      {navi.items.map((item) =>
        item.kind === 'link' ? (
          <li key={item.href}>
            <a href={item.href}>{item.label}</a>
          </li>
        ) : (
          <li key={item.label} style={item.widthPx ? { width: `${item.widthPx}px` } : undefined}>
            <p className="ar">{item.label}</p>
            <ul>
              {item.entries.map((entry, index) => (
                // biome-ignore lint/suspicious/noArrayIndexKey: ディビジョン名やリンクが重複し得るため位置で識別する
                <Entry key={index} entry={entry} />
              ))}
            </ul>
          </li>
        ),
      )}
      {navi.fillerWidthPx !== null && (
        <li style={{ width: `${navi.fillerWidthPx}px` }}>
          <p>{' '}</p>
        </li>
      )}
    </ul>
  )
}
