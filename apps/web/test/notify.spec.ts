import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { operatorErrorText } from '../src/admin/errorText.utils.ts'
import {
  notificationDetailSchema,
  notificationListSchema,
  notificationPreviewSchema,
} from '../src/api/notify.types.ts'
import { isCronRequest } from '../src/notify/cron.utils.ts'
import { notifyPaths } from '../src/notify/notify.paths.ts'
import {
  audienceFromQuery,
  audienceText,
  bodyExcerpt,
  checkedAudience,
  checkedFields,
  countsText,
  isoToLocalTime,
  keepPlan,
  localTimeToIso,
  presetBody,
  presetTime,
  quotaState,
  sameAudience,
  sameTiming,
  scheduleText,
  sentNotice,
  timingFromQuery,
} from '../src/notify/notify.utils.ts'
import { uncertainFailure } from '../src/notify/notifyError.utils.ts'

// 管理画面の「公式 LINE で送る」の、宛先の条件の読み方と言葉

describe('audienceFromQuery', () => {
  test('条件と絞り込みを読む', () => {
    assert.deepEqual(audienceFromQuery({ kind: 'reps', status: 'pending', division: '12' }), {
      kind: 'reps',
      status: 'pending',
      divisionId: 12,
    })
  })
  test('知らない値は既定にし、条件に使えない絞り込みは捨てる', () => {
    assert.deepEqual(audienceFromQuery({}), { kind: 'reps', status: '', divisionId: null })
    assert.deepEqual(audienceFromQuery({ kind: 'all', status: 'cancelled', division: 'x' }), {
      kind: 'reps',
      status: '',
      divisionId: null,
    })
    assert.deepEqual(audienceFromQuery({ kind: 'incomplete', status: 'approved', division: '3' }), {
      kind: 'incomplete',
      status: '',
      divisionId: 3,
    })
    assert.deepEqual(audienceFromQuery({ kind: 'past_reps', status: 'pending', division: '3' }), {
      kind: 'past_reps',
      status: '',
      divisionId: null,
    })
    assert.deepEqual(audienceFromQuery({ kind: ['reps', 'past_reps'], division: '0' }), {
      kind: 'reps',
      status: '',
      divisionId: null,
    })
  })
})

describe('notifyPaths', () => {
  test('送る画面を、選んだ条件で開く', () => {
    assert.equal(
      notifyPaths.createFor(5, { kind: 'reps', status: 'returned', divisionId: 9 }),
      '/admin/tournaments/5/notifications/new?kind=reps&status=returned&division=9',
    )
    assert.equal(
      notifyPaths.createFor(5, { kind: 'participants', status: '', divisionId: null }),
      '/admin/tournaments/5/notifications/new?kind=participants',
    )
  })
})

describe('keepPlan', () => {
  const back = '/admin/tournaments/5/notifications/new'
  const plan = { kind: 'incomplete', division: '9', timing: 'later', at: '2027-02-08T19:00' }
  test('送る画面に戻すときは、知らせに条件と時刻を付け直す', () => {
    const target = keepPlan(`${back}?error=x`, back, plan)
    const url = new URL(target, 'https://example.test')
    assert.equal(url.pathname, back)
    assert.equal(url.searchParams.get('error'), 'x')
    assert.equal(url.searchParams.get('kind'), 'incomplete')
    assert.equal(url.searchParams.get('division'), '9')
    assert.equal(url.searchParams.get('at'), '2027-02-08T19:00')
  })
  test('ログイン画面などほかの行き先はそのまま', () => {
    assert.equal(keepPlan('/admin/login', back, plan), '/admin/login')
  })
})

