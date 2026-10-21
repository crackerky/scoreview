import type {
  FormatInput,
  FormatParams,
  PreviewMatch,
  SavedFormat,
  ScheduleSetting,
  Sport,
  Tiebreak,
} from '@/api/format.types'

// 大会の形のウィザードの条件と入力の組み立て(Next.js と React に依存しない)。プレビュー、作成、
// 形の候補は、ここで作った同じ時間割の条件を api に渡す(運営者レビュー C3: プレビューは 3 コート、
// 作成は 4 コートの食い違い)。

/** 会場を使う 1 日。時刻は "HH:MM" */
export type DayCondition = { date: string; start: string; end: string }

/** ウィザードの「大会の条件」の欄 */
export type Conditions = {
  sport: Sport
  courts: number
  /** 1 試合の枠の長さ(分)。記録係の画面は試合時間として使う */
  slotMinutes: number
  days: DayCondition[]
  /** 昼休みの開始と長さ(分)。長さ 0 は昼休みなし */
  lunchStart: string
  lunchMinutes: number
  /** 同じチームの試合の間に最低空ける枠の数 */
  restSlots: number
  /** 段ごとに日を分ける(1 日目は予選、2 日目は順位決定戦) */
  stageDays: boolean
}

/** 選んだ形 */
export type Chosen = { template: string; params: FormatParams; title: string }

/** ウィザードの入力全体。プレビューと作成はこれから buildFormatInput で同じ入力を作る */
export type WizardForm = {
  cond: Conditions
  chosen: Chosen | null
  seeds: number[]
  tiebreak: Tiebreak
  forfeit: { winner: number; loser: number }
}

/** 競技ごとの初期値。枠の長さは運営会社の実績の最頻値、休みは連戦の実態(R3 5-2、5-3) */
export const sportDefaults: Record<Sport, { slotMinutes: number; restSlots: number }> = {
  four: { slotMinutes: 20, restSlots: 0 },
  seven: { slotMinutes: 60, restSlots: 1 },
  beach: { slotMinutes: 25, restSlots: 1 },
}

/** 会場の時間の初期値。7 人制の開始は 9:00(R3 5-1)、終了は 17:00 */
export const defaultDayStart = '09:00'
export const defaultDayEnd = '17:00'
export const defaultLunchStart = '12:00'
/** 日数の上限(api の maxScheduleDays と同じ) */
export const maxDays = 7

/** "YYYY-MM-DD" に n 日足す */
export function addDays(date: string, days: number): string {
  const parsed = new Date(`${date}T00:00:00Z`)
  if (Number.isNaN(parsed.getTime())) {
    return date
  }
  parsed.setUTCDate(parsed.getUTCDate() + days)
  return parsed.toISOString().slice(0, 10)
}

/** 日数を変える。増やした日は前の日の翌日で、前の日と同じ時間にする */
export function withDayCount(cond: Conditions, count: number): Conditions {
  const target = Math.min(Math.max(1, Math.trunc(count)), maxDays)
  const days = cond.days.slice(0, target)
  while (days.length < target) {
    const last = days[days.length - 1]
    days.push(
      last
        ? { date: addDays(last.date, 1), start: last.start, end: last.end }
        : { date: '', start: defaultDayStart, end: defaultDayEnd },
    )
  }
  return { ...cond, days, stageDays: target > 1 && cond.stageDays }
}

/** 競技を変える。枠の長さと休みを競技の初期値にする */
export function withSport(cond: Conditions, sport: Sport): Conditions {
  return { ...cond, sport, ...sportDefaults[sport] }
}

