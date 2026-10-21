import { Fragment } from 'react'
import type { TournamentStatus } from '@/api/common.types'
import type { TournamentListResponse } from '@/api/public.types'
import { scviewHref } from '@/i18n/href'
import { type Lang, pickText } from '@/i18n/lang'
import { type MessageKey, msg } from '@/i18n/messages'

// clone-spec A 3: 大会一覧 /sctnmt の #score_view の中身。
// 区分見出し(TeamListMIDA)は lang=en でも日本語。大会名だけが英語名になり、空なら <a> の中身も空。

const sectionLabel: Record<TournamentStatus, MessageKey> = {
  entry_open: 'sectionEntryOpen',
  entry_closed: 'sectionEntryClosed',
  finished: 'sectionFinished',
}

const headingBoxStyle = {
  margin: '2px',
  padding: '1px',
  borderColor: '#cccccc',
  borderWidth: '0px',
  borderStyle: 'solid',
} as const

export function TournamentListView({ data, lang }: { data: TournamentListResponse; lang: Lang }) {
  return (
    <>
      {data.sections.map((section) => (
        <Fragment key={section.status}>
          <div style={{ clear: 'both', height: '10px' }} />
          <div style={headingBoxStyle}>
            <div className="TeamListMIDA">{msg(lang, sectionLabel[section.status])}</div>
          </div>
          {section.tournaments.map((tournament) => (
            <div key={tournament.id} style={{ marginLeft: '14px', marginTop: '10px' }}>
              <div className="TeamListBLOCK" style={{ margin: '2px 0px', width: '98%' }}>
                <div style={{ padding: '4px', color: 'white' }} className="px12 FTitle">
                  <span className="teamimg_rnk">
                    <a href={scviewHref(tournament.id, lang)}>{pickText(lang, tournament.name)}</a>
                  </span>
                </div>
              </div>
            </div>
          ))}
        </Fragment>
      ))}
    </>
  )
}
