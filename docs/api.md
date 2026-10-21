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
| `score_mismatch` | 409 | 試合終了などで記録係が確認したスコア(`expectedScore`)が現在のスコアと違う。`message` に現在のスコアを書く |
| `too_many_requests` | 429 | ログイン試行の制限を超えた(同じメールアドレスで 15 分に 10 回失敗) |
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
// 試合の結果の種類(docs/format-engine.md「試合の結果の種類」)。管理用の応答だけに出す
type ResultType = 'normal' | 'forfeit' | 'double_forfeit' | 'no_contest' | 'abandoned' | 'unplayed'
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
    // 日程があるか(divisions.schedule_enabled)。false のディビジョンはナビのスケジュールと
    // ディビジョン成績に出ず、SCORE メニューではリンクのない項目になる
    scheduleEnabled: boolean
    menuSuffix: string               // SCORE メニューで名前の後ろに付く競技の略号("Ult" "Free" "DiscGolfT")
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
  // 結果の種類。通常以外は閲覧ページの状況列に種類を出す(本サイト独自。requirements の意図的な差分)
  resultType: 'normal' | 'forfeit' | 'double_forfeit' | 'no_contest' | 'abandoned' | 'unplayed'
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
  // rank 昇順(NULL は末尾)、同順位は勝ち数の降順、得失差の降順、sort_order 順
  standings: Array<Tally & { team: TeamRef; rank: number | null }>
  leaders: Array<{                   // 得点+アシストの合計の降順、上位 30 件、合計 0 は除く
    team: TeamRef; player: PlayerRef
    games: number; assists: number; goals: number   // games は選手のチームの試合数
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
      // text は "9-3"(試合記録の左右順)。スコアのない未試合は試合の CD("A1")
      | { kind: 'match'; matchId: number; status: MatchStatus; text: string }
    >>
    results: Array<Tally & { team: TeamRef; points: number; rank: number | null }>  // teams と同じ順
    // 本サイト独自の補足(requirements.md「意図的な差分」)。文は日本語
    notes: {
      reasons: Array<{ team: TeamRef; text: string }>     // 同点を分けた根拠「勝ち点同数、得失点差で上位」。順位を手で上書きしたプールは空
      conditions: Array<{ team: TeamRef; text: string }>  // 進行中のプールの勝ち上がり条件「次の A3 に勝てば 1 位確定」
    }
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
  showSotg: boolean                        // SOTG スコア一覧を出す(行は matches と同じ試合、値は空)
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

`/api/v1/admin/` 以下と `/api/v1/auth/logout`、`/api/v1/auth/me` は `Authorization: Bearer {sessionToken}` が必要。
トークンが無い、未知、期限切れなら 401(`unauthorized`)。

記録係(staff)は `tournament_staff` に登録された大会について、次の操作だけができる。
それ以外はすべて管理者(admin)専用で、staff が呼ぶと 403(`forbidden`)。

- 読み取り: 大会一覧(担当大会だけが返る)、大会、ディビジョン一覧、ディビジョン、試合一覧、試合(スコア入力画面のデータ)、試合の監査ログ、自分の担当試合
- 変更: 試合の状態、結果の種類、得点経過の追加・取り消し・得点者の付け直し、試合ごとの選手記録。
  ただし**記録係の割り当て**(`staff_assignments`)が覆う試合だけ。割り当ての無い試合を変えると 403

パスの資源が存在しなければ、権限の判定より先に 404 を返す。

### 共通の規則

- 以下の表のパスは `/api/v1` を省いて書く(`/admin/tournaments` は `/api/v1/admin/tournaments`)。
- 作成(`POST`)は 201 と作った資源、部分更新(`PATCH`)は 200 と更新後の資源、削除(`DELETE`)は 204(本文なし)を返す。
  一覧は `{ 複数形のキー: [...] }` で返す(例 `{ tournaments: [...] }`)。
- ボディは JSON 1 つ。未知のキー、型の違い、1 MiB を超えるボディは 400。
- `PATCH` のボディは作成と同じ形で、**ボディにあるキーだけ**を変える。入れ子のオブジェクト(`name` など)もキーごとに変える(`{"name":{"en":"X"}}` は `name.ja` を変えない)。
  キーに `null` を入れるとその値を消す。
- 作成でボディに無いキーは、各型の「既定」の値になる。
- URL の項目(ロゴ、バナー、写真、グループの画像)は `http(s)://` の絶対 URL か `/` で始まるサイト内のパスだけを受け付ける(`javascript:` などは 400)。
- 自由記述 HTML(`topHtml`)は**保存時に**無害化する。応答と閲覧用の `topHtml` は無害化後の値になる。

### 認証

```ts
type AdminUser = {
  id: number; email: string; displayName: string; role: 'admin' | 'staff'
  tournamentIds: number[]          // 担当大会(tournament_staff)。昇順
}
```

| メソッドとパス | ボディ | 応答 |
|---|---|---|
| `POST /api/v1/auth/login` | `{ email, password }` | `{ token, expiresAt, user: AdminUser }`。`expiresAt` は RFC 3339(UTC)。失敗は 401 で、メールアドレスの有無を区別しない文面。試行の制限を超えたら 429 |
| `POST /api/v1/auth/logout` | なし | 204 |
| `GET /api/v1/auth/me` | なし | `{ user: AdminUser }` |

- メールアドレスは前後の空白を除いた小文字で比べる。
- セッションの有効期間はログインから 24 時間(延長しない)。トークンは 32 バイトの乱数の base64url で、api は SHA-256 だけを保存する。
- ログイン試行の制限: 同じメールアドレスで 15 分に 10 回失敗すると、古い失敗が 15 分の窓から外れるまで照合せずに 429 を返す。存在しないアドレスへの試行も同じく数える。成功すると失敗の記録を消す。

### 大会とバナー

```ts
type TournamentInput = {
  name: I18nText                     // name.ja は必須
  shortName: string
  status: TournamentStatus           // 既定 'entry_open'
  listed: boolean                    // 既定 true。false なら大会一覧に出さない
  startDate: string | null           // "YYYY-MM-DD"。大会一覧の並び順
  logoUrl: string | null
  topHtml: I18nText                  // 保存時に無害化
  labels: { teamList: 'teams' | 'entries'; groups: 'groups' | 'results' }  // 既定 teams, groups
  showSchedule: boolean              // 既定 true
  // 不戦勝の規定スコア。既定 null(未設定)。winner > loser >= 0
  forfeitScore: { winner: number; loser: number } | null
}
type AdminTournament = { id: number } & TournamentInput
type Banner = { imageUrl: string; linkUrl: string | null }
```

| メソッドとパス | ボディ | 応答 |
|---|---|---|
| `GET /admin/tournaments` | なし | `{ tournaments: AdminTournament[] }`(開催日の新しい順。staff は担当大会だけ) |
| `POST /admin/tournaments` | `TournamentInput` | `AdminTournament` |
| `GET` / `PATCH` / `DELETE /admin/tournaments/{id}` | `PATCH` は `TournamentInput` の一部 | `AdminTournament` / 204。削除は配下を全部消す |
| `GET /admin/tournaments/{id}/banners` | なし | `{ banners: Banner[] }`(表示順) |
| `PUT /admin/tournaments/{id}/banners` | `{ banners: Banner[] }`(この順で表示。最大 30) | `{ banners: Banner[] }` |

### ディビジョンと最終順位

```ts
type DivisionInput = {
  name: I18nText                     // name.ja は必須
  kind: DivisionKind                 // 既定 'none'
  sortOrder: number                  // 既定 0
  scheduleEnabled: boolean           // 既定 true。大会トップの [スケジュール] をリンクにする
  standingsPublished: boolean        // 既定 false。大会トップの [最終順位] を公開する
  entryKind: EntryKind               // 既定 'team'
  statsKind: StatsKind               // 既定 'ultimate'
}
type AdminDivision = { id: number; tournamentId: number } & DivisionInput
```

| メソッドとパス | ボディ | 応答 |
|---|---|---|
| `GET /admin/tournaments/{id}/divisions` | なし | `{ divisions: AdminDivision[] }`(sort_order, id 順) |
| `POST /admin/tournaments/{id}/divisions` | `DivisionInput` | `AdminDivision` |
| `GET` / `PATCH` / `DELETE /admin/divisions/{id}` | | `AdminDivision` / 204 |
| `PUT /admin/divisions/{id}/final-ranks` | `Array<{ teamId: number; rank: number \| null }>` | `{ teams: AdminTeam[] }` |

最終順位は `teams.final_rank` をまとめて書く(配列に無いチームは変えない。同順位可。`null` で消す)。
公開は `PATCH /admin/divisions/{id}` の `standingsPublished` で切り替える。

### グループとトーナメント表

```ts
type GroupInput = {
  name: string                       // 必須。参照サイトの表記ゆれもそのまま
  kind: GroupKind                    // 既定 'pool'
  sortOrder: number                  // 既定 0
  linked: boolean                    // 既定 true
  published: boolean                 // 既定 true。false ならナビに出さない
  image: { url: string; width: number | null; height: number | null } | null  // kind='image' なら必須
}
type AdminGroup = {
  id: number; divisionId: number
  bracket: { pattern: string; ptnName: string } | null   // 生成済みのトーナメント表の識別子
  teams: Array<{ teamId: number; rankOverride: number | null }>  // group_teams の登録順
} & GroupInput
type MatchSideInput = {
  teamId: number | null
  placeholder: { code: string; outcome: 'win' | 'lose' } | null  // "[A1-勝ち]" は { code: 'A1', outcome: 'win' }
}
```

| メソッドとパス | ボディ | 応答 |
|---|---|---|
| `GET /admin/divisions/{id}/groups` | なし | `{ groups: AdminGroup[] }`(非公開も含む) |
| `POST /admin/divisions/{id}/groups` | `GroupInput` | `AdminGroup`。`kind='bracket'` は空の表で作る |
| `GET` / `PATCH` / `DELETE /admin/groups/{id}` | | `AdminGroup` / 204。削除しても試合は残る(group が null になる) |
| `PUT /admin/groups/{id}/teams` | `{ teams: Array<{ teamId; rankOverride }> }`(この順で対戦表と順位表に出す) | `AdminGroup` |
| `GET /admin/bracket-templates` | なし | `{ templates: Array<{ id: string; name: I18nText; teams: number }> }` |
| `POST /admin/groups/{id}/bracket` | `{ template: string; codePrefix: string; seeds: MatchSideInput[] }` | `AdminGroup` |

トーナメント表の生成は、テンプレート(`single4`、`single8`、`single16`)の格子と試合を 1 回で作る。
テンプレートは大会形式エンジンの型(`single4` は T4、`single8` は T8 全順位型(1〜8 位、12 試合)、`single16` は T16(1〜8 位、20 試合))で作る。

- `seeds` は表の上から順の枠で、数はテンプレートの `teams` と同じ。各枠は `teamId`(確定したチーム)か `placeholder`(未確定の参照。前のグループの試合など)のどちらか一方。確定したチームは同じディビジョンのチームに限り、グループのチームになる。
- 試合 CD は `codePrefix` と番号。番号は参照サイトと同じく決勝を 1 として 1 回戦へ向かって振り、その後に 3 位決定、5 位決定などを続ける。例 `single8` と `"B"` なら決勝が B1、準決勝が B2 と B3、1 回戦が B4〜B7(上から)、3 位決定戦が B8、5 位決定戦が B9、5〜8 位の敗者戦が B10 と B11、7 位決定戦が B12。
- 後続の試合の左右は前の試合を `placeholder` で参照する(B2 の左は `[B4-勝ち]`、B8 の左は `[B2-負け]`)。
- グループに試合が既にあるか、ディビジョンに同じ CD の試合があれば 409(作り直すときは試合を消してから)。
- 格子は閲覧用の `bracket` で返る。スコアが入るまで game セルの文字列は試合 CD。

### チームと選手

```ts
type TeamInput = {
  fullName: I18nText                 // fullName.ja は必須。加工せずに保存する
  shortName: I18nText                // shortName.ja は必須
  comment: I18nText
  barStyle: TeamBarStyle             // 既定 'default'
  sortOrder: number
  finalRank: number | null
}
type AdminTeam = { id: number; tournamentId: number; divisionId: number } & TeamInput
type PlayerInput = {
  number: string                     // "00" のような表記を保つ
  name: I18nText                     // name.ja は必須
  gender: Gender | null
  role: I18nText | null              // null なら選手、値があればスタッフ(role.ja 必須)
}
type AdminPlayer = { id: number; teamId: number } & PlayerInput
```

| メソッドとパス | ボディ | 応答 |
|---|---|---|
| `GET /admin/divisions/{id}/teams` | なし | `{ teams: AdminTeam[] }` |
| `POST /admin/divisions/{id}/teams` | `TeamInput` | `AdminTeam` |
| `GET` / `PATCH` / `DELETE /admin/teams/{id}` | | `AdminTeam` / 204 |
| `GET /admin/teams/{id}/players` | なし | `{ players: AdminPlayer[] }`(ID 順。スタッフを含む) |
| `POST /admin/teams/{id}/players` | `PlayerInput` | `AdminPlayer` |
| `GET` / `PATCH` / `DELETE /admin/players/{id}` | | `AdminPlayer` / 204 |

### 試合

```ts
type MatchInput = {
  groupId: number | null             // 同じディビジョンのグループ
  code: string                       // 試合 CD。未確定チームの参照に使う
  date: string | null                // "YYYY-MM-DD"
  time: string | null                // "HH:MM"
  field: string | null               // コート
  hasDetail: boolean                 // 既定 true
  left: MatchSideInput               // teamId は同じディビジョンのチーム。左右で同じチームは不可
  right: MatchSideInput
}
type AdminMatch = {
  id: number; divisionId: number; groupId: number | null
  code: string; date: string | null; time: string | null; field: string | null
  hasDetail: boolean; status: MatchStatus
  left: MatchSide; right: MatchSide  // 閲覧用と同じ形。ただし team が解決済みでも placeholder を返す
  winnerSide: Side | null            // 同点で勝者を決めた試合と、不戦勝・不成立の勝者
  resultType: ResultType
}
type MatchDetail = {
  match: AdminMatch
  context: {                         // 見出しとスコアシートに出す名前
    tournamentId: number; tournamentName: string; divisionName: string; groupName: string | null
  }
  inBracket: boolean                 // トーナメント(bracket グループ)の試合
  forfeitScore: { winner: number; loser: number } | null  // 不戦勝の規定スコア(大会、無ければ大会の形の設定)
  startedAt: string | null           // 経過 0 秒の時刻(RFC 3339、UTC)。得点経過が無ければ null
  events: Array<{                    // 時系列の昇順
    id: number; clientId: string     // clientId は端末が付けた UUID(無ければ api が付けた UUID)
    seq: number; elapsedSec: number
    kind: 'goal' | 'timeout' | 'pull' | 'end'; side: Side | null
    scorer: PlayerRef | null; assist: PlayerRef | null; callahan: boolean
    leftScore: number; rightScore: number       // その時点の累計
  }>
  playerStats: Array<{ playerId: number; side: Side; blocks: number; greatests: number }>
  rosters: { left: AdminPlayer[]; right: AdminPlayer[] }   // 左右のチームの選手とスタッフ
  scorer: ScorerClaim | null         // 試合の記録担当(期限内のもの。下記「試合の記録担当」)
  concurrentRecords: Array<{         // 別の利用者がほぼ同時(5 秒以内)に入れた同じ種類の記録の組(二重記録の疑い)
    kind: 'goal' | 'pull'
    firstClientId: string; secondClientId: string   // 得点経過の clientId。second が後
    firstUser: string; secondUser: string           // 記録した利用者の表示名
    secondsApart: number
  }>
  matchMinutes: number | null        // ディビジョンの時間割の 1 試合の枠の長さ(分)。設定が無ければ null
}
```

| メソッドとパス | ボディ | 応答 |
|---|---|---|
| `GET /admin/divisions/{id}/matches` | なし | `{ matches: AdminMatch[] }`(日付、時刻、コートの自然順) |
| `POST /admin/divisions/{id}/matches` | `MatchInput` | `AdminMatch`(未試合で作る) |
| `GET /admin/matches/{id}` | なし | `MatchDetail` |
| `PATCH /admin/matches/{id}` | `MatchInput` の一部 | `AdminMatch`。スコアと状態は変えない |
| `DELETE /admin/matches/{id}` | なし | 204 |
| `POST /admin/matches/{id}/events` | 下記 | 201 と `MatchDetail` |
| `DELETE /admin/matches/{id}/events/{eventId}` | なし(クエリ `clientId` で操作 ID) | 200 と `MatchDetail`(取り消し後の状態を返す) |
| `PATCH /admin/matches/{id}/events/{eventId}` | `{ clientId?, scorerId, assistId, callahan }` | `MatchDetail`。goal の得点者、アシスト、キャラハンを付け直す。終了した試合でも可 |
| `PUT /admin/matches/{id}/events/{eventId}/side` | `{ clientId?, side: Side }` | `MatchDetail`。goal を相手チームの得点に付け替える(得点者、アシスト、キャラハンは外す)。既に `side` の得点か、終了した試合なら 409。監査ログは `event.side` の 1 行 |
| `POST /admin/matches/{id}/scorer` | `{ deviceId: string; takeover?: boolean }` | `{ scorer: ScorerClaim; mine: boolean }`。下記「試合の記録担当」 |
| `DELETE /admin/matches/{id}/scorer` | なし(クエリ `deviceId`) | 204。その端末が記録担当なら外す |
| `PUT /admin/matches/{id}/player-stats` | `{ stats: Array<{ playerId; side; blocks; greatests }> }` | `MatchDetail` |
| `POST /admin/matches/{id}/status` | 下記 | `MatchDetail` |
| `POST /admin/matches/{id}/result` | 下記 | `MatchDetail`。通常以外の結果で試合を終える |
| `GET /admin/matches/{id}/audit` | なし | `{ entries: AuditEntry[] }`(新しい順。下記「監査ログ」) |

`{eventId}` は得点経過の `id`(正の整数)か `clientId`(UUID)。通信が切れている間に記録した得点経過は、端末が api の `id` を知らないので `clientId` で指す。

**操作 ID(`clientId`)と冪等性**: 試合を変える操作(得点経過の追加・取り消し・付け直し、状態の変更、結果の種類)には、端末が UUID の `clientId` を付けられる。
同じ `clientId` の操作が既に適用済みなら、2 回目は何もせずに現在の `MatchDetail` を返す(得点経過の追加なら 201、他は 200)。
端末は通信が切れている間の操作を溜め、つながったら順に送り直す。応答が届かずに送り直しても二重に登録されない。
UUID の形でない `clientId` は 400。得点経過の追加では `clientId` がその得点経過の `clientId` になる。

得点経過の追加のボディ:

```ts
{
  clientId?: string | null           // 操作 ID(UUID)
  kind: 'goal' | 'timeout' | 'pull' | 'end'
  side: Side | null                  // goal は得点した側、timeout は取った側、pull はプルした側。end は null
  elapsedSec?: number | null         // 省略か null なら、最初のイベントを記録した時刻からの経過秒
  scorerId?: number | null           // goal だけ。side のチームの選手
  assistId?: number | null           // goal だけ。得点者と別の選手。キャラハンなら null
  callahan?: boolean                 // goal だけ
  winnerSide?: Side | null           // end だけ。同点のトーナメントの試合では必須
  expectedScore?: { left: number; right: number } | null
                                     // end だけ。記録係が確認したスコア。現在のスコアと違えば 409(score_mismatch)
}
```

状態の変更のボディ:

```ts
{
  clientId?: string | null           // 操作 ID(UUID)
  status: MatchStatus
  winnerSide?: Side | null           // finished にするときだけ。同点のトーナメントの試合では必須
  score?: { left: number; right: number } | null  // 得点経過の無い試合(詳細なし)の結果。経過がある試合では指定不可
  expectedScore?: { left: number; right: number } | null  // finished にするときだけ。end と同じ
  downstream?: 'keep' | 'reset' | null  // 結果のある下流の試合の対戦が変わるときの承認(下記「大会の形と進行」)
}
```

結果の種類のボディ(`POST /admin/matches/{id}/result`):

```ts
{
  clientId?: string | null           // 操作 ID(UUID)
  resultType: Exclude<ResultType, 'normal'>
  winnerSide?: Side | null           // forfeit と no_contest は必須。abandoned は同点のトーナメントの試合で必須。
                                     // double_forfeit と unplayed は指定不可
  forfeitScore?: { winner: number; loser: number } | null
                                     // forfeit だけ。規定スコア(大会、無ければ大会の形の設定)があるときは
                                     // 省略するか同じ値。どちらも無く省略すれば勝者だけを記録する
  expectedScore?: { left: number; right: number } | null  // 記録係が確認したスコア。end と同じ
}
```

| 種類 | 記録する値 | 成績の集計(チーム、プール、勝ち点) |
|---|---|---|
| `forfeit`(不戦勝) | 規定スコア(大会、無ければ大会の形の設定。どちらも無ければ `forfeitScore`)を勝者の側に。それも無ければスコアなし(本部が後で決める)。`winnerSide` | 記録したスコアで勝ち・負け。スコアが無ければ勝ち・負けだけ |
| `double_forfeit`(両チーム不戦敗) | スコアなし(`null`) | 両チームとも負け。得失点なし |
| `no_contest`(不成立) | スコアなし。`winnerSide` | 勝ち・負けだけ。得失点なし |
| `abandoned`(途中打ち切り) | その時点の得点経過のスコア | 通常の試合と同じ |
| `unplayed`(未消化。途中打ち切りを未消化にした場合も) | その時点のスコア(あれば) | 数えない |

- 試合前でも試合中でも `finished` になる。終了した試合には 409(結果を変えるときは管理者が `status: 'live'` へ戻す)。
- 試合中へ戻すと `resultType` は `normal` に戻り、スコアは得点経過から計算し直す(規定スコアを引き継がない)。
- 勝者がいれば、通常の終了と同じく後続試合の未確定チームを埋める。両チーム不戦敗と未消化は埋めない。

試合のスコアと状態は、得点経過と状態の変更からだけ決まる。

- スコアは得点経過の goal の数で、追加と取り消しのたびに計算し直す。
- 状態は `scheduled → live → finished` の順に進む(飛び越しも可)。未試合の試合は最初のイベントで `live`、`end` の追加で `finished` になる。
- `finished → live` の戻し(`end` の取り消し、または状態の変更)は admin だけ。状態の変更で戻すと `end` の得点経過を消す。それ以外の戻しは 409。
- 終了した試合には得点経過を足せない(409)。取り消せるのは `end` だけ。
- 終了時、スコアに差があれば勝者はスコアで決まり、`winnerSide` は保存しない(スコアと矛盾する指定は 400)。同点なら `winnerSide` を保存し、トーナメントの試合で指定が無ければ 400。
- 試合が終わると、同じディビジョンで、この試合の CD を `placeholder` で参照する未試合の試合のチームを、勝者または敗者で埋める。終了を取り消すと空に戻す(既に始まった後続試合は変えない)。トーナメント表の格子のチーム名も同時に書き直す。

#### 試合の記録担当

同じ試合を 2 台で記録すると得点が二重に入るので、記録画面を開いた端末を「記録担当」にして、他の端末には引き継ぎを確かめさせる。

```ts
type ScorerClaim = {
  userId: number; name: string       // 記録担当の利用者と表示名(未登録ならメールアドレス)
  deviceId: string                   // 端末が作った UUID
  expiresAt: string                  // RFC 3339(UTC)。この時刻までに延長が無ければ空く
}
```

- `POST /admin/matches/{id}/scorer` は、記録担当がいない、期限切れ、または同じ利用者の同じ端末なら、この端末を記録担当にして期限を 30 秒後に延ばす(`mine: true`)。
  他の端末が期限内の記録担当なら、`takeover: true` のときだけ引き継ぐ。`false` なら変えずに今の記録担当を返す(`mine: false`)。
- 記録画面は開いている間 10 秒ごとに延長し、閉じるときに `DELETE` で外す。期限が切れた記録担当は `MatchDetail.scorer` に出ない。
- 記録担当は画面の案内のための仕組みで、得点経過の追加などの操作は記録担当でなくても受け付ける(通信が切れている間に溜めた操作を捨てないため)。
  同じ種類の記録がほぼ同時に別の利用者から入ったら `concurrentRecords` に出す。
- 権限は得点経過の追加と同じ(記録係は割り当てが覆う試合だけ)。`deviceId` が UUID でなければ 400。

### 監査ログ

スコア、得点経過、試合の状態、結果の種類、最終順位の変更は、変更と同じトランザクションで 1 操作 1 行の監査ログに残る(docs/format-engine.md「監査ログ」)。

```ts
type AuditMatchState = {
  status: MatchStatus; resultType: ResultType
  leftScore: number | null; rightScore: number | null; winnerSide: Side | null
}
type AuditEvent = {
  clientId: string; kind: 'goal' | 'timeout' | 'pull' | 'end'; side: Side | null; elapsedSec: number
  scorer: { id: number; number: string; name: string } | null   // 記録した時点の背番号と名前
  assist: { id: number; number: string; name: string } | null
  callahan: boolean
}
type AuditEntry = {
  id: number
  at: string                         // RFC 3339(UTC)
  user: { id: number; name: string; role: 'admin' | 'staff' | null } | null  // 消された利用者は name が空、role が null
  action: 'event.add' | 'event.remove' | 'event.update' | 'event.side' | 'match.status' | 'match.result' | 'team.final_rank'
    | 'match.align_teams'            // 対戦チームの差し替え(下記「大会の形と進行」)
  matchId: number | null; teamId: number | null
  // 変更の前と後。試合の操作は match(と得点経過の操作なら event)、最終順位は finalRank を持つ
  // note は操作の補足(「得点経過(9-8)と違う最終スコア(11-9)で本部が確定」「棄権の一括処理」)。無ければ null
  before: { match: AuditMatchState | null; event: AuditEvent | null; finalRank: number | null; note: string | null }
  after: { match: AuditMatchState | null; event: AuditEvent | null; finalRank: number | null; note: string | null }
}
```

最終順位の変更(`PUT /admin/divisions/{id}/final-ranks` と `PATCH /admin/teams/{id}` の `finalRank`)は、値が変わったチームごとに 1 行残す。

### 記録係の割り当て

記録係(staff)を、試合 1 つ、またはコート・日付・時間帯の範囲に割り当てる。
範囲の条件は `null` なら問わない(全部 `null` なら大会の全試合)。時間帯は `from` 以上 `to` 未満で、試合の開始時刻で判定する。
コートや時刻が決まっていない試合は、その条件のある割り当てでは覆わない。

```ts
type AssignmentInput = {
  userId: number                     // 役割 staff のユーザー
  matchId: number | null             // 同じ大会の試合。指定したら他の条件は null
  field: string | null               // コート
  date: string | null                // "YYYY-MM-DD"
  from: string | null                // "HH:MM"
  to: string | null                  // "HH:MM"。from より後
}
type Assignment = { id: number; userName: string; matchCode: string | null } & AssignmentInput
type AssignedMatch = AdminMatch & { tournamentId: number; tournamentName: string; divisionName: string }
```

| メソッドとパス | ボディ | 応答 |
|---|---|---|
| `GET /admin/tournaments/{id}/assignments` | なし | `{ assignments: Assignment[] }`(記録係、日付、時刻、コートの順) |
| `POST /admin/tournaments/{id}/assignments` | `AssignmentInput` | `Assignment`。記録係がまだ大会の担当でなければ担当にもする |
| `DELETE /admin/assignments/{id}` | なし | 204 |
| `GET /admin/me/matches` | なし | `{ matches: AssignedMatch[] }`。自分の割り当てが覆う試合(日付、時刻、コートの順)。誰でも呼べる |

### 大会の形と進行

大会形式エンジン(docs/format-engine.md)で、ディビジョンの形(プール、順位決定戦、時間割)を作り、当日の確定と訂正を行う。
進行状況の読み取り(`progress`、`format/matches`)は担当大会の記録係にも許し、それ以外は管理者だけ。
形の無いディビジョンへの進行の操作は 409。

```ts
type Sport = 'seven' | 'four' | 'beach'            // 7 人制 / 4on4 / ビーチ
type Tiebreak = 'current' | 'current_no_h2h' | 'jfda'  // 現行方式(直接対決あり)/ 直接対決なし / JFDA 競技規則
type FormatParams = object                          // テンプレートごとの JSON(docs/format-engine.md「形の指定の JSON」)
type ScheduleSetting = {
  date: string; start: string                      // "YYYY-MM-DD"、"HH:MM"
  slotMinutes: number                               // 1 試合の枠の長さ(分)
  courts: string[]                                  // コートの名前(この順に使う)
  teamRestSlots: number                             // 同じチームの試合の間に空ける枠の数(4on4 は 0)
  stageGapSlots: number                             // 段の間に空ける枠の数
  end: string | null                                // 終了時刻 "HH:MM"
}
type FormatInput = {
  template: 'pool_bands' | 'blocks' | 'round_robin'
  params: FormatParams
  seeds: number[]                                   // シード順のチーム。数は形のチーム数と同じ
  tiebreak: Tiebreak                                // 省略時 'current'
  forfeit: { winner: number; loser: number } | null // 不戦勝の規定スコア。null は 15-0。大会の設定があればそちらを順位計算に使う
  schedule: ScheduleSetting | null                  // null なら時刻とコートを入れない
}
type Violation = { kind: string; codes: string[]; shortMinutes: number; detail: string }  // detail は運営者向けの文
type FormatPreview = {
  title: string                                     // 「4 プール(各 3 チーム) → T4 × 3(1st / 5th / 9th)」
  startLetter: string                               // 最初のグループ文字(大会内の通し)
  teams: number; stages: string[]
  groups: Array<{
    index: number; letter: string; name: string; kind: GroupKind; stage: number
    slots: Array<{ position: number; label: string; seed: number | null; team: TeamRef | null }>  // label は「シード3」「A1」「[E2-勝ち]」
    bracket: { pattern; ptnName; columns; rows: BracketCell[][] } | null  // 閲覧用の格子と同じ形
  }>
  matches: Array<{
    code: string; group: number; round: number
    left: { label: string; team: TeamRef | null }; right: { label: string; team: TeamRef | null }
    date: string | null; time: string | null; field: string | null
  }>
  courts: string[]; violations: Violation[]; end: string | null   // end は最後の試合の終了時刻
  ruleText: string[]                                // タイブレークの案内文(参加チームへの案内にそのまま使える)
  advancement: string[]                             // 勝ち上がりの案内文
  ranked: number; unique: number                    // 全試合後に順位が付くチーム数、そのうち同順位でないチーム数
  replacesGroups: number; replacesMatches: number   // 適用すると消える既存のグループと試合の数
}
type SavedFormat = {
  divisionId: number; tournamentId: number; tournamentName: string; divisionName: string
  title: string; version: number                    // version は適用のたびに増える
  template: string; params: FormatParams; tiebreak: Tiebreak
  forfeit: { winner: number; loser: number }; schedule: ScheduleSetting | null
  seeds: number[]                                   // 一覧(GET /admin/formats)では空
}
// label は管理画面の表記(「A 組 1 位」「E2 の勝ちチーム」「シード 3」)。閲覧ページの "[E2-勝ち]" とは別
type ProgressSide = { label: string; team: TeamRef | null; score: number | null }
type ProgressMatch = {
  id: number; code: string; groupName: string; date: string | null; time: string | null; field: string | null
  groupLabel: string                                // 管理画面のグループ名(「予選 A 組」「1〜4 位を決めるトーナメント」)
  status: MatchStatus; left: ProgressSide; right: ProgressSide
}
type Progress = {
  divisionId: number; title: string; version: number; tiebreak: Tiebreak
  ruleText: string[]; advancement: string[]; slotMinutes: number; courts: string[]
  now: string                                       // 判定に使った現在時刻(ISO 8601)
  // いまやること。緊急度の順(overdue → overlap と mismatch → confirm_group と confirm_stage → tie →
  // court_delay)に並べて返す。overlap は同じチームが時刻の重なる 2 試合に入っている(両方終わったものは除く)
  todo: Array<{
    kind: 'tie' | 'confirm_group' | 'confirm_stage' | 'overdue' | 'court_delay' | 'mismatch' | 'overlap'
    text: string                                    // 運営者向けの文
    groupId: number | null; stageId: number | null; matchId: number | null; court: string | null
    minutes: number                                 // overdue と court_delay の遅れ(分)
  }>
  stages: Array<{ id; index; name; confirmed: boolean; confirmedUnplayed: boolean; needsConfirm: boolean }>
  groups: Array<{
    id: number; letter: string; name: string; kind: GroupKind; stage: number
    label: string                                   // 管理画面のグループ名(name のデータは変えない)
    state: 'waiting' | 'playing' | 'tied' | 'ready' | 'confirmed' | 'finished'
    confirmedUnplayed: boolean; remaining: number   // remaining は終わっていない試合の数
    slots: ProgressSide[]
    rows: Array<Tally & { team: TeamRef; rank: number; points: number; reason: string; unresolved: boolean }>  // プールの順位(枠が埋まっていれば)
    ties: TeamRef[][]                               // 未決の同順位の組(プールの全試合が終わった後だけ)
    conditions: Array<{ team: TeamRef; text: string }>  // 進行中のプールの勝ち上がり条件「次の A3 に勝てば 1 位確定」
  }>
  live: ProgressMatch[]                             // 試合中
  next: ProgressMatch[]                             // 次の時間帯(まだ始まっていない最も早い開始時刻の試合)
  final: Array<{ rank: number; team: TeamRef; tied: boolean }>
}
type Impact = {
  lines: string[]                                   // 「Qualifying pool A の 2 位と 3 位が入れ替わります(…)」
  matches: Array<{ matchId; code; groupName; beforeLeft: TeamRef | null; beforeRight; afterLeft; afterRight; hasResult: boolean }>
  needsApproval: boolean                            // 既に結果のある試合の対戦が変わる
}
```

| メソッドとパス | ボディ | 応答 |
|---|---|---|
| `POST /admin/divisions/{id}/format/proposals` | `{ teams, courts, slotMinutes, date, start, sport }` | `{ proposals: Array<{ id; title; template; params; recommended; note; matches; end; ranked; unique; violations }> }`(運営会社の頻出形が先頭で `recommended`) |
| `POST /admin/divisions/{id}/format/preview` | `FormatInput` | `FormatPreview`(保存しない) |
| `PUT /admin/divisions/{id}/format` | `FormatInput` | `FormatPreview`。ディビジョンのグループと試合を置き換える。結果の入った試合があれば 409 |
| `GET /admin/divisions/{id}/format` | なし | `{ format: SavedFormat \| null }` |
| `GET /admin/formats?exclude={divisionId}` | なし | `{ formats: SavedFormat[] }`(前回の大会の形のコピー元。新しい順に 30 件) |
| `GET /admin/divisions/{id}/progress` | なし | `Progress` |
| `GET /admin/divisions/{id}/format/matches` | なし | `{ matches: ProgressMatch[] }`(時刻、コートの順) |
| `POST /admin/groups/{id}/confirm` | `{ unplayed: boolean }` | 204。プールの順位を確定し、次の段の枠を埋める。結果待ちがあれば `unplayed: true`(未消化のまま確定)のときだけ。未決の同順位があれば 409 |
| `POST /admin/stages/{id}/confirm` | `{ unplayed: boolean }` | 204。段の全プールを確定し、段の全体順位を使えるようにする |
| `POST /admin/groups/{id}/rank-decisions` | `{ teamIds: number[]; reason: string }` | 204。未決の同順位の組を上位から並べる(ブリックマーク投げなど。`reason` 必須)。プールの全試合が終わる前は 409。指定はその時点の成績に結び付き、訂正で成績が変われば未決に戻る |
| `POST /admin/stages/{id}/rank-decisions` | `{ rank: number; teamIds: number[]; reason: string }` | 204。段の全体順位(各プールの `rank` 位どうし)の未決。段の全プールの試合が終わる前は 409 |
| `POST /admin/matches/{id}/correction-impact` | `{ score: { left; right }; winnerSide: Side \| null }` | `Impact`(保存しない)。保存のときに断る入力(トーナメントの同点で `winnerSide` が無い、対戦チームが決まっていない)は、ここで同じ 400 / 409 を返す |
| `POST /admin/matches/{id}/correction` | 上と同じ + `downstream: 'keep' \| 'reset' \| null`、`expectedScore?: { left; right } \| null`、`keepEvents?: boolean` | `Impact`。得点経過の無い試合の結果を入れる、または直す。`needsApproval` なのに `downstream` が null なら 409。得点経過のある試合は `keepEvents: true` のときだけ、得点経過を残したまま `score` で終了にする(得点経過の合計と違えば監査ログの `note` に残す。閲覧ページは試合の最終スコアを出す)。無ければ 400。`expectedScore` は本部が画面を開いたときのスコアで、今のスコア(スコアなしは 0-0)と違えば 409 `score_mismatch`(記録係の試合終了の照合と同じ。`message` に今のスコア) |
| `POST /admin/matches/{id}/align-teams` | なし | 204。既に始まった試合の左右を、訂正後の順位のチームに差し替える(スコアは変えない)。結果は新しいチームのものとして次の試合と最終順位に流す |
| `GET /admin/divisions/{id}/schedule-check` | なし | `{ violations: Violation[] }`。同じ大会の他のディビジョンの試合とのコートの重なりも `court_conflict`(`codes` の 2 つ目が相手の試合 CD)。形の適用で作る時間割は、他のディビジョンの試合のコートと時刻を避ける |
| `POST /admin/divisions/{id}/schedule-shift` | `{ court: string \| null; minutes: number }` | `{ violations }`。まだ始まっていない試合を、そのコート(null なら全コート)の最初の未試合の時刻から `minutes` 分ずらす(-360〜360、0 以外) |
| `POST /admin/matches/{id}/swap-schedule` | `{ otherMatchId: number }` | `{ violations }`。未試合の 2 試合の日付、時刻、コートを入れ替える |

- 試合の結果が変わるたび(得点経過、状態、結果の種類の変更)に、形のあるディビジョンでは枠と下流の未試合の試合の左右、グループのチーム、プールの順位(`rank_override`)、最終順位(`teams.final_rank`)を自動で書き直す。トーナメント内の勝ち上がりは自動、プールから次の段へは確定が要る。
- 既に始まった試合の左右は自動では変えない。食い違いは `progress.todo` の `mismatch` に出る(枠の側も、前の試合の勝者・敗者を待つ側も)。食い違う試合の結果は、勝者と敗者を次の試合へ流さず、最終順位にも使わない(結果を記録した対戦チームで勝者を決めるので、出ていないチームを勝者にしない)。食い違いのあるプールは確定できない。
- 結果を変える全経路(得点経過の追加と取り消し、状態の変更、結果の種類、訂正)で、既に結果のある下流の試合の対戦が新たに変わるなら、`downstream` を付けない限り 409。`downstream` は状態の変更(`POST /admin/matches/{id}/status`)と訂正だけが受け付ける。得点経過の `end` の取り消しで 409 になったら、状態の変更(`status: 'live'`)に `downstream` を付けて試合中へ戻す。
  - `reset`: 影響を受ける下流の試合の結果と得点経過を消して未試合に戻す。対戦チームは訂正後の順位で入れ直る(監査ログに `match.status` で残る)。
  - `keep`: 下流の結果を残す。その試合は `mismatch` になり、`align-teams` で差し替えるか結果を直すまで、結果を次の試合と最終順位に使わない。
  - 既に食い違いになっている試合は、さらに対戦が変わっても承認の対象にしない(前の承認で扱いが決まっている)。
- 対戦チームが決まっていない試合への結果の入力は、どの経路(得点経過、状態、結果の種類、訂正)でも 409。
- 形の適用、確定、同順位の指定、差し替えは監査ログに残る(`format.apply`、`group.confirm`、`stage.confirm`、`group.rank_decision`、`stage.rank_decision`、`match.align_teams`)。

### ギャラリー

```ts
type AdminGallery = {
  id: number; tournamentId: number
  name: string; sortOrder: number
  photos: Array<{ thumbUrl: string; fullUrl: string }>   // 表示順
}
```

| メソッドとパス | ボディ | 応答 |
|---|---|---|
| `GET /admin/tournaments/{id}/galleries` | なし | `{ galleries: AdminGallery[] }` |
| `POST /admin/tournaments/{id}/galleries` | `{ name, sortOrder }` | `AdminGallery` |
| `GET` / `PATCH` / `DELETE /admin/galleries/{id}` | `PATCH` は `{ name, sortOrder }` の一部 | `AdminGallery` / 204 |
| `PUT /admin/galleries/{id}/photos` | `{ photos: Array<{ thumbUrl; fullUrl }> }`(この順で表示) | `AdminGallery` |

### ユーザー

| メソッドとパス | ボディ | 応答 |
|---|---|---|
| `GET /admin/users` | なし | `{ users: AdminUser[] }`(役割、メールアドレス順) |
| `POST /admin/users` | `{ email, displayName, role, password }`(role の既定 'staff'。password は 12〜256 文字で必須) | `AdminUser` |
| `GET` / `PATCH` / `DELETE /admin/users/{id}` | `PATCH` は `{ email, displayName, role, password }` の一部 | `AdminUser` / 204 |
| `PUT /admin/users/{id}/tournaments` | `{ tournamentIds: number[] }`(担当大会を置き換える) | `AdminUser` |

- パスワードは応答に含めない。`PATCH` で `password` を変えると、そのユーザーの全セッションを消す。
- 自分自身の役割の変更と削除は 409(管理者がいなくなるのを防ぐ)。
- 最初の管理者は `go run ./cmd/createadmin -email ...` で作る(パスワードは環境変数 `SCOREVIEW_ADMIN_PASSWORD` か標準入力)。
