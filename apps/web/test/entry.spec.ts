import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { operatorErrorText } from '../src/admin/errorText.utils.ts'
import { type EntryQuestion, type EntrySummary, entryDetailSchema } from '../src/api/entry.types.ts'
import {
  answerFieldName,
  answersFromForm,
  answerText,
  countByStatus,
  dateTimeText,
  draftFromForm,
  formText,
  parseMembers,
  parseStatus,
  questionsFromForm,
  registeredText,
} from '../src/entry/entry.utils.ts'

// 申込の管理画面の、フォームの読み方と画面の形への直し方

describe('parseMembers', () => {
  test('背番号のある行とない行を読み、空行を飛ばす', () => {
    assert.deepEqual(parseMembers('7 山田 太郎\n\n佐藤 花子\r\n00　鈴木 一郎\n#12. 田中 次郎'), [
      { number: '7', name: '山田 太郎' },
      { number: '', name: '佐藤 花子' },
      { number: '00', name: '鈴木 一郎' },
      { number: '12', name: '田中 次郎' },
    ])
  })
  test('数字だけの名前や、4 桁以上の数は背番号にしない', () => {
    assert.deepEqual(parseMembers('2024年度 卒業生'), [{ number: '', name: '2024年度 卒業生' }])
    assert.deepEqual(parseMembers('1234 山田'), [{ number: '', name: '1234 山田' }])
  })
})

describe('questionsFromForm', () => {
  test('空の行と消す行を捨て、順の数で並べ、選択肢は選択式だけに残す', () => {
    const form = {
      'q0.id': '5',
      'q0.label': '宿泊',
      'q0.scope': 'team',
      'q0.kind': 'yes_no',
      'q0.order': '3',
      'q0.options': 'ゴミ',
      'q0.required': 'on',
      'q1.id': '6',
      'q1.label': '学年',
      'q1.scope': 'member',
      'q1.kind': 'short_text',
      'q1.order': '1',
      'q1.remove': 'on',
      'q2.label': ' 弁当 ',
      'q2.scope': 'team',
      'q2.kind': 'choice',
      'q2.order': '2',
      'q2.options': '要る\n\n 要らない \n',
      'q3.label': '',
      'q3.kind': 'choice',
    }
    assert.deepEqual(questionsFromForm(form, 4), [
      {
        id: null,
        scope: 'team',
        label: '弁当',
        kind: 'choice',
        required: false,
        options: ['要る', '要らない'],
      },
      { id: 5, scope: 'team', label: '宿泊', kind: 'yes_no', required: true, options: [] },
    ])
  })
})

const question = (id: number, scope: EntryQuestion['scope']): EntryQuestion => ({
  id,
  scope,
  label: `質問${id}`,
  kind: 'short_text',
  required: false,
  options: [],
  answered: 0,
})

describe('draftFromForm', () => {
  test('チーム向けの質問の答えだけを、空でないものだけ送る', () => {
    const draft = draftFromForm(
      {
        divisionId: '70',
        fullName: '熱海ディスクス',
        shortName: '熱海',
        repName: '山田 太郎',
        repPhone: '',
        repEmail: '',
        members: '7 山田 太郎',
        a1: ' はい ',
        a2: '',
        a3: '3年',
      },
      [question(1, 'team'), question(2, 'team'), question(3, 'member')],
    )
    assert.equal(draft.divisionId, 70)
    assert.deepEqual(draft.members, [{ number: '7', name: '山田 太郎' }])
    assert.deepEqual(draft.answers, [{ questionId: 1, value: 'はい' }])
  })
})

describe('フォームの欄の読み方', () => {
  test('formText は文字の欄だけを読み、無いか複数なら空', () => {
    assert.equal(formText({ a: 'x', b: ['1', '2'] }, 'a'), 'x')
    assert.equal(formText({ a: 'x', b: ['1', '2'] }, 'b'), '')
    assert.equal(formText({}, 'c'), '')
  })
  test('answersFromForm は答えの欄(answerFieldName)を読み、空の答えは送らない', () => {
    assert.equal(answerFieldName(12), 'a12')
    assert.deepEqual(
      answersFromForm({ a1: ' 3年 ', a2: ' ' }, [question(1, 'member'), question(2, 'member')]),
      [{ questionId: 1, value: '3年' }],
    )
  })
})

