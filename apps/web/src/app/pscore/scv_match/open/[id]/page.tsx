import { getMatch } from '@/api/public'
import type { MatchResponse } from '@/api/public.types'
import { loadPageData } from '@/app/_lib/notFound'
import { type IdPageProps, readIdPage } from '@/app/_lib/pageProps'
import { MatchTopMenu, MatchView } from '@/components/MatchView'
import { SiteFrame } from '@/components/SiteFrame'
import { tournamentPageTitle } from '@/components/title.utils'
import { type Lang, pickText } from '@/i18n/lang'

type MatchSide = MatchResponse['match']['left']

function fullName(side: MatchSide, lang: Lang): string {
  return side.fullName === null ? '' : pickText(lang, side.fullName)
}

// clone-spec D 2: 試合ページ。title は "-[{左の正式名} VS {右の正式名}]-"(D 0)
export default async function MatchPage(props: IdPageProps) {
  const [{ id, lang }, query] = await Promise.all([readIdPage(props), props.searchParams])
  const data = await loadPageData(id, getMatch)
  const { left, right } = data.match
  const suffix = `-[${fullName(left, lang)} VS ${fullName(right, lang)}]-`
  return (
    <>
      <title>{tournamentPageTitle(data.frame, lang, suffix)}</title>
      <SiteFrame
        variant={{ kind: 'tournament', frame: data.frame, lang }}
        topMenu={
          <MatchTopMenu matchId={data.match.id} lang={lang} scrollToNavi={query.refpos === '1'} />
        }
      >
        <MatchView data={data} now={new Date()} lang={lang} />
      </SiteFrame>
    </>
  )
}
