'use server'

import { redirect } from 'next/navigation'
import { formToObject } from '@/admin/form.utils'
import { ApiError } from '@/api/client'
import { entryLinks } from '@/api/entryLink'
import { entryPaths } from '@/entry/entry.paths'
import { draftFromForm, type FormFields } from '@/entry/entry.utils'
import {
  isLinkToken,
  joinFromForm,
  type LinkFormState,
  participantErrorText,
  registeredIdsFromForm,
  scorerChoiceFromForm,
} from '@/entry/entryLink.utils'

// 本人用のリンクの画面(/entry/{合言葉})の送信。ログインは無く、合言葉だけで本人を確かめる(api が確かめる)。
// 成功したら ?done= を付けて同じ画面に移る。入力や状態の誤りなら、参加者向けの文を返す(フォームは
// 書いた中身のまま残る。components/entry の submitLinkForm)。

async function runLinkAction(
  formData: FormData,
  run: (token: string, form: FormFields) => Promise<string>,
): Promise<LinkFormState> {
  const form = formToObject(formData)
  const token = typeof form.token === 'string' ? form.token : ''
  if (!isLinkToken(token)) {
    redirect('/')
  }
  let done: string
  try {
    done = await run(token, form)
  } catch (error) {
    // 入力と状態の誤り(4xx)は画面に文で出す。それ以外(api が落ちているなど)はエラーの画面へ
    if (!(error instanceof ApiError) || error.status >= 500) {
      throw error
    }
    return { error: participantErrorText(error.code, error.message) }
  }
  redirect(entryPaths.link(token, { done }))
}

export async function submitTeamEntryAction(
  _state: LinkFormState,
  formData: FormData,
): Promise<LinkFormState> {
  return runLinkAction(formData, async (token, form) => {
    const view = await entryLinks.get(token)
    await entryLinks.submitTeam(token, draftFromForm(form, view.team?.questions ?? []))
    return 'team'
  })
}

export async function resubmitTeamEntryAction(
  _state: LinkFormState,
  formData: FormData,
): Promise<LinkFormState> {
  return runLinkAction(formData, async (token, form) => {
    const view = await entryLinks.get(token)
    const entryId = Number(typeof form.entryId === 'string' ? form.entryId : '')
    await entryLinks.resubmitTeam(token, entryId, {
      ...draftFromForm(form, view.team?.questions ?? []),
      registeredIds: registeredIdsFromForm(form),
    })
    return 'resubmit'
  })
}

export async function joinTeamAction(
  _state: LinkFormState,
  formData: FormData,
): Promise<LinkFormState> {
  return runLinkAction(formData, async (token, form) => {
    const view = await entryLinks.get(token)
    await entryLinks.join(token, joinFromForm(form, view.member?.questions ?? []))
    return 'member'
  })
}

/** 参加が決まったチームの記録係を、LINE で登録したメンバーから選ぶ(代表の画面) */
export async function chooseScorerAction(
  _state: LinkFormState,
  formData: FormData,
): Promise<LinkFormState> {
  return runLinkAction(formData, async (token, form) => {
    const { entryId, memberId } = scorerChoiceFromForm(form)
    await entryLinks.chooseScorer(token, entryId, memberId)
    return memberId === null ? 'scorer-cleared' : 'scorer'
  })
}
