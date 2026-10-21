import { loadAdmin, requireUser } from '@/admin/session'
import { divisions, matches, tournaments } from '@/api/admin'
import type { IdPageProps } from '@/app/_lib/pageProps'
import { readAdminPage } from '@/app/admin/_lib/page'
import { PrintButton } from '@/components/admin/PrintButton'
import { Timetable } from '@/components/admin/Timetable'
import '@/styles/print-sheet.css'

// 大会の時間割(コート × 時刻)。全ディビジョンの試合を 1 日 1 枚(A4 横)に並べる。
export default async function TimetablePage(props: IdPageProps) {
  const { id } = await readAdminPage(props)
  const { token } = await requireUser('any')
  const [tournament, divisionList] = await loadAdmin(() =>
    Promise.all([tournaments.get(token, id), divisions.list(token, id)]),
  )
  const lists = await loadAdmin(() =>
    Promise.all(divisionList.divisions.map((division) => matches.list(token, division.id))),
  )
  const all = divisionList.divisions.flatMap((division, index) =>
    (lists[index]?.matches ?? []).map((match) => ({ ...match, divisionName: division.name.ja })),
  )
  return (
    <div className="pr">
      <title>{`${tournament.name.ja} 時間割`}</title>
      <div className="pr-toolbar">
        <a href={`/admin/tournaments/${id}`}>← 大会に戻る</a>
        <PrintButton />
        <span>A4 横、1 日 1 枚で印刷します。</span>
      </div>
      <Timetable title={tournament.name.ja} matches={all} />
    </div>
  )
}
