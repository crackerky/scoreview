import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import {
  assignmentFormSchema,
  bannersFormSchema,
  bracketFormSchema,
  elapsedSchema,
  eventFormSchema,
  finalRanksFromForm,
  groupFormSchema,
  groupTeamsFromForm,
  matchFormSchema,
  photosFormSchema,
  playerFormSchema,
  playerStatsFromForm,
  sideChoiceSchema,
  statusFormSchema,
  teamFormSchema,
  tournamentFormSchema,
  userCreateFormSchema,
  userFormSchema,
  userTournamentsFormSchema,
} from '../src/api/admin.types.ts'

// 管理画面のフォーム入力(FormData を平らにした文字列のオブジェクト)から api のボディへの変換と検証

describe('大会のフォーム', () => {
  test('日英の名前、チェックボックス、空欄の null をボディの形にする', () => {
    const body = tournamentFormSchema.parse({
      nameJa: ' 2026 架空カップ ',
      nameEn: '',
      shortName: '2026 Cup',
      status: 'entry_closed',
      listed: 'on',
      startDate: '2026-11-01',
      logoUrl: '',
      topHtmlJa: '<p>本文</p>',
      teamListLabel: 'teams',
      groupLabel: 'results',
    })
    assert.deepEqual(body, {
      name: { ja: '2026 架空カップ', en: '' },
      shortName: '2026 Cup',
      status: 'entry_closed',
      listed: true,
      startDate: '2026-11-01',
      logoUrl: null,
      topHtml: { ja: '<p>本文</p>', en: '' },
      labels: { teamList: 'teams', groups: 'results' },
      // チェックなしのチェックボックスはキーごと送られない
      showSchedule: false,
      // 不戦勝の規定スコアは未設定(空欄)なら null
      forfeitScore: null,
    })
  })

  test('不戦勝の規定スコアは勝ちと負けを両方入れ、勝ちを大きくする', () => {
    const base = { nameJa: 'x', status: 'finished', teamListLabel: 'teams', groupLabel: 'groups' }
    const parsed = tournamentFormSchema.parse({ ...base, forfeitWinner: '15', forfeitLoser: '0' })
    assert.deepEqual(parsed.forfeitScore, { winner: 15, loser: 0 })
    assert.equal(tournamentFormSchema.safeParse({ ...base, forfeitWinner: '15' }).success, false)
    assert.equal(
      tournamentFormSchema.safeParse({ ...base, forfeitWinner: '0', forfeitLoser: '0' }).success,
      false,
    )
  })

  test('名前が空、日付の形が違う、javascript: の URL は失敗する', () => {
    const base = { nameJa: 'x', status: 'finished', teamListLabel: 'teams', groupLabel: 'groups' }
    assert.equal(tournamentFormSchema.safeParse({ ...base, nameJa: '  ' }).success, false)
    assert.equal(
      tournamentFormSchema.safeParse({ ...base, startDate: '2026/11/01' }).success,
      false,
    )
    assert.equal(
      tournamentFormSchema.safeParse({ ...base, logoUrl: 'javascript:alert(1)' }).success,
      false,
    )
    assert.equal(
      tournamentFormSchema.safeParse({ ...base, logoUrl: '//evil.example/a.png' }).success,
      false,
    )
    assert.equal(
      tournamentFormSchema.safeParse({ ...base, logoUrl: '/brand/logo.svg' }).success,
      true,
    )
  })
})

describe('一覧を 1 行ずつ書く欄', () => {
  test('バナーは「画像 リンク(省略可)」の行', () => {
    const body = bannersFormSchema.parse({
      banners: '/b1.png https://example.com/\n\n  /b2.png  \r\n',
    })
    assert.deepEqual(body, {
      banners: [
        { imageUrl: '/b1.png', linkUrl: 'https://example.com/' },
        { imageUrl: '/b2.png', linkUrl: null },
      ],
    })
    assert.equal(
      bannersFormSchema.safeParse({ banners: 'data:image/png;base64,AA' }).success,
      false,
    )
  })

  test('写真は「サムネイル 拡大」の 2 つが必須', () => {
    assert.deepEqual(photosFormSchema.parse({ photos: '/t.jpg /f.jpg' }), {
      photos: [{ thumbUrl: '/t.jpg', fullUrl: '/f.jpg' }],
    })
    assert.equal(photosFormSchema.safeParse({ photos: '/t.jpg' }).success, false)
    assert.deepEqual(photosFormSchema.parse({}), { photos: [] })
  })
})

