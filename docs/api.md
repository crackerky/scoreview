# API 契約

web(Next.js)と api(Go)の境界の仕様である。
Go の DTO(`apps/api/internal/httpapi/dto*.go`)と TypeScript の zod スキーマ(`apps/web/src/api/*.types.ts`)は、どちらも本書に合わせる。
本書を変えるときは、同じコミットで両側を変える。

## 共通

- ベース URL は `/api/v1`。api はインターネットに公開せず、web のサーバー側からだけ呼ぶ。
- 全リクエストに `X-Internal-Token: {INTERNAL_API_TOKEN}` を付ける。値が一致しなければ 401 を返す。
- JSON のキーは camelCase。日付は `"YYYY-MM-DD"`、時刻は `"HH:MM"`(秒なし)。
- 値がないことは `null` で表す。キーを省略しない(zod で `.nullable()` として検証するため)。
- 表示言語は API では選ばない。日英の両方を `I18nText` で返し、web が `lang` に応じて選ぶ。英語が未登録なら `en` は空文字。
- 閲覧用の応答はすべて `frame` を含む。web は `frame` から大会ナビとサイドバーを描く。

### エラー

```json
{ "error": { "code": "not_found", "message": "TOURNAMENT_ID [999999]" } }
```

| code | HTTP | 用途 |
|---|---|---|
| `bad_request` | 400 | クエリやボディの検証失敗 |
| `unauthorized` | 401 | 内部トークンかセッションがない、または不正 |
| `forbidden` | 403 | 権限がない |
| `not_found` | 404 | ID が存在しない。`message` は参照サイトのエラー画面に出す `{種別}_ID [{id}]`(`TOURNAMENT_ID`、`ENTRY_TEAM_ID`、`ENTRY_PLAYER_ID`、`MATCH_ID`、`EVENT_DIVISION_ID`、`GROUP_ID`、`TGALLERY_ID`) |
| `conflict` | 409 | 一意制約違反、状態遷移の不正 |
| `internal` | 500 | それ以外 |

## 共通の型

TypeScript の記法で示す。

```ts
type I18nText = { ja: string; en: string }

type TournamentStatus = 'entry_open' | 'entry_closed' | 'finished'
type DivisionKind = 'men' | 'women' | 'mixed' | 'none'
type TeamBarStyle = 'default' | 'mixed' | 'women' | 'univ' | 'waku1' | 'waku2'
type EntryKind = 'team' | 'individual'
type StatsKind = 'ultimate' | 'discgolf'
type MatchStatus = 'scheduled' | 'live' | 'finished'
type GroupKind = 'pool' | 'bracket' | 'image'
type Side = 'left' | 'right'
type Gender = 'M' | 'F'

// 大会ナビとサイドバーを描くための骨格
type TournamentFrame = {
  id: number
  name: I18nText
  shortName: string
  labels: { teamList: 'teams' | 'entries'; groups: 'groups' | 'results' }
  showSchedule: boolean              // ナビの「スケジュール」「ディビジョン成績」
  divisions: Array<{
    id: number
    name: I18nText
    scheduleEnabled: boolean         // サイドバーの SCORE メニューに出すか(divisions.schedule_enabled)
    // published=true のグループだけ。sort_order, id 順。name.ja は前後の空白も名前の一部
    groups: Array<{ id: number; name: I18nText; linked: boolean }>
  }>
  galleries: Array<{ id: number; name: I18nText }>  // 英語名は "---" のことが多い(参照サイトの値)
  banners: Array<{ imageUrl: string; linkUrl: string | null }>
}

type TeamRef = { id: number; name: I18nText }     // name は短縮名(画面の表で使う名前)
type PlayerRef = { id: number; number: string; name: I18nText; gender: Gender | null }

// 日程や結果一覧の 1 行の片側
type MatchSide = {
  team: TeamRef | null
  placeholder: { code: string; outcome: 'win' | 'lose' } | null  // team が null のとき
  score: number | null
  // 対戦相手の決まり方の英語表記("[A1-Win]" "[MA-3rd]" "(A10)")。なければ空文字。
  // 英語版でチームの英語名が空のとき、名前の代わりに出す(参照サイトと同じ。clone-spec B 8)
  labelEn: string
}

type MatchRow = {
  matchId: number
  hasDetail: boolean                 // false なら状況列はリンクなしの「終了」
  code: string
  date: string | null
  time: string | null
  field: string | null               // コート。数字とは限らない("A" など)
  status: MatchStatus
  left: MatchSide
  right: MatchSide
}

// 成績表の 1 行(チームページ、ディビジョン、プール)
type Tally = {
  games: number; wins: number; draws: number; losses: number
  goalsFor: number; goalsAgainst: number
}
```

