import { getTeam } from '@/api/public'
import { loadPageData } from '@/app/_lib/notFound'
import { type IdPageProps, readIdPage } from '@/app/_lib/pageProps'
import { SiteFrame } from '@/components/SiteFrame'
import { TeamView } from '@/components/TeamView'
import { tournamentPageTitle } from '@/components/title.utils'
import { pickText } from '@/i18n/lang'

// clone-spec D 1: チームページ。title は "-{チームの正式名}-"(D 0)
export default async function TeamPage(props: IdPageProps) {
  const { id, lang } = await readIdPage(props)
  const data = await loadPageData(id, getTeam)
  // 本文の上の「HH:MM 時点」(api から読み終えた時刻)
  const now = new Date()
  const title = tournamentPageTitle(data.frame, lang, `-${pickText(lang, data.team.fullName)}-`)
  return (
    <>
      <title>{title}</title>
      <SiteFrame
        asOf={now}
        variant={{ kind: 'tournament', frame: data.frame, lang, scoreMenu: true }}
        topMenu={null}
      >
        <TeamView data={data} lang={lang} />
      </SiteFrame>
    </>
  )
}
