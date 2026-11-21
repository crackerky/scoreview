import { loadAdmin, requireUser } from '@/admin/session'
import { divisions, tournaments } from '@/api/admin'
import { entries } from '@/api/entry'
import type { IdPageProps } from '@/app/_lib/pageProps'
import { createEntryAction } from '@/app/admin/_actions/entry'
import { readAdminPage } from '@/app/admin/_lib/page'
import { AdminShell, Field, Hidden, Select, TextArea } from '@/components/admin/AdminParts'
import { EntryAnswerField } from '@/components/entry/EntryAnswerField'
import { entryPaths } from '@/entry/entry.paths'

// 電話や紙で受けた申込を、運営が代わりに入れる。LINE からの申込と同じ受付を通るので、同じ部門に同じ
// チーム名の申込を二重に入れることはない。入れた申込は「申込中」になり、承認すると部門のチームになる。
export default async function NewEntryPage(props: IdPageProps) {
  const { id, notice } = await readAdminPage(props)
  const { user, token } = await requireUser('admin')
  const [tournament, divisionList, questionList] = await loadAdmin(() =>
    Promise.all([
      tournaments.get(token, id),
      divisions.list(token, id),
      entries.questions(token, id),
    ]),
  )
  const teamQuestions = questionList.questions.filter((question) => question.scope === 'team')
  const formId = 'entry-form'
  const crumbs = [
    { href: `/admin/tournaments/${id}`, label: tournament.name.ja },
    { href: entryPaths.list(id), label: '申込' },
  ]
  if (divisionList.divisions.length === 0) {
    return (
      <AdminShell user={user} title="申込を代わりに入れる" notice={notice} crumbs={crumbs}>
        <p className="adm-muted">
          部門がまだありません。<a href={`/admin/tournaments/${id}/divisions/new`}>部門を作って</a>
          から申込を入れてください。
        </p>
      </AdminShell>
    )
  }
  return (
    <AdminShell
      user={user}
      title="申込を代わりに入れる"
      kicker="電話や紙で受けた申込"
      notice={notice}
      crumbs={crumbs}
      primary={{ label: '申込を入れる', form: formId }}
    >
      <form id={formId} action={createEntryAction} className="adm-stack">
        <Hidden values={{ back: entryPaths.create(id), tournamentId: id }} />
        <h2 className="adm-section-title">チーム</h2>
        <div className="adm-grid">
          <Select
            label="部門"
            name="divisionId"
            options={divisionList.divisions.map((division) => ({
              value: String(division.id),
              label: division.name.ja,
            }))}
          />
          <Field label="チーム名(正式)" name="fullName" required />
          <Field label="チーム名(短縮。日程に出る)" name="shortName" required />
        </div>
        <h2 className="adm-section-title">代表</h2>
        <div className="adm-grid">
          <Field label="代表の名前(フルネーム)" name="repName" required />
          <Field label="電話番号" name="repPhone" />
          <Field label="メールアドレス" name="repEmail" type="email" />
        </div>
        <h2 className="adm-section-title">メンバー</h2>
        <TextArea
          label="1 行に 1 人。背番号があれば名前の前に(例「7 山田 太郎」)"
          name="members"
          rows={10}
        />
        {teamQuestions.length > 0 ? (
          <>
            <h2 className="adm-section-title">質問</h2>
            {teamQuestions.map((question) => (
              <EntryAnswerField key={question.id} question={question} />
            ))}
          </>
        ) : null}
      </form>
    </AdminShell>
  )
}
