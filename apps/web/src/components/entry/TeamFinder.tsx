'use client'

import { useActionState, useState } from 'react'
import type { EntryLinkFind } from '@/api/entry.types'
import { type LinkFormState, teamMatches } from '@/entry/entryLink.utils'
import { FormError } from './FormError'

// メンバーが大会の中から自分のチームを探す画面の本体。チーム名(正式名か短縮名)を打つと絞り込み、部門ごとに
// 並べる。チームを押すと、そのチームのメンバー登録の画面に移る(api が新しいリンクを発行する)。チームが
// 多い大会(1 部門に 90 チーム以上)でも、名前の一部を打てば見つけられるようにする。

type Props = {
  token: string
  teams: EntryLinkFind['teams']
  action: (state: LinkFormState, formData: FormData) => Promise<LinkFormState>
}

export function TeamFinder({ token, teams, action }: Props) {
  const [state, formAction, pending] = useActionState(action, { error: null })
  const [query, setQuery] = useState('')
  const shown = teams.filter((team) => teamMatches(team, query))
  const divisions = [...new Map(shown.map((team) => [team.division.id, team.division])).values()]
  return (
    <form action={formAction} className="entry-form">
      <FormError error={state.error} attempt={state} />
      <input type="hidden" name="token" value={token} />
      <label className="entry-field">
        <span>チーム名で探す</span>
        <input
          type="search"
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          placeholder="チーム名の一部"
          autoComplete="off"
          enterKeyHint="search"
        />
      </label>
      <p className="entry-muted" role="status">
        {query === '' ? `${teams.length} チーム` : `${shown.length} チームが見つかりました`}
      </p>
      {divisions.map((division) => (
        <section key={division.id} className="entry-find-group">
          <h2 className="entry-subheading">{division.name.ja}</h2>
          <div className="entry-find-list">
            {shown
              .filter((team) => team.division.id === division.id)
              .map((team) => (
                <button
                  key={team.entryId}
                  type="submit"
                  name="entryId"
                  value={team.entryId}
                  className="entry-find-team"
                  disabled={pending}
                >
                  <span>{team.fullName}</span>
                  {team.shortName !== team.fullName ? <small>{team.shortName}</small> : null}
                </button>
              ))}
          </div>
        </section>
      ))}
      {shown.length === 0 ? (
        <p className="entry-muted">
          見つかりません。チームの代表が申し込んでいるか、代表に確かめてください。
        </p>
      ) : null}
    </form>
  )
}
