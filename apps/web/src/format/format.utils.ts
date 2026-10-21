// 参照サイトの表示書式を再現する整形関数。値の計算(順位、集計)は api の責務で、ここは文字列化だけを行う。

/** 日付・時刻が api 契約(YYYY-MM-DD / HH:MM)の形でないときに投げる。 */
export class FormatError extends Error {
  constructor(message: string) {
    super(message)
    this.name = 'FormatError'
  }
}

// PHP mbstring の mb_strwidth が幅 2 と数える範囲(East Asian Width の W と F)。
// 半角カナ(U+FF61〜U+FF9F)は幅 1 のまま。
const wideRanges: ReadonlyArray<readonly [number, number]> = [
  [0x1100, 0x115f],
  [0x2e80, 0x303e],
  [0x3041, 0x33ff],
  [0x3400, 0x4dbf],
  [0x4e00, 0x9fff],
  [0xa000, 0xa4cf],
  [0xac00, 0xd7a3],
  [0xf900, 0xfaff],
  [0xfe30, 0xfe4f],
  [0xff00, 0xff60],
  [0xffe0, 0xffe6],
  [0x1f300, 0x1f64f],
  [0x1f900, 0x1f9ff],
  [0x20000, 0x2fffd],
  [0x30000, 0x3fffd],
]

/** 1 文字(コードポイント)の表示幅。全角 2、半角 1。 */
export function charWidth(char: string): number {
  const code = char.codePointAt(0) ?? 0
  return wideRanges.some(([from, to]) => code >= from && code <= to) ? 2 : 1
}

/** 文字列全体の表示幅(mb_strwidth 相当)。 */
export function displayWidth(text: string): number {
  let width = 0
  for (const char of text) {
    width += charWidth(char)
  }
  return width
}

/**
 * 表示幅 maxWidth に収まるところで切る(mb_strimwidth($s, 0, maxWidth) 相当、省略記号なし)。
 * clone-spec B 7: 日程のディビジョン名は幅 10、チーム名は幅 16。末尾の空白は残す。
 */
export function trimToWidth(text: string, maxWidth: number): string {
  let width = 0
  let result = ''
  for (const char of text) {
    width += charWidth(char)
    if (width > maxWidth) {
      return result
    }
    result += char
  }
  return result
}

/** clone-spec B 7 の切り詰め幅。 */
export const scheduleDivisionWidth = 10
export const scheduleTeamWidth = 16

/**
 * numerator / denominator を小数 2 桁で四捨五入し、末尾の 0 と小数点を削った文字列にする。
 * 負の値は扱わない(勝率、平均、得点の割合だけに使う)。denominator は正であること。
 */
function roundTo2(numerator: number, denominator: number): string {
  // 浮動小数の誤差を避けるため、100 倍した値を整数で四捨五入する
  const hundredths = Math.floor((numerator * 200 + denominator) / (2 * denominator))
  const integer = Math.floor(hundredths / 100)
  const fraction = hundredths % 100
  if (fraction === 0) {
    return String(integer)
  }
  return `${integer}.${String(fraction).padStart(2, '0').replace(/0$/, '')}`
}

/**
 * 割合を百分率で出す(勝率、試合の得点の内訳)。分母 0 は "--%"。
 * clone-spec C 1.3: 83.33%、66.67%、80%、100%。引き分けは勝ちに数えないので wins をそのまま渡す。
 */
export function formatPercent(numerator: number, denominator: number): string {
  if (denominator === 0) {
    return '--%'
  }
  return `${roundTo2(numerator * 100, denominator)}%`
}

/** 勝率 = 勝ち ÷ 試合数(clone-spec C 1.3)。 */
export function formatWinRate(wins: number, games: number): string {
  return formatPercent(wins, games)
}

/** 選手ページの平均 = 値 ÷ 試合数(clone-spec D 3.1)。試合数 0 は "--"。 */
export function formatAverage(total: number, games: number): string {
  if (games === 0) {
    return '--'
  }
  return roundTo2(total, games)
}

const datePattern = /^(\d{4})-(\d{2})-(\d{2})$/
const timePattern = /^(\d{2}):(\d{2})$/

function splitDate(date: string): [string, string, string] {
  const match = datePattern.exec(date)
  if (!match?.[1] || !match[2] || !match[3]) {
    throw new FormatError(`date is not YYYY-MM-DD: ${date}`)
  }
  return [match[1], match[2], match[3]]
}

/** 表の日付列 "MM/DD"(clone-spec B 5)。null は空。 */
export function formatMonthDay(date: string | null): string {
  if (date === null) {
    return ''
  }
  const [, month, day] = splitDate(date)
  return `${month}/${day}`
}

/** "YYYY/MM/DD"(スケジュールの日付の選択肢、試合ページの日時)。null は空。 */
export function formatYmdSlash(date: string | null): string {
  if (date === null) {
    return ''
  }
  return splitDate(date).join('/')
}

/** "YYYYMMDD"(スケジュールの ymd クエリと選択肢の value)。 */
export function formatYmdCompact(date: string): string {
  return splitDate(date).join('')
}

/** "HH:MM"。null は空。 */
export function formatTime(time: string | null): string {
  if (time === null) {
    return ''
  }
  if (!timePattern.test(time)) {
    throw new FormatError(`time is not HH:MM: ${time}`)
  }
  return time
}

/** 表の時間列。clone-spec B 5: "HH:MM" の後ろに半角スペースが 1 つ付く。 */
export function formatTimeCell(time: string | null): string {
  return time === null ? '' : `${formatTime(time)} `
}

/** 試合の経過時間 "MM:SS"(分は 60 以上もそのまま出す。clone-spec D 2.3)。 */
export function formatElapsed(elapsedSec: number): string {
  const minutes = Math.floor(elapsedSec / 60)
  const seconds = elapsedSec % 60
  return `${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`
}

// clone-spec D 2.2: 試合ページの表示時刻は日本時間。hourCycle h23 で 0 時を "24" にしない
const jstClock = new Intl.DateTimeFormat('en-GB', {
  timeZone: 'Asia/Tokyo',
  hour: '2-digit',
  minute: '2-digit',
  second: '2-digit',
  hourCycle: 'h23',
})

/** 試合ページのスコアボード右下の "HH:MM:SS JST"(表示した時点の日本時間。clone-spec D 2.2)。 */
export function formatClockJst(now: Date): string {
  return `${jstClock.format(now)} JST`
}