describe('送る時刻(日本時間)', () => {
  test('画面の欄の値を RFC 3339 にし、戻す', () => {
    assert.equal(localTimeToIso('2027-02-08T19:00'), '2027-02-08T19:00:00+09:00')
    assert.equal(isoToLocalTime('2027-02-08T10:00:00Z'), '2027-02-08T19:00')
    assert.equal(localTimeToIso('2027-02-30T19:00'), null, '無い日')
    assert.equal(localTimeToIso('2027-02-08 19:00'), null)
    assert.equal(localTimeToIso(''), null)
  })
  test('予約の時刻を曜日つきで見せる', () => {
    assert.equal(scheduleText('2027-02-08T10:00:00Z'), '2月8日(月) 19:00')
    assert.equal(scheduleText('2027-02-08T16:30:00Z'), '2月9日(火) 01:30', '日本時間で日付が変わる')
  })
  test('クエリから読み、すぐ送るときは時刻を持たない', () => {
    assert.deepEqual(timingFromQuery({ timing: 'later', at: '2027-02-08T19:00' }), {
      kind: 'later',
      at: '2027-02-08T19:00',
    })
    assert.deepEqual(timingFromQuery({ at: '2027-02-08T19:00' }), { kind: 'now', at: '' })
    assert.equal(
      sameTiming(
        { kind: 'later', at: '2027-02-08T19:00' },
        { kind: 'later', at: '2027-02-08T19:30' },
      ),
      false,
    )
  })
  test('前日と当日の案内の時刻は大会の日から決める', () => {
    assert.equal(presetTime('2027-02-09', 'eve'), '2027-02-08T19:00')
    assert.equal(presetTime('2027-03-01', 'eve'), '2027-02-28T19:00', '月をまたぐ')
    assert.equal(presetTime('2027-02-09', 'day'), '2027-02-09T07:00')
    assert.match(presetBody('eve', 'https://score.example/tournaments/5'), /^明日は/)
    assert.match(
      presetBody('day', 'https://score.example/tournaments/5'),
      /score\.example\/tournaments\/5$/,
    )
  })
})

describe('定期の呼び出しの確かめ', () => {
  test('CRON_SECRET と一致する Authorization だけを受ける', () => {
    assert.equal(isCronRequest('Bearer s3cret', 's3cret'), true)
    assert.equal(isCronRequest('Bearer wrong!', 's3cret'), false)
    assert.equal(isCronRequest('Bearer s3cret', undefined), false, '設定が無ければ誰も受けない')
    assert.equal(isCronRequest(null, 's3cret'), false)
  })
})

