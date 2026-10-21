import { getDivisionSchedule } from '@/api/public'
import { loadPageData } from '@/app/_lib/notFound'
import { type IdPageProps, readIdPage } from '@/app/_lib/pageProps'
import { ScheduleView } from '@/components/ScheduleView'
import { SiteFrame } from '@/components/SiteFrame'
import { scheduleQueryFromSearchParams } from '@/components/schedule.utils'
import { tournamentPageTitle } from '@/components/title.utils'
import { pickText } from '@/i18n/lang'
import { msg } from '@/i18n/messages'

// clone-spec B 4: ディビジョン別のスケジュール。ymd のキーがなければ全日程(大会全体は最終日)
export default async function DivisionSchedulePage(props: IdPageProps) {
  const { id, lang } = await readIdPage(props)
  const query = scheduleQueryFromSearchParams(await props.searchParams)
  const data = await loadPageData(id, (divisionId) => getDivisionSchedule(divisionId, query))
  // 本文の上の「HH:MM 時点」(api から読み終えた時刻)
  const now = new Date()
  return (
    <>
      <title>{tournamentPageTitle(data.frame, lang, msg(lang, 'titleDivisionSchedule'))}</title>
      <SiteFrame
        heading={`${msg(lang, 'navSchedule')} ${data.division === null ? '' : pickText(lang, data.division.name)}`}
        asOf={now}
        variant={{ kind: 'tournament', frame: data.frame, lang }}
        topMenu={null}
      >
        <ScheduleView data={data} lang={lang} />
      </SiteFrame>
    </>
  )
}
