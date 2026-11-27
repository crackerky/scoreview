import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { type EntryDetail, type EntryQuestion, entryLinkSchema } from '../src/api/entry.types.ts'
import { entryPaths } from '../src/entry/entry.paths.ts'
import { parseMembers } from '../src/entry/entry.utils.ts'
import {
  findPastTeam,
  isLinkToken,
  joinFromForm,
  linkUnusableText,
  membersText,
  participantErrorText,
  pastTeamValues,
  profileValues,
  registeredIdsFromForm,
  resubmitValues,
  scorerChoiceFromForm,
  shareToLineUrl,
} from '../src/entry/entryLink.utils.ts'

// 本人用のリンクで開く画面(/entry/{合言葉})

const token = `${'A'.repeat(42)}_`

describe('isLinkToken', () => {
  test('base64url の 43 文字だけを合言葉として扱う', () => {
    assert.equal(isLinkToken(token), true)
    assert.equal(isLinkToken('A'.repeat(42)), false)
    assert.equal(isLinkToken(`${'A'.repeat(42)}/`), false)
    assert.equal(isLinkToken('../../admin'), false)
  })
})

test('entryPaths.link は結果をクエリで付ける', () => {
  assert.equal(entryPaths.link(token), `/entry/${token}`)
  assert.equal(entryPaths.link(token, { done: 'team' }), `/entry/${token}?done=team`)
})

test('shareToLineUrl は招待リンクを入れた文を LINE の送る画面に渡す', () => {
  const url = new URL(shareToLineUrl('熱海ディスクス', 'https://line.me/R/oaMessage/%40up/?x'))
  assert.equal(url.origin + url.pathname, 'https://line.me/R/share')
  const text = url.searchParams.get('text') ?? ''
  assert.match(text, /^熱海ディスクス のメンバー登録のお願いです。/)
  assert.match(text, /https:\/\/line\.me\/R\/oaMessage\/%40up\/\?x$/)
})

const question = (id: number): EntryQuestion => ({
  id,
  scope: 'member',
  label: `質問${id}`,
  kind: 'short_text',
  required: false,
  options: [],
  answered: 0,
})

describe('joinFromForm', () => {
  test('名簿の行を選んだら ID、名簿に無いなら null。空の答えは送らない', () => {
    const questions = [question(1), question(2)]
    assert.deepEqual(
      joinFromForm(
        { member: '15', name: '佐藤 花子', number: '12', a1: ' 3年 ', a2: '' },
        questions,
      ),
      { memberId: 15, name: '佐藤 花子', number: '12', answers: [{ questionId: 1, value: '3年' }] },
    )
    assert.equal(joinFromForm({ member: 'new', name: 'x', number: '' }, questions).memberId, null)
    assert.equal(joinFromForm({ name: 'x', number: '' }, questions).memberId, null)
  })
})

test('本人用のリンクの応答を parse できる', () => {
  const parsed = entryLinkSchema.parse({
    purpose: 'member',
    team: null,
    scorer: null,
    member: {
      tournamentName: 'ラバー 2027',
      team: {
        id: 1,
        division: { id: 2, name: { ja: 'オープン', en: '' } },
        fullName: '熱海ディスクス',
        shortName: '熱海',
        repName: '山田 太郎',
        channel: 'line',
        status: 'pending',
        members: 2,
        registered: 1,
        submittedAt: '2026-11-21T01:00:00Z',
        teamId: null,
      },
      questions: [],
      unclaimed: [{ id: 3, name: '鈴木 一郎', number: '', registered: false, playerId: null }],
      mine: { id: 4, name: '佐藤 花子', number: '12', registered: true, playerId: null },
      mineAnswers: [],
      profileName: '佐藤 花子',
      expiresAt: '2026-11-24T01:00:00Z',
    },
  })
  assert.equal(parsed.member?.mine?.name, '佐藤 花子')
})

describe('participantErrorText', () => {
  test('参加者の言葉にし、管理画面の言葉(申込一覧など)を使わない', () => {
    const cases: Array<[string, string, string]> = [
      [
        'conflict',
        'conflict: create entry: a team with the same name has already applied: the same value is already registered',
        'この部門には、同じチーム名の申込がもうあります。チーム名を確かめてください',
      ],
      [
        'conflict',
        'conflict: the tournament is not accepting entries',
        'この大会の申込は締め切りました',
      ],
      [
        'conflict',
        'conflict: someone else has already registered as this member',
        'その名前では、ほかの人がもう登録しました。名簿から選び直すか、「名簿に自分がいない」を選んでください',
      ],
      [
        'conflict',
        "conflict: only the representative can register as the representative's row",
        'その名前は代表の行です。名簿から自分の名前を選ぶか、「名簿に自分がいない」を選んでください',
      ],
      [
        'conflict',
        'conflict: the entry of this team has been cancelled',
        'このチームの申込は取り消されています',
      ],
      [
        'conflict',
        'conflict: only a returned entry can be resubmitted; it is pending',
        'この間に申込の状態が変わりました。画面を開き直して、今の状態を確かめてください',
      ],
      [
        'bad_request',
        'number: is used by another member',
        '背番号が、同じチームのほかの人と重なっています',
      ],
      [
        'bad_request',
        'members[1].number: is used by another member',
        'メンバーの 2 人目の背番号が、同じチームのほかの人と重なっています',
      ],
      [
        'bad_request',
        'fullName: must not contain line breaks or control characters',
        'チーム名(正式)は、改行を入れずに 1 行で書いてください',
      ],
      ['bad_request', 'repName: must be at most 100 characters', '代表の名前は 100 文字までです'],
      ['bad_request', 'name: is required', '名前を入れてください'],
      ['bad_request', 'answers[3]: is required', '必須の質問に答えてください'],
      ['bad_request', 'divisionId: is not a division of this tournament', '部門を選んでください'],
      ['not_found', 'TEAM_ENTRY_ID 5 not found', linkUnusableText],
      [
        'conflict',
        'conflict: resubmit entry 5: conflict: the members of the entry changed; reload it',
        'この間にメンバーが LINE で登録しました。画面を開き直してから、もう一度直してください',
      ],
    ]
    for (const [code, message, want] of cases) {
      assert.equal(participantErrorText(code, message), want, message)
    }
  })
})

