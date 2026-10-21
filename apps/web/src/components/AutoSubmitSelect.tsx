'use client'

import type { ComponentProps } from 'react'

/**
 * 値を変えたら所属する form を送信する select(参照サイトの onchange="Action('search')"、clone-spec B 1)。
 * form 自体は method=get の素の HTML なので、JavaScript がなくても送信ボタンで送れる。
 */
export function AutoSubmitSelect(props: Omit<ComponentProps<'select'>, 'onChange'>) {
  return <select {...props} onChange={(event) => event.currentTarget.form?.requestSubmit()} />
}