/** 条件の初期値。保存済みの時間割の設定があればそれを欄に出す(開き直したとき) */
export function initialConditions(saved: ScheduleSetting | null, defaultDate: string): Conditions {
  if (saved === null) {
    const sport: Sport = 'seven'
    return {
      sport,
      courts: 4,
      ...sportDefaults[sport],
      days: [{ date: defaultDate, start: defaultDayStart, end: defaultDayEnd }],
      lunchStart: defaultLunchStart,
      lunchMinutes: 0,
      stageDays: false,
    }
  }
  const days = saved.days.length > 0 ? saved.days : [saved]
  return {
    sport: saved.sport ?? (saved.teamRestSlots === 0 ? 'four' : 'seven'),
    courts: saved.courts.length,
    slotMinutes: saved.slotMinutes,
    restSlots: saved.teamRestSlots,
    days: days.map((day) => ({ date: day.date, start: day.start, end: day.end ?? defaultDayEnd })),
    lunchStart: saved.lunch?.start ?? defaultLunchStart,
    lunchMinutes: saved.lunch?.minutes ?? 0,
    stageDays: saved.stageDays,
  }
}

/** 引き継いだ時間割の設定を、新しい大会の開催日から始まる日付に置き換える(大会のコピー) */
export function shiftConditions(cond: Conditions, firstDate: string): Conditions {
  return {
    ...cond,
    days: cond.days.map((day, index) => ({ ...day, date: addDays(firstDate, index) })),
  }
}

/** 条件から api の時間割の設定を作る。date、start、end は 1 日目と同じ値 */
export function scheduleOf(cond: Conditions): ScheduleSetting {
  const [first] = cond.days
  return {
    date: first?.date ?? '',
    start: first?.start ?? defaultDayStart,
    end: first?.end ?? null,
    slotMinutes: cond.slotMinutes,
    courts: Array.from({ length: cond.courts }, (_, index) => String(index + 1)),
    teamRestSlots: cond.restSlots,
    stageGapSlots: 0,
    days: cond.days.map((day) => ({ date: day.date, start: day.start, end: day.end })),
    lunch: cond.lunchMinutes > 0 ? { start: cond.lunchStart, minutes: cond.lunchMinutes } : null,
    stageDays: cond.days.length > 1 && cond.stageDays,
    sport: cond.sport,
  }
}

/** プレビューと作成の入力。形が選ばれていなければ null */
export function buildFormatInput(form: WizardForm): FormatInput | null {
  if (form.chosen === null) {
    return null
  }
  return {
    template: form.chosen.template,
    params: form.chosen.params,
    seeds: form.seeds,
    tiebreak: form.tiebreak,
    forfeit: form.forfeit,
    schedule: scheduleOf(form.cond),
  }
}

/** 2 つの入力が同じか(プレビューした入力のまま作るかの判定) */
export function sameInput(a: FormatInput | null, b: FormatInput | null): boolean {
  return a !== null && b !== null && JSON.stringify(a) === JSON.stringify(b)
}

const tiebreakNames: Record<Tiebreak, string> = {
  current: '現行方式',
  current_no_h2h: '現行方式(直接対決なし)',
  jfda: 'JFDA 競技規則',
}

/**
 * 前回の形をコピーする。引き継ぐのは形(プール数、順位帯)、同点の決め方、不戦勝の規定スコアだけで、
 * 今回入れたコート数、日付、時刻などの条件は変えない。変えた欄の説明を notes に返す。
 */
export function copyFormat(
  form: WizardForm,
  saved: SavedFormat,
): { form: WizardForm; notes: string[] } {
  const notes = [`形を前回の「${saved.title}」にしました`]
  if (saved.tiebreak !== form.tiebreak) {
    notes.push(`同点の決め方を前回の「${tiebreakNames[saved.tiebreak]}」にしました`)
  }
  if (saved.forfeit.winner !== form.forfeit.winner || saved.forfeit.loser !== form.forfeit.loser) {
    notes.push(
      `不戦勝の規定スコアを前回の ${saved.forfeit.winner}-${saved.forfeit.loser} にしました`,
    )
  }
  notes.push('コート数、日付、時刻、昼休みは今回入れた条件のままです')
  return {
    form: {
      ...form,
      chosen: { template: saved.template, params: saved.params, title: saved.title },
      tiebreak: saved.tiebreak,
      forfeit: saved.forfeit,
    },
    notes,
  }
}

