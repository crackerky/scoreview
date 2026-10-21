import type { Lang } from './lang'

// 画面の固定文言の日英対応(clone-spec A〜D の jp/en 表)。参照サイトの誤綴り(Reague, deffence)と
// 全角空白・半角カナはそのまま再現する。表見出しの改行は "\n" で表し、描画側で <br /> にする。
// lang=en でも日本語のまま出る文言(大会一覧の区分見出しなど)は en にも日本語を入れる。

const jp = {
  // 大会ナビ(clone-spec A 1.3、E 3)。グループ成績は末尾に全角空白が付く
  navTournamentTop: '大会トップ',
  navTeamList: 'チーム一覧',
  navEntryList: 'エントリー一覧',
  navSchedule: 'スケジュール',
  navAllDivision: '全ディビジョン',
  navDivisionScore: 'ディビジョン成績',
  navGroupScore: 'グループ成績　',
  navResult: '成績',
  navGallery: 'ギャラリー',

  // 大会一覧の区分見出し(clone-spec A 3。lang=en でも日本語)
  sectionEntryOpen: 'エントリー受付中',
  sectionEntryClosed: 'エントリー締切り',
  sectionFinished: '大会終了',

  // 大会トップ(clone-spec A 4)
  logoAlt: 'ロゴ',
  topDivision: '開催ディビジョン',
  topInformation: '大会情報',
  topScheduleLink: '[スケジュール]',
  topStandingsLink: '[最終順位]',

  // チーム一覧の件数(clone-spec A 5。jp でも英語固定)
  teamListTeams: 'Teams',
  teamListPlayers: 'Players',

  // スケジュールと結果一覧(clone-spec B 5〜10、C 2.1(c)、D 1.1)
  colDivision: 'ディビジョン',
  colDate: '日付',
  colTime: '時間',
  colField: 'コート',
  colCode: 'CD',
  colTeam: 'チーム名',
  colDash: '-',
  colVs: 'VS',
  colStatus: '状況',
  scheduleScoreSeparator: 'x',
  listScoreSeparator: 'Ｘ',
  statusResult: '[対戦結果]',
  statusWaiting: '未試合',
  statusEnd: '終了',
  // 推測: 試合中の文言は参照サイトで未観測(clone-spec B 6)
  statusLive: '試合中',
  placeholderWin: '勝ち',
  placeholderLose: '負け',
  refreshWait: '５秒お待ちください',
  refreshOkAlt: 'Reflesh_OK',
  refreshNgAlt: 'RefleshWait',
  titleAllSchedule: '-全スケジュール-',
  titleDivisionSchedule: '-ディビジョンスケジュール-',

  // ディビジョン成績(clone-spec C 1)
  standings: '順位表',
  scoreboardLeaders: 'スコアボードリーダー',
  scoreboardLeadersBlock: 'スコアボードリーダー　（ブロック）',
  colTeams: 'チーム名',
  colGames: '試合数',
  colWins: '勝ち数',
  colDraws: '分け数',
  colLosses: '負け数',
  colGoalsFor: '得点',
  colGoalsAgainst: '失点',
  colGoalsDiff: '得失差',
  colWinRate: '勝率',
  colRank: '順位',
  colPoints: '勝ち点',
  colNumber: '#',
  colPlayer: '選手',
  colAssist: 'アシスト',
  colGoal: 'ゴール',
  colTotal: 'トータル',
  colBlock: 'ブロック',
  colEmpty: '--',

  // グループ成績(clone-spec C 2)
  matrixCorner: '- MATRIX -',
  resultCorner: '- RESULT -',
  leagueResult: 'リーグ結果',
  groupGameList: '結果一覧',
  placement: 'Placement',

  // チームページ(clone-spec D 1)
  teamGameList: 'チーム対戦一覧',
  playersList: '選手一覧',
  staffList: 'スタッフ一覧',
  colRole: '役割',
  colStaff: 'スタッフ',
  colTeamBlock: 'ブロック',
  sotgScoreList: 'SOTGｽｺｱ一覧',
  sotgLegend:
    '※１･･･ルールの理解　※２･･･ファール及び身体接触　※３･･･フェアプレイ　\n' +
    '※４･･･ポジティブな姿勢およびセルフマネジメント　※５･･･相手チームのスピリットとの比較',
  colOpponent: '相手チーム',
  colSotg1: '※１',
  colSotg2: '※２',
  colSotg3: '※３',
  colSotg4: '※４',
  colSotg5: '※５',
  colSotgTotal: '合計',
  sotgAverage: '< SOTG 平均 >',

  // 試合ページ(clone-spec D 2)。出場選手表の見出しは jp でも英語
  matchResultBadge: '-対戦結果-',
  fieldPrefix: '@Field:',
  colElapsed: '経過',
  colAssistPlayer: 'アシスト選手',
  colGoalPlayer: 'ゴール選手',
  colPoint: '点',
  colEvent: 'イベント',
  eventGoal: '得点',
  eventPull: '試合開始ｽﾛｰｵﾌ',
  eventTimeout: 'タイムアウト',
  eventEnd: '試合終了',
  eventMark: '★',
  lineupNumber: '#',
  lineupPlayer: 'Player',
  lineupAssist: 'Assist',
  lineupGoal: 'Goal',
  lineupBlock: 'Block',
  gameStatistics: '統計情報',
  statGoals: '得点数',
  statFromOffence: 'オフェンスからの得点数',
  statFromDefence: 'ディフェンスからの得点数',
  statFromTurnover: 'ターンオーバーからの得点数',
  statTimeouts: 'タイムアウト数',

  // 選手ページ(clone-spec D 3)
  colCallahan: 'キャラハン',
  colGreatest: 'グレイテスト',
  gameEvents: 'イベント',
  colVsTeam: '対戦チーム',
  dgPoints: '得点',
  dgCatch: 'キャッチ',
  dgThrowMiss: 'スローミス',
  dgCleanCatch: 'ｸﾘｰﾝｷｬｯﾁ',
  dgFollowCatch: 'ﾌｫﾛｰｷｬｯﾁ',
  dgFootFault: 'ﾌｯﾄﾌｧｰﾙ',

  // 試合ページの得点経過(clone-spec D 2.3。import-cache の試合 45322 などで観測)
  eventCallahan: 'キャラハンゴール',
  // スケジュールの検索フォームの送信ボタン。JavaScript がないとき(noscript)だけ出す本サイト独自の文言
  scheduleSearchSubmit: '検索',
} as const

