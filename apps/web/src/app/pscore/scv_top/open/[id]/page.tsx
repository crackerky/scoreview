import type { IdPageProps } from '@/app/_lib/pageProps'
import { TournamentTopPage } from '@/app/_lib/TournamentTopPage'

// clone-spec A 4: 大会ナビの「大会トップ」のリンク先。/scview/{id} と同じ内容
export default function ScvTopPage(props: IdPageProps) {
  return <TournamentTopPage {...props} />
}
