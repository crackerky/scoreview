import { getDivisionStandings } from '@/api/public'
import { loadPageData } from '@/app/_lib/notFound'
import { type IdPageProps, readIdPage } from '@/app/_lib/pageProps'
import { DivisionStandingsView } from '@/components/DivisionStandingsView'
import { SiteFrame } from '@/components/SiteFrame'
import { tournamentPageTitle } from '@/components/title.utils'
import { pickText } from '@/i18n/lang'

// clone-spec C 1: ディビジョン成績。title の接尾辞は "-{ディビジョン名}-"(参照サイトの実 HTML で確認)
export default async function DivisionStandingsPage(props: IdPageProps) {
  const { id, lang } = await readIdPage(props)
  const data = await loadPageData(id, getDivisionStandings)
  const suffix = `-${pickText(lang, data.division.name)}-`
  return (
    <>
      <title>{tournamentPageTitle(data.frame, lang, suffix)}</title>
      <SiteFrame
        variant={{ kind: 'tournament', frame: data.frame, lang, scoreMenu: true }}
        topMenu={null}
      >
        <DivisionStandingsView data={data} lang={lang} />
      </SiteFrame>
    </>
  )
}
