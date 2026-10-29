import { loadAdmin, requireUser } from '@/admin/session'
import { tournaments } from '@/api/admin'
import { copyTournamentAction, createTournamentAction } from '@/app/admin/_actions/tournament'
import type { SearchProps } from '@/app/admin/_lib/page'
import { AdminShell, Check, Field, Hidden, readNotice } from '@/components/admin/AdminParts'

// 大会を作る(大会一覧の「大会を作る」から)。前回の大会をコピーするか、白紙から作る
export default async function NewTournamentPage({ searchParams }: SearchProps) {
  const notice = readNotice(await searchParams)
  const { user, token } = await requireUser('admin')
  const list = await loadAdmin(() => tournaments.list(token))
  const back = '/admin/tournaments/new'
  return (
    <AdminShell user={user} title="大会を作る" notice={notice}>
      {list.tournaments.length > 0 ? (
        <section>
          <h2>前回の大会をコピーして作る</h2>
          <p className="adm-muted">
            部門、大会の形、不戦勝の規定スコア、お知らせ欄、バナー、時間割の設定を引き継ぎます。試合と結果は引き継ぎません。
          </p>
          <form action={copyTournamentAction}>
            <Hidden values={{ back }} />
            <div className="adm-grid">
              <label className="adm-field">
                <span>コピー元の大会</span>
                <select name="sourceId" required>
                  {list.tournaments.map((tournament) => (
                    <option key={tournament.id} value={tournament.id}>
                      {tournament.name.ja}
                      {tournament.startDate ? `(${tournament.startDate})` : ''}
                    </option>
                  ))}
                </select>
              </label>
              <Field label="新しい大会の開催日" name="startDate" type="date" required />
              <Field label="新しい大会の名前(空ならコピー元と同じ)" name="nameJa" />
            </div>
            <Check label="チームと選手も引き継ぐ" name="withTeams" />
            <button type="submit">コピーして作る</button>
          </form>
        </section>
      ) : null}
      <section>
        <h2>白紙から作る</h2>
        <p className="adm-muted">
          作った大会は準備中(公開の一覧に載りません)。準備ができたら大会の設定の「公開する」を押してください。
        </p>
        <form action={createTournamentAction}>
          <Hidden
            values={{
              back,
              status: 'entry_open',
              teamListLabel: 'teams',
              groupLabel: 'groups',
              showSchedule: 'on',
            }}
          />
          <div className="adm-grid">
            <Field label="大会名(日本語)" name="nameJa" required />
            <Field label="大会名(英語)" name="nameEn" />
            <Field label="開催日" name="startDate" type="date" />
          </div>
          <button type="submit">作成</button>
        </form>
      </section>
    </AdminShell>
  )
}
