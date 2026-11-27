'use server'

import { redirect } from 'next/navigation'
import { formToObject } from '@/admin/form.utils'
import { saveSession } from '@/admin/session'
import { ApiError } from '@/api/client'
import { entryLinks } from '@/api/entryLink'
import { entryPaths } from '@/entry/entry.paths'
import { draftFromForm, type FormFields } from '@/entry/entry.utils'
import {
  idFromForm,
  isLinkToken,
  joinFromForm,
  type LinkFormState,
  participantErrorText,
  registeredIdsFromForm,
  scorerChoiceFromForm,
  scorerLanding,
} from '@/entry/entryLink.utils'

// 本人用のリンクの画面(/entry/{合言葉})の送信。ログインは無く、合言葉だけで本人を確かめる(api が確かめる)。
// 成功したら ?done= を付けて同じ画面に移る。入力や状態の誤りなら、参加者向けの文を返す(フォームは
// 書いた中身のまま残る。components/entry の submitLinkForm)。

async function runLinkActionTo(
  formData: FormData,
  run: (token: string, form: FormFields) => Promise<string>,
): Promise<LinkFormState> {
  const form = formToObject(formData)
  const token = typeof form.token === 'string' ? form.token : ''
  if (!isLinkToken(token)) {
    redirect('/')
  }
  let next: string
  try {
    next = await run(token, form)
  } catch (error) {
    // 入力と状態の誤り(4xx)は画面に文で出す。それ以外(api が落ちているなど)はエラーの画面へ
    if (!(error instanceof ApiError) || error.status >= 500) {
      throw error
    }
    return { error: participantErrorText(error.code, error.message) }
  }
  redirect(next)
}

/** 送って、同じ画面に ?done= を付けて移る */
async function runLinkAction(
  formData: FormData,
  run: (token: string, form: FormFields) => Promise<string>,
): Promise<LinkFormState> {
  return runLinkActionTo(formData, async (token, form) =>
    entryPaths.link(token, { done: await run(token, form) }),
  )
}

/** チームを探す画面でチームを選ぶと、そのチームのメンバー登録の画面(新しく発行したリンク)へ移る */
export async function chooseTeamAction(
  _state: LinkFormState,
  formData: FormData,
): Promise<LinkFormState> {
  return runLinkActionTo(formData, async (token, form) => {
    const issued = await entryLinks.chooseTeam(token, idFromForm(form, 'entryId'))
    return entryPaths.link(issued.token)
  })
}

/** 代表が、招待リンクを使わずに登録したメンバーを外す */
export async function removeMemberAction(
  _state: LinkFormState,
  formData: FormData,
): Promise<LinkFormState> {
  return runLinkAction(formData, async (token, form) => {
    await entryLinks.removeMember(token, idFromForm(form, 'entryId'), idFromForm(form, 'memberId'))
    return 'removed'
  })
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

/**
 * チームの記録係が、公式 LINE の本人用のリンクから記録の画面に入る。api がリンクを確かめてセッションを
 * 発行し、管理画面のログインと同じ Cookie に入れて、自分の担当試合の一覧(管理画面のトップ)へ移る。
 * リンクを開いただけでは入らず、ボタンを押したときに入る(LINE がリンクの見本を作るために開いても入らない)。
 */
export async function scorerSignInAction(
  _state: LinkFormState,
  formData: FormData,
): Promise<LinkFormState> {
  const form = formToObject(formData)
  const token = typeof form.token === 'string' ? form.token : ''
  if (!isLinkToken(token)) {
    redirect('/')
  }
  try {
    const session = await entryLinks.scorerSession(token)
    await saveSession(session.token, session.expiresAt)
  } catch (error) {
    if (!(error instanceof ApiError) || error.status >= 500) {
      throw error
    }
    return { error: participantErrorText(error.code, error.message) }
  }
  // 公式 LINE の試合のボタンから開いたら、その試合のスコアの画面へ(記録できる試合かは、その画面が
  // 確かめる)
  redirect(scorerLanding(form.match))
}
