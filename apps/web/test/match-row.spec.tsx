import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import type { MatchRow } from '../src/api/common.types.ts'
import { scheduleResponseSchema } from '../src/api/public.types.ts'
import { MatchListRows, MatchRowLine } from '../src/components/MatchRowLine.tsx'
import { Tbl } from '../src/components/Tbl.tsx'
import { loadFixture } from './fixtures/load.ts'
import { renderHtml } from './render.ts'

const schedule = scheduleResponseSchema.parse(loadFixture('schedule-tournament-1'))

function rowAt(index: number) {
  const row = schedule.rows[index]
  assert.ok(row)
  return row
}

function renderScheduleRow(index: number, lang: 'jp' | 'en' = 'jp'): string {
  const row = rowAt(index)
  return renderHtml(
    <table>
      <tbody>
        <MatchRowLine
          row={row}
          lang={lang}
          variant={{ kind: 'schedule', division: row.division }}
        />
      </tbody>
    </table>,
  )
}

function renderListRow(row: MatchRow, lang: 'jp' | 'en' = 'jp'): string {
  return renderHtml(
    <Tbl>
      <MatchListRows rows={[row]} lang={lang} />
    </Tbl>,
  )
}

describe('スケジュールの行(clone-spec B 5〜8)', () => {
  test('終了・詳細ありの行: 状態クラス、ディビジョン種別、切り詰め、[対戦結果] リンク', () => {
    const html = renderScheduleRow(0)
    assert.match(html, /<tr class="H24"><td class="MC_END MC_MIX CM">ビーチミッ<\/td>/)
    assert.match(html, /<td class="MC_END CM">10\/18<\/td><td class="MC_END CM">09:00 <\/td>/)
    assert.match(html, /<td class="MC_END CM">1<\/td><td class="MC_END CM">A1<\/td>/)
    assert.match(
      html,
      /<td class="MC_END RM"><span class="teamLink"><a href="\/pscore\/scv_team\/open\/5001\?lang=jp">架空チームA<\/a><\/span><\/td>/,
    )
    assert.match(
      html,
      /<td class="MC_END CM">9<\/td><td class="MC_END CM">x<\/td><td class="MC_END CM">3<\/td>/,
    )
    assert.match(html, /<td class="MC_END LM"><span class="teamLink">/)
    assert.match(
      html,
      /<a class="Select_Link" href="\/pscore\/scv_match\/open\/9001\?lang=jp">\[対戦結果\]<\/a><\/td><\/tr>/,
    )
  })

  test('試合詳細のない終了試合は「終了」(en は END)', () => {
    assert.match(renderScheduleRow(1), /<td class="MC_END CM">終了<\/td><\/tr>/)
    assert.match(renderScheduleRow(1, 'en'), /<td class="MC_END CM">END<\/td><\/tr>/)
  })

  test('未試合の行: MC_MAE、スコア空、未確定チームは素のテキスト', () => {
    const html = renderScheduleRow(2)
    assert.match(html, /<td class="MC_MAE RM">\[A1-勝ち\]<\/td><td class="MC_MAE CM"><\/td>/)
    assert.match(html, /<td class="MC_MAE CM">x<\/td>/)
    assert.match(html, /<td class="MC_MAE LM">\[A2-負け\]<\/td>/)
    assert.match(html, /<td class="MC_MAE CM">未試合<\/td><\/tr>/)
    assert.match(renderScheduleRow(2, 'en'), /\[A1-Win\].*\[A2-Lose\].*Waiting/)
  })

  test('種別のないディビジョンは種別クラスを付けず、空白 2 つが残る', () => {
    const row = rowAt(0)
    const html = renderHtml(
      <table>
        <tbody>
          <MatchRowLine
            row={row}
            lang="en"
            variant={{ kind: 'schedule', division: { ...row.division, kind: 'none' } }}
          />
        </tbody>
      </table>,
    )
    assert.match(html, /<td class="MC_END {2}CM">Beach Mixe<\/td>/)
  })

  test('チーム名は表示幅 16 で切り詰める', () => {
    const row = rowAt(0)
    const longName = { ja: 'シロコロホルモンズ', en: "Aichi Syukutoku University Garnet's" }
    const withLong = { ...row, left: { ...row.left, team: { id: 1, name: longName } } }
    const variant = { kind: 'schedule', division: row.division } as const
    const jp = renderHtml(
      <table>
        <tbody>
          <MatchRowLine row={withLong} lang="jp" variant={variant} />
        </tbody>
      </table>,
    )
    assert.match(jp, />シロコロホルモン<\/a>/)
    const en = renderHtml(
      <table>
        <tbody>
          <MatchRowLine row={withLong} lang="en" variant={variant} />
        </tbody>
      </table>,
    )
    assert.match(en, />Aichi Syukutoku <\/a>/)
  })
})

describe('結果一覧の行(clone-spec C 2.1(c)、D 1.1)', () => {
  test('見出し行の列と幅、CD は左寄せ、区切りは全角 Ｘ、名前は切り詰めない', () => {
    const row = rowAt(0)
    const longRow = {
      ...row,
      left: { ...row.left, team: { id: 1, name: { ja: 'シロコロホルモンズ', en: '' } } },
    }
    const html = renderListRow(longRow)
    assert.match(
      html,
      /<tr><td class="MD1 CM" width="10%">CD<\/td><td class="MD1 CM" width="7%">日付<\/td>/,
    )
    assert.match(
      html,
      /<td class="MD1 CM" width="20%">チーム名<\/td><td class="MD1 CM" width="4%">-<\/td>/,
    )
    assert.match(html, /<td class="MD1 CM" width="18%">状況<\/td><\/tr>/)
    assert.match(
      html,
      /<tr class="H24"><td class="MC_END LM">A1<\/td><td class="MC_END CM">10\/18<\/td>/,
    )
    assert.match(html, /<td class="MC_END CM">Ｘ<\/td>/)
    assert.match(html, />シロコロホルモンズ<\/a>/)
  })

  test('英語の見出し', () => {
    const html = renderListRow(rowAt(0), 'en')
    assert.match(
      html,
      />CD<\/td>.*>Date<\/td>.*>Time<\/td>.*>Field<\/td>.*>Team<\/td>.*>Status<\/td>/,
    )
    assert.match(html, />\[Result\]<\/a>/)
  })

  test('推測: 試合中の行は MC_NOW', () => {
    const live = { ...rowAt(0), status: 'live' as const }
    assert.match(renderListRow(live), /<td class="MC_NOW LM">A1<\/td>/)
  })
})
