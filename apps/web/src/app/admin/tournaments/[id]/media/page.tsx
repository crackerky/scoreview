import { loadAdmin, requireUser } from '@/admin/session'
import { galleries, tournaments } from '@/api/admin'
import type { IdPageProps } from '@/app/_lib/pageProps'
import { createGalleryAction, putBannersAction } from '@/app/admin/_actions/tournament'
import { readAdminPage } from '@/app/admin/_lib/page'
import { AdminShell, Field, Hidden, TextArea } from '@/components/admin/AdminParts'

// 写真とバナー(大会ホームの入口から)。サイドバーのバナーと、大会のギャラリー
export default async function TournamentMediaPage(props: IdPageProps) {
  const { id, notice } = await readAdminPage(props)
  const { user, token } = await requireUser('admin')
  const [tournament, banners, galleryList] = await loadAdmin(() =>
    Promise.all([
      tournaments.get(token, id),
      tournaments.banners(token, id),
      galleries.list(token, id),
    ]),
  )
  const back = `/admin/tournaments/${id}/media`
  const crumbs = [{ href: `/admin/tournaments/${id}`, label: tournament.name.ja }]
  const bannerLines = banners.banners
    .map((banner) => [banner.imageUrl, banner.linkUrl ?? ''].join(' ').trim())
    .join('\n')
  return (
    <AdminShell user={user} title="写真とバナー" notice={notice} crumbs={crumbs}>
      <h2>ギャラリー</h2>
      {galleryList.galleries.length > 0 ? (
        <ul className="adm-rows">
          {galleryList.galleries.map((gallery) => (
            <li key={gallery.id}>
              <a className="adm-row" href={`/admin/galleries/${gallery.id}`}>
                <span className="adm-row-main">
                  <span className="adm-row-title">{gallery.name}</span>
                  <span className="adm-row-meta">写真 {gallery.photos.length} 枚</span>
                </span>
                <span className="adm-chevron" aria-hidden="true">
                  ›
                </span>
              </a>
            </li>
          ))}
        </ul>
      ) : (
        <p className="adm-muted">ギャラリーはまだありません。</p>
      )}
      <details>
        <summary>ギャラリーを作る</summary>
        <form action={createGalleryAction}>
          <Hidden values={{ back, tournamentId: id }} />
          <div className="adm-grid">
            <Field label="ギャラリー名" name="name" required />
            <Field label="並び順" name="sortOrder" type="number" defaultValue={0} />
          </div>
          <button type="submit">ギャラリーを作る</button>
        </form>
      </details>

      <h2>バナー</h2>
      <form action={putBannersAction}>
        <Hidden values={{ back, id }} />
        <TextArea
          label="1 行に「画像 URL リンク URL(省略可)」を空白区切りで(表示順)"
          name="banners"
          defaultValue={bannerLines}
        />
        <button type="submit">保存</button>
      </form>
    </AdminShell>
  )
}
