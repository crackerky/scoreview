import { getGallery } from '@/api/public'
import { loadPageData } from '@/app/_lib/notFound'
import { type IdPageProps, readIdPage } from '@/app/_lib/pageProps'
import { GalleryView } from '@/components/GalleryView'
import { SiteFrame } from '@/components/SiteFrame'
import { tournamentPageTitle } from '@/components/title.utils'

// clone-spec A 6: ギャラリー。本文は lang に関係なく同じで、ナビの文言だけが変わる
export default async function GalleryPage(props: IdPageProps) {
  const { id, lang } = await readIdPage(props)
  const data = await loadPageData(id, getGallery)
  return (
    <>
      <title>{tournamentPageTitle(data.frame, lang)}</title>
      <SiteFrame variant={{ kind: 'tournament', frame: data.frame, lang }} topMenu={null}>
        <GalleryView data={data} />
      </SiteFrame>
    </>
  )
}
