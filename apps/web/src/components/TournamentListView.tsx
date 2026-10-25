import { Fragment } from 'react'
import type { ListSection, TournamentListResponse, TournamentSummary } from '@/api/public.types'
import { scviewHref, teamListHref } from '@/i18n/href'
import { type Lang, pickText } from '@/i18n/lang'
import { type MessageKey, msg } from '@/i18n/messages'
import { fillTemplate, splitFinished } from './tournamentList.utils'

// clone-spec A 3: 大会一覧 /sctnmt の #score_view の中身。
// 区分見出し(TeamListMIDA)は lang=en でも日本語。大会名だけが英語名になり、空なら <a> の中身も空。
// 本サイト独自(data-scoreview-addition): 開催中の区分、受付中の大会のエントリー数(チーム一覧への
// リンク)、終了した大会のうち最近の 5 件より前を年ごとにたたむ(クライアントの要望)。

const sectionLabel: Record<ListSection, MessageKey> = {
  in_progress: 'sectionInProgress',
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

function TournamentEntry({
  tournament,
  section,
  lang,
}: {
  tournament: TournamentSummary
  section: ListSection
  lang: Lang
}) {
  return (
    <div style={{ marginLeft: '14px', marginTop: '10px' }}>
      <div className="TeamListBLOCK" style={{ margin: '2px 0px', width: '98%' }}>
        <div style={{ padding: '4px', color: 'white' }} className="px12 FTitle">
          <span className="teamimg_rnk">
            <a href={scviewHref(tournament.id, lang)}>{pickText(lang, tournament.name)}</a>
          </span>
        </div>
      </div>
      {/* 帯は全体が大会トップへのリンク(legacy.css)なので、エントリー数は帯の外の下に出す */}
      {section === 'entry_open' && (
        <div className="sv_list_entries" data-scoreview-addition="">
          <a href={teamListHref(tournament.id, lang)}>
            {fillTemplate(msg(lang, 'listEntryCount'), { n: tournament.teamCount })}
          </a>
        </div>
      )}
    </div>
  )
}

function FinishedTournaments({
  tournaments,
  lang,
}: {
  tournaments: TournamentSummary[]
  lang: Lang
}) {
  const { recent, older } = splitFinished(tournaments)
  return (
    <>
      {recent.map((tournament) => (
        <TournamentEntry
          key={tournament.id}
          tournament={tournament}
          section="finished"
          lang={lang}
        />
      ))}
      {older.map((group) => (
        <details key={group.year ?? 'undated'} className="sv_list_year" data-scoreview-addition="">
          <summary>
            {group.year === null
              ? msg(lang, 'listUndated')
              : fillTemplate(msg(lang, 'listYearGroup'), {
                  year: group.year,
                  n: group.tournaments.length,
                })}
          </summary>
          {group.tournaments.map((tournament) => (
            <TournamentEntry
              key={tournament.id}
              tournament={tournament}
              section="finished"
              lang={lang}
            />
          ))}
        </details>
      ))}
    </>
  )
}

export function TournamentListView({ data, lang }: { data: TournamentListResponse; lang: Lang }) {
  return (
    <>
      {data.sections.map((section) => (
        <Fragment key={section.section}>
          <div style={{ clear: 'both', height: '10px' }} />
          <div style={headingBoxStyle}>
            <div className="TeamListMIDA">{msg(lang, sectionLabel[section.section])}</div>
          </div>
          {section.section === 'finished' ? (
            <FinishedTournaments tournaments={section.tournaments} lang={lang} />
          ) : (
            section.tournaments.map((tournament) => (
              <TournamentEntry
                key={tournament.id}
                tournament={tournament}
                section={section.section}
                lang={lang}
              />
            ))
          )}
        </Fragment>
      ))}
    </>
  )
}
