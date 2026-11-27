'use client'

import { useRouter } from 'next/navigation'
import { type ChangeEvent, useState, useTransition } from 'react'
import {
  type BuildTeam,
  draftState,
  type PoolPaste,
  parsePoolTable,
  poolLetter,
  snakePools,
} from '@/admin/build.utils'
import { savePoolDraftAction } from '@/app/admin/_actions/format'

// 試合を組むの組分け。チームを組に入れるやり方は 3 つ。選んで入れる(チームを押してから組を押す)、
// 貼り付け・CSV(Excel で決めた組分けを貼るか CSV を選ぶ。申込のチームと照らす)、おまかせ(並びの順に
// 蛇行で配る)。決めた組は「組分けを保存」で下書きに残し、時間割の段で形を作るときに使う。

type Way = 'pick' | 'paste' | 'auto'

const ways: ReadonlyArray<{ id: Way; title: string; sub: string }> = [
  {
    id: 'pick',
    title: '選んで入れる',
    sub: 'チームを押してから、入れる組を押す。強さや同じ学校を見ながら手で決めるとき',
  },
  {
    id: 'paste',
    title: '貼り付け・CSV',
    sub: 'Excel やスプレッドシートで決めた組分けを、そのまま貼るかファイルで入れる',
  },
  { id: 'auto', title: 'おまかせ', sub: 'チームの並びの順に、組へ蛇行(A、B、C、C、B、A…)で配る' },
]

const sampleTable = 'A組\tB組\n(チーム名)\t(チーム名)\n(チーム名)\t(チーム名)'

type Props = {
  divisionId: number
  teams: BuildTeam[]
  /** 保存済みの下書き(無ければ空の組) */
  initialPools: number[][]
}

