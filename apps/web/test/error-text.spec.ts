import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { currentScoreOf, operatorErrorText } from '../src/admin/errorText.utils.ts'

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
