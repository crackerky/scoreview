import type { IdPageProps } from '@/app/_lib/pageProps'
import { TournamentTopPage } from '@/app/_lib/TournamentTopPage'

// clone-spec A 4: 大会一覧からのリンク先。/pscore/scv_top/open/{id} と同じ内容
export default function ScviewPage(props: IdPageProps) {
  return <TournamentTopPage {...props} />
}
