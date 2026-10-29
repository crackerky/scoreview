import { tournamentStatusOptions } from '@/admin/labels.utils'
import { loadAdmin, requireUser } from '@/admin/session'
import { tournaments } from '@/api/admin'
import type { IdPageProps } from '@/app/_lib/pageProps'
import {
  deleteTournamentAction,
  setListedAction,
  updateTournamentAction,
} from '@/app/admin/_actions/tournament'
import { readAdminPage } from '@/app/admin/_lib/page'
import {
  AdminShell,
  Check,
  DangerZone,
  DeleteForm,
  Field,
  Hidden,
  Select,
  TextArea,
} from '@/components/admin/AdminParts'

// 大会の設定(大会ホームの入口から)。公開の切り替え、基本情報、いちばん下に削除
export default async function TournamentSettingsPage(props: IdPageProps) {
  const { id, notice } = await readAdminPage(props)
  const { user, token } = await requireUser('admin')
  const tournament = await loadAdmin(() => tournaments.get(token, id))
  const back = `/admin/tournaments/${id}/settings`
  const crumbs = [{ href: `/admin/tournaments/${id}`, label: tournament.name.ja }]
  return (
    <AdminShell user={user} title="大会の設定" notice={notice} crumbs={crumbs}>
      <h2>公開</h2>
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

      <h2>基本情報</h2>
      <form action={updateTournamentAction}>
        <Hidden values={{ back, id }} />
        <div className="adm-grid">
          <Field label="大会名(日本語)" name="nameJa" defaultValue={tournament.name.ja} required />
          <Select
            label="エントリーの状態"
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

      <DangerZone>
        <p className="adm-muted">部門、チーム、試合、ギャラリーも全部消えます。</p>
        <DeleteForm action={deleteTournamentAction} values={{ back, id }} label="この大会" />
      </DangerZone>
    </AdminShell>
  )
}
