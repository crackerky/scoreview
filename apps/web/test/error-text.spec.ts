import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { currentScoreOf, operatorErrorText, validationParts } from '../src/admin/errorText.utils.ts'

// api のエラーを運営の言葉にする共通の変換表(src/admin/errorText.utils.ts。レビュー H6)。

describe('エラーの変換表', () => {
  test('項目と理由から運営の言葉の文を作り、接頭辞を出さない', () => {
    assert.equal(
      operatorErrorText('bad_request', 'winnerSide: required for a tied match in a bracket group'),
      'トーナメントは引き分けにできません。勝ったチームを選んでください',
    )
    assert.equal(
      operatorErrorText('bad_request', 'forfeitScore: the forfeit score is 15-0'),
      '不戦勝のスコアは大会の規定(15-0)で記録します。スコアの欄を空にしてください',
    )
    assert.equal(
      operatorErrorText(
        'bad_request',
        'score: 得点経過のある試合は、スコア入力画面で得点経過を直してください',
      ),
      '得点経過のある試合は、スコア入力画面で得点経過を直してください',
    )
  })

  test('規則に無い英語の理由は、項目の名前で尋ねる', () => {
    assert.equal(
      operatorErrorText('bad_request', 'name.ja: must be a string'),
      '「名前」の値を確かめてください',
    )
    assert.equal(operatorErrorText('bad_request', 'x: y'), '入力した値を確かめてください')
  })

  test('conflict は日本語の理由だけを出し、英語なら一般の文にする', () => {
    assert.equal(
      operatorErrorText('conflict', 'conflict: 予選 A 組に結果待ちの試合があります'),
      '予選 A 組に結果待ちの試合があります',
    )
    assert.equal(
      operatorErrorText('conflict', 'conflict: duplicate key'),
      'ほかの操作と重なったため保存できませんでした。画面を開き直してください',
    )
  })

  test('score_mismatch は今のスコアを出す', () => {
    const message = 'the score is now 11-9, not the confirmed 9-8'
    assert.deepEqual(currentScoreOf(message), { left: 11, right: 9 })
    assert.equal(
      operatorErrorText('score_mismatch', message),
      'この間に結果が 11-9 に変わりました。今の結果を確かめてから保存してください',
    )
  })

  test('どのコードも英語や内部の言葉を出さない', () => {
    const codes = ['not_found', 'forbidden', 'unauthorized', 'too_many_requests', 'unavailable']
    for (const code of codes) {
      const text = operatorErrorText(code, 'internal english text')
      assert.match(text, /[ぁ-ん]/, code)
      assert.doesNotMatch(text, /[a-z_]{4,}/, code)
    }
  })
})

test('validationParts は「項目: 理由」を分け、項目が無ければ全体を理由にする', () => {
  assert.deepEqual(validationParts('members[2].name: is required'), {
    field: 'members[2].name',
    reason: 'is required',
  })
  assert.deepEqual(validationParts('body is too large'), { field: '', reason: 'body is too large' })
  test('文の下のボタンの誤り', () => {
    assert.equal(
      operatorErrorText(
        'bad_request',
        'buttons[2].url: must be an https URL of at most 1000 characters',
      ),
      'リンクの URL は https で始まるもの(1000 字まで)にしてください',
    )
    assert.equal(
      operatorErrorText('bad_request', 'buttons[0].label: is required'),
      'リンクのボタンの文字を書いてください',
    )
    assert.equal(
      operatorErrorText('bad_request', 'buttons: must be at most 3'),
      'ボタンは 3 つまでです',
    )
    assert.equal(
      operatorErrorText('bad_request', 'body: {招待リンク} cannot be used for this audience'),
      '「招待リンク」はこの宛先には差し込めません。招待リンクとまだ登録していない人はこの大会に申し込んだ代表に、チーム名は友だち全員のほかに使えます',
    )
    assert.equal(
      operatorErrorText('bad_request', 'confirmAll: is required to send to every friend'),
      '友だち全員に送るときは、人数を確かめる欄に印を付けてください',
    )
  })
})
