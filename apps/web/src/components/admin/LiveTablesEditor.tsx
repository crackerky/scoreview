'use client'

import { useState } from 'react'
import type { LiveTablesConfig } from '@/api/public.types'
import { type TableKind, TablesPanel, tableNames } from '@/components/live/LiveTables'
import {
  type ColumnKey,
  columnCount,
  columnDefs,
  fitColumns,
  markSets,
  poolMatrix,
  type TablePart,
} from '@/live/live.tables'
import type { LiveData } from '@/live/live.types'

// 管理画面「観戦アプリの表」の中身(Claude Design「Score View Admin」の 2a)。左の見本を押すか右の
// ボタンで変えたい部分を選び、右で見せ方を選ぶ。見せ方は hidden 欄 config に入れ、外のフォームの
// 「保存する」で送る。

const parts: Array<{ key: TablePart; label: string }> = [
  { key: 'cols', label: '右の列' },
  { key: 'mark', label: '勝ち負けの記号' },
  { key: 'live', label: '試合中のマス' },
  { key: 'order', label: 'チームの並び' },
  { key: 'hl', label: '進出圏の色' },
]

function Switch({ on }: { on: boolean }) {
  return (
    <span className={on ? 'ltc-switch on' : 'ltc-switch'} aria-hidden="true">
      <span className="ltc-knob" />
    </span>
  )
}

function Option({
  on,
  label,
  onChoose,
  children,
}: {
  on: boolean
  label: string
  onChoose: () => void
  children?: React.ReactNode
}) {
  return (
    <button type="button" className="ltc-option" aria-pressed={on} onClick={onChoose}>
      <span className="ltc-option-head">
        <span className="ltc-radio" aria-hidden="true" />
        <span>{label}</span>
      </span>
      {children}
    </button>
  )
}

