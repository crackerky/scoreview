import { getPlayer } from '@/api/public'
import { loadPageData } from '@/app/_lib/notFound'
import { type IdPageProps, readIdPage } from '@/app/_lib/pageProps'
import { PlayerView } from '@/components/PlayerView'
import { SiteFrame } from '@/components/SiteFrame'
import { tournamentPageTitle } from '@/components/title.utils'
import { pickText } from '@/i18n/lang'

// clone-spec D 3: 選手ページ(スタッフも同じ)。title は "-{名前}-"(D 0)
export default async function PlayerPage(props: IdPageProps) {
  const { id, lang } = await readIdPage(props)
  const data = await loadPageData(id, getPlayer)
  const title = tournamentPageTitle(data.frame, lang, `-${pickText(lang, data.player.name)}-`)
  return (
    <>
      <title>{title}</title>
      <SiteFrame
        variant={{ kind: 'tournament', frame: data.frame, lang, scoreMenu: true }}
        topMenu={null}
      >
        <PlayerView data={data} lang={lang} />
      </SiteFrame>
    </>
  )
}
