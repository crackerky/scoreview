import type { Gender } from '@/api/common.types'

// 名簿(チームと選手)の貼り付けの読み取りと、登録前のプレビュー(運営者レビュー H4: エクセルの見出し行
// 「No」がチームになり、番号の列が正式名に入った)。表計算からの貼り付け(タブ区切り)と、メモ帳からの
// 貼り付け(カンマか読点区切り)の両方を受ける。

/** 貼り付けを行と列に分ける。空行は飛ばす。どの行にもタブが無ければカンマと読点で分ける */
export function splitRows(text: string): string[][] {
  const lines = text.split(/\r?\n/).filter((line) => line.trim() !== '')
  const tabbed = lines.some((line) => line.includes('\t'))
  return lines.map((line) => line.split(tabbed ? '\t' : /,|、/).map((cell) => cell.trim()))
}

/** 列の数(いちばん長い行に合わせる) */
export function columnCount(rows: readonly string[][]): number {
  return rows.reduce((most, row) => Math.max(most, row.length), 0)
}

const headerWords = {
  name: /^(チーム名?|正式名|チーム名称|団体名|team(\s*name)?|name)$/i,
  short: /^(略称|短縮名|略名|表示名|short(\s*name)?)$/i,
  team: /^(チーム名?|所属|団体名|team)$/i,
  number: /^(背番号|番号|no\.?|#|number)$/i,
  person: /^(氏名|名前|選手名?|player|name)$/i,
  gender: /^(性別|gender|sex)$/i,
  // 英語名の列(無ければ api が日本語名を入れる。英語の閲覧ページで名前が空にならないように)
  english:
    /^(英語名|英名|英語表記|(チーム名?|氏名|名前)\s*[(（]英語[)）]|english(\s*name)?|name\s*\(en\))$/i,
  other: /^(代表者?|連絡先|備考|メール|電話|no\.?|#|番号)$/i,
}

/** 1 行目が見出しに見えるか(「No」「チーム名」「略称」などの言葉がある) */
export function looksLikeHeader(row: readonly string[]): boolean {
  return row.some((cell) => Object.values(headerWords).some((word) => word.test(cell)))
}

function findColumn(header: readonly string[] | null, word: RegExp): number | null {
  if (header === null) {
    return null
  }
  const index = header.findIndex((cell) => word.test(cell))
  return index >= 0 ? index : null
}

const numeric = /^\d+$/

/** 列の中身が全部数字か(番号の列を名前の列に選ばないため) */
function numericColumn(rows: readonly string[][], column: number): boolean {
  const values = rows.map((row) => row[column] ?? '').filter((value) => value !== '')
  return values.length > 0 && values.every((value) => numeric.test(value))
}

/**
 * チームの名簿の列の割り当て。short が null なら略称は正式名と同じ。nameEn が null なら英語名は空で送り、
 * api が日本語名を入れる
 */
export type TeamColumns = {
  header: boolean
  name: number
  short: number | null
  nameEn: number | null
}

/** チームの名簿の列を推測する。見出しがあれば言葉で、無ければ数字だけでない最初の列を名前にする */
export function guessTeamColumns(rows: readonly string[][]): TeamColumns {
  const header = rows[0] && looksLikeHeader(rows[0]) ? rows[0] : null
  const body = header ? rows.slice(1) : rows
  const count = columnCount(rows)
  let name = findColumn(header, headerWords.name)
  if (name === null) {
    name = 0
    while (name < count - 1 && numericColumn(body, name)) {
      name++
    }
  }
  let short = findColumn(header, headerWords.short)
  if (short === null && header === null && count > name + 1 && !numericColumn(body, name + 1)) {
    // 見出しの無い「正式名、短縮名」(これまでの貼り付けの形)
    short = name + 1
  }
  return { header: header !== null, name, short, nameEn: findColumn(header, headerWords.english) }
}

/** 登録前のチームの行。fullNameEn は英語名の列の値(空なら api が日本語名を入れる)。warnings は画面に
 * 出す注意(登録は止めない) */
export type TeamRow = {
  fullName: string
  shortName: string
  fullNameEn: string
  warnings: string[]
}

/** 列の割り当てでチームの行を作る。existing は登録済みのチームの正式名(重複の注意に使う) */
export function teamRowsOf(
  rows: readonly string[][],
  columns: TeamColumns,
  existing: readonly string[] = [],
): TeamRow[] {
  const body = columns.header ? rows.slice(1) : rows
  const seen = new Map<string, number>()
  const out: TeamRow[] = []
  for (const row of body) {
    const fullName = row[columns.name] ?? ''
    if (fullName === '') {
      continue
    }
    const short = columns.short === null ? '' : (row[columns.short] ?? '')
    const warnings: string[] = []
    if (numeric.test(fullName)) {
      warnings.push('名前が数字だけです(番号の列を選んでいませんか)')
    }
    if (looksLikeHeader([fullName])) {
      warnings.push('見出しのようです(1 行目は見出し、を確かめてください)')
    }
    if (existing.includes(fullName)) {
      warnings.push('同じ名前のチームが登録済みです')
    }
    if (seen.has(fullName)) {
      warnings.push(`${(seen.get(fullName) ?? 0) + 1} 行目と同じ名前です`)
    }
    if (!seen.has(fullName)) {
      seen.set(fullName, out.length)
    }
    const fullNameEn = columns.nameEn === null ? '' : (row[columns.nameEn] ?? '')
    out.push({ fullName, shortName: short === '' ? fullName : short, fullNameEn, warnings })
  }
  return out
}

/** 選手の名簿の列の割り当て。number、gender、nameEn(英語名。無ければ api が日本語名を入れる)は無くてもよい */
export type PlayerColumns = {
  header: boolean
  team: number
  number: number | null
  name: number
  gender: number | null
  nameEn: number | null
}

/** 選手の名簿の列を推測する(見出しの言葉から。無ければ チーム名、背番号、氏名、性別 の順とみなす) */
export function guessPlayerColumns(rows: readonly string[][]): PlayerColumns {
  const header = rows[0] && looksLikeHeader(rows[0]) ? rows[0] : null
  const count = columnCount(rows)
  const at = (index: number) => (index < count ? index : null)
  return {
    header: header !== null,
    team: findColumn(header, headerWords.team) ?? 0,
    number: header ? findColumn(header, headerWords.number) : at(1),
    name: findColumn(header, headerWords.person) ?? (count > 2 ? 2 : count - 1),
    gender: header ? findColumn(header, headerWords.gender) : at(3),
    nameEn: findColumn(header, headerWords.english),
  }
}

/** 性別の書き方を M / F に。読めなければ null */
export function genderOf(value: string): Gender | null {
  if (/^(男|男性|m|male|メン|men)$/i.test(value)) {
    return 'M'
  }
  if (/^(女|女性|f|female|ウィメン|women)$/i.test(value)) {
    return 'F'
  }
  return null
}

/** 登録前の選手の行。teamId はチーム名から見つけたチーム(見つからなければ null で登録しない) */
export type PlayerRow = {
  teamId: number | null
  teamName: string
  number: string
  name: string
  nameEn: string
  gender: Gender | null
  warnings: string[]
}

/** 列の割り当てで選手の行を作る。チームは正式名か略称で探す */
export function playerRowsOf(
  rows: readonly string[][],
  columns: PlayerColumns,
  teams: ReadonlyArray<{ id: number; fullName: string; shortName: string }>,
): PlayerRow[] {
  const body = columns.header ? rows.slice(1) : rows
  const out: PlayerRow[] = []
  const seen = new Set<string>()
  for (const row of body) {
    const name = row[columns.name] ?? ''
    if (name === '') {
      continue
    }
    const teamName = row[columns.team] ?? ''
    const team = teams.find((item) => item.fullName === teamName || item.shortName === teamName)
    const number = columns.number === null ? '' : (row[columns.number] ?? '')
    const genderText = columns.gender === null ? '' : (row[columns.gender] ?? '')
    const gender = genderOf(genderText)
    const warnings: string[] = []
    if (!team) {
      warnings.push(`チーム「${teamName}」が見つかりません(この行は登録しません)`)
    }
    if (numeric.test(name)) {
      warnings.push('氏名が数字だけです(背番号の列を選んでいませんか)')
    }
    if (genderText !== '' && gender === null) {
      warnings.push(`性別「${genderText}」が読めません(空欄で登録します)`)
    }
    const key = `${team?.id ?? teamName}|${number}`
    if (number !== '' && seen.has(key)) {
      warnings.push('同じチームに同じ背番号があります')
    }
    seen.add(key)
    const nameEn = columns.nameEn === null ? '' : (row[columns.nameEn] ?? '')
    out.push({ teamId: team?.id ?? null, teamName, number, name, nameEn, gender, warnings })
  }
  return out
}

/** 英語名の一括の補い(POST /admin/divisions/{id}/english-names)の結果の通知 */
export function filledEnglishText(filled: {
  teams: number
  players: number
  groups: number
}): string {
  if (filled.teams + filled.players + filled.groups === 0) {
    return '英語名が空のチーム、選手、組はありませんでした'
  }
  return (
    `英語名が空だったところに日本語名を入れました(チーム ${filled.teams}、選手 ${filled.players}、` +
    `組 ${filled.groups})`
  )
}
