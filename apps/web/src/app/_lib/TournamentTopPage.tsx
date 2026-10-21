import { getTournamentTop } from '@/api/public'
import { SiteFrame } from '@/components/SiteFrame'
import { TournamentTopView } from '@/components/TournamentTopView'
import { tournamentPageTitle } from '@/components/title.utils'
import { msg } from '@/i18n/messages'
import { loadPageData } from './notFound'
import { type IdPageProps, readIdPage } from './pageProps'

/**
 * 大会トップ。/scview/{id} と /pscore/scv_top/open/{id} は同じ内容を返す(リダイレクトしない。
 * clone-spec A 0)。違いは存在しない ID のエラー画面の文面だけ(各ルートの not-found.tsx)。
 */
export async function TournamentTopPage(props: IdPageProps) {
  const { id, lang } = await readIdPage(props)
  const data = await loadPageData(id, getTournamentTop)
  return (
    <>
      <title>{tournamentPageTitle(data.frame, lang)}</title>
      <SiteFrame
        heading={msg(lang, 'navTournamentTop')}
        variant={{ kind: 'tournament', frame: data.frame, lang }}
      >
        <TournamentTopView data={data} lang={lang} />
      </SiteFrame>
    </>
  )
}
