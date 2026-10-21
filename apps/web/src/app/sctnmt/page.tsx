import { getTournaments } from '@/api/public'
import { SiteFrame } from '@/components/SiteFrame'
import { TournamentListView } from '@/components/TournamentListView'
import { tournamentListTitle } from '@/components/title.utils'
import { parseLang } from '@/i18n/lang'

type Props = { searchParams: Promise<Record<string, string | string[] | undefined>> }

// clone-spec A 3: 大会一覧。/ はここへリダイレクトする(next.config.ts)
export default async function TournamentListPage({ searchParams }: Props) {
  const lang = parseLang((await searchParams).lang)
  const data = await getTournaments()
  return (
    <>
      <title>{tournamentListTitle()}</title>
      <SiteFrame variant={{ kind: 'list' }}>
        <TournamentListView data={data} lang={lang} />
      </SiteFrame>
    </>
  )
}