/**
 * 試合の片側の決まり方を運営の言葉にする。「A1」→「A組1位」、「[E2-勝ち]」→「E2 の勝者」、
 * 「[E4-負け]」→「E4 の負けチーム」。それ以外(「シード3」など)はそのまま。
 */
export function describeSide(label: string): string {
  const match = /^\[([A-Z]+\d+)-(勝ち|負け)\]$/.exec(label)
  if (match) {
    return match[2] === '勝ち' ? `${match[1]} の勝者` : `${match[1]} の負けチーム`
  }
  const rank = /^([A-Z]+)(\d+)$/.exec(label)
  if (rank) {
    return `${rank[1]}組${rank[2]}位`
  }
  return label
}

const bracketWords: Record<string, string> = {
  FINAL: '決勝',
  Semifinals: '準決勝',
  Quarterfinals: '準々決勝',
  Placement: '順位',
}

/**
 * 管理画面のプレビューのトーナメント表の見出しを日本語にする(閲覧ページの格子は参照サイトの再現なので
 * 英語のまま。ここは管理画面の表示だけ)。
 */
export function adminBracketText(text: string): string {
  const word = bracketWords[text]
  if (word) {
    return word
  }
  const round = /^(\d+)(?:st|nd|rd|th) Round$/.exec(text)
  if (round) {
    return `${round[1]} 回戦`
  }
  const range = /^(\d+)(?:st|nd|rd|th) - (\d+)(?:st|nd|rd|th)$/.exec(text)
  if (range) {
    return `${range[1]}〜${range[2]} 位`
  }
  const rank = /^([A-Z]+) rank (\d+)$/.exec(text)
  if (rank) {
    return `${rank[1]} の ${rank[2]} 位`
  }
  const next = /^(\d+)(?:st|nd|rd|th) placement T →$/.exec(text)
  if (next) {
    return `${next[1]} 位からのトーナメントへ →`
  }
  const place = /^(\d+)(?:st|nd|rd|th)(\(tied\))?$/.exec(text)
  if (place) {
    return `${place[1]} 位${place[2] ? '(同順位)' : ''}`
  }
  if (/^\[[A-Z]+\d+-(勝ち|負け)\]$/.test(text)) {
    return describeSide(text)
  }
  return text
}

/** プレビューの時間割の 1 日分(時刻 × コート)。キーは `${time}|${court}` */
export type PreviewDay = {
  date: string | null
  times: string[]
  cells: Map<string, PreviewMatch>
}

/** プレビューの試合を日付ごとの時間割にする(日付順。日付の無い試合は最後) */
export function buildPreviewDays(matches: readonly PreviewMatch[]): PreviewDay[] {
  const byDate = new Map<string | null, PreviewDay>()
  for (const match of matches) {
    if (match.time === null || match.field === null) {
      continue
    }
    const day: PreviewDay = byDate.get(match.date) ?? {
      date: match.date,
      times: [],
      cells: new Map(),
    }
    if (!day.times.includes(match.time)) {
      day.times.push(match.time)
    }
    day.cells.set(`${match.time}|${match.field}`, match)
    byDate.set(match.date, day)
  }
  const days = [...byDate.values()]
  for (const day of days) {
    day.times.sort()
  }
  return days.sort((a, b) => {
    if (a.date === null || b.date === null) {
      return a.date === null ? 1 : -1
    }
    return a.date.localeCompare(b.date)
  })
}

/** 「2 日目の 16:00 終了」「12:40 終了」 */
export function endText(end: string | null, endDay: number | null): string {
  if (end === null) {
    return '時刻未定'
  }
  return endDay !== null && endDay > 1 ? `${endDay} 日目の ${end} 終了` : `${end} 終了`
}
