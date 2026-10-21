import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import {
  filledEnglishText,
  genderOf,
  guessPlayerColumns,
  guessTeamColumns,
  playerRowsOf,
  splitRows,
  teamRowsOf,
} from '../src/admin/roster.utils.ts'

// 名簿の貼り付けのプレビュー(運営者レビュー H4)。

const entrySheet = [
  'No\tチーム名\t略称\t代表者',
  '1\t架空ブルーウィングス\t架空BW\t代表者1',
  '2\t架空シーガルズ\t架空SG\t代表者2',
].join('\n')

describe('チームの名簿', () => {
  test('エクセルのエントリー表: 見出し行を外し、チーム名と略称の列を見出しから選ぶ', () => {
    const rows = splitRows(entrySheet)
    const columns = guessTeamColumns(rows)
    assert.deepEqual(columns, { header: true, name: 1, short: 2, nameEn: null })
    assert.deepEqual(teamRowsOf(rows, columns), [
      { fullName: '架空ブルーウィングス', shortName: '架空BW', fullNameEn: '', warnings: [] },
      { fullName: '架空シーガルズ', shortName: '架空SG', fullNameEn: '', warnings: [] },
    ])
  })

  test('見出しが無くても、番号だけの列は名前に選ばない', () => {
    const rows = splitRows('1\t架空A\n2\t架空B')
    assert.deepEqual(guessTeamColumns(rows), { header: false, name: 1, short: null, nameEn: null })
  })

  test('メモ帳の「正式名、短縮名」は従来どおり', () => {
    const rows = splitRows('架空クラブ東京、トウキョウ\n\n架空クラブ大阪,オオサカ\n架空A')
    const columns = guessTeamColumns(rows)
    assert.deepEqual(columns, { header: false, name: 0, short: 1, nameEn: null })
    assert.deepEqual(
      teamRowsOf(rows, columns).map((row) => [row.fullName, row.shortName]),
      [
        ['架空クラブ東京', 'トウキョウ'],
        ['架空クラブ大阪', 'オオサカ'],
        ['架空A', '架空A'],
      ],
    )
  })

  test('英語名の列があれば見出しから選び、その値を英語名にする(無ければ空で api が日本語名を入れる)', () => {
    const rows = splitRows(
      ['チーム名\t略称\t英語名', '架空フライヤーズ東京\tフライヤーズ\tFictional Flyers'].join('\n'),
    )
    const columns = guessTeamColumns(rows)
    assert.deepEqual(columns, { header: true, name: 0, short: 1, nameEn: 2 })
    assert.equal(teamRowsOf(rows, columns)[0]?.fullNameEn, 'Fictional Flyers')
  })

  test('列を選び違えたときと重複には注意を出す', () => {
    const rows = splitRows(entrySheet)
    const wrong = teamRowsOf(rows, { header: false, name: 0, short: null, nameEn: null }, ['架空A'])
    assert.match(wrong[0]?.warnings.join('') ?? '', /見出し/)
    assert.match(wrong[1]?.warnings.join('') ?? '', /数字だけ/)
    const dup = teamRowsOf(
      splitRows('架空A\n架空B\n架空A'),
      { header: false, name: 0, short: null, nameEn: null },
      ['架空B'],
    )
    assert.deepEqual(
      dup.map((row) => row.warnings),
      [[], ['同じ名前のチームが登録済みです'], ['1 行目と同じ名前です']],
    )
  })
})

describe('選手の名簿', () => {
  const teams = [
    { id: 1, fullName: '架空ブルーウィングス', shortName: '架空BW' },
    { id: 2, fullName: '架空シーガルズ', shortName: '架空SG' },
  ]
  test('見出しからチーム名、背番号、氏名、性別の列を選び、略称でもチームを見つける', () => {
    const rows = splitRows(
      [
        'チーム名\t背番号\t氏名\t性別',
        '架空BW\t7\t架空 太郎\t男',
        '架空シーガルズ\t00\t架空 花子\t女',
        '架空ナイツ\t3\t架空 次郎\tM',
      ].join('\n'),
    )
    const columns = guessPlayerColumns(rows)
    assert.deepEqual(columns, {
      header: true,
      team: 0,
      number: 1,
      name: 2,
      gender: 3,
      nameEn: null,
    })
    const players = playerRowsOf(rows, columns, teams)
    assert.deepEqual(
      players.map((row) => [row.teamId, row.number, row.name, row.gender]),
      [
        [1, '7', '架空 太郎', 'M'],
        [2, '00', '架空 花子', 'F'],
        [null, '3', '架空 次郎', 'M'],
      ],
    )
    assert.match(players[2]?.warnings[0] ?? '', /見つかりません/)
  })

  test('選手の英語名の列', () => {
    const rows = splitRows(
      ['チーム名\t背番号\t氏名\t氏名(英語)', '架空BW\t7\t架空 太郎\tTaro KAKU'].join('\n'),
    )
    const columns = guessPlayerColumns(rows)
    assert.equal(columns.nameEn, 3)
    assert.equal(playerRowsOf(rows, columns, teams)[0]?.nameEn, 'Taro KAKU')
  })

  test('性別の書き方', () => {
    assert.equal(genderOf('女性'), 'F')
    assert.equal(genderOf('male'), 'M')
    assert.equal(genderOf('?'), null)
  })
})

describe('英語名の一括の補い', () => {
  test('入れた数を通知に出す', () => {
    assert.equal(
      filledEnglishText({ teams: 12, players: 3, groups: 7 }),
      '英語名が空だったところに日本語名を入れました(チーム 12、選手 3、組 7)',
    )
    assert.equal(
      filledEnglishText({ teams: 0, players: 0, groups: 0 }),
      '英語名が空のチーム、選手、組はありませんでした',
    )
  })
})
