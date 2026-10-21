// 管理画面のフォーム送信の「処理中」表示と二重送信の防止(src/components/admin/SubmitFeedback.tsx)の判断。
// DOM に触れない純粋な関数にしてテストする。

/** 送信中のボタンに付ける属性。admin.css がこの属性で「処理中」を出す */
export const BUSY_ATTRIBUTE = 'aria-busy'

/** この属性を付けたフォームは対象外(送信を画面側で取り消すことがあるフォーム。ログアウトの確認など) */
export const SKIP_ATTRIBUTE = 'data-no-busy'

/**
 * 画面が変わらないまま処理中の表示を解く上限(ミリ秒)。Vercel の関数と Neon の DB が止まっている
 * ところからの最初の操作は数秒かかる。それより十分長く、応答が来なかったときに押し直せる長さにする
 */
export const BUSY_TIMEOUT_MS = 20_000

/**
 * submit イベントの扱い。
 * - block: 同じボタンの送信が終わっていない(押し直しで大会やチームが二重に作られるのを防ぐ)
 * - mark: サーバーで処理する送信(Server Action)。ボタンを処理中にする
 * - ignore: GET のフォーム(ブラウザが普通に読み込み中を出す)と、対象外のフォーム
 */
export type SubmitDecision = 'block' | 'mark' | 'ignore'

/** Resource Timing の 1 件のうち、判断に使う項目 */
export type ResourceEntry = { name: string; initiatorType: string; startTime: number }

function pathAndQuery(url: string, origin: string): string | null {
  if (!URL.canParse(url, origin)) {
    return null
  }
  const parsed = new URL(url, origin)
  return parsed.origin === origin ? parsed.pathname + parsed.search : null
}

/**
 * entry が送信した Server Action の応答か。Server Action は送信した画面の URL への fetch で、
 * 応答の読み終わりで Resource Timing に載る。画面の取り直し(router.refresh)と先読みの fetch は
 * URL に _rsc が付くので除く
 */
export function isActionResponse(
  entry: ResourceEntry,
  submission: { pageUrl: string; submittedAt: number },
  origin: string,
): boolean {
  if (entry.initiatorType !== 'fetch' || entry.startTime < submission.submittedAt) {
    return false
  }
  const target = pathAndQuery(entry.name, origin)
  if (target === null || new URL(entry.name, origin).searchParams.has('_rsc')) {
    return false
  }
  return target === pathAndQuery(submission.pageUrl, origin)
}

export function submitDecision(input: {
  submitterBusy: boolean
  /**
   * form の method 属性(無ければ null)。form.method は使わない: React が画面の切り替え後に作る
   * Server Action のフォームは method 属性を持たず、form.method が既定の get になるため
   */
  methodAttribute: string | null
  skip: boolean
}): SubmitDecision {
  if (input.skip || input.methodAttribute?.toLowerCase() === 'get') {
    return 'ignore'
  }
  return input.submitterBusy ? 'block' : 'mark'
}