export type MessageKey = keyof typeof jp

const en: Record<MessageKey, string> = {
  navTournamentTop: 'Tournament Top',
  navTeamList: 'Team List',
  navEntryList: 'Entry List',
  navSchedule: 'Schedule',
  navAllDivision: 'All Division',
  navDivisionScore: 'Division Score',
  navGroupScore: 'Group Score',
  navResult: 'Result',
  navGallery: 'Gallery',

  sectionEntryOpen: 'エントリー受付中',
  sectionEntryClosed: 'エントリー締切り',
  sectionFinished: '大会終了',

  logoAlt: 'ロゴ',
  topDivision: 'Division',
  topInformation: 'Information',
  topScheduleLink: '[Schedule]',
  topStandingsLink: '[Standings]',

  teamListTeams: 'Teams',
  teamListPlayers: 'Players',

  colDivision: 'Division',
  colDate: 'Date',
  colTime: 'Time',
  colField: 'Field',
  colCode: 'CD',
  colTeam: 'Team',
  colDash: '-',
  colVs: 'VS',
  colStatus: 'Status',
  scheduleScoreSeparator: 'x',
  listScoreSeparator: 'Ｘ',
  statusResult: '[Result]',
  statusWaiting: 'Waiting',
  statusEnd: 'END',
  // 推測(jp と同じく未観測)
  statusLive: 'Playing',
  placeholderWin: 'Win',
  placeholderLose: 'Lose',
  refreshWait: 'Please Wait 5 seconds. (Reload)',
  refreshOkAlt: 'Reflesh_OK',
  refreshNgAlt: 'RefleshWait',
  titleAllSchedule: '-All Schedule-',
  titleDivisionSchedule: '-Division Schedule-',

  standings: 'Standings',
  scoreboardLeaders: 'Scoreboard Leaders',
  scoreboardLeadersBlock: 'Scoreboard Leaders *Block*',
  colTeams: 'Teams',
  colGames: 'Games',
  colWins: 'Wins',
  colDraws: 'Draw',
  colLosses: 'Losses',
  colGoalsFor: 'Goals\nfor',
  colGoalsAgainst: 'Goals\nagainst',
  colGoalsDiff: 'Goals\ndiff',
  colWinRate: 'Win %',
  colRank: 'Rank',
  // 推測: 勝ち点列は HTML コメント内にしか無く、英語表記は未観測(clone-spec C 2.1(b))
  colPoints: 'Points',
  colNumber: '#',
  colPlayer: 'Player',
  colAssist: 'Assist',
  colGoal: 'Goal',
  colTotal: 'Tot.',
  colBlock: 'Block',
  colEmpty: '--',

  matrixCorner: '- MATRIX -',
  resultCorner: '- RESULT -',
  leagueResult: 'Reague Result',
  groupGameList: 'GameList for the Group',
  placement: 'Placement',

  teamGameList: 'GameList for the Team',
  playersList: 'Players List',
  staffList: 'Staffs List',
  colRole: 'Task',
  colStaff: 'Staff',
  colTeamBlock: 'block',
  sotgScoreList: 'SOTG ScoreList',
  // 推測: ※4 の後半と ※5 は clone-spec D 1.1 で途中までしか記録が無い
  sotgLegend:
    '* 1 ･･･Rules Knowledge & Use  * 2 ･･･Fouls & Body Contact  * 3 ･･･Fair-Mindedness  \n' +
    "* 4 ･･･Positive Attitude & Self-Control  * 5 ･･･Compare with Opponent's Spirit",
  colOpponent: 'VS Team',
  colSotg1: '* 1',
  colSotg2: '* 2',
  colSotg3: '* 3',
  colSotg4: '* 4',
  colSotg5: '* 5',
  colSotgTotal: 'Total',
  sotgAverage: '< SOTG Avg. >',

  matchResultBadge: '--Result--',
  fieldPrefix: '@Field:',
  colElapsed: 'Time',
  colAssistPlayer: 'Assist Player',
  colGoalPlayer: 'Goal Player',
  colPoint: 'P.',
  colEvent: 'Event',
  eventGoal: 'Goal',
  eventPull: '1st Pull',
  eventTimeout: 'Timeout',
  eventEnd: 'End',
  eventMark: '*',
  lineupNumber: '#',
  lineupPlayer: 'Player',
  lineupAssist: 'Assist',
  lineupGoal: 'Goal',
  lineupBlock: 'Block',
  gameStatistics: 'Game Statistics',
  statGoals: 'Goals',
  statFromOffence: 'Goals from starting on offence',
  statFromDefence: 'Goals from starting on deffence',
  statFromTurnover: 'Goals from turnovers',
  statTimeouts: 'Time outs',

  colCallahan: 'Callahan',
  colGreatest: 'Greatest',
  gameEvents: 'Game Events',
  colVsTeam: 'VS Team',
  // 推測: ディスクゴルフの選手成績の英語見出しは未観測(clone-spec D 3.1)
  dgPoints: 'Points',
  dgCatch: 'Catch',
  dgThrowMiss: 'Throw Miss',
  dgCleanCatch: 'Clean Catch',
  dgFollowCatch: 'Follow Catch',
  dgFootFault: 'Foot Fault',

  // 推測: キャラハンゴールの英語表記は未観測
  eventCallahan: 'Callahan Goal',
  scheduleSearchSubmit: 'Search',
}

const dictionaries: Record<Lang, Record<MessageKey, string>> = { jp, en }

/** 固定文言を言語に応じて返す。 */
export function msg(lang: Lang, key: MessageKey): string {
  return dictionaries[lang][key]
}

/**
 * SCV_MIDA_M 見出しなどの "< 文言 >" 表記。clone-spec C 0: jp は山括弧の内側に半角空白、
 * en は空白なし(例 "< 結果一覧 >" / "<GameList for the Group>")。
 */
export function angleHeading(lang: Lang, text: string): string {
  return lang === 'en' ? `<${text}>` : `< ${text} >`
}

/** 未確定チームの表記 "[A1-勝ち]" / "[A1-Win]"(clone-spec B 8)。 */
export function placeholderLabel(lang: Lang, code: string, outcome: 'win' | 'lose'): string {
  return `[${code}-${msg(lang, outcome === 'win' ? 'placeholderWin' : 'placeholderLose')}]`
}