## 閲覧用エンドポイント

### `GET /api/v1/tournaments`

大会一覧。`listed=true` の大会だけを返す。

```ts
{
  sections: Array<{
    status: TournamentStatus        // entry_open, entry_closed, finished の順。大会が 0 件の区分は返さない
    tournaments: Array<{ id: number; name: I18nText }>  // start_date 降順(NULL は先頭)、同日は id 降順
  }>
}
```

### `GET /api/v1/tournaments/{id}`

大会トップ。`listed=false` でも返す。

```ts
{
  frame: TournamentFrame
  logoUrl: string | null            // null なら web が既定ロゴを出す
  topHtml: I18nText                 // サーバー側で無害化済み
  divisions: Array<{
    id: number; name: I18nText
    scheduleLinked: boolean         // divisions.schedule_enabled
    standingsLinked: boolean        // divisions.standings_published
  }>
}
```

### `GET /api/v1/tournaments/{id}/teams`

チーム一覧。

```ts
{
  frame: TournamentFrame
  divisions: Array<{
    id: number; name: I18nText; kind: DivisionKind; entryKind: EntryKind
    teams: Array<{
      id: number
      name: I18nText                // 正式名
      barStyle: TeamBarStyle
      memberCount: number | null    // スタッフを除く選手数。individual なら null
    }>
  }>
}
```

### `GET /api/v1/tournaments/{id}/schedule` と `GET /api/v1/divisions/{id}/schedule`

スケジュール。前者が大会全体(参照サイトの `scv_search_tn`)、後者がディビジョン別(`scv_search_div`)。

クエリ:

| 名前 | 意味 |
|---|---|
| `ymd` | `YYYYMMDD`。**キー自体がない**とき、大会全体は最終日、ディビジョン別は全日程を既定にする。`ymd=`(空)は全日程 |
| `team` | 大会全体ではチームの正式名(日本語)、ディビジョン別ではチーム ID。左右どちらかがそのチームの試合 |
| `field` | コート。文字列の完全一致(数字とは限らない) |
| `sort` | `division`、`date`、`time`、`field`、`code`、`leftTeam`、`rightTeam` |
| `dir` | `asc` か `desc`。`sort` があるときだけ有効 |

既定の並びは日付、時刻、コートの昇順(コートは自然順: 数字のコートを数値順で先に、それ以外を文字列順で後に)。
`sort` を指定したときも、同じ値の中ではこの既定順を保つ(安定ソート)。
`leftTeam` と `rightTeam` は短縮名で、大文字小文字を区別しない。

```ts
{
  frame: TournamentFrame
  scope: 'tournament' | 'division'
  division: { id: number; name: I18nText } | null      // scope=division のとき
  filters: {
    dates: string[]                                    // 試合のある日付の昇順
    // 大会全体: 大会の全チーム。value は正式名(日本語)
    // ディビジョン別: そのディビジョンのチーム。value はチーム ID の文字列
    teams: Array<{ value: string; name: I18nText }>    // name は正式名
    fields: string[]                                   // 範囲内の試合で使われたコートの異なり値(自然順)
    selected: { ymd: string; team: string; field: string }  // 実際に適用した値(既定値を含む)
  }
  sort: { key: string; dir: 'asc' | 'desc' } | null
  rows: Array<MatchRow & { division: { id: number; name: I18nText; kind: DivisionKind } }>
}
```

### `GET /api/v1/divisions/{id}/standings`

ディビジョン成績。

```ts
{
  frame: TournamentFrame
  division: { id: number; name: I18nText }
  standings: Array<Tally & { team: TeamRef; rank: number | null }>  // rank 昇順(NULL は末尾)、同順位は sort_order 順
  leaders: Array<{                   // 得点+アシストの合計の降順、上位 30 件、合計 0 は除く
    team: TeamRef; player: PlayerRef
    games: number; assists: number; goals: number
  }>
  blockLeaders: Array<{ team: TeamRef; player: PlayerRef; games: number; blocks: number }>  // 上位 10 件、0 件は除く
}
```

### `GET /api/v1/groups/{id}`

