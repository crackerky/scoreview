import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { type ScheduleResponse, scheduleResponseSchema } from '../src/api/public.types.ts'
import { RefreshButtons } from '../src/components/RefreshButtons.tsx'
import { ScheduleView } from '../src/components/ScheduleView.tsx'
import { tournamentPageTitle } from '../src/components/title.utils.ts'
import { msg } from '../src/i18n/messages.ts'
import { loadFixture } from './fixtures/load.ts'
import { renderHtml } from './render.ts'

const tournamentSchedule = scheduleResponseSchema.parse(loadFixture('schedule-tournament-1'))
const divisionSchedule = scheduleResponseSchema.parse(loadFixture('schedule-division-101'))

describe('検索フォーム(clone-spec B 3〜5)', () => {
  const html = renderHtml(<ScheduleView data={tournamentSchedule} lang="jp" />)

  test('GET のフォームと hidden の lang、select の既定値(大会全体は api が返した最終日)', () => {
    assert.match(
      html,
      /^<div class="ITBL"><form accept-charset="utf-8" method="get"><input type="hidden" name="lang" value="jp"\/><table><tbody><tr>/,
    )
    assert.match(
      html,
      /<td class="NINI CM BOLD" width="40px"><label for="ymd">日付<\/label><\/td><td class="" width="120px"><select name="ymd" class="NWAKU {2}WMAX" id="ymd"><option value=""><\/option><option value="20261017">2026\/10\/17<\/option><option value="20261018" selected="">2026\/10\/18<\/option><\/select><\/td><td class="NINI" width="10px"><\/td>/,
    )
  })

  test('大会全体の team の value は正式名、コートは異なり値', () => {
    assert.match(
      html,
      /<td class="NINI CM BOLD" width="60px"><label for="team">チーム名<\/label><\/td><td class="" width="180px"><select name="team" class="NWAKU {2}WMAX" id="team"><option value="" selected=""><\/option><option value="架空チームA">架空チームA<\/option>/,
    )
    assert.match(
      html,
      /<td class="NINI CM BOLD" width="60px"><label for="field">コート<\/label><\/td><td class="" width="100px"><select name="field" class="NWAKU {2}WMAX" id="field"><option value="" selected=""><\/option><option value="1">1<\/option><option value="2">2<\/option><\/select>/,
    )
  })

  test('ディビジョン別は ymd 空が選ばれ、team の value はチーム ID', () => {
    const div = renderHtml(<ScheduleView data={divisionSchedule} lang="en" />)
    assert.match(
      div,
      /<select name="ymd" class="NWAKU {2}WMAX" id="ymd"><option value="" selected=""><\/option><option value="20261018">2026\/10\/18<\/option>/,
    )
    assert.match(div, /<option value="5001">Fictional A<\/option>/)
    assert.match(div, />Date<\/label><\/td>.*>Team<\/label><\/td>.*>Field<\/label><\/td>/)
  })

  test('更新ボタンは現在の絞り込みで開き直す。最初は NG だけ見える', () => {
    assert.match(
      html,
      /<td class="RM" width="120px" style="height:33px"><a id="Refresh_OK" style="display:none" href="\/pscore\/scv_search_tn\/open\/1\?lang=jp&amp;ymd=20261018&amp;team=&amp;field="><img src="\/brand\/refresh.svg" alt="Reflesh_OK"\/><\/a><a id="Refresh_NG" href="\/pscore\/scv_search_tn\/open\/1\?lang=jp&amp;ymd=20261018&amp;team=&amp;field="><img src="\/brand\/refresh-ng.svg" alt="RefleshWait"\/><\/a>\u00a0<noscript><input type="submit" value="検索"\/><\/noscript><\/td>/,
    )
  })

  test('NG の alert の文面(jp は全角の５)', () => {
    assert.equal(msg('jp', 'refreshWait'), '５秒お待ちください')
    assert.equal(msg('en', 'refreshWait'), 'Please Wait 5 seconds. (Reload)')
    const en = renderHtml(
      <RefreshButtons reloadHref="/x" waitMessage="wait" okAlt="Reflesh_OK" ngAlt="RefleshWait" />,
    )
    assert.match(en, /^<a id="Refresh_OK" style="display:none" href="\/x">/)
  })
})

