import { divisionKindOptions, tournamentStatusOptions } from '@/admin/labels.utils'
import { loadAdmin, requireUser } from '@/admin/session'
import { assignments, divisions, galleries, matches, tournaments, users } from '@/api/admin'
import type { IdPageProps } from '@/app/_lib/pageProps'
import {
  createDivisionAction,
  createGalleryAction,
  deleteTournamentAction,
  putBannersAction,
  setListedAction,
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
import { AssignmentSection } from '@/components/admin/AssignmentSection'

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
      {isAdmin ? (
        <form action={setListedAction} className="adm-publish">
          <Hidden values={{ back, id, listed: tournament.listed ? '' : 'on' }} />
          {tournament.listed ? (
            <>
              <span>公開中(大会一覧に載っています)</span>
              <button type="submit" className="secondary">
                非公開にする
              </button>
            </>
          ) : (
            <>
              <span className="adm-notice">
                準備中です。閲覧ページは URL を知っている人だけが見られ、大会一覧には載りません。
              </span>
              <button type="submit">公開する</button>
            </>
          )}
        </form>
      ) : null}
      <p className="adm-actions">
        <a href={`/scview/${id}`}>閲覧ページを開く</a>
        <a href={`/admin/tournaments/${id}/timetable`}>時間割を印刷</a>
      </p>
      <h2>部門</h2>
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
      {isAdmin ? (
        <AdminSections
          id={id}
          token={token}
          tournament={tournament}
          divisionIds={divisionList.divisions.map((division) => division.id)}
          back={back}
        />
      ) : null}
    </AdminShell>
  )
}

async function AdminSections({
  id,
  token,
  tournament,
  divisionIds,
  back,
}: {
  id: number
  token: string
  tournament: Awaited<ReturnType<typeof tournaments.get>>
  divisionIds: number[]
  back: string
}) {
  const [banners, galleryList, assignmentList, userList, matchLists] = await loadAdmin(() =>
    Promise.all([
      tournaments.banners(token, id),
      galleries.list(token, id),
      assignments.list(token, id),
      users.list(token),
      Promise.all(divisionIds.map((divisionId) => matches.list(token, divisionId))),
    ]),
  )
  const bannerLines = banners.banners
    .map((banner) => [banner.imageUrl, banner.linkUrl ?? ''].join(' ').trim())
    .join('\n')
  return (
    <>
      <h3>部門を作る</h3>
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
        </div>
        <details>
          <summary>詳しい設定</summary>
          <div className="adm-grid">
            <Field label="並び順" name="sortOrder" type="number" defaultValue={0} />
          </div>
        </details>
        <button type="submit">作成</button>
      </form>

      <AssignmentSection
        tournamentId={id}
        back={back}
        assignments={assignmentList.assignments}
        staff={userList.users.filter((user) => user.role === 'staff')}
        matches={matchLists.flatMap((list) => list.matches)}
      />

      <h2>基本情報</h2>
      <form action={updateTournamentAction}>
        <Hidden values={{ back, id }} />
        <div className="adm-grid">
          <Field label="大会名(日本語)" name="nameJa" defaultValue={tournament.name.ja} required />
          <Select
            label="区分"
            name="status"
            options={tournamentStatusOptions}
            defaultValue={tournament.status}
          />
          <Field label="開催日" name="startDate" type="date" defaultValue={tournament.startDate} />
          <Field
            label="不戦勝のスコア(勝ったチーム)"
            name="forfeitWinner"
            type="number"
            defaultValue={tournament.forfeitScore?.winner ?? null}
          />
          <Field
            label="不戦勝のスコア(負けたチーム)"
            name="forfeitLoser"
            type="number"
            defaultValue={tournament.forfeitScore?.loser ?? null}
          />
        </div>
        <p className="adm-muted">
          不戦勝のスコアは大会の規程に合わせて入れてください(例 15 と
          0)。空欄のままだと、記録係が不戦勝を記録するときにスコアの入力を求めます。
        </p>
        <Check label="大会一覧に載せる(公開)" name="listed" defaultChecked={tournament.listed} />
        <TextArea
          label="大会トップのお知らせ欄(日本語。HTML も書けます。保存時に安全な形に直します)"
          name="topHtmlJa"
          defaultValue={tournament.topHtml.ja}
          rows={8}
        />
        <details>
          <summary>詳しい設定(英語の名前、ロゴ、大会メニューの言い方)</summary>
          <div className="adm-grid">
            <Field label="大会名(英語)" name="nameEn" defaultValue={tournament.name.en} />
            <Field label="短い名前" name="shortName" defaultValue={tournament.shortName} />
            <Field label="ロゴ URL" name="logoUrl" defaultValue={tournament.logoUrl} />
            <Select
              label="大会メニューのチーム一覧の名前"
              name="teamListLabel"
              options={[
                { value: 'teams', label: 'チーム一覧' },
                { value: 'entries', label: 'エントリー一覧' },
              ]}
              defaultValue={tournament.labels.teamList}
            />
            <Select
              label="大会メニューの組の成績の名前"
              name="groupLabel"
              options={[
                { value: 'groups', label: 'グループ成績' },
                { value: 'results', label: '成績' },
              ]}
              defaultValue={tournament.labels.groups}
            />
          </div>
          <Check
            label="大会メニューにスケジュールと部門の成績を出す"
            name="showSchedule"
            defaultChecked={tournament.showSchedule}
          />
          <TextArea
            label="大会トップのお知らせ欄(英語)"
            name="topHtmlEn"
            defaultValue={tournament.topHtml.en}
            rows={8}
          />
        </details>
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
      <p className="adm-muted">部門、チーム、試合、ギャラリーも全部消えます。</p>
      <DeleteForm action={deleteTournamentAction} values={{ back, id }} label="この大会" />
    </>
  )
}
