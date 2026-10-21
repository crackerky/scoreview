import { Fragment } from 'react'
import type { TeamBarStyle } from '@/api/common.types'
import type { TeamListResponse } from '@/api/public.types'
import { teamHref } from '@/i18n/href'
import { type Lang, pickText } from '@/i18n/lang'
import { msg } from '@/i18n/messages'

// clone-spec A 5: チーム一覧 /pscore/scv_team_list/open/{id} の #score_view の中身。
// 1 チーム = 187px の色帯(名前)+ 30px の黒帯(x人数)。float: left で 1 行に 3 組並ぶ。
// 個人戦(entryKind=individual)は名前がリンクなしの文字で、見出しは "Players"、黒帯は空。

const barClass: Record<TeamBarStyle, string> = {
  default: 'FTitle',
  mixed: 'FTitle_Mix',
  women: 'FTitle_Women',
  univ: 'FTitle_Univ',
  waku1: 'FTitle_Waku1',
  waku2: 'FTitle_Waku2',
}

const nbsp = ' '
const nbsp3 = nbsp.repeat(3)

const headingBoxStyle = {
  margin: '2px',
  padding: '1px',
  borderColor: '#cccccc',
  borderWidth: '0px',
  borderStyle: 'solid',
  textAlign: 'center',
} as const

const barTextStyle = { padding: '4px', color: 'white' } as const

type Division = TeamListResponse['divisions'][number]

/** "{ディビジョン名}   -- 32 Teams --   "(件数の単位は jp でも英語) */
function headingText(division: Division, lang: Lang): string {
  const unit = msg(lang, division.entryKind === 'individual' ? 'teamListPlayers' : 'teamListTeams')
  return `${pickText(lang, division.name)}${nbsp3}--${nbsp}${division.teams.length} ${unit}${nbsp}--${nbsp3}`
}

export function TeamListView({ data, lang }: { data: TeamListResponse; lang: Lang }) {
  return (
    <>
      {data.divisions.map((division) => (
        <Fragment key={division.id}>
          <div style={headingBoxStyle}>
            <div className="TeamListMIDA">{headingText(division, lang)}</div>
          </div>
          <div style={{ marginLeft: '14px', marginTop: '10px' }}>
            {division.teams.map((team) => (
              <Fragment key={team.id}>
                <div className="TeamListBLOCK" style={{ margin: '2px 0px', width: '187px' }}>
                  <div style={barTextStyle} className={`px12 ${barClass[team.barStyle]}`}>
                    <span className="teamimg_rnk">
                      {division.entryKind === 'individual' ? (
                        pickText(lang, team.name)
                      ) : (
                        <a href={teamHref(team.id, lang)}>{pickText(lang, team.name)}</a>
                      )}
                    </span>
                  </div>
                </div>
                <div className="TeamListBLOCK" style={{ margin: '2px 8px 2px 0px', width: '30px' }}>
                  <div style={barTextStyle} className="px12 AreaSepa RM">
                    {team.memberCount === null ? nbsp : `x${team.memberCount}`}
                  </div>
                </div>
              </Fragment>
            ))}
          </div>
          <div style={{ clear: 'both', height: '10px' }} />
        </Fragment>
      ))}
    </>
  )
}
