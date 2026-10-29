import type { FlowEvent } from '@/live/live.types'
import { leadPoints } from '@/live/live.utils'

// 得点差の折れ線(Claude Design「StageCam」の得点の流れ)。上が左のチームのリード、下が右のチームの
// リード。中央の点線が同点。線を描き、点を順に出す。

const width = 322
const plotLeft = 16
const plotWidth = 290

export function LeadChart({
  events,
  leftName,
  rightName,
  leftColor,
  rightColor,
  height = 118,
}: {
  events: FlowEvent[]
  leftName: string
  rightName: string
  leftColor: string
  rightColor: string
  height?: number
}) {
  if (events.length === 0) {
    return null
  }
  const top = 22
  const bottom = height - 26
  const middle = (top + bottom) / 2
  const points = leadPoints(events, plotWidth, bottom - top).map((point) => ({
    ...point,
    x: plotLeft + point.x,
    y: top + point.y,
  }))
  const line = points.reduce(
    (path, point) => `${path} H${point.x.toFixed(1)} V${point.y.toFixed(1)}`,
    `M${plotLeft},${middle}`,
  )
  const area = `${line} V${middle} Z`
  const clipTop = `live-clip-top-${height}`
  const clipBottom = `live-clip-bottom-${height}`
  return (
    <svg
      width="100%"
      viewBox={`0 0 ${width} ${height}`}
      role="img"
      aria-label={`得点差の流れ。最後は ${events.at(-1)?.leftScore}-${events.at(-1)?.rightScore}`}
      style={{ display: 'block', maxWidth: width }}
    >
      <defs>
        <clipPath id={clipTop}>
          <rect x={0} y={0} width={width} height={middle} />
        </clipPath>
        <clipPath id={clipBottom}>
          <rect x={0} y={middle} width={width} height={height - middle} />
        </clipPath>
      </defs>
      <line
        x1={plotLeft}
        y1={middle}
        x2={plotLeft + plotWidth}
        y2={middle}
        stroke="rgba(255,255,255,.3)"
        strokeDasharray="3 4"
      />
      <path
        d={area}
        fill={leftColor}
        clipPath={`url(#${clipTop})`}
        style={{ animation: 'live-fade .8s .3s both' }}
      />
      <path
        d={area}
        fill={rightColor}
        clipPath={`url(#${clipBottom})`}
        style={{ animation: 'live-fade .8s .3s both' }}
      />
      <path
        d={line}
        fill="none"
        stroke="#fff"
        strokeWidth={2.5}
        strokeLinejoin="round"
        strokeDasharray={1400}
        style={{ animation: 'live-draw 1.3s cubic-bezier(.3,.6,.3,1) both' }}
      />
      {points.map((point, index) => (
        <circle
          // biome-ignore lint/suspicious/noArrayIndexKey: 得点の順番そのものが識別子(並びは変わらない)
          key={index}
          cx={point.x}
          cy={point.y}
          r={3.5}
          fill={point.side === 'left' ? leftColor : rightColor}
          stroke="#fff"
          strokeWidth={1.5}
          style={{
            transformBox: 'fill-box',
            transformOrigin: 'center',
            animation: `live-pop .35s ${(0.15 + (index / events.length) * 1.1).toFixed(2)}s both`,
          }}
        />
      ))}
      <text x={plotLeft} y={13} fill="#c9d0d8" fontSize={10.5} fontWeight={700}>
        ▲ {leftName} リード
      </text>
      <text x={plotLeft} y={bottom + 15} fill="#c9d0d8" fontSize={10.5} fontWeight={700}>
        ▼ {rightName} リード
      </text>
    </svg>
  )
}
