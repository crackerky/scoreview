import {
  type Audience,
  type AudienceKind,
  type AudienceStatus,
  audienceKinds,
  audienceStatuses,
  type JobStatus,
  type NotificationCounts,
  type NotificationPreview,
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
    label: '承認したチームの代表とメンバー',
    hint: '前日の案内など、出場が決まった人みんなに送るとき。メンバーは LINE で登録した人だけです',
  },
  past_reps: {
    label: '前の大会に申し込んだ代表',
    hint: 'この大会にまだ申し込んでいない人だけ。受付を始めた知らせに使います',
  },
}

/** reps で選べる申込の状態 */
export const audienceStatusLabels: Readonly<Record<AudienceStatus, string>> = {
  '': 'すべて',
  pending: '申込中',
  returned: '差し戻し',
  approved: '承認',
}

export const jobStatusLabels: Readonly<Record<JobStatus, string>> = {
  // 送っている最中か、結果を書く前に止まった(LINE が受け付けたかは分からない)
  pending: '途中で止まった',
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
    parts.push(`途中で止まった ${counts.pending} 人`)
  }
  return parts.length === 0 ? '宛先なし' : parts.join(' ・ ')
}

/** 送った直後の知らせ。送れなかった人がいれば誤りの色で出し、送り直しは記録の画面へ案内する */
export function sentNotice(counts: NotificationCounts): ['ok' | 'error', string] {
  const unsent = counts.failed + counts.pending
  if (unsent > 0) {
    return [
      'error',
      `${unsent} 人に送れませんでした。この画面の「送り直す」で、その人たちにだけ送れます`,
    ]
  }
  return ['ok', `${counts.sent} 人に送りました`]
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
 * 送れなかったときの戻り先に、宛先の条件を付け直す。back は送る画面のパス(safeAdminPath はクエリを
 * 通さないので、条件は落ちている)。ログイン画面などほかの行き先はそのまま返す。
 */
export function keepAudience(target: string, back: string, audience: Audience): string {
  if (!target.startsWith(`${back}?`)) {
    return target
  }
  const params = new URLSearchParams(target.slice(back.length + 1))
  for (const [key, value] of Object.entries(audienceQuery(audience))) {
    if (value !== undefined) {
      params.set(key, value)
    }
  }
  return `${back}?${params.toString()}`
}
