import type { GroupResponse } from '@/api/public.types'
import { type Lang, pickText } from '@/i18n/lang'

// 本サイト独自の補足(docs/requirements.md「意図的な差分」)。参照サイトのリーグ結果の下に、
// 同点を分けた根拠と、進行中のプールの勝ち上がり条件を小さく出す。文は api が日本語で作るので、
// 日本語表示のときだけ出す(英語表示は参照サイトどおりにする)。

type Notes = NonNullable<GroupResponse['pool']>['notes']

export function PoolNotes({ notes, lang }: { notes: Notes; lang: Lang }) {
  if (lang !== 'jp' || (notes.reasons.length === 0 && notes.conditions.length === 0)) {
    return null
  }
  return (
    <div className="sv_pool_notes">
      {notes.conditions.length > 0 ? (
        <>
          <p className="sv_pool_notes_head">勝ち上がり条件(残り試合の勝敗から計算)</p>
          <ul>
            {notes.conditions.map((note) => (
              <li key={`c-${note.team.id}`}>
                {pickText(lang, note.team.name)}: {note.text}
              </li>
            ))}
          </ul>
        </>
      ) : null}
      {notes.reasons.length > 0 ? (
        <>
          <p className="sv_pool_notes_head">順位の根拠</p>
          <ul>
            {notes.reasons.map((note) => (
              <li key={`r-${note.team.id}`}>
                {pickText(lang, note.team.name)}: {note.text}
              </li>
            ))}
          </ul>
        </>
      ) : null}
    </div>
  )
}
