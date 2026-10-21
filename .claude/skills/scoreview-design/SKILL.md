---
name: scoreview-design
description: scoreview(score.clubjr.com を模倣するスコアサイト)の設計判断の基準。依存の向き、模倣の再現度の線引き、API 契約、GORM の安全な書き方、ドメイン計算の置き場所、エラー設計、テスト方針を定める。Use when このリポジトリで機能・API・テーブル・ページを設計/実装するとき、参照サイトの挙動をどこまで再現するか迷ったとき、取込ツールやブラケット表に手を入れるとき。
---

# scoreview の設計判断

背景と詳細は docs/(requirements, tech-stack, architecture, api)と docs/clone-spec/ にある。
ここには実装中に毎回参照する判断基準だけを置く。

## 1. 模倣の再現度

- 閲覧ページの仕様は docs/clone-spec/ である。DOM 構造、クラス名、列幅、文言(日英)、条件分岐は記録どおりに再現する。参照サイトの誤綴り(`Tounament`, `Reague Result`, `21th`, `Reflesh_OK`)も、データや固定文言として出ているものはそのまま出す。
- 再現しない挙動は requirements.md の「意図的な差分」表にあるものだけ(エラー時の 500、`href="#"`+onclick、POST による絞り込み、無害化しない HTML、他社ブランド素材)。新しく差分を作るときは、この表に理由つきで追記してから実装する。
- 参照サイトの CSS と画像はコピーしない。clone-spec に記録した値(色、幅、余白)で CSS を書き直し、ブランド素材は `apps/web/src/brand/` の設定値を使う。
- clone-spec に「未観測」とある状態(試合中 MC_NOW、未開始の試合ページ、没収試合)は、CSS 定義と既存の文言体系から最小限に作り、推測であることをコードコメントに書く。

## 2. 依存の向き

```
api:  domain ◀ store ◀ httpapi ◀ cmd/*      bracket ▶ domain、importer ▶ store
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
- 取込ツールは参照サイトの ID をそのまま主キーに入れる。入れた後で `setval` 相当の処理(シーケンスを最大 ID に進める)を store の関数として用意し、そこだけ理由を書いて `//nolint:forbidigo` を許す。

## 5. ドメインの規則(参照サイトで確認済み)

- ディビジョン順位は `teams.final_rank` の値を出す。勝率から計算しない。
- プール順位は 勝ち点(3/1/0)→ 2 チーム同点なら直接対決 → 得失差 → 得点。`group_teams.rank_override` があれば優先し、0 なら空欄。
- 勝率は 勝ち ÷ 試合数(引き分けは勝ちに数えない)。試合数 0 は `--%`。小数 2 桁で四捨五入し末尾の 0 を削る。この整形は web の `format.ts` で行う。
- 選手の並びは (得点+アシスト) 降順 → 得点降順 → 背番号の数値昇順。
- 得点経過は API では古い順で返し、web で新しい順に並べ替えて出す。
- チーム名の表示幅の切り詰め(日程 16、ディビジョン名 10、全角 2 半角 1)は web で行う。短縮名そのものはデータ(`name_short_*`)。

## 6. エラー

- Go: domain の sentinel(`ErrNotFound` など)と構造化エラー(`NotFoundError{Resource, ID}`)を返し、`%w` で包む。HTTP ステータスへの変換は `httpapi.writeError` だけで行う。ハンドラで `http.Error` を書かない。
- web: `apiFetch` が `ApiError` を投げる。`not_found` は参照サイトと同じ文面のエラー画面(HTTP 404)にする。それ以外は握りつぶさず Next.js のエラー境界へ流す。
- 失敗を空配列や 0 で隠さない。

## 7. セキュリティ

- api は `X-Internal-Token` を全リクエストで検証する。Vercel では公開ルートを付けない(vercel.json の rewrites に api を入れない)。
- 自由記述 HTML(`top_html_*`)は `internal/sanitize` の許可リストで無害化してから返す。web で `dangerouslySetInnerHTML` を使えるのは `SanitizedHtml.tsx` だけ(Biome で強制)。
- パスワードは argon2id。セッショントークンは平文を保存せず SHA-256 を保存する。ログイン失敗の応答でメールアドレスの有無を区別しない。
- 管理画面の更新は Server Actions から api に中継する。web 側でも `auth/me` で権限を確かめてから画面を出す。

## 8. テスト

- Go: domain はテーブル駆動の単体テスト(clone-spec の実例をケースにする)。store と httpapi は `testdb.New(t)` の実 PostgreSQL で統合テスト。
- web: `node --import tsx --test`。整形関数と、モック応答からのコンポーネント描画(renderToStaticMarkup)で DOM のクラス名と文言を検証する。
- 取込ツールのパーサのテストには、参照サイトの HTML から個人名を除いた最小のフィクスチャを使う。`.local/` の生データをテストから読まない(CI に存在しないため)。

## 9. 禁止事項

- 新しい lint/format ツール、テストランナー、CSS フレームワーク(Tailwind 等)を足さない。見た目は参照サイトのクラス名を持つ素の CSS で書く。
- `.local/` のデータ(実在の個人名を含む)をコミットしない、公開環境の DB に入れない。
- 参照サイトへの自動アクセスは取込ツールと画面比較だけにし、間隔を 1 秒以上空ける。
