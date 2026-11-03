import { connection } from 'next/server'
import { japaneseDate } from '@/admin/tournamentList.utils'
import { getTournaments } from '@/api/public'
import type { TournamentListResponse } from '@/api/public.types'
import { SiteShell } from '@/components/site/SiteShell'

// サイトの入口。開催中とこれからの大会を並べ、押すと観戦アプリ(/live/{id})を開く。終わった大会は
// たたんで下に置く。一覧に載せない設定の大会は api が返さない。

type Section = TournamentListResponse['sections'][number]

const sectionNames: Record<Section['section'], string> = {
  in_progress: '開催中',
  entry_open: 'エントリー受付中',
  entry_closed: 'これからの大会',
  finished: '終わった大会',
}

function TournamentLinks({ section }: { section: Section }) {
  return (
    <ul className="site-list">
      {section.tournaments.map((tournament) => (
        <li key={tournament.id}>
          <a className="site-card" href={`/live/${tournament.id}`}>
            <span className="site-card-name">{tournament.name.ja}</span>
            <span className="site-card-meta">
              {tournament.startDate ? japaneseDate(tournament.startDate) : '日程未定'}
              {tournament.teamCount > 0 ? ` ・ ${tournament.teamCount}チーム` : ''}
            </span>
          </a>
        </li>
      ))}
    </ul>
  )
}

export default async function HomePage() {
  // 要求ごとに描く。ビルド時に描くと一覧がその時点で固まり、しかも本番のビルドでは api を呼ぶ
  // BACKEND_INTERNAL_URL(Vercel Services の binding)が無くて落ちる
  await connection()
  const { sections } = await getTournaments()
  const current = sections.filter(
    (section) => section.section !== 'finished' && section.tournaments.length > 0,
  )
  const finished = sections.find((section) => section.section === 'finished')
  return (
    <SiteShell>
      <title>SCORE VIEW</title>
      {current.length === 0 ? (
        <p className="site-empty">いま開催中とこれからの大会はありません。</p>
      ) : null}
      {current.map((section) => (
        <section key={section.section} className="site-section">
          <h1 className={section.section === 'in_progress' ? 'site-heading now' : 'site-heading'}>
            {sectionNames[section.section]}
          </h1>
          <TournamentLinks section={section} />
        </section>
      ))}
      {finished && finished.tournaments.length > 0 ? (
        <details className="site-section site-finished">
          <summary className="site-heading">{sectionNames.finished}</summary>
          <TournamentLinks section={finished} />
        </details>
      ) : null}
    </SiteShell>
  )
}
