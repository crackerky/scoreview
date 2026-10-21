import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { impactSchema, progressSchema, savedFormatOneSchema } from '../src/api/format.types.ts'
import { loadFixture } from './fixtures/load.ts'

// fixture は開発用の api(Go)の実際の応答を保存したもの(架空のチーム)。Go の DTO と zod の食い違いを見つける

describe('大会の形と進行の応答のスキーマ', () => {
  test('進行状況: 確定待ち 3 プールと、未決の同順位 1 つ', () => {
    const progress = progressSchema.parse(loadFixture('format-progress'))
    const kinds = progress.todo.map((todo) => todo.kind)
    assert.equal(kinds.filter((kind) => kind === 'confirm_group').length, 3)
    assert.equal(kinds.filter((kind) => kind === 'tie').length, 1)
    const poolD = progress.groups.find((group) => group.name === 'Qualifying pool D')
    assert.equal(poolD?.state, 'tied')
    assert.equal(poolD?.ties[0]?.length, 3)
  })

  test('保存済みの形とシード', () => {
    const saved = savedFormatOneSchema.parse(loadFixture('format-saved'))
    assert.equal(saved.format?.template, 'pool_bands')
    assert.equal(saved.format?.seeds.length, 12)
  })

  test('訂正の影響', () => {
    const impact = impactSchema.parse(loadFixture('format-impact'))
    assert.equal(impact.needsApproval, false)
    assert.match(impact.lines[0] ?? '', /2 位と 3 位が入れ替わります/)
  })
})