describe('チームと選手', () => {
  test('チームの正式名は前後の空白も含めて加工しない', () => {
    const body = teamFormSchema.parse({
      fullNameJa: ' メッターによるお料理向上委員会 ',
      shortNameJa: 'お料理向上委員会',
      barStyle: 'mixed',
      sortOrder: '',
      finalRank: '3',
    })
    assert.equal(body.fullName.ja, ' メッターによるお料理向上委員会 ')
    assert.equal(body.sortOrder, 0)
    assert.equal(body.finalRank, 3)
    assert.equal(
      teamFormSchema.safeParse({ ...body, fullNameJa: 'x', shortNameJa: 'x', barStyle: 'x' })
        .success,
      false,
    )
  })

  test('選手の役割が空なら選手、あればスタッフ', () => {
    const player = playerFormSchema.parse({ number: '00', nameJa: '選手', gender: '', roleJa: '' })
    assert.equal(player.role, null)
    assert.equal(player.gender, null)
    assert.equal(player.number, '00')
    const staff = playerFormSchema.parse({
      nameJa: '監督',
      gender: 'F',
      roleJa: 'マネージャ',
      roleEn: 'Manager',
    })
    assert.deepEqual(staff.role, { ja: 'マネージャ', en: 'Manager' })
  })

  test('最終順位は rank-{id} の欄から作り、空欄は順位なし', () => {
    assert.deepEqual(finalRanksFromForm({ 'rank-5': '1', 'rank-6': '', back: '/admin' }), [
      { teamId: 5, rank: 1 },
      { teamId: 6, rank: null },
    ])
    assert.throws(() => finalRanksFromForm({ 'rank-5': '0' }))
  })
})

describe('グループとトーナメント表', () => {
  test('画像のグループは画像 URL が必須', () => {
    const base = { name: 'Image', kind: 'image', linked: 'on', published: 'on' }
    assert.equal(groupFormSchema.safeParse(base).success, false)
    const body = groupFormSchema.parse({ ...base, imageUrl: '/r.png', imageWidth: '500' })
    assert.deepEqual(body.image, { url: '/r.png', width: 500, height: null })
    assert.equal(groupFormSchema.parse({ name: 'Pool', kind: 'pool' }).image, null)
  })

  test('グループのチームは「順」の数字の順に並べ、空欄は入れない', () => {
    const body = groupTeamsFromForm({
      'team-10': '2',
      'team-11': '',
      'team-12': '1',
      'rank-12': '0',
      'rank-10': '',
    })
    assert.deepEqual(body, {
      teams: [
        { teamId: 12, rankOverride: 0 },
        { teamId: 10, rankOverride: null },
      ],
    })
  })

  test('チームか「A1-win」の参照を読む', () => {
    assert.deepEqual(sideChoiceSchema.parse('12'), { teamId: 12, placeholder: null })
    assert.deepEqual(sideChoiceSchema.parse('B1-lose'), {
      teamId: null,
      placeholder: { code: 'B1', outcome: 'lose' },
    })
    assert.deepEqual(sideChoiceSchema.parse(''), { teamId: null, placeholder: null })
    assert.equal(sideChoiceSchema.safeParse('B1').success, false)
  })

  test('トーナメント表の枠は全部選ぶ', () => {
    const body = bracketFormSchema.parse({
      template: 'single4',
      codePrefix: 'F',
      seedCount: '4',
      'seed-0': '1',
      'seed-1': 'A9-win',
      'seed-2': '3',
      'seed-3': '4',
    })
    assert.equal(body.seeds.length, 4)
    assert.deepEqual(body.seeds[1], { teamId: null, placeholder: { code: 'A9', outcome: 'win' } })
    assert.equal(
      bracketFormSchema.safeParse({
        template: 'single4',
        codePrefix: 'F',
        seedCount: '2',
        'seed-0': '1',
      }).success,
      false,
    )
  })
})