test('membersText は parseMembers が読める形に戻す', () => {
  const members = [
    { name: '山田 太郎', number: '7' },
    { name: '佐藤 花子', number: '' },
  ]
  assert.equal(membersText(members), '7 山田 太郎\n佐藤 花子')
  assert.deepEqual(parseMembers(membersText(members)), members)
})

test('resubmitValues は今の中身を入れ、本人が登録した人はメンバー表に入れない', () => {
  const entry: EntryDetail = {
    id: 1,
    division: { id: 2, name: { ja: 'オープン', en: '' } },
    fullName: '熱海ディスクス',
    shortName: '熱海',
    repName: '山田 太郎',
    channel: 'line',
    status: 'returned',
    members: 2,
    registered: 1,
    submittedAt: '2026-11-21T01:00:00Z',
    teamId: null,
    tournamentId: 7,
    repPhone: '090',
    repEmail: '',
    returnNote: '短縮名を直してください',
    inviteCode: 'a'.repeat(26),
    inviteUrl: '',
    decidedAt: null,
    scorerMemberId: null,
    memberList: [
      { id: 3, name: '佐藤 花子', number: '12', registered: true, playerId: null },
      { id: 4, name: '鈴木 一郎', number: '3', registered: false, playerId: null },
    ],
    answers: [{ questionId: 9, value: 'yes' }],
  }
  assert.deepEqual(resubmitValues(entry), {
    divisionId: '2',
    fullName: '熱海ディスクス',
    shortName: '熱海',
    repName: '山田 太郎',
    repPhone: '090',
    repEmail: '',
    members: '3 鈴木 一郎',
    a9: 'yes',
  })
})

test('registeredIdsFromForm は registered の欄の数だけを読む', () => {
  assert.deepEqual(registeredIdsFromForm({ registered: ['3', '5', 'x'] }), [3, 5])
  assert.deepEqual(registeredIdsFromForm({ registered: '7' }), [7])
  assert.deepEqual(registeredIdsFromForm({}), [])
})

test('期限切れの案内は、代表とメンバーの両方の受け取り方を書く', () => {
  assert.match(linkUnusableText, /代表の方は公式 LINE にもう一度「エントリー」/)
  assert.match(linkUnusableText, /メンバーの方は、代表の方から届いた招待リンク/)
})

describe('参加者の台帳', () => {
  const atami = {
    fullName: '熱海ディスクス',
    shortName: '熱海',
    members: [
      { name: '山田 太郎', number: '7' },
      { name: '佐藤 花子', number: '' },
    ],
  }
  test('前回の入力を代表の欄に入れる(まだ無ければ空)', () => {
    assert.deepEqual(profileValues({ name: '山田 太郎', phone: '090', email: 'a@example.com' }), {
      repName: '山田 太郎',
      repPhone: '090',
      repEmail: 'a@example.com',
    })
    assert.deepEqual(profileValues(null), {})
  })
  test('前のチームをチーム名、短縮名、メンバー表の欄にする', () => {
    assert.deepEqual(pastTeamValues(atami), {
      fullName: '熱海ディスクス',
      shortName: '熱海',
      members: '7 山田 太郎\n佐藤 花子',
    })
  })
  test('書いたチーム名に合う前のチームを、前後の空白と大文字小文字を除いて探す', () => {
    const izu = { fullName: 'Izu Flyers', shortName: 'Izu', members: [] }
    assert.equal(findPastTeam([atami, izu], ' izu flyers '), izu)
    assert.equal(findPastTeam([atami, izu], '熱海'), undefined)
    assert.equal(findPastTeam([atami, izu], ' '), undefined)
  })
  test('本人用のリンクの応答の前回の入力と前のチームを parse できる', () => {
    const parsed = entryLinkSchema.parse({
      purpose: 'team',
      team: {
        tournament: { id: 1, name: 'ラバー 2027' },
        open: true,
        divisions: [],
        questions: [],
        mine: [],
        profile: { name: '山田 太郎', phone: '', email: '' },
        pastTeams: [atami],
        expiresAt: '2026-11-24T01:00:00Z',
      },
      member: null,
      scorer: null,
    })
    assert.equal(parsed.team?.pastTeams[0]?.members.length, 2)
  })
})

test('記録係を選ぶフォームは、空を「外す」にし、数でない値は外すとして読む', () => {
  assert.deepEqual(scorerChoiceFromForm({ entryId: '12', memberId: '34' }), {
    entryId: 12,
    memberId: 34,
  })
  assert.deepEqual(scorerChoiceFromForm({ entryId: '12', memberId: '' }), {
    entryId: 12,
    memberId: null,
  })
  assert.deepEqual(scorerChoiceFromForm({ entryId: '12', memberId: 'x' }), {
    entryId: 12,
    memberId: null,
  })
})

test('記録係のリンクの応答を parse できる', () => {
  const parsed = entryLinkSchema.parse({
    purpose: 'scorer',
    team: null,
    member: null,
    scorer: {
      teamName: '熱海ディスクス',
      tournamentName: 'ラバー 2027',
      memberName: '佐藤 花子',
      expiresAt: '2026-11-24T01:00:00Z',
    },
  })
  assert.equal(parsed.scorer?.memberName, '佐藤 花子')
})