グループ成績。`kind` に応じて `pool`、`bracket`、`image` のどれか 1 つが非 null になる。

```ts
{
  frame: TournamentFrame
  group: { id: number; name: string; kind: GroupKind; division: { id: number; name: I18nText } }
  pool: {
    teams: TeamRef[]                 // group_teams.sort_order 順(登録順)
    // teams と同じ順の N×N。対角 self、下三角 lower、上三角は試合があれば match、なければ empty
    matrix: Array<Array<
      | { kind: 'self' | 'lower' | 'empty' }
      | { kind: 'match'; matchId: number; status: MatchStatus; text: string }  // text は "9-3"(試合記録の左右順)
    >>
    results: Array<Tally & { team: TeamRef; points: number; rank: number | null }>  // teams と同じ順
  } | null
  bracket: {
    pattern: string                  // "T8A"
    ptnName: string
    columns: number
    rows: BracketCell[][]
  } | null
  image: { url: string; width: number | null; height: number | null } | null
  matches: MatchRow[]                // 日付、時刻の昇順
}

type BracketCell = {
  borders: { top: boolean; bottom: boolean; left: boolean; right: boolean }
  kind: '' | 'team' | 'win' | 'lose' | 'game' | 'txt_yen' | 'txt_sharp' | 'txt_and' | 'rank' | 'txt_left' | 'txt_right'
  text: string                       // team 参照がなければこの文字列を出す
  team: TeamRef | null               // チーム名のセル
  matchId: number | null             // game セル。スコア文字列は text に入る
  status: MatchStatus | null         // game セルの試合状態(試合中は赤で出す)
}
```

### `GET /api/v1/teams/{id}`

チームページ。

```ts
{
  frame: TournamentFrame
  team: {
    id: number
    fullName: I18nText; shortName: I18nText
    comment: I18nText
    layout: 'standard' | 'roster_only'     // roster_only はディスクゴルフ団体戦(成績表と対戦一覧を出さない)
  }
  record: Tally & { rank: number | null }
  matches: MatchRow[]                      // 日付、時刻の昇順
  players: Array<{
    player: PlayerRef
    games: number                          // チームの試合数(record.games)。全選手で同じ値
    assists: number; goals: number; blocks: number
    absent: boolean                        // 欠場。出場記録のあるチームで、どの試合にも出ていない選手
  }>                                       // (得点+アシスト)降順、得点降順、背番号の数値昇順
  staff: Array<{ id: number; role: I18nText; name: I18nText; gender: Gender | null }>
                                           // 役割の順(監督、コーチ、トレーナー、マネージャ、その他)、同じ役割は ID 順
}
```

### `GET /api/v1/matches/{id}`

試合ページ。

```ts
{
  frame: TournamentFrame
  match: {
    id: number; code: string
    // スコアボードのグループ欄。試合に別の文字があればそれ(大文字小文字の違う旧名、2017 年は空)、
    // なければグループの名前。グループがなければ null
    groupName: string | null
    date: string | null; time: string | null; field: string | null
    status: MatchStatus
    // fullName は正式名(ページの title 用)。team が null なら null
    left: { team: TeamRef | null; fullName: I18nText | null; score: number | null }
    right: { team: TeamRef | null; fullName: I18nText | null; score: number | null }
    note: string                             // スコアボード下の行の中央の備考。多くは空
    hideBlocks: boolean                      // 出場選手表に Block 列を出さない
    hideStats: boolean                       // 統計表を出さない
    breadcrumb: { divisionId: number; divisionName: I18nText; groupId: number | null }
  }
  // 時系列の昇順(古い順)。画面では web が逆順にして新しい順で出す
  events: Array<{
    elapsedSec: number
    kind: 'goal' | 'timeout' | 'pull' | 'end'
    side: Side | null
    scorer: PlayerRef | null; assist: PlayerRef | null
    callahan: boolean                        // キャラハンゴール(kind=goal のときだけ true になりうる)
    leftScore: number; rightScore: number   // そのイベント時点の累計
  }>
  lineups: {
    left: Array<{ player: PlayerRef; assists: number; goals: number; blocks: number }>
    right: Array<{ player: PlayerRef; assists: number; goals: number; blocks: number }>
  }                                          // (得点+アシスト)降順、得点降順、背番号の数値昇順
  // fromTurnover は fromDefense と同じ値(ディフェンス開始で得点するにはターンオーバーが要る。
  // 参照サイトの実データ 12 試合すべてで一致)。保存せずプルの向きから計算する
  stats: {
    left: { goals: number; fromOffense: number; fromDefense: number; fromTurnover: number; timeouts: number }
    right: { goals: number; fromOffense: number; fromDefense: number; fromTurnover: number; timeouts: number }
  }
}
```