export function LiveTablesEditor({ data, saved }: { data: LiveData; saved: LiveTablesConfig }) {
  const [config, setConfig] = useState(saved)
  const [kind, setKind] = useState<TableKind>('pool')
  const [selected, setSelected] = useState<TablePart>('mark')
  const [editing, setEditing] = useState(true)
  const change = (patch: Partial<LiveTablesConfig>) => setConfig({ ...config, ...patch })
  const toggleColumn = (key: ColumnKey) =>
    setConfig({ ...config, columns: { ...config.columns, [key]: !config.columns[key] } })
  const dirty = JSON.stringify(config) !== JSON.stringify(saved)
  const none = config.shown && !config.poolShown && !config.bracketShown
  const warning = !config.shown
    ? 'いまは観戦アプリに表を出していません。左は編集用の表示です。'
    : none
      ? '表を1つも選んでいません。出さないときは、上のスイッチを切ってください。'
      : !(kind === 'pool' ? config.poolShown : config.bracketShown)
        ? 'この表は観戦アプリに出しません。上のスイッチで切り替えます。'
        : ''
  const firstPool = data.pools[0]
  const sample = firstPool ? poolMatrix(firstPool, data.matches, config).rows[0] : undefined
  const count = columnCount(config)
  const orderSample = (order: LiveTablesConfig['order']) =>
    firstPool ? poolMatrix(firstPool, data.matches, { ...config, order }).rows : []
  const kindShown = (item: TableKind) => (item === 'pool' ? config.poolShown : config.bracketShown)
  return (
    <div className="ltc">
      <input type="hidden" name="config" value={JSON.stringify(config)} />
      <button
        type="button"
        className="ltc-master"
        aria-pressed={config.shown}
        onClick={() => change({ shown: !config.shown })}
      >
        <Switch on={config.shown} />
        <span className="ltc-master-text">
          <strong>観戦アプリに「表で見る」を出す</strong>
          <span>引き出し(順位)の入口ごと出し入れします</span>
        </span>
      </button>
      <div className="ltc-columns">
        <div className="ltc-preview-column">
          <div className="ltc-modes">
            <button type="button" aria-pressed={editing} onClick={() => setEditing(true)}>
              編集
            </button>
            <button type="button" aria-pressed={!editing} onClick={() => setEditing(false)}>
              観戦者の見え方
            </button>
          </div>
          <div className="ltc-phone">
            <div className="ltc-phone-screen">
              <div className="lt-body">
                <TablesPanel
                  data={data}
                  config={config}
                  kind={kind}
                  edit={
                    editing
                      ? { selected, onPick: setSelected, onToggleColumn: toggleColumn }
                      : undefined
                  }
                />
              </div>
            </div>
          </div>
          <span className="ltc-note">
            {editing
              ? '出さない列も薄く残しています。見出しを押すと出し入れできます。'
              : '保存すると、観戦者にはこう見えます。'}
          </span>
        </div>

        <div className="ltc-settings">
          <div className="ltc-tabs">
            {(['pool', 'ko'] as const).map((item) => (
              <div key={item} className={item === kind ? 'ltc-tab current' : 'ltc-tab'}>
                <button type="button" className="ltc-tab-name" onClick={() => setKind(item)}>
                  {tableNames[item]}
                </button>
                <button
                  type="button"
                  className="ltc-tab-switch"
                  aria-label={`${tableNames[item]}を観戦アプリに出す`}
                  aria-pressed={kindShown(item)}
                  onClick={() =>
                    change(
                      item === 'pool'
                        ? { poolShown: !config.poolShown }
                        : { bracketShown: !config.bracketShown },
                    )
                  }
                >
                  <Switch on={kindShown(item)} />
                  <span>{kindShown(item) ? '出す' : '出さない'}</span>
                </button>
              </div>
            ))}
          </div>
          {warning !== '' ? <div className="ltc-warn">{warning}</div> : null}

          {kind === 'pool' ? (
            <div className="ltc-pool">
              <p className="ltc-hint">
                左の表で変えたい所を押すか、ここから選んでください。列は見出しを押すと出し入れできます。
              </p>
              <div className="ltc-parts">
                {parts.map((part) => (
                  <button
                    key={part.key}
                    type="button"
                    aria-pressed={selected === part.key}
                    onClick={() => {
                      setSelected(part.key)
                      setEditing(true)
                    }}
                  >
                    {part.label}
                  </button>
                ))}
              </div>
              <hr />
              {selected === 'cols' ? (
                <section className="ltc-section">
                  <h2>右に並べる列</h2>
                  <div className="ltc-checks">
                    {columnDefs.map((def) => (
                      <button
                        key={def.key}
                        type="button"
                        className="ltc-check"
                        aria-pressed={config.columns[def.key]}
                        onClick={() => toggleColumn(def.key)}
                      >
                        <span className="ltc-box" aria-hidden="true">
                          {config.columns[def.key] ? '✓' : ''}
                        </span>
                        <span className="ltc-check-label">{def.label}</span>
                        <span
                          className="ltc-mini-heads"
                          style={{ opacity: config.columns[def.key] ? 1 : 0.35 }}
                        >
                          {def.heads.map((head, index) => (
                            <span key={head} className="ltc-mini-col">
                              <span className="ltc-mini-head">{head}</span>
                              <span className="ltc-mini-value">
                                {sample?.stats[def.key][index] ?? ''}
                              </span>
                            </span>
                          ))}
                        </span>
                      </button>
                    ))}
                  </div>
                  <span className={count <= fitColumns ? 'ltc-fit ok' : 'ltc-fit'}>
                    {count <= fitColumns
                      ? `いま${count}列。スマホで横にずらさずに見られます。`
                      : `いま${count}列。スマホでは横にずらして見ます(${fitColumns}列までなら1画面)。`}
                  </span>
                </section>
              ) : null}
              {selected === 'mark' ? (
                <section className="ltc-section">
                  <h2>勝ち負けの記号</h2>
                  <div className="ltc-options">
                    {(
                      [
                        ['maru', 'まる・さんかく・ばつ'],
                        ['wdl', '英字'],
                        ['kanji', '漢字'],
                      ] as const
                    ).map(([key, label]) => (
                      <Option
                        key={key}
                        on={config.marks === key}
                        label={label}
                        onChoose={() => change({ marks: key })}
                      >
                        <span className="ltc-mark-cells">
                          {markSets[key].marks.map((mark, index) => (
                            <span key={mark} className="ltc-mark-cell">
                              <strong>{mark}</strong>
                              <span
                                className={
                                  index === 2 ? 'ltc-mark-score lt-lost' : 'ltc-mark-score'
                                }
                              >
                                {['7-5', '6-6', '3-7'][index]}
                              </span>
                            </span>
                          ))}
                        </span>
                      </Option>
                    ))}
                  </div>
                  <span className="ltc-note">
                    左から 勝ち・引き分け・負け。表の下の説明も同じ記号に変わります。
                  </span>
                </section>
              ) : null}
              {selected === 'live' ? (
                <section className="ltc-section">
                  <h2>試合中のマス</h2>
                  <div className="ltc-options">
                    <Option
                      on={config.live === 'score'}
                      label="いまの点を出す"
                      onChoose={() => change({ live: 'score' })}
                    >
                      <span className="ltc-live-cell">
                        <span>● 試合中</span>
                        <strong>4-3</strong>
                      </span>
                      <span className="ltc-option-sub">その場の点を橙で出します</span>
                    </Option>
                    <Option
                      on={config.live === 'blank'}
                      label="終わるまで空ける"
                      onChoose={() => change({ live: 'blank' })}
                    >
                      <span className="ltc-live-cell">
                        <span>● 試合中</span>
                        <strong />
                      </span>
                      <span className="ltc-option-sub">結果が出てから表に入ります</span>
                    </Option>
                  </div>
                </section>
              ) : null}
              {selected === 'order' ? (
                <section className="ltc-section">
                  <h2>チームの並び</h2>
                  <div className="ltc-options">
                    {(
                      [
                        ['entry', '登録順', '表の中の位置が変わりません。紙の表と同じ'],
                        ['rank', '順位順', '上から1位。試合が終わるたびに入れ替わります'],
                      ] as const
                    ).map(([key, label, sub]) => (
                      <Option
                        key={key}
                        on={config.order === key}
                        label={label}
                        onChoose={() => change({ order: key })}
                      >
                        <span className="ltc-order-rows">
                          {orderSample(key).map((row) => (
                            <span key={row.teamId} className="ltc-order-row">
                              <span className="lt-bar" style={{ background: row.color }} />
                              <span className="ltc-order-name">{row.name}</span>
                              <span>{row.stats.rank[0]}位</span>
                            </span>
                          ))}
                        </span>
                        <span className="ltc-option-sub">{sub}</span>
                      </Option>
                    ))}
                  </div>
                </section>
              ) : null}
              {selected === 'hl' ? (
                <section className="ltc-section">
                  <h2>進出圏の色</h2>
                  <div className="ltc-options">
                    {([true, false] as const).map((on) => (
                      <Option
                        key={String(on)}
                        on={config.highlight === on}
                        label={on ? '色をつける' : 'つけない'}
                        onChoose={() => change({ highlight: on })}
                      >
                        <span className="ltc-rank-cells">
                          {[1, 2, 3].map((rank) => (
                            <span
                              key={rank}
                              className="ltc-rank-cell"
                              style={{
                                background:
                                  on && rank <= (firstPool?.advance ?? 0) ? '#ffe3c2' : '#fff',
                              }}
                            >
                              {rank}
                            </span>
                          ))}
                        </span>
                      </Option>
                    ))}
                  </div>
                  <span className="ltc-note">
                    {firstPool && firstPool.advance > 0
                      ? `各組の上位${firstPool.advance}チームが優勝を争うトーナメントへ進みます。`
                      : '大会の形を決めると、優勝を争うトーナメントへ進む順位に色がつきます。'}
                    「順位」の列を出しているときだけ色がつきます。
                  </span>
                </section>
              ) : null}
            </div>
          ) : (
            <section className="ltc-ko">
              <h2>トーナメント表</h2>
              <p>
                大会の形の組み合わせから自動で描きます。勝ち上がりは赤い線、まだ決まっていない枠は点線の枠で出します。ここで決めるのは、観戦アプリに出すかどうかだけです。
              </p>
            </section>
          )}
          <p className="ltc-note">
            {dirty
              ? '「保存する」を押すと、観戦アプリにすぐ反映されます。開いている人の画面も 15 秒ほどで切り替わります。'
              : 'いまの見せ方は保存済みです。'}
          </p>
        </div>
      </div>
    </div>
  )
}
