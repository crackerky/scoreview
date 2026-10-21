import { barStyleOptions } from '@/admin/labels.utils'
import { loadAdmin, requireUser } from '@/admin/session'
import { players, teams } from '@/api/admin'
import type { AdminPlayer } from '@/api/admin.types'
import type { IdPageProps } from '@/app/_lib/pageProps'
import {
  createPlayerAction,
  deletePlayerAction,
  deleteTeamAction,
  updatePlayerAction,
  updateTeamAction,
} from '@/app/admin/_actions/team'
import { readAdminPage } from '@/app/admin/_lib/page'
import {
  AdminShell,
  DeleteForm,
  Field,
  Hidden,
  Select,
  TextArea,
} from '@/components/admin/AdminParts'

const genderOptions = [
  { value: '', label: '未登録' },
  { value: 'M', label: '男性' },
  { value: 'F', label: '女性' },
]

export default async function AdminTeamPage(props: IdPageProps) {
  const { id, notice } = await readAdminPage(props)
  const { user, token } = await requireUser('admin')
  const [team, playerList] = await loadAdmin(() =>
    Promise.all([teams.get(token, id), players.list(token, id)]),
  )
  const back = `/admin/teams/${id}`
  const crumbs = [
    { href: `/admin/tournaments/${team.tournamentId}`, label: '大会' },
    { href: `/admin/divisions/${team.divisionId}`, label: 'ディビジョン' },
  ]
  return (
    <AdminShell user={user} title={team.fullName.ja} notice={notice} crumbs={crumbs}>
      <p>
        <a href={`/pscore/scv_team/open/${id}`}>閲覧ページを開く</a>
      </p>
      <h2>選手とスタッフ</h2>
      {playerList.players.map((player) => (
        <PlayerForm key={player.id} player={player} back={back} />
      ))}
      <form action={createPlayerAction}>
        <fieldset>
          <legend>選手を足す</legend>
          <Hidden values={{ back, teamId: id }} />
          <PlayerFields />
          <button type="submit">追加</button>
        </fieldset>
      </form>

      <h2>チームの設定</h2>
      <form action={updateTeamAction}>
        <Hidden values={{ back, id }} />
        <div className="adm-grid">
          <Field
            label="正式名(日本語)"
            name="fullNameJa"
            defaultValue={team.fullName.ja}
            required
          />
          <Field
            label="短縮名(日本語)"
            name="shortNameJa"
            defaultValue={team.shortName.ja}
            required
          />
          <Field label="正式名(英語)" name="fullNameEn" defaultValue={team.fullName.en} />
          <Field label="短縮名(英語)" name="shortNameEn" defaultValue={team.shortName.en} />
          <Select
            label="色帯"
            name="barStyle"
            options={barStyleOptions}
            defaultValue={team.barStyle}
          />
          <Field label="並び順" name="sortOrder" type="number" defaultValue={team.sortOrder} />
          <Field label="最終順位" name="finalRank" type="number" defaultValue={team.finalRank} />
        </div>
        <TextArea
          label="コメント(日本語)"
          name="commentJa"
          defaultValue={team.comment.ja}
          rows={3}
        />
        <TextArea label="コメント(英語)" name="commentEn" defaultValue={team.comment.en} rows={3} />
        <button type="submit">保存</button>
      </form>
      <DeleteForm
        action={deleteTeamAction}
        values={{ back, id, divisionId: team.divisionId }}
        label="このチーム"
      />
    </AdminShell>
  )
}

function PlayerFields({ player }: { player?: AdminPlayer }) {
  return (
    <div className="adm-grid">
      <Field label="背番号" name="number" defaultValue={player?.number} />
      <Field label="名前(日本語)" name="nameJa" defaultValue={player?.name.ja} required />
      <Field label="名前(英語)" name="nameEn" defaultValue={player?.name.en} />
      <Select
        label="性別"
        name="gender"
        options={genderOptions}
        defaultValue={player?.gender ?? ''}
      />
      <Field
        label="役割(スタッフのとき。例 マネージャ)"
        name="roleJa"
        defaultValue={player?.role?.ja}
      />
      <Field label="役割(英語)" name="roleEn" defaultValue={player?.role?.en} />
    </div>
  )
}

function PlayerForm({ player, back }: { player: AdminPlayer; back: string }) {
  return (
    <fieldset>
      <legend>
        {player.number ? `#${player.number} ` : ''}
        {player.name.ja}
        {player.role ? `(${player.role.ja})` : ''}
      </legend>
      <form action={updatePlayerAction}>
        <Hidden values={{ back, id: player.id }} />
        <PlayerFields player={player} />
        <button type="submit">保存</button>
      </form>
      <DeleteForm action={deletePlayerAction} values={{ back, id: player.id }} label="この選手" />
    </fieldset>
  )
}