export function PoolBuilder({ divisionId, teams, initialPools }: Props) {
  const router = useRouter()
  const [pools, setPools] = useState<number[][]>(initialPools)
  const [way, setWay] = useState<Way>('pick')
  const [picked, setPicked] = useState<number | null>(null)
  const [text, setText] = useState('')
  const [pasted, setPasted] = useState<PoolPaste | null>(null)
  const [dirty, setDirty] = useState(false)
  const [message, setMessage] = useState<{ tone: 'ok' | 'error'; text: string } | null>(null)
  const [pending, startTransition] = useTransition()
  const byId = new Map(teams.map((team) => [team.id, team]))
  const placed = new Set(pools.flat())
  const left = teams.filter((team) => !placed.has(team.id))
  const state = draftState(
    pools,
    teams.map((team) => team.id),
  )

  function change(next: number[][]) {
    setPools(next)
    setDirty(true)
    setMessage(null)
  }

  function place(poolIndex: number) {
    if (picked === null) {
      return
    }
    change(
      pools.map((pool, index) =>
        index === poolIndex
          ? [...pool.filter((id) => id !== picked), picked]
          : pool.filter((id) => id !== picked),
      ),
    )
    setPicked(null)
  }

  function remove(teamId: number) {
    change(pools.map((pool) => pool.filter((id) => id !== teamId)))
  }

  function moveUp(poolIndex: number, row: number) {
    change(
      pools.map((pool, index) => {
        if (index !== poolIndex || row === 0) {
          return pool
        }
        const next = [...pool]
        const above = next[row - 1]
        const here = next[row]
        if (above === undefined || here === undefined) {
          return pool
        }
        next[row - 1] = here
        next[row] = above
        return next
      }),
    )
  }

  function setCount(count: number) {
    const size = Math.max(1, Math.min(16, count))
    // 減らした組のチームは、まだ入っていないチームに戻す
    change(Array.from({ length: size }, (_, index) => pools[index] ?? []))
  }

  function readTable(source: string) {
    setText(source)
    setPasted(parsePoolTable(source, teams))
  }

  function readFile(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0]
    if (file) {
      file
        .text()
        .then(readTable, () => setMessage({ tone: 'error', text: 'ファイルを読めませんでした' }))
    }
  }

  function save() {
    startTransition(async () => {
      const result = await savePoolDraftAction(divisionId, pools)
      if (result.ok) {
        setDirty(false)
        setMessage({ tone: 'ok', text: '組分けを保存しました' })
        router.refresh()
      } else {
        setMessage({ tone: 'error', text: result.error })
      }
    })
  }

  const poolCards = (
    <div className="bld-pools">
      {pools.map((pool, index) => (
        <section
          // 組は並びの位置で決まる(A、B、C…)
          // biome-ignore lint/suspicious/noArrayIndexKey: 組は位置そのものが名前
          key={index}
          className={`bld-pool${picked !== null ? ' can' : ''}`}
          aria-label={`${poolLetter(index)} 組`}
        >
          <button
            type="button"
            className="bld-pool-head"
            onClick={() => place(index)}
            disabled={picked === null}
          >
            <strong>{poolLetter(index)} 組</strong>
            <span>{picked === null ? `${pool.length} チーム` : 'ここに入れる'}</span>
          </button>
          {pool.length === 0 ? (
            <p className="adm-muted">チームを選んでから、組の名前を押す</p>
          ) : (
            <ol>
              {pool.map((id, row) => (
                <li key={id}>
                  <span className="bld-pool-row">
                    <span>{byId.get(id)?.shortName ?? `チーム ${id}`}</span>
                    <span className="bld-pool-ops">
                      <button
                        type="button"
                        className="bld-op"
                        onClick={() => moveUp(index, row)}
                        disabled={row === 0}
                        aria-label={`${byId.get(id)?.shortName ?? id}を上へ`}
                      >
                        ↑
                      </button>
                      <button
                        type="button"
                        className="bld-op"
                        onClick={() => remove(id)}
                        aria-label={`${byId.get(id)?.shortName ?? id}を組から外す`}
                      >
                        ×
                      </button>
                    </span>
                  </span>
                </li>
              ))}
            </ol>
          )}
        </section>
      ))}
    </div>
  )

  return (
    <div className="adm-stack">
      <div className="bld-ways">
        {ways.map((item) => (
          <button
            key={item.id}
            type="button"
            className="bld-way"
            aria-pressed={way === item.id}
            onClick={() => setWay(item.id)}
          >
            <strong>{item.title}</strong>
            <span className="adm-muted">{item.sub}</span>
          </button>
        ))}
      </div>

      <div className="adm-actions bld-count">
        <span>組の数</span>
        <button
          type="button"
          className="secondary"
          onClick={() => setCount(pools.length - 1)}
          aria-label="組を減らす"
        >
          −
        </button>
        <strong>{pools.length}</strong>
        <button
          type="button"
          className="secondary"
          onClick={() => setCount(pools.length + 1)}
          aria-label="組を増やす"
        >
          ＋
        </button>
        <span className="adm-muted">
          {state.placed}/{state.total} チームが組に入っています
        </span>
      </div>

      {way === 'pick' ? (
        <div className="bld-split">
          <div>
            <h3 className="bld-h">まだ組に入っていないチーム</h3>
            <div className="bld-list">
              {left.length === 0 ? <p className="adm-muted">全部のチームが組に入りました</p> : null}
              {left.map((team) => (
                <button
                  key={team.id}
                  type="button"
                  className="bld-team"
                  aria-pressed={picked === team.id}
                  onClick={() => setPicked(picked === team.id ? null : team.id)}
                >
                  <span>{team.shortName}</span>
                  {team.fullName !== team.shortName ? <small>{team.fullName}</small> : null}
                </button>
              ))}
            </div>
          </div>
          <div>
            <h3 className="bld-h">組</h3>
            {poolCards}
          </div>
        </div>
      ) : null}

      {way === 'paste' ? (
        <div className="bld-split">
          <div className="adm-stack">
            <p className="adm-muted">
              Excel の表をそのままコピーして貼れます。1
              行目が組の名前でその下にチーム名を縦に並べた形と、「組, チーム名」の 2 列を 1 行 1
              チームにした形(CSV)の、どちらでも読みます。
            </p>
            <label className="adm-field">
              <span>組分けの表</span>
              <textarea
                rows={10}
                value={text}
                placeholder={sampleTable}
                onChange={(event) => setText(event.target.value)}
              />
            </label>
            <div className="adm-actions">
              <button type="button" onClick={() => readTable(text)} disabled={text.trim() === ''}>
                読み込む
              </button>
              <label className="bld-file">
                CSV を選ぶ
                <input type="file" accept=".csv,.tsv,.txt,text/csv" onChange={readFile} />
              </label>
            </div>
          </div>
          <div className="adm-stack">
            {pasted === null ? (
              <p className="adm-muted">読み込むと、申込のチームと照らした結果がここに出ます。</p>
            ) : (
              <>
                {pasted.unknown.length > 0 ? (
                  <div className="adm-notice error">
                    申込のチームと合わない名前が {pasted.unknown.length}{' '}
                    つあります。表を直して読み込み直してください。
                    <ul>
                      {pasted.unknown.map((item) => (
                        <li key={`${item.pool}-${item.name}`}>
                          {item.pool} 組の「{item.name}」
                          {item.near ? `(近いのは「${item.near}」)` : ''}
                        </li>
                      ))}
                    </ul>
                  </div>
                ) : null}
                {pasted.duplicates.length > 0 ? (
                  <p className="adm-notice error">
                    2 回出てくるチーム: {pasted.duplicates.join('、')}
                  </p>
                ) : null}
                {pasted.missing.length > 0 ? (
                  <p className="adm-muted">
                    表に無いチーム(あとで選んで入れられます):{' '}
                    {pasted.missing.map((id) => byId.get(id)?.shortName ?? id).join('、')}
                  </p>
                ) : null}
                <p>
                  {pasted.names
                    .map((name, index) => `${name} 組 ${pasted.pools[index]?.length ?? 0} チーム`)
                    .join('、')}
                </p>
                <div className="adm-actions">
                  <button
                    type="button"
                    onClick={() => {
                      change(pasted.pools)
                      setWay('pick')
                    }}
                    disabled={pasted.pools.length === 0}
                  >
                    この組分けにする({pasted.names.length} 組)
                  </button>
                </div>
              </>
            )}
          </div>
        </div>
      ) : null}

      {way === 'auto' ? (
        <div className="adm-stack">
          <p className="adm-muted">
            チームの並び(申込を受けた順)のまま、{pools.length}{' '}
            組へ蛇行で配ります。配ったあとは「選んで入れる」で入れ替えられます。
          </p>
          <div className="adm-actions">
            <button
              type="button"
              onClick={() => {
                change(
                  snakePools(
                    teams.map((team) => team.id),
                    pools.length,
                  ),
                )
                setWay('pick')
              }}
            >
              {pools.length} 組に配る
            </button>
          </div>
        </div>
      ) : null}

      {way !== 'pick' ? poolCards : null}

      <div className="adm-actions bld-save">
        <button type="button" onClick={save} disabled={pending || !dirty}>
          組分けを保存
        </button>
        {state.complete ? (
          <a className="adm-primary-link" href={`/admin/divisions/${divisionId}/build?step=times`}>
            時間割へ進む
          </a>
        ) : null}
        <span className={message?.tone === 'error' ? 'bld-error' : 'adm-muted'} role="status">
          {message?.text ??
            (dirty
              ? '保存していない変更があります'
              : state.complete
                ? '全部のチームが組に入りました'
                : 'まだ組に入っていないチームか、2 チーム未満の組があります')}
        </span>
      </div>
    </div>
  )
}
