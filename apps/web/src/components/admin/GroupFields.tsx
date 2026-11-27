import { groupShown } from '@/admin/group.utils'
import { groupKindChoices } from '@/admin/labels.utils'
import type { AdminGroup } from '@/api/admin.types'
import { Check, Field, Hidden } from './AdminParts'

// 組を手で足すときと、組の設定を直すときの欄。組がスコアサイトでどう出るか(種類、並ぶ順、出すかどうか)を
// 言葉で添える。画像の欄は、種類で画像を選んだときだけ出す(.grp-form:has で切り替える。CSS だけなので、
// 画面を開いた直後から効く)。

type Props =
  | {
      mode: 'add'
      /** 足す組の並び順の初めの値(今の組の一番後ろ) */
      nextSortOrder: number
    }
  | { mode: 'edit'; group: AdminGroup }

export function GroupFields(props: Props) {
  const group = props.mode === 'edit' ? props.group : undefined
  const kind = group?.kind ?? 'pool'
  return (
    <>
      <Field
        label="組の名前"
        name="name"
        defaultValue={group?.name}
        placeholder="例 A組、決勝トーナメント"
        required
      />
      <fieldset>
        <legend>種類</legend>
        {groupKindChoices.map((choice) => (
          <label key={choice.value} className="adm-check adm-audience">
            <input
              type="radio"
              name="kind"
              value={choice.value}
              defaultChecked={kind === choice.value}
            />
            <span>
              {choice.label}
              <small className="adm-muted">{choice.hint}</small>
            </span>
          </label>
        ))}
      </fieldset>
      <div className="grp-image adm-grid">
        <Field
          label="見せる画像のアドレス(https:// で始まり、開くと画像そのものが出るもの)"
          name="imageUrl"
          defaultValue={group?.image?.url}
        />
        {group ? (
          <>
            <Field
              label="画像の幅(px。空なら画像のとおり)"
              name="imageWidth"
              type="number"
              defaultValue={group.image?.width}
            />
            <Field
              label="画像の高さ(px。空なら画像のとおり)"
              name="imageHeight"
              type="number"
              defaultValue={group.image?.height}
            />
          </>
        ) : null}
      </div>
      <label className="adm-field">
        <span>スコアサイトでの並び順</span>
        <input
          type="number"
          name="sortOrder"
          defaultValue={group?.sortOrder ?? (props.mode === 'add' ? props.nextSortOrder : 0)}
        />
        <small className="adm-muted">
          {group
            ? '部門の組の中で、小さい数ほど先に出ます。同じ数なら、先に作った組が先です。'
            : '部門の組の中で、小さい数ほど先に出ます。そのままなら、今ある組の一番後ろに入ります。'}
        </small>
      </label>
      {group ? (
        <>
          {/* 今のスコアサイトは、リンクにしない組(linked が偽)を出さない。出すかどうかの欄を 1 つにし、
              出すときはリンクにもする */}
          <Hidden values={{ linked: 'on' }} />
          <Check label="スコアサイトに出す" name="published" defaultChecked={groupShown(group)} />
        </>
      ) : null}
    </>
  )
}