describe('言葉', () => {
  test('宛先の条件の説明に、状態と部門を足す', () => {
    const names = (id: number) => (id === 9 ? 'オープン' : undefined)
    assert.equal(
      audienceText({ kind: 'reps', status: 'pending', divisionId: 9 }, names),
      '申し込んだ代表 ・ 確認待ち ・ オープン',
    )
    assert.equal(
      audienceText({ kind: 'participants', status: '', divisionId: 7 }, names),
      '参加が決まったチームの代表とメンバー ・ 消した部門',
    )
  })
  test('結果の人数は 0 人の結果を書かない', () => {
    assert.equal(
      countsText({ pending: 0, sent: 12, skipped: 1, failed: 0 }),
      '送った 12 人 ・ ブロック中 1 人',
    )
    assert.equal(countsText({ pending: 0, sent: 0, skipped: 0, failed: 0 }), '宛先なし')
  })
  test('送れなかった人がいれば送り直しを、送っている途中の人がいれば待つことを案内する', () => {
    assert.deepEqual(sentNotice({ pending: 0, sent: 12, skipped: 1, failed: 0 }), [
      'ok',
      '12 人に送りました',
    ])
    assert.deepEqual(sentNotice({ pending: 1, sent: 10, skipped: 0, failed: 2 }), [
      'error',
      '2 人に送れませんでした。この画面の「送り直す」で、その人たちにだけ送れます',
    ])
    assert.deepEqual(sentNotice({ pending: 3, sent: 10, skipped: 0, failed: 0 }), [
      'error',
      '3 人は送っている途中です。少しあとで、この画面を開き直してください',
    ])
    assert.equal(
      countsText({ pending: 2, sent: 0, skipped: 0, failed: 1 }),
      '送れなかった 1 人 ・ 送っている途中 2 人',
    )
  })
  test('api が送らなかったと答えた失敗のほかは、送ったかどうかが分からないとみなす', () => {
    const apiError = (status: number) =>
      Object.assign(new Error('x'), { name: 'ApiError', status, code: 'internal' })
    assert.equal(uncertainFailure(apiError(500)), true)
    assert.equal(uncertainFailure(apiError(504)), true)
    assert.equal(uncertainFailure(new TypeError('fetch failed')), true)
    assert.equal(
      uncertainFailure(new SyntaxError('Unexpected token <')),
      true,
      '本文が JSON でない',
    )
    assert.equal(uncertainFailure(apiError(409)), false, '送らなかったと分かっている')
    assert.equal(uncertainFailure(apiError(400)), false)
  })
  test('数えた宛先と選んでいる宛先を比べる', () => {
    const checked = { kind: 'reps', status: 'pending', divisionId: 9 } as const
    const form = { kind: 'reps', status: 'pending', division: '9', ...checkedFields(checked) }
    assert.equal(sameAudience(audienceFromQuery(form), checkedAudience(form)), true)
    const changed = { ...form, kind: 'participants' }
    assert.equal(sameAudience(audienceFromQuery(changed), checkedAudience(changed)), false)
    // 条件に使わない絞り込みの違いは、変えたことにしない(参加者の状態は捨てる)
    const participants = {
      kind: 'participants',
      status: '',
      division: '',
      ...checkedFields({ kind: 'participants', status: '', divisionId: null }),
    }
    assert.equal(
      sameAudience(
        audienceFromQuery({ ...participants, status: 'approved' }),
        checkedAudience(participants),
      ),
      true,
    )
  })
  test('一覧には運営が書いた文だけを短く出す', () => {
    assert.equal(bodyExcerpt('ルーキーズファイナル\n\n明日は\n8 時集合です'), '明日は 8 時集合です')
    assert.equal(bodyExcerpt('大会\n\nあいうえお', 3), 'あいう…')
  })
  test('今月の配信数で足りるか', () => {
    assert.deepEqual(quotaState({ limit: 200, used: 190 }, 10), {
      text: '今月は 200 通のうち 190 通を使い、残りは 10 通です',
      enough: true,
    })
    assert.equal(quotaState({ limit: 200, used: 190 }, 11).enough, false)
    assert.equal(quotaState({ limit: null, used: 5000 }, 900).enough, true)
    assert.equal(quotaState(null, 900).enough, true, '読めなければ api に任せる')
  })
  test('api の誤りを運営の言葉にする', () => {
    assert.equal(operatorErrorText('bad_request', 'body: is required'), '送る文を書いてください')
    assert.equal(
      operatorErrorText('bad_request', 'audience: has no one who can receive the message'),
      'この条件で送れる人がいません(ブロックしている人には送れません)',
    )
    assert.equal(
      operatorErrorText('bad_request', 'kind: must be reps, incomplete, participants or past_reps'),
      '宛先の条件を選び直してください',
    )
    assert.notEqual(
      operatorErrorText('bad_request', 'kind: must be men, women, mixed or none'),
      '宛先の条件を選び直してください',
      '部門の種類の誤りは宛先の言葉にしない',
    )
    assert.equal(
      operatorErrorText(
        'conflict',
        'conflict: this needs 12 messages but only 3 are left this month',
      ),
      '今月の残りの配信数が足りないので、送りませんでした。宛先を絞ってください',
    )
    assert.equal(
      operatorErrorText(
        'conflict',
        'conflict: the LINE settings are missing, so nothing can be sent',
      ),
      '公式 LINE の設定が無いので送れません',
    )
  })
})

describe('応答の形', () => {
  const summary = {
    id: 1,
    tournamentId: 5,
    audience: { kind: 'reps', status: '', divisionId: null },
    body: '大会\n\n明日は 8 時集合です',
    automatic: false,
    sentBy: '運営',
    createdAt: '2027-02-08T09:00:00Z',
    sendAt: null,
    state: 'sent',
    note: '',
    counts: { pending: 0, sent: 1, skipped: 0, failed: 0 },
    canResend: false,
  }
  test('送った記録と、その詳細', () => {
    assert.equal(
      notificationListSchema.parse({ notifications: [summary], canSend: true }).notifications[0]
        ?.audience.kind,
      'reps',
    )
    const detail = notificationDetailSchema.parse({
      ...summary,
      jobs: [
        { name: '山田', teams: '熱海', status: 'sent', error: '', sentAt: '2027-02-08T09:00:01Z' },
      ],
    })
    assert.equal(detail.jobs[0]?.status, 'sent')
    assert.throws(() => notificationDetailSchema.parse({ ...summary, jobs: [{ status: 'read' }] }))
  })
  test('送る前の確認(LINE の設定が無ければ配信数は null)', () => {
    const preview = notificationPreviewSchema.parse({
      heading: '大会',
      recipients: [{ name: '山田', teams: ['熱海'], reachable: false }],
      reachable: 0,
      unreachable: 1,
      quota: null,
      canSend: false,
    })
    assert.equal(preview.recipients[0]?.reachable, false)
  })
})
