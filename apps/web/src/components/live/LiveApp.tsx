'use client'

import {
  type KeyboardEvent,
  type PointerEvent,
  type ReactNode,
  useCallback,
  useEffect,
  useRef,
  useState,
  type WheelEvent,
} from 'react'
import {
  type FlowEvent,
  type LiveData,
  type LiveMatch,
  type LiveSide,
  liveDataSchema,
  liveMatchDetailSchema,
} from '@/live/live.types'
import {
  type LiveTab,
  placements,
  type ScoreHits,
  scoreHits,
  tabMatches,
  timeGroups,
  winnerOf,
} from '@/live/live.utils'
import type { SwitchTournament } from '@/live/liveData'
import { LeadChart } from './LeadChart'

// 観戦アプリ(Claude Design「StageCam」)。今日の試合をカードの輪に並べ、横にめくって選び、真ん中の
// カードを押すと試合を大きく開く。点が入ると数字が飛び込む演出を出す。データは一定の間隔で取り直す。
// マイチーム(★)はこの端末に保存する。動きを減らす設定の端末では最初の演出を出さない(live.css も)。

/** カードの輪の、隣のカードとの角度(度)と半径(px) */
const cardStep = 40
const ringRadius = 290
/** 横にめくるとき、この距離(px)で 1 枚ぶん進む */
const swipePerCard = 120
/** 取り直す間隔(ミリ秒)。点は 1〜2 分に 1 回なので 15 秒で足りる */
const refreshMs = 15_000
/** 点が入った演出を出しておく時間(ミリ秒) */
const hitMs = 1100

const tabs: Array<[LiveTab, string]> = [
  ['mine', 'マイチーム'],
  ['live', 'ライブ'],
  ['done', '結果'],
]

const frameColors = {
  live: ['#ff8a1f', '#b85a0a'],
  sched: ['#19a7a0', '#0d6b67'],
  done: ['#9aa6b4', '#5f6975'],
} as const

/** カードの上の札。これからの試合は始まる時刻を出す */
function statusWord(match: LiveMatch): string {
  if (match.status === 'live') {
    return '試合中'
  }
  if (match.status === 'done') {
    return '終了'
  }
  return match.time === '' ? 'このあと' : `${match.time} 開始`
}

/** 色を暗くする(負けたチームの地の色) */
function shade(hex: string, amount: number): string {
  const value = Number.parseInt(hex.slice(1), 16)
  const channel = (shift: number) => Math.round(((value >> shift) & 255) * (1 - amount))
  return `rgb(${channel(16)},${channel(8)},${channel(0)})`
}

function favoritesKey(tournamentId: number): string {
  return `sv-live-favorites-${tournamentId}`
}

function readFavorites(tournamentId: number): number[] {
  try {
    const raw = window.localStorage.getItem(favoritesKey(tournamentId))
    const parsed: unknown = raw === null ? [] : JSON.parse(raw)
    return Array.isArray(parsed)
      ? parsed.filter((item): item is number => Number.isInteger(item))
      : []
  } catch {
    // 保存を使えない端末(プライベートブラウズなど)では、マイチームを覚えないだけにする
    return []
  }
}

function writeFavorites(tournamentId: number, favorites: number[]): void {
  try {
    window.localStorage.setItem(favoritesKey(tournamentId), JSON.stringify(favorites))
  } catch {
    // 保存を使えない端末では、開いている間だけ覚える
  }
}

function prefersReducedMotion(): boolean {
  return window.matchMedia('(prefers-reduced-motion: reduce)').matches
}

type SideView = {
  name: string
  sub: string
  won: boolean
  background: string
  opacity: number
  score: string
  star: boolean
}

function sideView(match: LiveMatch, side: 'left' | 'right', favorites: number[]): SideView {
  const own: LiveSide = match[side]
  const winner = winnerOf(match)
  const won = winner === side
  // 終わった試合は、どちらが勝ったかをひと目で分かるように負けた側を暗く沈める
  const lost = winner !== null && !won
  return {
    name: own.name,
    sub: own.pending ? '未定' : '',
    won,
    background: lost ? shade(own.color, 0.62) : own.color,
    opacity: lost ? 0.45 : 1,
    score: match.status === 'sched' || own.score === null ? '' : String(own.score),
    star: own.teamId !== null && favorites.includes(own.teamId),
  }
}