describe('試合とスコア入力', () => {
  test('試合の登録は日時の形と、左右で同じチームでないことを確かめる', () => {
    const body = matchFormSchema.parse({
      code: 'A2',
      date: '2026-10-18',
      time: '09:30',
      field: 'B',
      hasDetail: 'on',
      groupId: '',
      left: '5',
      right: 'A1-win',
    })
    assert.equal(body.groupId, null)
    assert.deepEqual(body.right, { teamId: null, placeholder: { code: 'A1', outcome: 'win' } })
    assert.equal(
      matchFormSchema.safeParse({ ...body, left: '5', right: '5', time: '' }).success,
      false,
    )
    assert.equal(matchFormSchema.safeParse({ left: '', right: '', time: '9:30' }).success, false)
  })

  test('経過は「分:秒」か分', () => {
    assert.equal(elapsedSchema.parse('12:05'), 725)
    assert.equal(elapsedSchema.parse('3'), 180)
    assert.equal(elapsedSchema.parse(''), null)
    assert.equal(elapsedSchema.safeParse('12:75').success, false)
  })

  test('得点は得点者とアシスト、経過が空欄なら null(api が決める)', () => {
    const body = eventFormSchema.parse({
      kind: 'goal',
      side: 'left',
      scorerId: '7',
      assistId: '',
      elapsed: '',
    })
    assert.deepEqual(body, {
      kind: 'goal',
      side: 'left',
      elapsedSec: null,
      scorerId: 7,
      assistId: null,
      callahan: false,
      winnerSide: null,
    })
  })

  test('キャラハンにアシストは付けられない。得点以外は選手を送らない', () => {
    const callahan = { kind: 'goal', side: 'right', scorerId: '7', assistId: '8', callahan: 'on' }
    assert.equal(eventFormSchema.safeParse(callahan).success, false)
    const timeout = eventFormSchema.parse({ kind: 'timeout', side: 'right', scorerId: '7' })
    assert.equal(timeout.scorerId, null)
    assert.equal(eventFormSchema.safeParse({ kind: 'pull' }).success, false, 'プルは左右が必要')
    const end = eventFormSchema.parse({ kind: 'end', side: 'left', winnerSide: 'right' })
    assert.equal(end.side, null)
    assert.equal(end.winnerSide, 'right')
  })

  test('状態の変更のスコアは左右とも入れるか両方空欄', () => {
    assert.deepEqual(
      statusFormSchema.parse({ status: 'finished', leftScore: '3', rightScore: '9' }),
      {
        status: 'finished',
        winnerSide: null,
        score: { left: 3, right: 9 },
      },
    )
    assert.equal(statusFormSchema.parse({ status: 'live', winnerSide: 'left' }).winnerSide, null)
    assert.equal(statusFormSchema.safeParse({ status: 'finished', leftScore: '3' }).success, false)
  })

  test('選手記録は出場か数のある選手だけを送る', () => {
    const body = playerStatsFromForm({
      'played-left-1': 'on',
      'blocks-left-1': '0',
      'greatests-left-1': '0',
      'blocks-left-2': '2',
      'greatests-left-2': '',
      'blocks-right-3': '0',
      'greatests-right-3': '0',
    })
    assert.deepEqual(body.stats, [
      { playerId: 1, side: 'left', blocks: 0, greatests: 0 },
      { playerId: 2, side: 'left', blocks: 2, greatests: 0 },
    ])
    assert.throws(() => playerStatsFromForm({ 'blocks-left-1': '-1' }))
  })
})

describe('ユーザー', () => {
  test('更新ではパスワードの空欄は送らない。作成では必須', () => {
    const update = userFormSchema.parse({ email: 'a@example.com', role: 'staff', password: '' })
    assert.equal(update.password, undefined)
    assert.equal(
      userFormSchema.safeParse({ email: 'a@example.com', role: 'staff', password: 'short' })
        .success,
      false,
    )
    assert.equal(
      userCreateFormSchema.safeParse({ email: 'a@example.com', role: 'staff' }).success,
      false,
    )
    assert.equal(userFormSchema.safeParse({ email: 'not-an-email', role: 'staff' }).success, false)
  })

  test('担当大会はチェックした値の一覧(1 つでも 0 でもよい)', () => {
    assert.deepEqual(userTournamentsFormSchema.parse({ tournamentIds: ['1', '3'] }), {
      tournamentIds: [1, 3],
    })
    assert.deepEqual(userTournamentsFormSchema.parse({ tournamentIds: '2' }), {
      tournamentIds: [2],
    })
    assert.deepEqual(userTournamentsFormSchema.parse({}), { tournamentIds: [] })
  })
})

describe('記録係の割り当てのフォーム', () => {
  test('空欄の条件は null(全コート、全日程、終日)', () => {
    assert.deepEqual(
      assignmentFormSchema.parse({ userId: '5', field: '3', from: '10:00', to: '' }),
      {
        userId: 5,
        matchId: null,
        field: '3',
        date: null,
        from: '10:00',
        to: null,
      },
    )
  })

  test('試合を選んだら範囲の条件は送らない。時間帯は始まりより後で終わる', () => {
    assert.deepEqual(assignmentFormSchema.parse({ userId: '5', matchId: '9', field: '3' }), {
      userId: 5,
      matchId: 9,
      field: null,
      date: null,
      from: null,
      to: null,
    })
    assert.equal(
      assignmentFormSchema.safeParse({ userId: '5', from: '12:00', to: '10:00' }).success,
      false,
    )
  })
})
