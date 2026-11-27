import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import type { EntryAdminDetail, EntryExport } from '../src/api/entry.types.ts'
import { csvCell, entriesCsv, entriesCsvName } from '../src/entry/entryCsv.utils.ts'

// 申込の書き出し(CSV)

const entry = (
  id: number,
  members: EntryAdminDetail['memberList'],
  answers: EntryAdminDetail['answers'],
) =>
  ({
    id,
    division: { id: 2, name: { ja: 'オープン', en: '' } },
    fullName: `チーム${id}`,
    shortName: `T${id}`,
    repName: '山田 太郎',
    channel: 'desk',
    status: 'pending',
    members: members.length,
    registered: 0,
    submittedAt: '2026-11-13T05:05:00Z',
    teamId: null,
    scorerChosen: false,
    tournamentId: 7,
    repPhone: '090-0000-0000',
    repEmail: '',
    returnNote: '',
    inviteCode: 'x',
    inviteUrl: '',
    decidedAt: null,
    scorerMemberId: null,
    memberList: members,
    answers,
  }) satisfies EntryAdminDetail

describe('csvCell', () => {
  test('区切りと引用符と改行を含む値は引用符で囲む', () => {
    assert.equal(csvCell('a,b'), '"a,b"')
    assert.equal(csvCell('say "hi"'), '"say ""hi"""')
    assert.equal(csvCell('1行目\n2行目'), '"1行目\n2行目"')
    assert.equal(csvCell('熱海'), '熱海')
  })
  test('式として実行されうる値の先頭に引用符を付ける', () => {
    assert.equal(csvCell('=HYPERLINK("x")'), '"\'=HYPERLINK(""x"")"')
    assert.equal(csvCell('+81'), "'+81")
    assert.equal(csvCell('@SUM(A1)'), "'@SUM(A1)")
    assert.equal(csvCell('090-0000-0000'), '090-0000-0000')
  })
})

describe('entriesCsv', () => {
  const data: EntryExport = {
    questions: [
      {
        id: 1,
        scope: 'team',
        label: '宿泊',
        kind: 'yes_no',
        required: true,
        options: [],
        answered: 1,
      },
      {
        id: 2,
        scope: 'member',
        label: '学年',
        kind: 'short_text',
        required: false,
        options: [],
        answered: 0,
      },
    ],
    entries: [
      entry(
        10,
        [
          { id: 1, name: '山田 太郎', number: '7', registered: false, playerId: null, answers: [] },
          {
            id: 2,
            name: '佐藤 花子',
            number: '',
            registered: true,
            playerId: null,
            answers: [{ questionId: 2, value: '2 年' }],
          },
        ],
        [{ questionId: 1, value: 'yes' }],
      ),
      entry(11, [], []),
    ],
  }
  const csv = entriesCsv(data)
  const lines = csv.slice(1).split('\r\n')
  test('BOM で始まり、CRLF で区切る', () => {
    assert.equal(csv.charCodeAt(0), 0xfeff)
    assert.equal(lines.at(-1), '')
  })
  test('見出しはチーム向けの質問をチームの欄の後ろに、メンバー向けの質問をメンバーの欄の後ろに並べる', () => {
    assert.equal(
      lines[0],
      '申込番号,状態,部門,チーム名(正式),チーム名(短縮),代表,電話番号,メールアドレス,受けた日時,経路,宿泊,背番号,名前,LINE の登録,学年',
    )
  })
  test('メンバー 1 人を 1 行にし、チームの欄と答えを繰り返す。メンバーの無い申込も 1 行出す', () => {
    assert.equal(
      lines[1],
      '10,確認待ち,オープン,チーム10,T10,山田 太郎,090-0000-0000,,11月13日 14:05,運営が入力,はい,7,山田 太郎,まだ,',
    )
    assert.equal(lines[2]?.endsWith(',はい,,佐藤 花子,済み,2 年'), true, 'メンバー本人の答え')
    assert.equal(lines[3]?.startsWith('11,確認待ち'), true)
    assert.equal(lines[3]?.endsWith(',,,,,'), true)
    assert.equal(lines.length, 5)
  })
  test('ファイル名は日本時間の日付', () => {
    assert.equal(entriesCsvName(7, new Date('2026-11-13T16:00:00Z')), 'entries-7-20261114.csv')
  })
})
