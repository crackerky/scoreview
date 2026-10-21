# CLAUDE.md — scoreview

score.clubjr.com の大会スコア速報サイトを模倣し、自社仕様へ作り変えるためのリポジトリ。
web = Next.js 16(TypeScript、`apps/web`)、api = Go 1.26 + GORM(`apps/api`)、Vercel Services で同時デプロイ。
設計判断(依存の向き、模倣の線引き、API 契約、GORM の安全な書き方、ドメイン規則)は **scoreview-design スキル**を参照。閲覧ページの仕様は `docs/clone-spec/`、API の正本は `docs/api.md`。

## TypeScript の規律

- `any` を使わない — 型検査から深刻な誤りを隠すため。型が本当に不明なら `unknown` で受けて絞り込む。
- 型アサーション `x as T` と非 null 断言 `y!` はランタイムで何も検査しない。使うときは同じ行に理由をコメントで書く。理由が書けないなら zod で parse する。
- api の応答は必ず `src/api/*.types.ts` の zod スキーマで parse してから使う。`res.json()` の戻り値に型を付けて使わない — 契約違反が実行時まで見えなくなるため。
- `@ts-ignore` / `@ts-nocheck` 禁止。`@ts-expect-error` はテストコードでのみ可。
- throw する値は `Error` のサブクラスだけにする(api 呼び出しは `ApiError`)。文字列やオブジェクトを throw しない。
- 空の catch を書かない。何もしないことが正当な場合だけ、その理由をコメントで書く。
- default export は Next.js が要求するファイル(`src/app/**` の page/layout/route、`*.config.*`)以外で使わない。

## Go の規律

- エラーは `fmt.Errorf("...: %w", err)` で文脈を付けて返す。`%v` で包まない — `errors.Is/As` で原因を判定できなくなるため。
- エラーを捨てるときは `_ = f()` と書き、同じ行か直前に理由をコメントで書く。理由のない `_ =` を書かない。
- `panic` を制御フローに使わない。プログラムの不変条件が壊れた場合だけ使い、理由をコメントに書く。
- HTTP ステータスはハンドラで決めない。domain のエラー(`domain.NotFound(...)`、`domain.Invalid(...)`、sentinel)を返し、`httpapi.writeError` に変換させる。
- GORM の `Raw` / `Exec` 禁止(lint で検出)。`Where` / `Order` / `Select` / `Joins` などの第 1 引数はリテラルか定数にし、値は `?` で渡す(`internal/store/sqlsafety_test.go` が検査)。
- I/O をする関数は第 1 引数に `context.Context` を取り、GORM には `db.WithContext(ctx)` で渡す。
- パッケージレベルの可変変数を作らない(テスト用ヘルパー `internal/testdb` を除く)— テストの並列実行で状態が漏れるため。
- 公開する型と関数には、契約(何を保証し、何を返すか)をドキュメントコメントで書く。

## 変更の規律

- タスクに関係するコードだけを変更する。指示されていないクリーンアップや、無関係な既存コメントの変更をしない — 変更そのものがバグと負債の入口になるため。気づいた改善点は変更せず報告する。
- 依頼されていない既存機能を削除しない。
- 今必要と分かっている問題だけを解く。将来のための汎用化、設定項目、抽象化を先回りで作らない。
- リファクタリングは機能変更・バグ修正と別のコミットにする。1 コミットは 100 行程度を目安にし、大きく超えるなら分割を検討する(生成物、機械的な一括変更、自動整形は対象外)。
- API の形を変えるときは、`docs/api.md`、Go の DTO、web の zod スキーマを同じコミットで変える。
- スキーマを変えるときは `apps/api/migrations/` に新しい番号の SQL を足す。既存のマイグレーションファイルを書き換えない — 適用済みの環境と食い違うため。

## 関数と命名

- 関数パラメータを再代入・変更しない(Biome の `noParameterAssign` が TS 側を検査)。
- 1 文字の名前を避ける(慣用的なループ index、Go のレシーバ名を除く)。TS は camelCase、型とコンポーネントは PascalCase、ファイルは `xxx.types.ts` / `xxx.utils.ts` / `XxxComponent.tsx`。Go は標準の命名(`ErrXxx`、短いパッケージ名)。
- 関数の長さに行数制限は設けない。基準は「読み手が素早く理解できるか」。説明が要る塊ができたら名前のある関数に切り出す。
- 行長は 100 文字(Biome と揃える)。Go は gofmt に従う。

## コメント

- why と、コードで表現できない制約(参照サイトの仕様への準拠、単位、順序依存)を書く。コードを読めば分かる what は書かない。
- 参照サイトの挙動を再現しているコードには、根拠の clone-spec の章を書く(例 `// clone-spec B 7: 表示幅 16 で切り詰め`)。「未観測」の状態を推測で作った箇所には「推測」と書く。

## コマンド

Node は 22 以上(`.nvmrc`)。シェルで pnpm が見つからないときは `nvm use` してから実行する。

- `pnpm check` — Biome、型検査、web のビルドとテスト、golangci-lint、Go のビルドとテストを全部通す。コミット時に husky が実行する。
- `pnpm lint` / `pnpm format` — Biome のチェック / 自動修正(TS のみ。`apps/api` は対象外)。
- `pnpm typecheck` / `pnpm build` / `pnpm test` — web の型検査 / ビルド / テスト(`node --import tsx --test`)。
- `pnpm go:lint` / `pnpm go:build` / `pnpm go:test` — api の golangci-lint / ビルド / テスト。
- `cd apps/api && go run ./cmd/devdb` — 開発用 PostgreSQL(ポート 54329、データは `.local/devdb`)を起動してマイグレーションを当てる。Ctrl+C で停止。
- `cd apps/api && DATABASE_URL=... INTERNAL_API_TOKEN=... go run ./cmd/api` — api を 8080 で起動。値は `.env.example` 参照。
- `pnpm --filter @scoreview/web dev` — web を 3000 で起動(`apps/web/.env.local` に `BACKEND_INTERNAL_URL` と `INTERNAL_API_TOKEN`)。
- 依存の追加は `pnpm add -E`(厳密ピン)、Go は `go get <module>@<version>`(バージョンを明示)。

## gotcha

- golangci-lint は PATH ではなく `$(go env GOPATH)/bin/golangci-lint` にある(`pnpm go:lint` はこのパスを使う)。
- Go のテストは embedded-postgres で本物の PostgreSQL を起動する。初回はバイナリを `~/.cache/scoreview-embedded-postgres` にダウンロードするので時間がかかる。テストは `testdb.Run` を `TestMain` で呼ぶパッケージでだけ DB を使える。
- `.local/` は git 管理外で、参照サイトの生 HTML と取込データ(実在の個人名を含む)がある。コミットしない、公開環境に入れない、テストから読まない。
- 参照サイト(score.clubjr.com)へ自動でアクセスするときは 1 秒以上の間隔を空ける。取込ツールは `.local/import-cache/` に保存し、同じページを取り直さない。
- 大会一覧のパスは `/sctnmt`(`/` はそこへリダイレクト)。各ページの URL は参照サイトと同じなので、同じ ID で両方を開いて見比べられる。
