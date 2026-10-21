import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { tournamentFrameSchema } from '../src/api/common.types.ts'
import { Sidebar } from '../src/components/Sidebar.tsx'
import { loadFixture } from './fixtures/load.ts'
import { renderHtml } from './render.ts'

const frame = tournamentFrameSchema.parse(loadFixture('frame-1'))

describe('サイドバーの SCORE メニュー', () => {
  test('scoreMenu=false ならメニューを出さずバナーだけ', () => {
    const html = renderHtml(<Sidebar frame={frame} lang="jp" scoreMenu={false} />)
    assert.doesNotMatch(html, /menu_list/)
    assert.match(html, /sub_banners/)
  })

  test('ディビジョン名に (Ult) を付け、ディビジョン成績とグループへリンクする', () => {
    const html = renderHtml(<Sidebar frame={frame} lang="jp" scoreMenu={true} />)
    const division = frame.divisions[0]
    assert.ok(division)
    assert.match(html, /<div class="sub_score_title">SCORE<\/div>/)
    assert.ok(
      html.includes(
        `<a href="/pscore/scv_division/open/${division.id}?lang=jp">${division.name.ja}(Ult)</a>`,
      ),
    )
    for (const group of division.groups.filter((g) => g.linked)) {
      assert.ok(
        html.includes(`<a href="/pscore/scv_group/open/${group.id}?lang=jp">${group.name}</a>`),
      )
    }
  })

  test('英語では英語名に (Ult) を付ける', () => {
    const html = renderHtml(<Sidebar frame={frame} lang="en" scoreMenu={true} />)
    const division = frame.divisions[0]
    assert.ok(division)
    assert.ok(html.includes(`${division.name.en}(Ult)</a>`))
    assert.match(html, /\?lang=en/)
  })

  test('スケジュールのないディビジョンは載せない', () => {
    const noSchedule = {
      ...frame,
      divisions: frame.divisions.map((division) => ({ ...division, scheduleEnabled: false })),
    }
    const html = renderHtml(<Sidebar frame={noSchedule} lang="jp" scoreMenu={true} />)
    assert.doesNotMatch(html, /scv_division/)
    assert.match(html, /menu_list/)
  })
})
