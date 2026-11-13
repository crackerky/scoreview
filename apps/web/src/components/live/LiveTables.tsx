'use client'

import type { MouseEvent } from 'react'
import type { LiveTablesConfig } from '@/api/public.types'
import { BracketTree } from '@/components/tables/BracketTree'
import { PoolMatrixTable, type TableEdit } from '@/components/tables/PoolMatrixTable'
import { bracketViews, columnCount, columnDefs, matrixLegend, poolMatrix } from '@/live/live.tables'
import type { LiveData } from '@/live/live.types'

// 観戦アプリの「表で見る」の中身(星取表とトーナメント表。Claude Design「Score View Admin」の 2b)。
// 観戦アプリの全画面の表と、管理画面「観戦アプリの表」の見本の両方で使う。見本(edit)では、押した
// 部分を選び、出さない列も薄く残して見出しで出し入れできるようにする。

export type TableKind = 'pool' | 'ko'

/** 出せる表の種類(見せ方で出すもの、かつ中身のあるもの)。並びは星取表、トーナメント表 */
export function tableKinds(data: LiveData, config: LiveTablesConfig): TableKind[] {
  const kinds: TableKind[] = []
  if (config.poolShown && data.pools.length > 0) {
    kinds.push('pool')
  }
  if (config.bracketShown && bracketViews(data.tables.brackets, data.matches).length > 0) {
    kinds.push('ko')
  }
  return kinds
}

export const tableNames: Record<TableKind, string> = { pool: '星取表', ko: 'トーナメント表' }

export function TablesPanel({
  data,
  config,
  kind,
  edit,
}: {
  data: LiveData
  config: LiveTablesConfig
  kind: TableKind
  edit?: TableEdit
}) {
  // 見本では、押した場所の data-part / data-col から選ぶ部分を決める(右の欄のボタンでも選べる)
  const pick = (event: MouseEvent<HTMLDivElement>) => {
    if (!edit || !(event.target instanceof Element)) {
      return
    }
    const column = event.target.closest('[data-col]')?.getAttribute('data-col')
    const known = columnDefs.find((def) => def.key === column)
    if (known) {
      edit.onToggleColumn(known.key)
      edit.onPick('cols')
      return
    }
    const part = event.target.closest('[data-part]')?.getAttribute('data-part')
    if (
      part === 'cols' ||
      part === 'mark' ||
      part === 'live' ||
      part === 'order' ||
      part === 'hl'
    ) {
      edit.onPick(part)
    }
  }
  return (
    // biome-ignore lint/a11y/noStaticElementInteractions: 見本の表を押して選ぶ近道(右の欄のボタンでも同じことができる)
    // biome-ignore lint/a11y/useKeyWithClickEvents: 同上
    <div className={edit ? 'lt-panel lt-editing' : 'lt-panel'} onClick={pick}>
      <div className="lt-title">
        <strong>{data.tournament.name}</strong>
        <span>{kind === 'pool' ? '予選リーグ 星取表' : '決勝トーナメント表'}</span>
      </div>
      {kind === 'pool' ? (
        <PoolTables data={data} config={config} edit={edit} />
      ) : (
        <BracketTables data={data} />
      )}
    </div>
  )
}

function PoolTables({
  data,
  config,
  edit,
}: {
  data: LiveData
  config: LiveTablesConfig
  edit?: TableEdit
}) {
  return (
    <div className="lt-stack">
      {data.pools.map((pool) => (
        <PoolMatrixTable
          key={pool.id}
          table={poolMatrix(pool, data.matches, config)}
          config={config}
          edit={edit}
        />
      ))}
      <span className="lt-legend">{matrixLegend(config, columnCount(config))}</span>
    </div>
  )
}

function BracketTables({ data }: { data: LiveData }) {
  const views = bracketViews(data.tables.brackets, data.matches)
  return (
    <div className="lt-stack">
      {views.map((view) => (
        <section key={view.name} className="lt-bracket">
          <h3>{view.name}</h3>
          {view.trees.map((tree) => (
            <BracketTree key={tree.root.match.id} tree={tree} />
          ))}
        </section>
      ))}
      <span className="lt-legend">
        赤い線が勝ち上がり。線の上の数字はその試合の得点です。まだ決まっていない枠は点線で出しています。
      </span>
    </div>
  )
}
