import type { AdminMatch } from '@/api/admin.types'

// 時間割(コート × 時刻の表)の組み立て。会場に貼る紙と本部の控えに使う。

export type TimetableMatch = AdminMatch & { divisionName: string }

export type TimetableDay = {
  date: string | null
  fields: string[]
  times: string[]
  /** `${time}|${field}` → その枠の試合(通常は 1 つ) */
  cells: Map<string, TimetableMatch[]>
  /** 時刻かコートが決まっていない試合(表の下に一覧で出す) */
  unplaced: TimetableMatch[]
}

/** コートの並び。数字のコートを数値順で先に、それ以外を文字列順で後に(日程の既定の並びと同じ) */
export function compareFields(a: string, b: string): number {
  const numeric = /^\d+$/
  const aNum = numeric.test(a)
  const bNum = numeric.test(b)
  if (aNum && bNum) {
    return Number(a) - Number(b)
  }
  if (aNum !== bNum) {
    return aNum ? -1 : 1
  }
  return a.localeCompare(b, 'ja')
}

export function cellKey(time: string, field: string): string {
  return `${time}|${field}`
}

/** 試合を日付ごとの時間割にする。日付の無い試合は最後の日(date が null)にまとめる */
export function buildTimetable(matches: TimetableMatch[]): TimetableDay[] {
  const byDate = new Map<string | null, TimetableMatch[]>()
  for (const match of matches) {
    const list = byDate.get(match.date) ?? []
    list.push(match)
    byDate.set(match.date, list)
  }
  const dates = [...byDate.keys()].sort((a, b) => {
    if (a === null || b === null) {
      return a === null ? 1 : -1
    }
    return a.localeCompare(b)
  })
  return dates.map((date) => {
    const list = byDate.get(date) ?? []
    const cells = new Map<string, TimetableMatch[]>()
    const unplaced: TimetableMatch[] = []
    const fields = new Set<string>()
    const times = new Set<string>()
    for (const match of list) {
      if (match.time === null || match.field === null) {
        unplaced.push(match)
        continue
      }
      fields.add(match.field)
      times.add(match.time)
      const key = cellKey(match.time, match.field)
      cells.set(key, [...(cells.get(key) ?? []), match])
    }
    return {
      date,
      fields: [...fields].sort(compareFields),
      times: [...times].sort(),
      cells,
      unplaced,
    }
  })
}
