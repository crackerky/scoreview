import { divisionKindOptions, tournamentStatusOptions } from '@/admin/labels.utils'
import { loadAdmin, requireUser } from '@/admin/session'
import { divisions, galleries, tournaments } from '@/api/admin'
import type { IdPageProps } from '@/app/_lib/pageProps'
import {
  createDivisionAction,
  createGalleryAction,
  deleteTournamentAction,
  putBannersAction,
  updateTournamentAction,
} from '@/app/admin/_actions/tournament'
import { readAdminPage } from '@/app/admin/_lib/page'
import {
  AdminShell,
  Check,
  DeleteForm,
  Field,
  Hidden,
  Select,
  TextArea,
} from '@/components/admin/AdminParts'

export default async function AdminTournamentPage(props: IdPageProps) {
  const { id, notice } = await readAdminPage(props)
  const { user, token } = await requireUser('any')
  const [tournament, divisionList] = await loadAdmin(() =>
    Promise.all([tournaments.get(token, id), divisions.list(token, id)]),
  )
  const back = `/admin/tournaments/${id}`
  const isAdmin = user.role === 'admin'
  return (
    <AdminShell user={user} title={tournament.name.ja} notice={notice}>
      <p>
        <a href={`/scview/${id}`}>閲覧ページを開く</a>
      </p>
      <h2>ディビジョン</h2>
      <ul className="adm-list">
        {divisionList.divisions.map((division) => (
          <li key={division.id}>
            <a href={`/admin/divisions/${division.id}`}>
              {division.name.ja}
              {division.name.en ? ` / ${division.name.en}` : ''}
            </a>
          </li>
        ))}
      </ul>
      {isAdmin ? <AdminSections id={id} token={token} tournament={tournament} back={back} /> : null}
    </AdminShell>
  )
}

async function AdminSections({
  id,
  token,
  tournament,
  back,
}: {
  id: number
  token: string
  tournament: Awaited<ReturnType<typeof tournaments.get>>
  back: string
}) {
  const [banners, galleryList] = await loadAdmin(() =>
    Promise.all([tournaments.banners(token, id), galleries.list(token, id)]),
  )
  const bannerLines = banners.banners
    .map((banner) => [banner.imageUrl, banner.linkUrl ?? ''].join(' ').trim())
    .join('\n')
  return (
    <>
      <h3>ディビジョンを作る</h3>
      <form action={createDivisionAction}>
        <Hidden
          values={{
            back,
            tournamentId: id,
            scheduleEnabled: 'on',
            entryKind: 'team',
            statsKind: 'ultimate',
          }}
        />
        <div className="adm-grid">
          <Field label="名前(日本語)" name="nameJa" required />
          <Field label="名前(英語)" name="nameEn" />
          <Select
            label="区分(日程の色)"
            name="kind"
            options={divisionKindOptions}
            defaultValue="mixed"
          />
          <Field label="並び順" name="sortOrder" type="number" defaultValue={0} />
        </div>
        <button type="submit">作成</button>
      </form>

      <h2>基本情報</h2>
      <form action={updateTournamentAction}>
        <Hidden values={{ back, id }} />
        <div className="adm-grid">
          <Field label="大会名(日本語)" name="nameJa" defaultValue={tournament.name.ja} required />
          <Field label="大会名(英語)" name="nameEn" defaultValue={tournament.name.en} />
          <Field label="短い名前" name="shortName" defaultValue={tournament.shortName} />
          <Select
            label="区分"
            name="status"
            options={tournamentStatusOptions}
            defaultValue={tournament.status}
          />
          <Field label="開催日" name="startDate" type="date" defaultValue={tournament.startDate} />
          <Field label="ロゴ URL" name="logoUrl" defaultValue={tournament.logoUrl} />
          <Select
            label="ナビのチーム一覧の名前"
            name="teamListLabel"
            options={[
              { value: 'teams', label: 'チーム一覧' },
              { value: 'entries', label: 'エントリー一覧' },
            ]}
            defaultValue={tournament.labels.teamList}
          />
          <Select
            label="ナビのグループの名前"
            name="groupLabel"
            options={[
              { value: 'groups', label: 'グループ成績' },
              { value: 'results', label: '成績' },
            ]}
            defaultValue={tournament.labels.groups}
          />
        </div>
        <Check label="大会一覧に載せる" name="listed" defaultChecked={tournament.listed} />
        <Check
          label="ナビにスケジュールとディビジョン成績を出す"
          name="showSchedule"
          defaultChecked={tournament.showSchedule}
        />
        <TextArea
          label="自由記述(日本語。HTML。保存時に無害化します)"
          name="topHtmlJa"
          defaultValue={tournament.topHtml.ja}
          rows={8}
        />
        <TextArea
          label="自由記述(英語。HTML)"
          name="topHtmlEn"
          defaultValue={tournament.topHtml.en}
          rows={8}
        />
        <button type="submit">保存</button>
      </form>

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

      <h2>ギャラリー</h2>
      <ul className="adm-list">
        {galleryList.galleries.map((gallery) => (
          <li key={gallery.id}>
            <a href={`/admin/galleries/${gallery.id}`}>
              {gallery.name}(写真 {gallery.photos.length} 枚)
            </a>
          </li>
        ))}
      </ul>
      <form action={createGalleryAction}>
        <Hidden values={{ back, tournamentId: id }} />
        <div className="adm-grid">
          <Field label="ギャラリー名" name="name" required />
          <Field label="並び順" name="sortOrder" type="number" defaultValue={0} />
        </div>
        <button type="submit">ギャラリーを作る</button>
      </form>

      <h2>大会の削除</h2>
      <p className="adm-muted">ディビジョン、チーム、試合、ギャラリーも全部消えます。</p>
      <DeleteForm action={deleteTournamentAction} values={{ back, id }} label="この大会" />
    </>
  )
}
