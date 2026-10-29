'use server'

import { field, idField, inputError, runActionTo } from '@/admin/action'
import { withNotice } from '@/admin/form.utils'
import { parseReplacements, type SourceDivision } from '@/admin/newTournament.utils'
import { requireUser } from '@/admin/session'
import { divisions, teams, tournaments } from '@/api/admin'
import { tournamentFormSchema } from '@/api/admin.types'

// 大会を作る 3 ステップ(Claude Design「Score View Admin」)の Server Actions。コピー元のチームを読み、
// 最後の「公開する」か「下書きのまま保存」で、コピー(または白紙)、チームの入れ替え、公開をまとめて行う。

/** コピー元の大会の部門とチーム(2 ステップ目の一覧)。並びはコピー先と同じ(並び順、ID の順) */
export async function loadSourceTeams(sourceId: number): Promise<SourceDivision[]> {
  const { token } = await requireUser('admin')
  const list = await divisions.list(token, sourceId)
  return Promise.all(
    list.divisions.map(async (division) => ({
      name: division.name.ja,
      teams: (await teams.list(token, division.id)).teams.map((team) => team.fullName.ja),
    })),
  )
}

export async function createFromWizardAction(formData: FormData): Promise<never> {
  return runActionTo(formData, async ({ token, form }) => {
    const name = field(form, 'nameJa').trim()
    const startDate = field(form, 'startDate')
    if (name === '') {
      throw inputError('大会名を入れてください')
    }
    if (!/^\d{4}-\d{2}-\d{2}$/.test(startDate)) {
      throw inputError('日付を入れてください')
    }
    const publish = field(form, 'publish') === '1'
    const created =
      field(form, 'mode') === 'copy'
        ? await copyWithReplacements(token, idField(form, 'sourceId'), name, startDate, form)
        : await tournaments.create(
            token,
            tournamentFormSchema.parse({
              nameJa: name,
              startDate,
              status: 'entry_open',
              teamListLabel: 'teams',
              groupLabel: 'groups',
              showSchedule: 'on',
            }),
          )
    if (publish) {
      await tournaments.update(token, created.id, { listed: true })
    }
    return withNotice(
      `/admin/tournaments/${created.id}`,
      'ok',
      publish
        ? '公開しました。大会一覧に「エントリー受付中」で載ります'
        : '下書きのまま保存しました',
    )
  })
}

/** チームごとコピーし、出ないチームを新しい名前に入れ替える(チーム数と組分けは変えない) */
async function copyWithReplacements(
  token: string,
  sourceId: number,
  name: string,
  startDate: string,
  form: Record<string, string | string[]>,
) {
  const replacements = parseReplacements(field(form, 'replacements'))
  const created = await tournaments.copy(token, sourceId, {
    name: { ja: name, en: '' },
    startDate,
    withTeams: true,
  })
  if (replacements.length === 0) {
    return created
  }
  const copied = await divisions.list(token, created.id)
  for (const replacement of replacements) {
    const division = copied.divisions[replacement.division]
    if (division === undefined) {
      continue
    }
    const team = (await teams.list(token, division.id)).teams[replacement.team]
    if (team === undefined) {
      continue
    }
    await teams.update(token, team.id, {
      fullName: { ja: replacement.name, en: '' },
      shortName: { ja: replacement.name, en: '' },
    })
  }
  return created
}
