import type { EntryAdminDetail, EntryExport } from '@/api/entry.types'
import { answerText, channelLabels, dateTimeText, genderLabels, statusLabels } from './entry.utils'

// 申込の書き出し(CSV)。運営が Excel で名簿として開く前提で、メンバー 1 人を 1 行にし、チームの欄と
// チーム向けの質問への答えを各行に繰り返す(並べ替えや絞り込みをしても行がチームから離れないように)。
// メンバー向けの質問への答えは、メンバーの欄の後ろに、その人の行にだけ書く。大会が性別と生年月日を聞くなら、
// メンバーの欄のすぐ後ろにその列を足す。

/** Excel が UTF-8 と読むための BOM。無いと日本語が化ける */
const bom = '﻿'

/**
 * 1 つの欄を CSV の書き方にする。= + - @ とタブ、改行で始まる値は、表計算ソフトが式として実行しうる
 * ので先頭に ' を付ける(CSV インジェクション。OWASP の推奨)。区切り、引用符、改行を含む値は引用符で囲む。
 */
export function csvCell(value: string): string {
  const safe = /^[=+\-@\t\r\n]/.test(value) ? `'${value}` : value
  return /[",\r\n]/.test(safe) ? `"${safe.replaceAll('"', '""')}"` : safe
}

function csvLine(cells: readonly string[]): string {
  return cells.map(csvCell).join(',')
}

const memberColumns = ['背番号', '名前', 'LINE の登録'] as const

type ExportMember = EntryAdminDetail['memberList'][number]

/** 大会が聞く本人の項目の列(見出しと、メンバーの値の取り出し) */
function personalColumns(
  personal: EntryExport['personal'],
): Array<{ label: string; value: (member: ExportMember) => string }> {
  return [
    ...(personal.gender
      ? [
          {
            label: '性別',
            value: (member: ExportMember) => (member.gender ? genderLabels[member.gender] : ''),
          },
        ]
      : []),
    ...(personal.birthdate
      ? [{ label: '生年月日', value: (member: ExportMember) => member.birthdate ?? '' }]
      : []),
  ]
}

function teamCells(entry: EntryAdminDetail, answers: readonly string[]): string[] {
  return [
    String(entry.id),
    statusLabels[entry.status],
    entry.division.name.ja,
    entry.fullName,
    entry.repName,
    entry.repPhone,
    entry.repEmail,
    dateTimeText(entry.submittedAt),
    channelLabels[entry.channel],
    ...answers,
  ]
}

/** 書き出す申込を CSV の文字列(BOM つき、改行は CRLF)にする */
export function entriesCsv(data: EntryExport): string {
  const questions = data.questions.filter((question) => question.scope === 'team')
  const memberQuestions = data.questions.filter((question) => question.scope === 'member')
  const personal = personalColumns(data.personal)
  const header = [
    '申込番号',
    '状態',
    '部門',
    'チーム名',
    '代表',
    '電話番号',
    'メールアドレス',
    '受けた日時',
    '経路',
    ...questions.map((question) => question.label),
    ...memberColumns,
    ...personal.map((column) => column.label),
    ...memberQuestions.map((question) => question.label),
  ]
  const lines = [csvLine(header)]
  for (const entry of data.entries) {
    const given = new Map(entry.answers.map((answer) => [answer.questionId, answer.value]))
    const answers = questions.map((question) => {
      const value = given.get(question.id)
      return value === undefined ? '' : answerText(question, value)
    })
    const team = teamCells(entry, answers)
    if (entry.memberList.length === 0) {
      lines.push(
        csvLine([...team, '', '', '', ...personal.map(() => ''), ...memberQuestions.map(() => '')]),
      )
      continue
    }
    for (const member of entry.memberList) {
      const own = new Map(member.answers.map((answer) => [answer.questionId, answer.value]))
      const memberAnswers = memberQuestions.map((question) => {
        const value = own.get(question.id)
        return value === undefined ? '' : answerText(question, value)
      })
      lines.push(
        csvLine([
          ...team,
          member.number,
          member.name,
          member.registered ? '済み' : 'まだ',
          ...personal.map((column) => column.value(member)),
          ...memberAnswers,
        ]),
      )
    }
  }
  return `${bom}${lines.join('\r\n')}\r\n`
}

/** 書き出すファイルの名前(ASCII だけにする。日本語の名前はブラウザによって化けるため) */
export function entriesCsvName(tournamentId: number, now: Date): string {
  const jst = new Date(now.getTime() + 9 * 60 * 60 * 1000)
  const day = jst.toISOString().slice(0, 10).replaceAll('-', '')
  return `entries-${tournamentId}-${day}.csv`
}
