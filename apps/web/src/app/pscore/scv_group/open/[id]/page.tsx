import { getGroup } from '@/api/public'
import { loadPageData } from '@/app/_lib/notFound'
import { type IdPageProps, readIdPage } from '@/app/_lib/pageProps'
import { GroupView } from '@/components/GroupView'
import { SiteFrame } from '@/components/SiteFrame'
import { tournamentPageTitle } from '@/components/title.utils'

// clone-spec C 2: グループ成績。title の接尾辞は "-{グループ名}-"
export default async function GroupPage(props: IdPageProps) {
  const { id, lang } = await readIdPage(props)
  const data = await loadPageData(id, getGroup)
  return (
    <>
      <title>{tournamentPageTitle(data.frame, lang, `-${data.group.name}-`)}</title>
      <SiteFrame variant={{ kind: 'tournament', frame: data.frame, lang }} topMenu={null}>
        <GroupView data={data} lang={lang} />
      </SiteFrame>
    </>
  )
}
