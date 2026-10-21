import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import type { BracketCell } from '../src/api/common.types.ts'
import { bracketCellClass, matrixColumnWidth } from '../src/components/group.utils.ts'

function cell(
  kind: BracketCell['kind'],
  borders: Partial<BracketCell['borders']> = {},
): BracketCell {
  return {
    borders: { top: false, bottom: false, left: false, right: false, ...borders },
    kind,
    text: '',
    team: null,
    matchId: null,
    status: null,
  }
}

describe('対戦マトリクスの列幅(clone-spec C 2.1(a))', () => {
  test('78 / チーム数 の %', () => {
    assert.equal(matrixColumnWidth(3), '26%')
    assert.equal(matrixColumnWidth(4), '19.5%')
    assert.equal(matrixColumnWidth(5), '15.6%')
  })
  test('割り切れないときは有効 14 桁(推測)', () => {
    assert.equal(matrixColumnWidth(7), '11.142857142857%')
  })
})

describe('ブラケットのセルの class(clone-spec C 2.3)', () => {
  test('空セルは "WDiv{N} "', () => {
    assert.equal(bracketCellClass(cell(''), 4), 'WDiv4 ')
  })
  test('罫線は Left, Right, Top, Btm の順、各クラスの後ろに空白', () => {
    assert.equal(
      bracketCellClass(cell('', { top: true, bottom: true, left: true, right: true }), 4),
      'WDiv4 BdLeft BdRight BdTop BdBtm ',
    )
    assert.equal(bracketCellClass(cell('', { left: true, top: true }), 3), 'WDiv3 BdLeft BdTop ')
  })
  test('種別ごとの末尾の空白は参照サイトの原文ママ', () => {
    assert.equal(
      bracketCellClass(cell('game', { right: true, top: true }), 4),
      'WDiv4 BdRight BdTop game ',
    )
    assert.equal(bracketCellClass(cell('team', { bottom: true }), 4), 'WDiv4 BdBtm team ')
    assert.equal(
      bracketCellClass(cell('win', { left: true, bottom: true }), 4),
      'WDiv4 BdLeft BdBtm win  ',
    )
    assert.equal(bracketCellClass(cell('lose', { bottom: true }), 3), 'WDiv3 BdBtm lose ')
    assert.equal(bracketCellClass(cell('txt_yen'), 3), 'WDiv3 txt_yen   ')
    assert.equal(bracketCellClass(cell('txt_sharp'), 4), 'WDiv4 txt_sharp ')
    assert.equal(
      bracketCellClass(cell('txt_and', { left: true, top: true }), 4),
      'WDiv4 BdLeft BdTop txt_and   ',
    )
  })
})
