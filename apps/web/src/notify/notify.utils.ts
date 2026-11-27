import {
  type Audience,
  type AudienceKind,
  type AudienceStatus,
  audienceKinds,
  audienceStatuses,
  type BatchState,
  type JobStatus,
  type NotificationCounts,
  type NotificationPreview,
  type SkipReason,
} from '@/api/notify.types'

// 管理画面の「公式 LINE で送る」の言葉と、宛先の条件の読み書き(docs/api.md「公式 LINE で送る」)。

type Query = Record<string, string | string[] | undefined>

/** 宛先の条件の名前と説明 */
export const audienceKindLabels: Readonly<Record<AudienceKind, { label: string; hint: string }>> = {
  reps: { label: '申し込んだ代表', hint: '取り消した申込を除きます。申込の状態で絞れます' },
  incomplete: {
    label: 'メンバーの登録が揃っていない代表',
    hint: '代表が書いたメンバーのうち、まだ LINE で登録していない人がいるチームの代表',
  },
  participants: {
    label: '参加が決まったチームの代表とメンバー',
    hint: '前日の案内など、出場が決まった人みんなに送るとき。メンバーは LINE で登録した人だけです',
  },
  no_scorer: {
    label: '記録係を選んでいないチームの代表',
    hint: '参加が決まったチームのうち、まだ記録係を選んでいないチームの代表。記録係を選んでもらう頼みと催促に使います',
  },
  past_reps: {
    label: '前の大会に申し込んだ代表',
    hint: 'この大会にまだ申し込んでいない人だけ。受付を始めた知らせに使います',
  },
}

/** reps で選べる申込の状態 */
export const audienceStatusLabels: Readonly<Record<AudienceStatus, string>> = {
  '': 'すべて',
  pending: '確認待ち',
  returned: '直し待ち',
  approved: '参加決定',
}

export const jobStatusLabels: Readonly<Record<JobStatus, string>> = {
  // 送っている最中か、結果を書く前に止まった(LINE が受け付けたかは分からない)
  pending: '送っている途中',
  sent: '送った',
  skipped: 'ブロック中で送らなかった',
  failed: '送れなかった',
}

export const jobStatusTones = {
  pending: 'warn',
  sent: 'ok',
  skipped: 'done',
  failed: 'live',
} as const satisfies Record<JobStatus, 'live' | 'done' | 'sched' | 'warn' | 'ok'>

function pick(query: Query, key: string): string {
  const value = query[key]
  return typeof value === 'string' ? value : ''
}

/**
 * クエリ(送る画面の ?kind=&status=&division=)かフォームの欄から宛先の条件を読む。知らない値は既定
 * (申し込んだ代表、すべての状態と部門)にし、条件に使えない絞り込み(代表のほかの状態、前の大会の
 * 代表の部門)は捨てる。api の 400 で画面を止めないため。
 */
export function audienceFromQuery(query: Query): Audience {
  const kind = audienceKinds.find((value) => value === pick(query, 'kind')) ?? 'reps'
  const status = audienceStatuses.find((value) => value === pick(query, 'status')) ?? ''
  const division = pick(query, 'division')
  const divisionId = /^[1-9][0-9]{0,15}$/.test(division) ? Number(division) : null
  return {
    kind,
    status: kind === 'reps' ? status : '',
    divisionId: kind === 'past_reps' ? null : divisionId,
  }
}

/** 宛先の条件を送る画面のクエリにする(既定の値は付けない) */
export function audienceQuery(audience: Audience): Record<string, string | undefined> {
  return {
    kind: audience.kind,
    status: audience.status || undefined,
    division: audience.divisionId === null ? undefined : String(audience.divisionId),
  }
}

/** 宛先の条件の 1 行の説明(送った記録の一覧と詳細)。divisionName は部門の名前の引き方 */
export function audienceText(
  audience: Audience,
  divisionName: (id: number) => string | undefined,
): string {
  const parts = [audienceKindLabels[audience.kind].label]
  if (audience.status !== '') {
    parts.push(audienceStatusLabels[audience.status])
  }
  if (audience.divisionId !== null) {
    parts.push(divisionName(audience.divisionId) ?? '消した部門')
  }
  return parts.join(' ・ ')
}

/** 送った結果の人数の 1 行(0 人の結果は書かない) */
export function countsText(counts: NotificationCounts): string {
  const parts: string[] = []
  if (counts.sent > 0) {
    parts.push(`送った ${counts.sent} 人`)
  }
  if (counts.skipped > 0) {
    parts.push(`ブロック中 ${counts.skipped} 人`)
  }
  if (counts.failed > 0) {
    parts.push(`送れなかった ${counts.failed} 人`)
  }
  if (counts.pending > 0) {
    parts.push(`送っている途中 ${counts.pending} 人`)
  }
  return parts.length === 0 ? '宛先なし' : parts.join(' ・ ')
}

