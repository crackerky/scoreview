import { getTournamentSchedule } from '@/api/public'
import { loadPageData } from '@/app/_lib/notFound'
import { type IdPageProps, readIdPage } from '@/app/_lib/pageProps'
import { ScheduleView } from '@/components/ScheduleView'
import { SiteFrame } from '@/components/SiteFrame'
import { scheduleQueryFromSearchParams } from '@/components/schedule.utils'
import { tournamentPageTitle } from '@/components/title.utils'
import { msg } from '@/i18n/messages'

// clone-spec B: 大会全体のスケジュール。絞り込みと並べ替えは GET のクエリ(ymd / team / field / sortkey1)
export default async function TournamentSchedulePage(props: IdPageProps) {
  const { id, lang } = await readIdPage(props)
  const query = scheduleQueryFromSearchParams(await props.searchParams)
  const data = await loadPageData(id, (tournamentId) => getTournamentSchedule(tournamentId, query))
  return (
    <>
      <title>{tournamentPageTitle(data.frame, lang, msg(lang, 'titleAllSchedule'))}</title>
      <SiteFrame variant={{ kind: 'tournament', frame: data.frame, lang }} topMenu={null}>
        <ScheduleView data={data} lang={lang} />
      </SiteFrame>
    </>
  )
}
