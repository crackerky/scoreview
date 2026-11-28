import { loadAdmin, requireUser } from '@/admin/session'
import { tournaments } from '@/api/admin'
import { entries } from '@/api/entry'
import type { EntryQuestion } from '@/api/entry.types'
import type { IdPageProps } from '@/app/_lib/pageProps'
import { saveEntryQuestionsAction, savePersonalAskAction } from '@/app/admin/_actions/entry'
import { readAdminPage } from '@/app/admin/_lib/page'
import { AdminShell, Check, Hidden } from '@/components/admin/AdminParts'
import { EntryQuestionsForm } from '@/components/admin/EntryQuestionsForm'
import { entryPaths } from '@/entry/entry.paths'

// 申込で聞く質問(Lイベントの「アンケート」と参加フォームの追加項目にあたる)。部門、チーム名、代表の
// 連絡先、メンバーの名前と背番号はどの大会でも聞くので、ここには大会ごとに足す質問だけを置く。
// 1 行が 1 問。下の空の行に書くと足し、文言を消すか「消す」に印を付けると消す。順は「順」の数で並べ替える。
// 上の「メンバー本人に聞く項目」は、Lイベントの顧客情報入力項目にあたる性別と生年月日。質問と違い、答えを
// 参加者の台帳に残して次の大会で前もって入れ、性別は選手の性別にもする。

/** 1 回に足せる空の行の数 */
const blankRows = 3

export default async function EntryQuestionsPage(props: IdPageProps) {
  const { id, notice } = await readAdminPage(props)
  const { user, token } = await requireUser('admin')
  const [tournament, list] = await loadAdmin(() =>
    Promise.all([tournaments.get(token, id), entries.questions(token, id)]),
  )
  const rows: Array<EntryQuestion | null> = [
    ...list.questions,
    ...Array.from({ length: blankRows }, () => null),
  ]
  const formId = 'entry-questions-form'
  return (
    <AdminShell
      user={user}
      title="申込で聞く質問"
      kicker="チーム名、代表の連絡先、メンバーの名前と背番号は、どの大会でも聞きます"
      notice={notice}
      crumbs={[
        { href: `/admin/tournaments/${id}`, label: tournament.name.ja },
        { href: entryPaths.list(id), label: '申込' },
      ]}
      primary={{ label: '保存する', form: formId }}
    >
      <h2>メンバー本人に聞く項目</h2>
      <p className="adm-muted">
        印を付けた項目は、メンバーが LINE で登録するときに必ず答えます。答えは運営の画面と CSV
        にだけ出て、次の大会では前回の答えが最初から入ります。性別は、参加を決めたときに選手の性別にもなります。
      </p>
      <form action={savePersonalAskAction} className="adm-actions">
        <Hidden values={{ back: entryPaths.questions(id), tournamentId: id }} />
        <Check label="性別" name="gender" defaultChecked={list.personal.gender} />
        <Check label="生年月日" name="birthdate" defaultChecked={list.personal.birthdate} />
        <button type="submit" className="secondary">
          聞く項目を保存
        </button>
      </form>
      <h2>質問</h2>
      <EntryQuestionsForm
        formId={formId}
        back={entryPaths.questions(id)}
        tournamentId={id}
        rows={rows}
        action={saveEntryQuestionsAction}
      />
    </AdminShell>
  )
}