/** 送った直後の知らせ。送れなかった人がいれば誤りの色で出し、送り直しは記録の画面へ案内する */
export function sentNotice(counts: NotificationCounts): ['ok' | 'error', string] {
  if (counts.failed > 0) {
    return [
      'error',
      `${counts.failed} 人に送れませんでした。この画面の「送り直す」で、その人たちにだけ送れます`,
    ]
  }
  if (counts.pending > 0) {
    return [
      'error',
      `${counts.pending} 人は送っている途中です。少しあとで、この画面を開き直してください`,
    ]
  }
  return ['ok', `${counts.sent} 人に送りました`]
}

/** 宛先の条件が同じか(送る画面で、数えた宛先と選んでいる宛先を比べる) */
export function sameAudience(left: Audience, right: Audience): boolean {
  return (
    left.kind === right.kind && left.status === right.status && left.divisionId === right.divisionId
  )
}

/** 数えた宛先の条件を、送るフォームの hidden 欄の値にする(画面で選んでいる条件と比べるため) */
export function checkedFields(audience: Audience): Record<string, string> {
  return {
    checkedKind: audience.kind,
    checkedStatus: audience.status,
    checkedDivision: audience.divisionId === null ? '' : String(audience.divisionId),
  }
}

/** 送るフォームの hidden 欄から、数えた宛先の条件を読む */
export function checkedAudience(form: Record<string, string | string[] | undefined>): Audience {
  return audienceFromQuery({
    kind: form.checkedKind,
    status: form.checkedStatus,
    division: form.checkedDivision,
  })
}

/** 送る画面で書いた文をブラウザに残す鍵(宛先を確かめるために画面を読み直しても文を消さない) */
export function draftStorageKey(tournamentId: number): string {
  return `notification-draft-${tournamentId}`
}

/** 届く文から、運営が書いた部分(1 行目の大会名と空行のあと)を一覧用に短くする */
export function bodyExcerpt(body: string, limit = 60): string {
  const separator = body.indexOf('\n\n')
  const text = (separator < 0 ? body : body.slice(separator + 2)).replace(/\s+/g, ' ').trim()
  const chars = [...text]
  return chars.length > limit ? `${chars.slice(0, limit).join('')}…` : text
}

/**
 * 今月の配信数の 1 行と、今回の送信で足りるか。quota が null(LINE から読めない)なら足りるとして送らせ、
 * 足りなければ api が 409 で止める。
 */
export function quotaState(
  quota: NotificationPreview['quota'],
  needed: number,
): { text: string; enough: boolean } {
  if (quota === null) {
    return { text: '今月の配信数を LINE から読めませんでした', enough: true }
  }
  if (quota.limit === null) {
    return { text: `今月は ${quota.used} 通を使いました(上限なし)`, enough: true }
  }
  const left = Math.max(quota.limit - quota.used, 0)
  return {
    text: `今月は ${quota.limit} 通のうち ${quota.used} 通を使い、残りは ${left} 通です`,
    enough: needed <= left,
  }
}

/**
 * 送れなかったときの戻り先に、宛先の条件と送る時刻(query)を付け直す。back は送る画面のパス
 * (safeAdminPath はクエリを通さないので、条件は落ちている)。ログイン画面などほかの行き先はそのまま返す。
 */
export function keepPlan(
  target: string,
  back: string,
  query: Record<string, string | undefined>,
): string {
  if (!target.startsWith(`${back}?`)) {
    return target
  }
  const params = new URLSearchParams(target.slice(back.length + 1))
  for (const [key, value] of Object.entries(query)) {
    if (value !== undefined) {
      params.set(key, value)
    }
  }
  return `${back}?${params.toString()}`
}

// 送る時刻(予約)。画面の欄は日本時間の datetime-local(「2027-02-08T19:00」)で、api には RFC 3339 で渡す。

/** 送る時刻の選び方。at は日本時間の欄の値(すぐ送るときは空) */
export type Timing = { kind: 'now'; at: '' } | { kind: 'later'; at: string }

/** 日本時間(UTC+9。夏時間なし) */
const jstOffsetMs = 9 * 60 * 60 * 1000

const localTimePattern = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})$/

/** 日本時間の欄の値を RFC 3339 にする。形が違うか、無い日時(2 月 30 日など)なら null */
export function localTimeToIso(value: string): string | null {
  const found = localTimePattern.exec(value)
  if (!found) {
    return null
  }
  const [year, month, day, hour, minute] = found.slice(1).map(Number) as [
    number,
    number,
    number,
    number,
    number,
  ] // 正規表現の 5 つの数字の組
  const utc = Date.UTC(year, month - 1, day, hour, minute) - jstOffsetMs
  if (isoToLocalTime(new Date(utc).toISOString()) !== value) {
    return null
  }
  return `${value}:00+09:00`
}