describe('表示', () => {
  const summary = (status: EntrySummary['status']): EntrySummary => ({
    id: 1,
    division: { id: 1, name: { ja: 'オープン', en: '' } },
    fullName: 'a',
    shortName: 'a',
    repName: 'r',
    channel: 'desk',
    status,
    members: 0,
    registered: 0,
    submittedAt: '2026-11-13T05:05:00Z',
    teamId: null,
    scorerChosen: false,
  })
  test('状態ごとに数える', () => {
    assert.deepEqual(countByStatus([summary('pending'), summary('pending'), summary('approved')]), {
      pending: 2,
      returned: 0,
      approved: 1,
      cancelled: 0,
    })
  })
  test('知らない状態のクエリは絞らない', () => {
    assert.equal(parseStatus('pending'), 'pending')
    assert.equal(parseStatus('deleted'), null)
    assert.equal(parseStatus(['pending']), null)
  })
  test('受けた時刻は日本時間で出す(UTC の 15 時は翌日)', () => {
    assert.equal(dateTimeText('2026-11-13T05:05:00Z'), '11月13日 14:05')
    assert.equal(dateTimeText('2026-11-13T15:30:00Z'), '11月14日 00:30')
  })
  test('LINE で登録した進み具合と、はい/いいえ', () => {
    assert.equal(registeredText(8, 6), '8 人中 6 人が LINE で登録済み')
    assert.equal(registeredText(0, 0), 'メンバー未記入')
    assert.equal(answerText({ kind: 'yes_no' }, 'yes'), 'はい')
    assert.equal(answerText({ kind: 'short_text' }, 'yes'), 'yes')
  })
})

describe('api の応答とエラー', () => {
  test('申込の応答を parse できる', () => {
    const parsed = entryDetailSchema.parse({
      id: 1,
      division: { id: 2, name: { ja: 'オープン', en: '' } },
      fullName: '熱海ディスクス',
      shortName: '熱海',
      repName: '山田 太郎',
      channel: 'desk',
      status: 'approved',
      members: 1,
      registered: 0,
      submittedAt: '2026-11-13T05:05:00Z',
      teamId: 9,
      scorerChosen: false,
      tournamentId: 7,
      repPhone: '',
      repEmail: '',
      returnNote: '',
      inviteCode: 'abcdefghijklmnopqrstuvwxyz',
      inviteUrl: '',
      decidedAt: '2026-11-13T06:00:00Z',
      scorerMemberId: null,
      memberList: [
        {
          id: 3,
          name: '山田 太郎',
          number: '7',
          registered: false,
          playerId: 11,
          withoutInvite: false,
        },
      ],
      answers: [{ questionId: 4, value: 'yes' }],
    })
    assert.equal(parsed.memberList[0]?.playerId, 11)
  })
  test('申込の英語の理由を運営の言葉にする', () => {
    assert.equal(
      operatorErrorText(
        'conflict',
        'conflict: create entry: a team with the same name has already applied: the same value is already registered',
      ),
      'この部門には同じチーム名の申込があります。申込一覧で確かめてください',
    )
    assert.equal(
      operatorErrorText('bad_request', 'members[2].number: is used by another member'),
      'メンバー表の 3 人目の背番号が、ほかの人と重なっています',
    )
    assert.equal(
      operatorErrorText('bad_request', 'answers[4]: is required'),
      '必須の質問に答えてください',
    )
    assert.equal(
      operatorErrorText('bad_request', 'questions[0].options: must have 1 to 30 options'),
      '質問 1 の選択肢を確かめてください(選択式は 1 つ以上、同じ選択肢は 1 回だけ)',
    )
  })
})
