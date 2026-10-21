import { loadAdmin, requireUser } from '@/admin/session'
import { sideLabelsOf } from '@/admin/timetable.utils'
import { divisions, matches } from '@/api/admin'
import { formats } from '@/api/format'
import type { IdPageProps } from '@/app/_lib/pageProps'
import { readAdminPage } from '@/app/admin/_lib/page'
import { PrintButton } from '@/components/admin/PrintButton'
import { ScoreSheet } from '@/components/admin/ScoreSheet'
import '@/styles/print-sheet.css'

// ディビジョンの全試合のスコアシートを 1 試合 1 枚で続けて印刷する(大会の前日に各コートの分を
// まとめて刷るため)。?date=YYYY-MM-DD で日付を絞る。
export default async function DivisionSheetsPage(props: IdPageProps) {
  const { id } = await readAdminPage(props)
  const query = await props.searchParams
  const date = typeof query.date === 'string' && query.date !== '' ? query.date : null
  const { token } = await requireUser('any')
  const [division, list, formatList] = await loadAdmin(() =>
    Promise.all([
      divisions.get(token, id),
      matches.list(token, id),
      formats.matchesIfFormatted(token, id),
    ]),
  )
  const labels = sideLabelsOf(formatList?.matches ?? [])
  const dates = [...new Set(list.matches.flatMap((match) => (match.date ? [match.date] : [])))]
  const targets = list.matches.filter((match) => date === null || match.date === date)
  const details = await loadAdmin(() =>
    Promise.all(targets.map((match) => matches.detail(token, match.id))),
  )
  return (
    <div className="pr">
      <title>{`${division.name.ja} スコアシート`}</title>
      <div className="pr-toolbar">
        <a href={`/admin/divisions/${id}`}>← 部門に戻る</a>
        <PrintButton />
        <span>
          {details.length} 試合 / A4 縦 1 試合 1 枚{dates.length > 1 ? ' / 日付: ' : ''}
        </span>
        {dates.length > 1
          ? [null, ...dates].map((value) => (
              <a key={value ?? 'all'} href={`?date=${value ?? ''}`}>
                {value ?? '全日程'}
              </a>
            ))
          : null}
      </div>
      {details.map((detail) => (
        <ScoreSheet key={detail.match.id} detail={detail} labels={labels.get(detail.match.id)} />
      ))}
    </div>
  )
}