describe('一覧表の見出し(clone-spec B 2、B 5)', () => {
  test('11 列の幅、並べ替えリンク、左チームの後ろの空白', () => {
    const html = renderHtml(<ScheduleView data={tournamentSchedule} lang="jp" />)
    const base = '/pscore/scv_search_tn/open/1\\?lang=jp&amp;ymd=20261018&amp;team=&amp;field='
    assert.match(
      html,
      new RegExp(
        `<div class="TBL"><table><tbody><tr><td class="MD1 CM" width="13%"><a class="MD1" href="${base}&amp;sortkey1=ASC_mdv.DIVISION_SORTKEY">ディビジョン</a></td>`,
      ),
    )
    for (const [label, column] of [
      ['日付', 'mat.MATCH_DATE'],
      ['時間', 'mat.START_TIME'],
      ['コート', 'mat.FIELD_NO'],
      ['CD', 'mat.MATCH_CD'],
    ]) {
      assert.match(
        html,
        new RegExp(
          `<td class="MD1 CM" width="7%"><a class="MD1" href="${base}&amp;sortkey1=ASC_${column}">${label}</a></td>`,
        ),
      )
    }
    assert.match(
      html,
      new RegExp(
        `<td class="MD1 RM" width="17%"><a class="MD1" href="${base}&amp;sortkey1=ASC_lt_name">チーム名</a> </td><td class="MD1 CM" width="4%">-</td><td class="MD1 CM" width="3%">VS</td><td class="MD1 CM" width="4%">-</td><td class="MD1 LM" width="17%"><a class="MD1" href="${base}&amp;sortkey1=ASC_rt_name">チーム名</a></td><td class="MD1 CM" width="14%">状況</td></tr>`,
      ),
    )
  })

  test('昇順で並べ替え中の列は MD1_ASC + ▼、次は DSC_', () => {
    const html = renderHtml(<ScheduleView data={divisionSchedule} lang="jp" />)
    assert.match(
      html,
      /<a class="MD1_ASC" href="\/pscore\/scv_search_div\/open\/101\?lang=jp&amp;ymd=&amp;team=&amp;field=&amp;sortkey1=DSC_mat.FIELD_NO">コート▼<\/a>/,
    )
    assert.match(html, /<a class="MD1" href="[^"]*sortkey1=ASC_mat.MATCH_DATE">日付<\/a>/)
  })

  test('降順で並べ替え中の列は MD1_DSC + ▲、次は ASC_', () => {
    const data: ScheduleResponse = { ...divisionSchedule, sort: { key: 'leftTeam', dir: 'desc' } }
    const html = renderHtml(<ScheduleView data={data} lang="en" />)
    assert.match(html, /<a class="MD1_DSC" href="[^"]*sortkey1=ASC_lt_name">Team▲<\/a> <\/td>/)
    assert.match(html, /<a class="MD1" href="[^"]*sortkey1=ASC_mat.FIELD_NO">Field<\/a>/)
  })
})

describe('一覧表の行(clone-spec B 6〜8)', () => {
  test('ディビジョン列だけに種別クラス、無い種別は空白 2 つ', () => {
    const row = tournamentSchedule.rows[2]
    assert.ok(row)
    const data: ScheduleResponse = {
      ...tournamentSchedule,
      rows: [{ ...row, division: { ...row.division, kind: 'none' } }],
    }
    const html = renderHtml(<ScheduleView data={data} lang="jp" />)
    assert.match(
      html,
      /<tr class="H24"><td class="MC_MAE {2}CM">ビーチミッ<\/td><td class="MC_MAE CM">10\/18<\/td><td class="MC_MAE CM">11:00 <\/td>/,
    )
    assert.match(html, /<td class="MC_MAE RM">\[A1-勝ち\]<\/td>/)
    assert.match(html, /<td class="MC_MAE CM">未試合<\/td><\/tr>/)
  })

  test('種別ありは MC_END MC_MIX CM', () => {
    const html = renderHtml(<ScheduleView data={tournamentSchedule} lang="en" />)
    assert.match(html, /<tr class="H24"><td class="MC_END MC_MIX CM">Beach Mixe<\/td>/)
    assert.match(html, />\[Result\]<\/a><\/td><\/tr>/)
    assert.match(html, /<td class="MC_END CM">END<\/td><\/tr>/)
  })
})

describe('<title>(clone-spec B 4、B 10)', () => {
  test('全スケジュールとディビジョンスケジュール', () => {
    const frame = tournamentSchedule.frame
    assert.equal(
      tournamentPageTitle(frame, 'jp', msg('jp', 'titleAllSchedule')),
      '[SCV] 2026 架空ビーチカップ -全スケジュール-',
    )
    assert.equal(
      tournamentPageTitle(frame, 'en', msg('en', 'titleDivisionSchedule')),
      '[SCV] 2026 Fictional Beach Cup -Division Schedule-',
    )
  })
})
