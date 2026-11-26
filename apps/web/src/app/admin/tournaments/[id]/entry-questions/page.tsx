import { loadAdmin, requireUser } from '@/admin/session'
import { tournaments } from '@/api/admin'
import { entries } from '@/api/entry'
import type { EntryQuestion } from '@/api/entry.types'
import type { IdPageProps } from '@/app/_lib/pageProps'
import { saveEntryQuestionsAction } from '@/app/admin/_actions/entry'
import { readAdminPage } from '@/app/admin/_lib/page'
import { AdminShell } from '@/components/admin/AdminParts'
import { EntryQuestionsForm } from '@/components/admin/EntryQuestionsForm'
import { entryPaths } from '@/entry/entry.paths'

// 申込で聞く質問(Lイベントの「アンケート」と参加フォームの追加項目にあたる)。部門、チーム名、代表の
// 連絡先、メンバーの名前と背番号はどの大会でも聞くので、ここには大会ごとに足す質問だけを置く。
// 1 行が 1 問。下の空の行に書くと足し、文言を消すか「消す」に印を付けると消す。順は「順」の数で並べ替える。

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