export function LiveApp({
  initial,
  tournaments,
  initialMatch,
}: {
  initial: LiveData
  tournaments: SwitchTournament[]
  /** 共有されたリンク(?match=)の試合。開いた状態で始める */
  initialMatch: number | null
}) {
  const tournamentId = initial.tournament.id
  const [data, setData] = useState(initial)
  const [tab, setTab] = useState<LiveTab>('live')
  const [pos, setPos] = useState(0)
  const [dragX, setDragX] = useState(0)
  const [tilt, setTilt] = useState(0)
  const [dragging, setDragging] = useState(false)
  const [camera, setCamera] = useState<'start' | 'rest'>('rest')
  const [cameraTransition, setCameraTransition] = useState('none')
  const [intro, setIntro] = useState(false)
  const [hits, setHits] = useState<ScoreHits>({})
  const [zoomId, setZoomId] = useState<number | null>(null)
  const [zoomVisible, setZoomVisible] = useState(false)
  const [dolly, setDolly] = useState(false)
  const [sheet, setSheet] = useState(false)
  const [flow, setFlow] = useState<FlowEvent[] | null>(null)
  const [zoomHit, setZoomHit] = useState<{ side: 'left' | 'right'; key: number } | null>(null)
  const [drawer, setDrawer] = useState(false)
  const [open, setOpen] = useState<Record<string, boolean>>({})
  const [pull, setPull] = useState(false)
  const [kebab, setKebab] = useState(false)
  const [picker, setPicker] = useState(false)
  const [favorites, setFavorites] = useState<number[]>([])
  const [draftFavorites, setDraftFavorites] = useState<number[]>([])
  const [toast, setToast] = useState('')
  const viewport = useRef<HTMLElement>(null)
  const pointer = useRef<{ x: number; y: number; axis: 'x' | 'y' | null; pos: number } | null>(null)
  const lastWheel = useRef(0)
  const timers = useRef<number[]>([])

  const later = useCallback((fn: () => void, ms: number) => {
    timers.current.push(window.setTimeout(fn, ms))
  }, [])

  const flash = useCallback(
    (text: string) => {
      setToast(text)
      later(() => setToast(''), 1700)
    },
    [later],
  )

  // マイチームを読み、最初にカメラが降りてくる演出を出す(動きを減らす設定なら出さない)
  useEffect(() => {
    const saved = readFavorites(tournamentId)
    setFavorites(saved)
    if (saved.length > 0) {
      setTab('mine')
    }
    if (!prefersReducedMotion()) {
      setCamera('start')
      setIntro(true)
      later(() => {
        setCamera('rest')
        setCameraTransition('transform 2.6s cubic-bezier(.3,.05,.15,1)')
      }, 80)
      later(() => setIntro(false), 2900)
    }
    const pending = timers.current
    return () => {
      for (const timer of pending) {
        window.clearTimeout(timer)
      }
    }
  }, [tournamentId, later])

  const loadFlow = useCallback(
    async (matchId: number) => {
      try {
        const response = await fetch(`/live/${tournamentId}/matches/${matchId}`, {
          cache: 'no-store',
        })
        if (!response.ok) {
          setFlow([])
          return
        }
        setFlow(liveMatchDetailSchema.parse(await response.json()).events)
      } catch {
        // 通信できないときは、次に取り直すまで今の表示のままにする
      }
    },
    [tournamentId],
  )

  // 一定の間隔で取り直し、点が増えた試合に演出を出す。画面が見えていない間は取り直さない
  useEffect(() => {
    const refresh = async () => {
      if (document.visibilityState !== 'visible') {
        return
      }
      try {
        const response = await fetch(`/live/${tournamentId}/data`, { cache: 'no-store' })
        if (!response.ok) {
          return
        }
        const next = liveDataSchema.parse(await response.json())
        setData((previous) => {
          const found = scoreHits(previous.matches, next.matches)
          if (Object.keys(found).length > 0) {
            setHits((current) => ({ ...current, ...found }))
            later(() => setHits({}), hitMs)
          }
          return { ...next, tournament: { ...next.tournament, date: previous.tournament.date } }
        })
      } catch {
        // 通信できないときは、次の間隔でもう一度取り直す
      }
    }
    const timer = window.setInterval(refresh, refreshMs)
    document.addEventListener('visibilitychange', refresh)
    return () => {
      window.clearInterval(timer)
      document.removeEventListener('visibilitychange', refresh)
    }
  }, [tournamentId, later])

  // 開いている試合に点が入ったら、得点の流れを取り直してから(帯に今の得点者を出すため)、大きな数字の
  // 演出と得点者の帯を出す
  useEffect(() => {
    if (zoomId === null || !zoomVisible) {
      return
    }
    const side = hits[zoomId]
    if (!side) {
      return
    }
    void loadFlow(zoomId).then(() => {
      setZoomHit({ side, key: Date.now() })
      later(() => setZoomHit(null), hitMs)
    })
  }, [hits, zoomId, zoomVisible, loadFlow, later])

  // 共有された試合のリンクから来たら、その試合を開いた状態で始める
  const sharedOpened = useRef(false)
  useEffect(() => {
    if (sharedOpened.current || initialMatch === null) {
      return
    }
    sharedOpened.current = true
    if (initial.matches.some((match) => match.id === initialMatch)) {
      setZoomId(initialMatch)
      setDolly(true)
      setZoomVisible(true)
      void loadFlow(initialMatch)
    }
  }, [initialMatch, initial.matches, loadFlow])

  const list = tabMatches(data.matches, tab, favorites)
  const count = list.length
  const current = Math.min(pos, Math.max(0, count - 1))
  const float = current - dragX / swipePerCard
  const front = Math.max(0, Math.min(count - 1, Math.round(float)))
  const clamp = (index: number) => Math.max(0, Math.min(count - 1, index))
  const go = (index: number) => setPos(clamp(index))

  const openZoom = (match: LiveMatch) => {
    setZoomId(match.id)
    setDolly(true)
    setCameraTransition('transform .62s cubic-bezier(.55,0,.35,1)')
    setSheet(false)
    setKebab(false)
    setFlow(null)
    void loadFlow(match.id)
    later(() => setZoomVisible(true), 430)
  }

  const closeZoom = () => {
    setZoomVisible(false)
    setSheet(false)
    setKebab(false)
    later(() => {
      setDolly(false)
      setCameraTransition('transform .62s cubic-bezier(.3,0,.2,1)')
    }, 180)
    later(() => setZoomId(null), 820)
  }

  const onPointerDown = (event: PointerEvent<HTMLElement>) => {
    if (zoomId !== null) {
      return
    }
    pointer.current = { x: event.clientX, y: event.clientY, axis: null, pos: current }
  }

  const onPointerMove = (event: PointerEvent<HTMLElement>) => {
    const start = pointer.current
    if (!start) {
      return
    }
    const dx = event.clientX - start.x
    const dy = event.clientY - start.y
    if (start.axis === null && Math.hypot(dx, dy) > 6) {
      start.axis = Math.abs(dx) >= Math.abs(dy) ? 'x' : 'y'
    }
    if (start.axis === 'x') {
      setDragX(dx)
      setDragging(true)
    } else if (start.axis === 'y') {
      setTilt(Math.max(-12, Math.min(10, -dy * 0.07)))
      setDragging(true)
    }
  }

  const onPointerUp = (event: PointerEvent<HTMLElement>) => {
    const start = pointer.current
    pointer.current = null
    if (!start) {
      return
    }
    if (start.axis === null && event.type === 'pointerup') {
      tapAt(event.clientX, event.clientY)
      return
    }
    if (start.axis === 'x') {
      setPos(clamp(Math.round(start.pos - (event.clientX - start.x) / swipePerCard)))
      setDragX(0)
    } else {
      setTilt(0)
      setCameraTransition('transform .6s cubic-bezier(.2,.8,.2,1)')
    }
    setDragging(false)
  }

  // 左右の端を押すと 1 枚めくり、真ん中を押すと試合を開く
  const tapAt = (x: number, y: number) => {
    const box = viewport.current?.getBoundingClientRect()
    if (!box) {
      return
    }
    const fx = (x - box.left) / box.width
    const fy = (y - box.top) / box.height
    if (fx < 0.27) {
      go(front - 1)
    } else if (fx > 0.73) {
      go(front + 1)
    } else if (fy > 0.15 && fy < 0.8) {
      const match = list[front]
      if (match) {
        openZoom(match)
      }
    }
  }

  const onWheel = (event: WheelEvent<HTMLElement>) => {
    if (zoomId !== null) {
      return
    }
    const now = Date.now()
    if (now - lastWheel.current < 420) {
      return
    }
    const delta = Math.abs(event.deltaX) > Math.abs(event.deltaY) ? event.deltaX : event.deltaY
    if (Math.abs(delta) < 15) {
      return
    }
    lastWheel.current = now
    go(front + (delta > 0 ? 1 : -1))
  }

  const onKeyDown = (event: KeyboardEvent<HTMLElement>) => {
    if (event.key === 'ArrowLeft') {
      go(front - 1)
    } else if (event.key === 'ArrowRight') {
      go(front + 1)
    } else if (event.key === 'Enter' || event.key === ' ') {
      const match = list[front]
      if (match) {
        event.preventDefault()
        openZoom(match)
      }
    }
  }

  const changeTab = (next: LiveTab) => {
    if (next !== tab) {
      setTab(next)
      setPos(0)
    }
  }

  const share = async (matchId: number | null) => {
    setKebab(false)
    const url = `${window.location.origin}/live/${tournamentId}${matchId === null ? '' : `?match=${matchId}`}`
    try {
      if (navigator.share) {
        await navigator.share({ title: data.tournament.name, url })
        return
      }
      await navigator.clipboard.writeText(url)
      flash('リンクをコピーしました')
    } catch {
      // 共有を取り消したとき、またはコピーできない端末では何もしない
    }
  }

  const toggleFavorite = (teamId: number) => {
    const next = favorites.includes(teamId)
      ? favorites.filter((item) => item !== teamId)
      : [...favorites, teamId]
    setFavorites(next)
    writeFavorites(tournamentId, next)
    flash(next.includes(teamId) ? 'マイチームにしました' : 'マイチームから外しました')
  }

  // Escape で、いちばん上に開いているもの(メニュー、引き出し、試合)を閉じる
  useEffect(() => {
    const onKey = (event: globalThis.KeyboardEvent) => {
      if (event.key !== 'Escape') {
        return
      }
      if (picker) {
        setPicker(false)
      } else if (pull || kebab) {
        setPull(false)
        setKebab(false)
      } else if (drawer) {
        setDrawer(false)
      } else if (zoomId !== null) {
        closeZoom()
      }
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  })

  const teams = uniqueTeams(data.matches)
  const zoomMatch =
    zoomId === null ? null : (data.matches.find((match) => match.id === zoomId) ?? null)
  const tabLabel = tabs.find(([key]) => key === tab)?.[1] ?? 'ライブ'

  let worldZ = -290
  let worldY = 0
  let worldX = -4 + tilt
  if (camera === 'start') {
    worldZ = -60
    worldY = -220
    worldX = 14
  }
  if (dolly) {
    worldZ = 350
  }
  const cardTransition = dragging ? 'none' : 'transform .6s cubic-bezier(.2,.8,.2,1), opacity .6s'

  return (
    <div className="live">
      <div className="live-screen">
        <div className="live-chrome" style={{ opacity: dolly ? 0 : 1 }}>
          <div className="live-bar">
            <button
              type="button"
              className="live-rank-button"
              onClick={() => {
                setDrawer(true)
                setPull(false)
                setKebab(false)
              }}
            >
              順位
            </button>
            <button
              type="button"
              className="live-title-button"
              aria-expanded={pull}
              onClick={() => {
                setPull(!pull)
                setKebab(false)
              }}
            >
              <span>{data.tournament.name}</span>
              <span className="live-caret" style={{ transform: `rotate(${pull ? 180 : 0}deg)` }}>
                ▼
              </span>
            </button>
            <KebabButton
              onClick={() => {
                setKebab(!kebab)
                setPull(false)
              }}
            />
          </div>
          <div className="live-tabs">
            {tabs.map(([key, label]) => (
              <button
                key={key}
                type="button"
                className="live-tab"
                aria-pressed={tab === key}
                onClick={() => changeTab(key)}
              >
                <span>{label}</span>
                <span className="live-tab-bar" />
              </button>
            ))}
            <span className="live-counter live-num" aria-live="polite">
              {count > 0 ? `${front + 1} / ${count}` : ''}
            </span>
          </div>
        </div>

        <section
          ref={viewport}
          className="live-viewport"
          aria-roledescription="カードの輪"
          aria-label={
            list[front]
              ? `${list[front].left.name} 対 ${list[front].right.name}。Enter で開く`
              : '試合'
          }
          // biome-ignore lint/a11y/noNoninteractiveTabindex: キーボードで輪をめくれるようにする
          tabIndex={0}
          onPointerDown={onPointerDown}
          onPointerMove={onPointerMove}
          onPointerUp={onPointerUp}
          onPointerLeave={onPointerUp}
          onWheel={onWheel}
          onKeyDown={onKeyDown}
        >
          <div className="live-rig">
            <div
              className="live-world"
              style={{
                transform: `translateZ(${worldZ}px) translateY(${worldY}px) rotateX(${worldX}deg)`,
                transition: dragging ? 'none' : cameraTransition,
              }}
            >
              <div className="live-field" style={{ opacity: dolly ? 0 : 1 }}>
                <div className="live-field-lines" />
              </div>
              {data.sponsor ? (
                <div className="live-sponsor" style={{ opacity: dolly ? 0 : 1 }}>
                  <div className="live-sponsor-frame">
                    <img src={data.sponsor.imageUrl} alt="スポンサー" />
                  </div>
                  <span className="live-sponsor-label">OFFICIAL SPONSOR</span>
                </div>
              ) : null}
              {list.map((match, index) => (
                <Card
                  key={match.id}
                  match={match}
                  angle={(index - float) * cardStep}
                  front={index === front}
                  hit={hits[match.id]}
                  favorites={favorites}
                  transition={cardTransition}
                />
              ))}
            </div>
          </div>
          {intro ? <IntroTitle name={data.tournament.name} /> : null}
          {count === 0 ? (
            <div className="live-empty">
              {tab === 'live'
                ? 'いま試合中の試合はありません'
                : tab === 'mine'
                  ? 'マイチームがまだありません。右上の「…」から選べます'
                  : '終わった試合はまだありません'}
            </div>
          ) : null}
        </section>

        <div className="live-strip-wrap" style={{ opacity: dolly ? 0 : 1 }}>
          <div className="live-strip">
            {timeGroups(list).map((group) => {
              const hasFront = group.indexes.includes(front)
              return (
                <div key={`${group.time}-${group.indexes[0]}`} className="live-strip-group">
                  <div className="live-strip-dots">
                    {group.indexes.map((index) => {
                      const match = list[index]
                      if (!match) {
                        return null
                      }
                      const color = frameColors[match.status][0]
                      const on = index === front
                      const size = on ? 16 : 10
                      return (
                        <button
                          key={match.id}
                          type="button"
                          className="live-strip-dot"
                          aria-label={`${match.time} ${match.left.name} 対 ${match.right.name}`}
                          onClick={() => go(index)}
                          style={{
                            width: size,
                            height: size,
                            borderColor: color,
                            background: on || match.status !== 'sched' ? color : 'transparent',
                          }}
                        />
                      )
                    })}
                  </div>
                  <span
                    className="live-strip-time live-num"
                    style={{ color: hasFront ? '#fff' : '#59636e' }}
                  >
                    {group.time}
                  </span>
                </div>
              )
            })}
          </div>
        </div>

        {zoomMatch ? (
          <ZoomView
            match={zoomMatch}
            visible={zoomVisible}
            favorites={favorites}
            flow={flow}
            hit={zoomHit}
            sheet={sheet}
            tabLabel={tabLabel}
            onClose={closeZoom}
            onSheet={() => setSheet(!sheet)}
            onKebab={() => setKebab(!kebab)}
            onFavorite={toggleFavorite}
          />
        ) : null}

        {pull || kebab ? (
          // biome-ignore lint/a11y/noStaticElementInteractions: メニューの外を押したら閉じる(Escape でも閉じる)
          // biome-ignore lint/a11y/useKeyWithClickEvents: 同上
          <div
            className="live-catcher"
            onClick={() => {
              setPull(false)
              setKebab(false)
            }}
          />
        ) : null}
        <div
          className="live-pop"
          style={{
            top: 92,
            left: '50%',
            width: 300,
            opacity: pull ? 1 : 0,
            transform: `translate(-50%, ${pull ? 0 : -8}px)`,
            pointerEvents: pull ? 'auto' : 'none',
          }}
          aria-hidden={!pull}
        >
          {tournaments.map((item) => (
            <a
              key={item.id}
              className="live-pop-item"
              href={`/live/${item.id}`}
              aria-current={item.id === tournamentId ? 'page' : undefined}
              tabIndex={pull ? 0 : -1}
            >
              <span className="live-pop-text">
                <span>{item.name}</span>
                <span className="live-pop-meta">{item.meta}</span>
              </span>
              {item.id === tournamentId ? <span style={{ color: '#b84800' }}>✓</span> : null}
            </a>
          ))}
        </div>
        <div
          className="live-pop"
          style={{
            top: 88,
            right: 12,
            width: 220,
            zIndex: 31,
            opacity: kebab ? 1 : 0,
            transform: `translateY(${kebab ? 0 : -6}px) scale(${kebab ? 1 : 0.92})`,
            transformOrigin: 'top right',
            pointerEvents: kebab ? 'auto' : 'none',
          }}
          aria-hidden={!kebab}
        >
          <button
            type="button"
            className="live-pop-item"
            tabIndex={kebab ? 0 : -1}
            onClick={() => {
              setKebab(false)
              setDraftFavorites(favorites)
              setPicker(true)
            }}
          >
            マイチームを選ぶ
          </button>
          <button
            type="button"
            className="live-pop-item"
            tabIndex={kebab ? 0 : -1}
            onClick={() => share(zoomMatch?.id ?? null)}
          >
            {zoomMatch ? 'この試合を共有' : 'この大会を共有'}
          </button>
          <a
            className="live-pop-item"
            href={
              zoomMatch
                ? `/pscore/scv_match/open/${zoomMatch.id}?lang=jp`
                : `/scview/${tournamentId}?lang=jp`
            }
            tabIndex={kebab ? 0 : -1}
          >
            {zoomMatch ? '試合の詳しいページ' : 'いつもの表示で見る'}
          </a>
        </div>

        <Drawer
          open={drawer}
          data={data}
          sections={open}
          onToggle={(key) => setOpen({ ...open, [key]: !open[key] })}
          onClose={() => setDrawer(false)}
          onOpenMatch={(match) => {
            setDrawer(false)
            openZoom(match)
          }}
        />

        {picker ? (
          <TeamPicker
            teams={teams}
            selected={draftFavorites}
            onToggle={(teamId) =>
              setDraftFavorites(
                draftFavorites.includes(teamId)
                  ? draftFavorites.filter((item) => item !== teamId)
                  : [...draftFavorites, teamId],
              )
            }
            onCancel={() => setPicker(false)}
            onSave={() => {
              setFavorites(draftFavorites)
              writeFavorites(tournamentId, draftFavorites)
              setPicker(false)
              setTab('mine')
              setPos(0)
              flash(
                draftFavorites.length > 0
                  ? `マイチームを ${draftFavorites.length} チームにしました`
                  : 'マイチームを外しました',
              )
            }}
          />
        ) : null}

        {toast !== '' ? (
          <div className="live-toast" role="status">
            {toast}
          </div>
        ) : null}
      </div>
    </div>
  )
}

function KebabButton({ onClick }: { onClick: () => void }) {
  return (
    <button
      type="button"
      className="live-icon-button live-kebab"
      aria-label="その他"
      onClick={onClick}
    >
      <span />
      <span />
      <span />
    </button>
  )
}

function WinMark({ big = false }: { big?: boolean }) {
  return <span className={big ? 'live-win big' : 'live-win'}>勝ち</span>
}

function uniqueTeams(matches: LiveMatch[]): Array<{ id: number; name: string; color: string }> {
  const seen = new Map<number, { id: number; name: string; color: string }>()
  for (const match of matches) {
    for (const side of [match.left, match.right]) {
      if (side.teamId !== null && !seen.has(side.teamId)) {
        seen.set(side.teamId, { id: side.teamId, name: side.name, color: side.color })
      }
    }
  }
  return [...seen.values()].sort((a, b) => a.name.localeCompare(b.name, 'ja'))
}

function IntroTitle({ name }: { name: string }) {
  const letters = [...name]
  return (
    <div className="live-intro" aria-hidden="true">
      <div className="live-intro-line">
        {letters.map((letter, index) => (
          <span
            // biome-ignore lint/suspicious/noArrayIndexKey: 大会名の文字の位置そのものが識別子
            key={index}
            className="live-intro-letter live-stage-font"
            style={{
              fontSize: letters.length > 12 ? 26 : 34,
              animationDelay: `${(0.35 + index * 0.06).toFixed(2)}s`,
            }}
          >
            {letter}
          </span>
        ))}
      </div>
    </div>
  )
}

function Card({
  match,
  angle,
  front,
  hit,
  favorites,
  transition,
}: {
  match: LiveMatch
  angle: number
  front: boolean
  hit: 'left' | 'right' | undefined
  favorites: number[]
  transition: string
}) {
  const away = Math.abs(angle)
  const [frame, edge] = frameColors[match.status]
  const left = sideView(match, 'left', favorites)
  const right = sideView(match, 'right', favorites)
  const slam = '.55s cubic-bezier(.2,.8,.2,1)'
  return (
    <div
      className="live-card"
      aria-hidden={!front}
      style={{
        transform: `rotateY(${angle.toFixed(2)}deg) translateZ(${ringRadius}px)`,
        transition,
        opacity: away > 118 ? 0 : Math.max(0.15, 1 - away / 150),
        visibility: away > 125 ? 'hidden' : 'visible',
      }}
    >
      <div
        className="live-card-edge"
        style={{ left: -8, background: edge, transform: 'rotateY(-90deg)' }}
      />
      <div
        className="live-card-edge"
        style={{ right: -8, background: edge, transform: 'rotateY(90deg)' }}
      />
      <div
        className="live-card-face"
        style={{
          borderColor: frame,
          boxShadow: `${
            match.status === 'live' && front
              ? '0 0 34px rgba(255,138,31,.5)'
              : '0 10px 30px rgba(0,0,0,.5)'
          }, inset 0 0 0 2px rgba(255,255,255,.3)`,
        }}
      >
        <div className="live-card-half" style={{ background: left.background }}>
          <span className="live-card-name live-card-font" style={{ opacity: left.opacity }}>
            {left.won ? <WinMark /> : null}
            {left.name}
            {left.star ? <span className="live-star"> ★</span> : null}
          </span>
          <span
            className="live-card-score live-num"
            style={{
              opacity: left.opacity,
              animation: hit === 'left' ? `live-slam ${slam}` : 'none',
            }}
          >
            {left.score}
          </span>
        </div>
        {/* 真ん中の帯は「何の試合か」だけ。時刻とコートは試合を開いたときに出す */}
        <div className="live-card-mid">{match.stage}</div>
        <div className="live-card-half" style={{ background: right.background }}>
          <span
            className="live-card-score live-num"
            style={{
              opacity: right.opacity,
              animation: hit === 'right' ? `live-slam ${slam}` : 'none',
            }}
          >
            {right.score}
          </span>
          <span className="live-card-name live-card-font" style={{ opacity: right.opacity }}>
            {right.won ? <WinMark /> : null}
            {right.name}
            {right.star ? <span className="live-star"> ★</span> : null}
          </span>
        </div>
        <div className="live-card-dim" style={{ opacity: Math.min(0.6, away / 90) }} />
      </div>
      <div className="live-card-status" style={{ background: frame }}>
        {match.status === 'live' ? <span className="live-dot" /> : null}
        {statusWord(match)}
      </div>
    </div>
  )
}

function ZoomView({
  match,
  visible,
  favorites,
  flow,
  hit,
  sheet,
  tabLabel,
  onClose,
  onSheet,
  onKebab,
  onFavorite,
}: {
  match: LiveMatch
  visible: boolean
  favorites: number[]
  flow: FlowEvent[] | null
  hit: { side: 'left' | 'right'; key: number } | null
  sheet: boolean
  tabLabel: string
  onClose: () => void
  onSheet: () => void
  onKebab: () => void
  onFavorite: (teamId: number) => void
}) {
  const left = sideView(match, 'left', favorites)
  const right = sideView(match, 'right', favorites)
  const lastGoal = flow?.at(-1)
  const sweepName = hit
    ? lastGoal && lastGoal.side === hit.side && lastGoal.scorer
      ? lastGoal.scorer.name
      : match[hit.side].name
    : ''
  const slam = 'live-slam .6s cubic-bezier(.2,.8,.2,1)'
  const statusText =
    match.status === 'live'
      ? '試合中'
      : match.status === 'done'
        ? '試合終了'
        : `${match.time} 開始予定`
  const favoriteButton = (side: LiveSide): ReactNode =>
    side.teamId === null ? null : (
      <button
        type="button"
        className="live-fav-button"
        aria-pressed={favorites.includes(side.teamId)}
        onClick={() => side.teamId !== null && onFavorite(side.teamId)}
      >
        {favorites.includes(side.teamId) ? '★ マイチーム' : '☆ マイチームにする'}
      </button>
    )
  return (
    <section
      className="live-zoom"
      aria-label={`${left.name} 対 ${right.name}`}
      style={{ opacity: visible ? 1 : 0, pointerEvents: visible ? 'auto' : 'none' }}
    >
      <div className="live-zoom-half top" style={{ background: left.background }}>
        <div className="live-zoom-team" style={{ opacity: left.opacity }}>
          {left.won ? <WinMark big /> : <span className="live-zoom-sub">{left.sub}</span>}
          <span className="live-zoom-name live-stage-font">
            {left.name}
            {left.star ? <span className="live-star"> ★</span> : null}
          </span>
          <span>{favoriteButton(match.left)}</span>
        </div>
        <span
          className="live-zoom-score live-stage-font"
          style={{
            alignSelf: 'flex-end',
            opacity: left.opacity,
            animation: hit?.side === 'left' ? slam : 'none',
          }}
        >
          {left.score}
        </span>
      </div>
      <div className="live-zoom-half bottom" style={{ background: right.background }}>
        <span
          className="live-zoom-score live-stage-font"
          style={{
            alignSelf: 'flex-start',
            opacity: right.opacity,
            animation: hit?.side === 'right' ? slam : 'none',
          }}
        >
          {right.score}
        </span>
        <div className="live-zoom-team" style={{ opacity: right.opacity }}>
          <span>{favoriteButton(match.right)}</span>
          <span className="live-zoom-name live-stage-font">
            {right.name}
            {right.star ? <span className="live-star"> ★</span> : null}
          </span>
          {right.won ? <WinMark big /> : <span className="live-zoom-sub">{right.sub}</span>}
        </div>
      </div>
      <div className="live-zoom-seam">
        <span>{match.stage}</span>
      </div>
      {hit ? (
        <div key={hit.key} className="live-sweep live-stage-font">
          {sweepName} 得点
        </div>
      ) : null}
      <div className="live-zoom-bar">
        <button type="button" className="live-pill-button" onClick={onClose}>
          ‹ {tabLabel}
        </button>
        <span className="live-crumb live-num">{match.time}</span>
        <span className="live-crumb">{match.court}</span>
        {match.stage !== match.divisionName ? (
          <span className="live-crumb">{match.divisionName}</span>
        ) : null}
        <span style={{ marginLeft: 'auto' }}>
          <KebabButton onClick={onKebab} />
        </span>
      </div>
      <div
        className="live-sheet"
        style={{ background: sheet ? 'rgba(10,12,16,.92)' : 'rgba(10,12,16,.74)' }}
      >
        <button type="button" className="live-sheet-toggle" aria-expanded={sheet} onClick={onSheet}>
          <span
            className="live-status-pill"
            style={{ background: match.status === 'live' ? '#ff5449' : '#fff' }}
          >
            {match.status === 'live' ? <span className="live-dot" /> : null}
            {statusText}
          </span>
          <span style={{ flex: 1, textAlign: 'right', fontSize: 13, fontWeight: 800 }}>
            得点の流れ
          </span>
          <span
            className="live-caret"
            style={{ fontSize: 10, transform: `rotate(${sheet ? 180 : 0}deg)` }}
          >
            ▲
          </span>
        </button>
        <div className="live-sheet-body" style={{ gridTemplateRows: sheet ? '1fr' : '0fr' }}>
          <div>
            <div className="live-sheet-scroll">
              {sheet ? <FlowList match={match} flow={flow} /> : null}
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}

function FlowList({ match, flow }: { match: LiveMatch; flow: FlowEvent[] | null }) {
  if (flow === null) {
    return <div className="live-flow-empty">読み込み中…</div>
  }
  if (flow.length === 0) {
    return (
      <div className="live-flow-empty">
        {match.status === 'sched' ? '始まると、ここに得点が流れます。' : '得点の記録はありません。'}
      </div>
    )
  }
  const muted = '#6b7480'
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
      <LeadChart
        events={flow}
        leftName={match.left.name}
        rightName={match.right.name}
        leftColor={match.left.color}
        rightColor={match.right.color}
      />
      <div>
        {[...flow].reverse().map((event, index) => {
          const team = event.side === 'left' ? match.left : match.right
          return (
            // biome-ignore lint/suspicious/noArrayIndexKey: 得点の順番(新しい順)が識別子
            <div key={index} className="live-flow-row">
              <span className="live-flow-time live-num">{event.elapsed}</span>
              <div className="live-flow-who">
                <div className="live-flow-scorer">
                  <span className="live-flow-team" style={{ background: team.color }}>
                    {team.name}
                  </span>
                  <span className="live-flow-name">
                    {event.scorer ? event.scorer.name : '—'}
                    {event.callahan ? '(キャラハン)' : ''}
                  </span>
                  {event.scorer ? (
                    <span
                      className="live-num"
                      style={{ color: '#9aa3ad', fontSize: 14, fontWeight: 700 }}
                    >
                      #{event.scorer.number}
                    </span>
                  ) : null}
                </div>
                {event.assist ? (
                  <span className="live-flow-assist">
                    アシスト {event.assist.name} #{event.assist.number}
                  </span>
                ) : null}
              </div>
              <span className="live-flow-score live-num">
                <span style={{ color: event.side === 'left' ? '#fff' : muted }}>
                  {event.leftScore}
                </span>
                <span style={{ color: muted }}>-</span>
                <span style={{ color: event.side === 'right' ? '#fff' : muted }}>
                  {event.rightScore}
                </span>
              </span>
            </div>
          )
        })}
      </div>
    </div>
  )
}

function placeStatus(match: LiveMatch): string {
  if (match.status === 'live') {
    return '試合中'
  }
  if (match.status === 'done') {
    return '終了'
  }
  return [match.time, match.court].filter((item) => item !== '').join(' ')
}

function Drawer({
  open,
  data,
  sections,
  onToggle,
  onClose,
  onOpenMatch,
}: {
  open: boolean
  data: LiveData
  sections: Record<string, boolean>
  onToggle: (key: string) => void
  onClose: () => void
  onOpenMatch: (match: LiveMatch) => void
}) {
  const finals = placements(data.matches)
  const tab = open ? 0 : -1
  return (
    <>
      {/* biome-ignore lint/a11y/noStaticElementInteractions: 引き出しの外を押したら閉じる(閉じるボタンもある) */}
      {/* biome-ignore lint/a11y/useKeyWithClickEvents: 同上 */}
      <div
        className="live-scrim"
        onClick={onClose}
        style={{ opacity: open ? 1 : 0, pointerEvents: open ? 'auto' : 'none' }}
      />
      <nav
        className="live-drawer"
        aria-label="順位"
        aria-hidden={!open}
        style={{ transform: `translateX(${open ? '0%' : '-105%'})` }}
      >
        <div className="live-drawer-head">
          <strong>{data.tournament.name}</strong>
          {data.tournament.date ? (
            <span style={{ fontSize: 12, color: '#9aa3ad' }}>{data.tournament.date}</span>
          ) : null}
        </div>
        {data.pools.length > 0 ? (
          <section className="live-drawer-section">
            <h2>予選の順位</h2>
            {data.pools.map((pool, index) => {
              const key = `pool-${pool.id}`
              // 組は閉じておき、順位決定戦まで一目で見渡せるようにする(順位決定戦が無ければ最初の組を開く)
              const expanded = sections[key] ?? (finals.length === 0 && index === 0)
              return (
                <div key={key} className="live-acc">
                  <button
                    type="button"
                    className="live-acc-toggle"
                    aria-expanded={expanded}
                    tabIndex={tab}
                    onClick={() => onToggle(key)}
                  >
                    <span>{pool.name}</span>
                    <span style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                      <span className="live-acc-badge">{pool.done ? '確定' : '途中経過'}</span>
                      <span
                        className="live-caret"
                        style={{ fontSize: 10, transform: `rotate(${expanded ? 180 : 0}deg)` }}
                      >
                        ▼
                      </span>
                    </span>
                  </button>
                  <div
                    className="live-acc-body"
                    style={{ gridTemplateRows: expanded ? '1fr' : '0fr' }}
                  >
                    <div>
                      <div className="live-acc-rows">
                        {pool.rows.map((row) => (
                          <div key={row.teamId} className="live-acc-row">
                            <span className="live-num">
                              {row.rank === null ? '' : `${row.rank}位`}
                            </span>
                            <span className="live-team-dot" style={{ background: row.color }} />
                            <span className="live-acc-name">{row.name}</span>
                            <span className="live-acc-right live-num">{row.points}点</span>
                          </div>
                        ))}
                      </div>
                    </div>
                  </div>
                </div>
              )
            })}
          </section>
        ) : null}
        {finals.length > 0 ? (
          <section className="live-drawer-section">
            <h2>順位決定戦</h2>
            {finals.map((item) => (
              <button
                key={item.match.id}
                type="button"
                className="live-place"
                tabIndex={tab}
                onClick={() => onOpenMatch(item.match)}
              >
                <span className="live-place-head">
                  <span className="live-place-title">{item.match.stage}</span>
                  <span className="live-place-when">{placeStatus(item.match)}</span>
                </span>
                {item.result ? (
                  item.result.map((row) => (
                    <span key={row.place} className="live-place-row">
                      <span className="live-place-rank live-num">{row.place}位</span>
                      <span className="live-team-dot" style={{ background: row.color }} />
                      <span>{row.name}</span>
                    </span>
                  ))
                ) : item.match.left.pending || item.match.right.pending ? null : (
                  // 勝ち上がりが決まるまではチーム名を出さない(何位決定戦かと時刻だけで足りる)
                  <span className="live-place-row live-place-pair">
                    {item.match.left.name} 対 {item.match.right.name}
                  </span>
                )}
              </button>
            ))}
          </section>
        ) : null}
        <a
          className="live-drawer-link"
          href={`/scview/${data.tournament.id}?lang=jp`}
          tabIndex={tab}
        >
          いつもの表示で見る ›
        </a>
        <a className="live-drawer-link" href="/sctnmt" tabIndex={tab}>
          大会一覧へ ›
        </a>
        <button
          type="button"
          className="live-drawer-link"
          style={{
            width: '100%',
            border: 0,
            borderTop: '1px solid #222832',
            background: 'transparent',
            cursor: 'pointer',
          }}
          tabIndex={tab}
          onClick={onClose}
        >
          閉じる
        </button>
      </nav>
    </>
  )
}

function TeamPicker({
  teams,
  selected,
  onToggle,
  onCancel,
  onSave,
}: {
  teams: Array<{ id: number; name: string; color: string }>
  selected: number[]
  onToggle: (teamId: number) => void
  onCancel: () => void
  onSave: () => void
}) {
  return (
    <div className="live-modal-back">
      <div className="live-modal" role="dialog" aria-modal="true" aria-label="マイチームを選ぶ">
        <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
          <strong style={{ fontSize: 17 }}>マイチームを選ぶ</strong>
          <span style={{ fontSize: 12.5, color: '#59636e' }}>
            選んだチームの試合が「マイチーム」に並びます。この端末に覚えます。
          </span>
        </div>
        <div className="live-modal-list">
          {teams.map((team) => (
            <button
              key={team.id}
              type="button"
              className="live-pick"
              aria-pressed={selected.includes(team.id)}
              onClick={() => onToggle(team.id)}
            >
              <span className="live-pick-star">{selected.includes(team.id) ? '★' : '☆'}</span>
              <span className="live-team-dot" style={{ background: team.color }} />
              <span>{team.name}</span>
            </button>
          ))}
        </div>
        <div className="live-modal-actions">
          <button type="button" className="live-modal-cancel" onClick={onCancel}>
            キャンセル
          </button>
          <button type="button" className="live-modal-save" onClick={onSave}>
            保存
          </button>
        </div>
      </div>
    </div>
  )
}
