---
name: scoreview-design
description: scoreview(フライングディスク大会の観戦アプリと運営管理画面)の設計判断の基準。依存の向き、API 契約、GORM の安全な書き方、ドメイン計算の置き場所、エラー設計、テスト方針を定める。Use when このリポジトリで機能・API・テーブル・ページを設計/実装するとき、ブラケット表や大会の形に手を入れるとき。
---

# scoreview の設計判断

背景と詳細は docs/(requirements, tech-stack, architecture, api, line-integration)にある。docs/clone-spec/ は、削除した模倣ページのための参照サイトの調査記録で、データの意味(順位、表記)の根拠として残している。
ここには実装中に毎回参照する判断基準だけを置く。

## 1. 画面の方針

- 公開する画面は大会の一覧(`/`)、観戦アプリ(`/live/{id}`)、個人情報保護方針だけ。参照サイトの模倣ページは 2026-11 に削除したので、参照サイトの DOM や文言に合わせる理由はもう無い。見た目は Claude Design(「StageCam」「Score View Admin」)に合わせる。
- 次の機能は LINE からのエントリーと通知、記録係の LINE からの入力(docs/line-integration.md)。URL は LINE のメッセージに埋め込むと変えられないので、`/live/{id}` と line-integration.md に書いたパスを固定する。

## 2. 依存の向き

```
api:  domain ◀ store ◀ httpapi ◀ cmd/*      bracket ▶ domain
web:  src/api(クライアント + zod)◀ src/components ◀ src/app
```

- `domain` は GORM、HTTP、JSON タグを import しない。順位、勝率の元になる集計、得点経過のスコア、オフェンス/ディフェンス開始の判定、選手の並び順はここに置く。
- `store` は GORM のモデル型を外に出さない。戻り値は domain の型か、ページ単位のビュー型(store 内で定義)にする。
- `httpapi` のハンドラは「パラメータ検証 → store → DTO 変換 → writeJSON」以外をしない。計算が要るなら domain に関数を作る。
- web のコンポーネントは api の応答型(zod から推論した型)だけを受け取り、fetch しない。fetch はページ(Server Component)で 1 回だけ行う。

## 3. API 契約

- 正本は docs/api.md。Go の DTO と web の zod スキーマを同じコミットで合わせる。片側だけ変えない。
- 閲覧用の応答は必ず `frame`(大会ナビとサイドバーの骨格)を含める。ページが大会 ID を知らなくても 1 往復で描けるようにするため。
- 日英は API で選ばず、`I18nText {ja, en}` で両方返す。英語が未登録なら空文字で返し、web も補完しない(参照サイトが空表示するため)。
- 値がないことは `null`。キーを省略しない。

## 4. DB と GORM

- スキーマは `apps/api/migrations/` の goose SQL が正本。GORM の AutoMigrate は使わない。列を足すときは新しい番号の SQL ファイルを追加する(既存ファイルを書き換えない)。
- `db.Raw` と `db.Exec` は golangci-lint(forbidigo)で禁止している。`Where`/`Order`/`Select`/`Joins` などの第 1 引数は文字列リテラルか定数にし、値は `?` で渡す(`internal/store/sqlsafety_test.go` が検査する)。
- 並び順のように実行時に選ぶ列は、許可リストの map(`sqlOrderBy := map[string]string{...}`)から定数を引く。利用者の入力を列名に使わない。
- 複数テーブルの更新(試合イベントの追加とスコアの更新など)は `db.Transaction` の中で行う。

## 5. ドメインの規則(参照サイトで確認済み)

- ディビジョン順位は `teams.final_rank` の値を出す。勝率から計算しない。
- プール順位は 勝ち点(3/1/0)→ 2 チーム同点なら直接対決 → 得失差 → 得点。`group_teams.rank_override` があれば優先し、0 なら空欄。
- 勝率は 勝ち ÷ 試合数(引き分けは勝ちに数えない)。試合数 0 は `--%`。小数 2 桁で四捨五入し末尾の 0 を削る。この整形は web の `format.ts` で行う。
- 選手の並びは (得点+アシスト) 降順 → 得点降順 → 背番号の数値昇順。
- 得点経過は API では古い順で返し、web で新しい順に並べ替えて出す。
- チーム名の表示幅の切り詰め(日程 16、ディビジョン名 10、全角 2 半角 1)は web で行う。短縮名そのものはデータ(`name_short_*`)。

## 6. エラー

- Go: domain の sentinel(`ErrNotFound` など)と構造化エラー(`NotFoundError{Resource, ID}`)を返し、`%w` で包む。HTTP ステータスへの変換は `httpapi.writeError` だけで行う。ハンドラで `http.Error` を書かない。
- web: `apiFetch` が `ApiError` を投げる。`not_found` は `notFound()` で「ページが見つかりません」の画面(HTTP 404。`src/app/not-found.tsx`)にする。それ以外は握りつぶさず Next.js のエラー境界へ流す。
- 失敗を空配列や 0 で隠さない。

## 7. セキュリティ

- api は `X-Internal-Token` を全リクエストで検証する。Vercel では公開ルートを付けない(vercel.json の rewrites に api を入れない)。
- 自由記述 HTML(`top_html_*`)は `internal/sanitize` の許可リストで無害化してから返す。web で `dangerouslySetInnerHTML` を使えるのは `SanitizedHtml.tsx` だけ(Biome で強制)。
- パスワードは argon2id。セッショントークンは平文を保存せず SHA-256 を保存する。ログイン失敗の応答でメールアドレスの有無を区別しない。
- 管理画面の更新は Server Actions から api に中継する。web 側でも `auth/me` で権限を確かめてから画面を出す。

## 8. 将来の LINE 連携に備える構造

詳細は docs/line-integration.md。フェーズ 1 から次を守る。

- 試合のスコアと状態(`matches.left_score` / `right_score` / `status` / `winner_side`)を変えるのは、store の試合更新用の関数群だけにする。ハンドラから `matches` のこれらの列を直接 Update しない(取込ツールとシードが行を作るときの初期値は除く)。LINE 通知はこの関数のトランザクション内に後から足す。取込ツールとシードは「通知しない」経路として区別できる引数にする。
- セッションは `user_id` にだけ結びつける。ログイン処理は「本人確認 → ユーザー特定 → セッション発行」の順に書き、本人確認の部分に LINE を後から足せるようにする。ただし今は差し替え用のインターフェースを作らない。
- ページの URL 体系を変えない(LINE のメッセージに埋めた URL は後から直せない)。
- チームの正式名は加工せずに保存する(後で大会をまたぐ `clubs` に束ねるため)。

## 9. テスト

- Go: domain はテーブル駆動の単体テスト(clone-spec の実例をケースにする)。store と httpapi は `testdb.New(t)` の実 PostgreSQL で統合テスト。
- web: `node --import tsx --test`。整形関数と、モック応答からのコンポーネント描画(renderToStaticMarkup)で DOM のクラス名と文言を検証する。
- `.local/` の生データをテストから読まない(CI に存在しないため)。

## 10. 禁止事項

- 新しい lint/format ツール、テストランナー、CSS フレームワーク(Tailwind 等)を足さない。見た目は画面ごとの素の CSS(site.css、live.css、admin.css)で書く。
- `.local/` のデータ(実在の個人名を含む)をコミットしない、公開環境の DB に入れない。
- 参照サイトへ自動でアクセスしない(取込ツールと画面比較は削除した)。