/** 時刻(ISO)を日本時間の欄の値にする */
export function isoToLocalTime(iso: string): string {
  return new Date(new Date(iso).getTime() + jstOffsetMs).toISOString().slice(0, 16)
}

const weekdays = ['日', '月', '火', '水', '木', '金', '土'] as const

/** 予約の時刻の見せ方(「2月8日(日) 19:00」。日本時間) */
export function scheduleText(iso: string): string {
  const jst = new Date(new Date(iso).getTime() + jstOffsetMs)
  const clock = `${String(jst.getUTCHours()).padStart(2, '0')}:${String(jst.getUTCMinutes()).padStart(2, '0')}`
  return `${jst.getUTCMonth() + 1}月${jst.getUTCDate()}日(${weekdays[jst.getUTCDay()]}) ${clock}`
}

/** クエリかフォームの欄(timing と at)から送る時刻の選び方を読む。知らない値はすぐ送る */
export function timingFromQuery(query: Query): Timing {
  return pick(query, 'timing') === 'later'
    ? { kind: 'later', at: pick(query, 'at') }
    : { kind: 'now', at: '' }
}

/** 送る時刻の選び方をクエリにする(すぐ送るときは付けない) */
export function timingQuery(timing: Timing): Record<string, string | undefined> {
  return timing.kind === 'later' ? { timing: 'later', at: timing.at } : {}
}

export function sameTiming(left: Timing, right: Timing): boolean {
  return left.kind === right.kind && left.at === right.at
}

/** 数えた送る時刻を、送るフォームの hidden 欄の値にする */
export function checkedTimingFields(timing: Timing): Record<string, string> {
  return { checkedTiming: timing.kind, checkedAt: timing.at }
}

/** 送るフォームの hidden 欄から、数えた送る時刻を読む */
export function checkedTiming(form: Query): Timing {
  return timingFromQuery({ timing: form.checkedTiming, at: form.checkedAt })
}

/**
 * 前日と当日の案内のひな形。送る時刻は運営会社と決めるまでの仮の既定(前日の夜と当日の朝。送る
 * 画面で直せる)。
 */
export const reminderPresets = {
  eve: {
    label: '前日の案内',
    dayOffset: -1,
    time: '19:00',
    greeting: '明日はよろしくお願いします。',
  },
  day: {
    label: '当日の案内',
    dayOffset: 0,
    time: '07:00',
    greeting: '本日はよろしくお願いします。',
  },
} as const

export type ReminderPreset = keyof typeof reminderPresets

export function parsePreset(value: string | string[] | undefined): ReminderPreset | null {
  return value === 'eve' || value === 'day' ? value : null
}

/** 案内の送る時刻(日本時間の欄の値)。大会の日(YYYY-MM-DD)から決める */
export function presetTime(startDate: string, preset: ReminderPreset): string {
  const { dayOffset, time } = reminderPresets[preset]
  const day = new Date(`${startDate}T00:00:00Z`)
  day.setUTCDate(day.getUTCDate() + dayOffset)
  return `${day.toISOString().slice(0, 10)}T${time}`
}

/** 案内の文のひな形(集合時刻と会場は運営が書き足す)。scoreUrl はスコアサイトの大会の URL */
export function presetBody(preset: ReminderPreset, scoreUrl: string): string {
  return [
    reminderPresets[preset].greeting,
    '',
    '集合時刻: ',
    '会場: ',
    '',
    '試合の予定と結果は、こちらで見られます。',
    scoreUrl,
  ].join('\n')
}

/**
 * 記録係を選んでもらう頼みの文のひな形(宛先は「記録係を選んでいないチームの代表」)。代表は「エントリー」
 * で開く申込の画面から、LINE で登録したメンバーを記録係に選ぶ。
 */
export const scorerRequestBody = [
  '参加が決まったチームの代表の方へ',
  '',
  '試合のスコアを入れる記録係を、チームから 1 人選んでください。記録係は、自分たちの試合のスコアを入れます。',
  '',
  'メニューの「エントリー」を押すか「エントリー」と送ると、申込の画面が開きます。記録係は、LINE でメンバー登録をした人から選べます。',
].join('\n')

/** 送る時刻に送れなかった理由の言葉 */
export const skipReasonLabels: Readonly<Record<Exclude<SkipReason, ''>, string>> = {
  no_recipients: '送る時刻に、条件に当てはまる人がいませんでした',
  quota:
    '送る時刻に、今月の残りの配信数が足りませんでした(一部の人にだけ届かないよう、送っていません)',
  line_not_configured: '送る時刻に、公式 LINE の設定がありませんでした',
}

/** 送信の段階の短い言葉(一覧の印) */
export const batchStateLabels: Readonly<Record<BatchState, string>> = {
  scheduled: '予約',
  sent: '送った',
  cancelled: '取りやめ',
  skipped: '送れなかった',
}
