import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import {
  divisionHref,
  divisionScheduleHref,
  galleryHref,
  groupHref,
  matchHref,
  playerHref,
  pscoreHref,
  scviewHref,
  scvTopHref,
  teamHref,
  teamListHref,
  tournamentListHref,
  tournamentScheduleHref,
} from '../src/i18n/href.ts'
import { parseLang, pickText } from '../src/i18n/lang.ts'
import { angleHeading, msg, placeholderLabel } from '../src/i18n/messages.ts'

describe('parseLang', () => {
  test('en だけが英語、省略・不正値・jp は日本語', () => {
    assert.equal(parseLang('en'), 'en')
    assert.equal(parseLang('jp'), 'jp')
    assert.equal(parseLang(undefined), 'jp')
    assert.equal(parseLang('EN'), 'jp')
    assert.equal(parseLang(['en', 'jp']), 'en')
  })
})

describe('pickText', () => {
  test('英語が空でも日本語で補わない', () => {
    assert.equal(pickText('en', { ja: '大会', en: '' }), '')
    assert.equal(pickText('jp', { ja: '大会', en: 'Cup' }), '大会')
  })
})

describe('href', () => {
  test('参照サイトと同じパスに lang を引き継ぐ', () => {
    assert.equal(tournamentListHref('en'), '/sctnmt?lang=en')
    assert.equal(scviewHref(525, 'jp'), '/scview/525?lang=jp')
    assert.equal(scvTopHref(525, 'en'), '/pscore/scv_top/open/525?lang=en')
    assert.equal(teamListHref(525, 'jp'), '/pscore/scv_team_list/open/525?lang=jp')
    assert.equal(tournamentScheduleHref(525, 'jp'), '/pscore/scv_search_tn/open/525?lang=jp')
    assert.equal(divisionScheduleHref(949, 'jp'), '/pscore/scv_search_div/open/949?lang=jp')
    assert.equal(divisionHref(949, 'jp'), '/pscore/scv_division/open/949?lang=jp')
    assert.equal(groupHref(6623, 'jp'), '/pscore/scv_group/open/6623?lang=jp')
    assert.equal(galleryHref(362, 'jp'), '/pscore/scv_tgallery/open/362?lang=jp')
    assert.equal(teamHref(17322, 'en'), '/pscore/scv_team/open/17322?lang=en')
    assert.equal(matchHref(45280, 'jp'), '/pscore/scv_match/open/45280?lang=jp')
    assert.equal(playerHref(604138, 'jp'), '/pscore/scv_player/open/604138?lang=jp')
  })

  test('追加のクエリは lang の後ろに付き、値はエンコードされる', () => {
    assert.equal(
      pscoreHref('scv_search_tn', 1, 'jp', { ymd: '', team: '架空 A' }),
      '/pscore/scv_search_tn/open/1?lang=jp&ymd=&team=%E6%9E%B6%E7%A9%BA+A',
    )
  })
})

describe('messages', () => {
  test('ナビの文言(グループ成績は jp だけ末尾に全角空白)', () => {
    assert.equal(msg('jp', 'navGroupScore'), 'グループ成績　')
    assert.equal(msg('en', 'navGroupScore'), 'Group Score')
    assert.equal(msg('en', 'navEntryList'), 'Entry List')
  })

  test('大会一覧の区分見出しは en でも日本語', () => {
    assert.equal(msg('en', 'sectionFinished'), '大会終了')
  })

  test('参照サイトの誤綴りを再現する', () => {
    assert.equal(msg('en', 'leagueResult'), 'Reague Result')
    assert.equal(msg('en', 'statFromDefence'), 'Goals from starting on deffence')
  })

  test('山括弧見出しは jp だけ内側に空白', () => {
    assert.equal(angleHeading('jp', '結果一覧'), '< 結果一覧 >')
    assert.equal(angleHeading('en', 'GameList for the Group'), '<GameList for the Group>')
  })

  test('未確定チームの表記', () => {
    assert.equal(placeholderLabel('jp', 'A1', 'win'), '[A1-勝ち]')
    assert.equal(placeholderLabel('jp', 'E9', 'lose'), '[E9-負け]')
    assert.equal(placeholderLabel('en', 'S3', 'win'), '[S3-Win]')
    assert.equal(placeholderLabel('en', 'S3', 'lose'), '[S3-Lose]')
  })
})
