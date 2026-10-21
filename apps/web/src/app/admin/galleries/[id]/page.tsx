import { loadAdmin, requireUser } from '@/admin/session'
import { galleries } from '@/api/admin'
import type { IdPageProps } from '@/app/_lib/pageProps'
import { deleteGalleryAction, updateGalleryAction } from '@/app/admin/_actions/gallery'
import { readAdminPage } from '@/app/admin/_lib/page'
import { AdminShell, DeleteForm, Field, Hidden, TextArea } from '@/components/admin/AdminParts'

export default async function AdminGalleryPage(props: IdPageProps) {
  const { id, notice } = await readAdminPage(props)
  const { user, token } = await requireUser('admin')
  const gallery = await loadAdmin(() => galleries.get(token, id))
  const back = `/admin/galleries/${id}`
  const lines = gallery.photos.map((photo) => `${photo.thumbUrl} ${photo.fullUrl}`).join('\n')
  const crumbs = [{ href: `/admin/tournaments/${gallery.tournamentId}`, label: '大会' }]
  return (
    <AdminShell user={user} title={gallery.name} notice={notice} crumbs={crumbs}>
      <p>
        <a href={`/pscore/scv_tgallery/open/${id}`}>閲覧ページを開く</a>
      </p>
      <form action={updateGalleryAction}>
        <Hidden values={{ back, id }} />
        <div className="adm-grid">
          <Field label="ギャラリー名" name="name" defaultValue={gallery.name} required />
          <Field label="並び順" name="sortOrder" type="number" defaultValue={gallery.sortOrder} />
        </div>
        <TextArea
          label="写真。1 行に「サムネイル URL 拡大画像 URL」を空白区切りで(表示順)"
          name="photos"
          defaultValue={lines}
          rows={10}
        />
        <button type="submit">保存</button>
      </form>
      <DeleteForm
        action={deleteGalleryAction}
        values={{ back, id, tournamentId: gallery.tournamentId }}
        label="このギャラリー"
      />
    </AdminShell>
  )
}