### `GET /api/v1/players/{id}`

選手ページ(スタッフも同じエンドポイント)。

```ts
{
  frame: TournamentFrame
  player: PlayerRef & { role: I18nText | null }    // role があればスタッフ
  team: TeamRef
  prev: PlayerRef & { role: I18nText | null }      // 同じチーム内の ID 順で 1 つ前(先頭なら末尾へ循環)
  next: PlayerRef & { role: I18nText | null }
  statsKind: StatsKind
  // callahans は本人が得点者のキャラハンゴールの数
  stats: { games: number; assists: number; goals: number; callahans: number; greatests: number; blocks: number }
  // 本人が得点かアシストをしたイベント。試合の日時順、試合内は経過時間順
  events: Array<{
    code: string; date: string | null; opponent: TeamRef | null
    elapsedSec: number
    callahan: boolean               // キャラハンゴール(イベント名に使う)
    scorer: PlayerRef | null; assist: PlayerRef | null
  }>
}
```

### `GET /api/v1/galleries/{id}`

```ts
{
  frame: TournamentFrame
  gallery: { id: number; name: string; photos: Array<{ thumbUrl: string; fullUrl: string }> }
}
```

## 管理用エンドポイント

`/api/v1/admin/` 以下は `Authorization: Bearer {sessionToken}` が必要。
記録係(staff)は `tournament_staff` に登録された大会の試合とイベントだけを操作できる。
それ以外はすべて管理者(admin)専用。

### 認証

| メソッドとパス | ボディ | 応答 |
|---|---|---|
| `POST /api/v1/auth/login` | `{ email, password }` | `{ token, expiresAt, user }`。失敗は 401(メールアドレスの有無を区別しない文面) |
| `POST /api/v1/auth/logout` | なし | 204 |
| `GET /api/v1/auth/me` | なし | `{ user: { id, email, displayName, role } }` |

### 登録と編集

各リソースは `GET`(一覧または単体)、`POST`(作成)、`PATCH`(部分更新)、`DELETE` を持つ。
ボディのキーは閲覧用の型に合わせて camelCase にする(例 `nameJa` ではなく `name: { ja, en }`)。
詳細は実装時に本節へ追記する。

| リソース | パス |
|---|---|
| 大会 | `/api/v1/admin/tournaments`、`/api/v1/admin/tournaments/{id}` |
| バナー | `/api/v1/admin/tournaments/{id}/banners` |
| ディビジョン | `/api/v1/admin/tournaments/{id}/divisions`、`/api/v1/admin/divisions/{id}` |
| グループ | `/api/v1/admin/divisions/{id}/groups`、`/api/v1/admin/groups/{id}`、`PUT /api/v1/admin/groups/{id}/teams` |
| トーナメント表の生成 | `POST /api/v1/admin/groups/{id}/bracket`(ボディ `{ template }`)、`GET /api/v1/admin/bracket-templates` |
| チーム | `/api/v1/admin/divisions/{id}/teams`、`/api/v1/admin/teams/{id}` |
| 選手とスタッフ | `/api/v1/admin/teams/{id}/players`、`/api/v1/admin/players/{id}` |
| 試合 | `/api/v1/admin/divisions/{id}/matches`、`/api/v1/admin/matches/{id}` |
| 試合イベント | `POST /api/v1/admin/matches/{id}/events`、`DELETE /api/v1/admin/matches/{id}/events/{eventId}` |
| 試合ごとの選手記録 | `PUT /api/v1/admin/matches/{id}/player-stats` |
| 試合の状態 | `POST /api/v1/admin/matches/{id}/status`(ボディ `{ status, winnerSide }`) |
| 最終順位 | `PUT /api/v1/admin/divisions/{id}/final-ranks`(ボディ `[{ teamId, rank }]`) |
| ギャラリー | `/api/v1/admin/tournaments/{id}/galleries`、`/api/v1/admin/galleries/{id}`、`PUT /api/v1/admin/galleries/{id}/photos` |
| ユーザー | `/api/v1/admin/users`、`PUT /api/v1/admin/users/{id}/tournaments` |
