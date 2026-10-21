import { getTeamList } from '@/api/public'
import { loadPageData } from '@/app/_lib/notFound'
import { type IdPageProps, readIdPage } from '@/app/_lib/pageProps'
import { SiteFrame } from '@/components/SiteFrame'
import { TeamListView } from '@/components/TeamListView'
import { tournamentPageTitle } from '@/components/title.utils'

// clone-spec A 5: チーム一覧(個人戦の大会ではエントリー一覧)
export default async function TeamListPage(props: IdPageProps) {
  const { id, lang } = await readIdPage(props)
  const data = await loadPageData(id, getTeamList)
  return (
    <>
      <title>{tournamentPageTitle(data.frame, lang)}</title>
      <SiteFrame variant={{ kind: 'tournament', frame: data.frame, lang }} topMenu={null}>
        <TeamListView data={data} lang={lang} />
      </SiteFrame>
    </>
  )
}
