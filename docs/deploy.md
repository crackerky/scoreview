# デプロイ手順

Vercel の 1 プロジェクトに、web(Next.js)と api(Go)を Vercel Services としてまとめてデプロイする。
構成の考え方は [architecture.md](architecture.md) にある。
本書の手順は、2026-10-21 の初回デプロイ(Vercel CLI 65.0.0、個人の Hobby スコープ、`https://scoreview-delta.vercel.app`)で通した。

## 前提

- GitHub リポジトリに push してあり、Vercel の Git 連携でデプロイする。CLI(`vercel deploy`)を使う場合も `.vercelignore` が `.local/`(参照サイトの生データと実在の個人名を含む取込データ)を除外する。
- 本番の DB は PostgreSQL 17 相当。Vercel Marketplace の Neon を想定する(料金は契約前に確認する。[tech-stack.md](tech-stack.md))。

## 1. Vercel プロジェクトを作る

1. リポジトリ直下で `vercel link --yes --project scoreview --scope <スコープ>` を実行する。プロジェクトの作成、GitHub リポジトリとの連携、Services(web は Next.js、api は Go)の検出まで済む。Framework Preset は Services になる。
   - このコマンドは `.gitignore` に `.vercel` と `.env*` を書き足す。`.env*` は `.env.example` まで無視させるので、書き足しは取り消す(`.vercel/` と `.env.local` は元から無視している)。
2. 関数の実行リージョンを DB と同じ `sin1`(シンガポール)にする。Vercel Marketplace の Neon は東京を選べない(選べるのは cle1、iad1、pdx1、fra1、lhr1、syd1、sin1、gru1)。api は 1 回の要求で何度も DB を呼ぶので、関数を DB の近くに置く。設定はプロジェクトの `serverlessFunctionRegion`(`vercel api /v9/projects/<id> --method PATCH`)。
3. 公開されるのは web だけである。`vercel.json` の rewrites は web にしか向けておらず、api には web の binding(`BACKEND_INTERNAL_URL`)からだけ届く。

## 2. DB を用意する

1. `vercel integration add neon -n scoreview-db -m region=sin1` で Neon を追加し、プロジェクトに接続する。初回はブラウザで Marketplace の規約に同意する必要があり、コマンドが同意の URL を返す(同意の後に同じコマンドをもう一度実行する)。`DATABASE_URL`(接続プール経由)と `DATABASE_URL_UNPOOLED`(直結)などが Production、Preview、Development の全部に入る。
   - このコマンドは Neon のエージェント向けスキル(`.agents/`、`.claude/skills/neon*`、`skills-lock.json`)をリポジトリに置く。このリポジトリでは使わないので消す。
2. マイグレーションを当てる。手元から本番の接続文字列で次を実行する(接続文字列はシェルの履歴に残さないよう、環境変数ファイルから読む)。

```sh
cd apps/api
DATABASE_URL='<DATABASE_URL_UNPOOLED の値>' go run ./cmd/migrate
```

Neon には接続プール経由(pooled)と直結の 2 つの接続文字列がある。マイグレーションは直結、実行時の api はプール経由を使う。値は `vercel env pull <リポジトリの外のファイル> --environment production` で取り出す。
接続プールは transaction モードなので、セッションをまたぐ状態(セッション単位の advisory lock など)は使えない。ログイン試行の lock はトランザクション単位(`pg_advisory_xact_lock`)なので問題ない。

## 3. 環境変数

| 名前 | 置く場所 | 値 |
|---|---|---|
| `DATABASE_URL` | api | Neon の接続プール経由の接続文字列(Marketplace が入れる) |
| `INTERNAL_API_TOKEN` | web と api(同じ値) | `openssl rand -hex 32` で作る。`change-me` を含む値では api が起動しない |
| `BACKEND_INTERNAL_URL` | web | Vercel の binding が入れる(手で設定しない) |
| `PUBLIC_SITE_URL` | web | 本番の公開 URL(現在は `https://scoreview-delta.vercel.app`)。記録係の画面の送信元の確認に使う。独自ドメインに移したら値を変えて再デプロイする |
| `REFERENCE_ASSET_MODE` | web | `block`(既定)。本番に取込データを入れないので変えない |
| `PORT` | api | Vercel が入れる |

Neon の接続は Preview にも同じ DB を入れる。Preview で本番と別の DB を使うには、Neon のブランチを作って Preview の `DATABASE_URL` を差し替える(まだしていない)。

## 4. 最初の管理者とデータ

```sh
cd apps/api
# 最初の管理者(パスワードは標準入力か環境変数 SCOREVIEW_ADMIN_PASSWORD から読む。引数には書かない)
DATABASE_URL='<直結の接続文字列>' go run ./cmd/createadmin -email <管理者のメールアドレス>

# 架空データを入れる。正式公開の前は本番もデモ環境なので入れる。正式公開の DB には入れない
DATABASE_URL='<直結の接続文字列>' go run ./cmd/seed
```

参照サイトから取り込んだデータ(`go run ./cmd/importer`)は実在の個人名を含むので、Preview にも本番にも入れない。開発用 DB(`go run ./cmd/devdb`)だけで使う。

## 5. デプロイ後の確認

- `/sctnmt` が開き、大会一覧が出る。
- `/admin` でログインできる。ログイン試行の制限(同じアドレスで 5 回失敗した後は間隔が延びる)が働く。
- api の URL を直接開けない(公開されていない)ことを確かめる。
- 応答ヘッダーに `Content-Security-Policy`、`Strict-Transport-Security`、`Permissions-Policy`、`X-Robots-Tag: noindex, nofollow` が付いている(noindex は正式公開のときに `next.config.ts` から外す)。

## 6. 運用

- マイグレーションは、新しいマイグレーションを含むコミットをデプロイする前に当てる。列の追加だけのマイグレーションは、先に当てても旧版の api が動き続ける。
- 依存の脆弱性は CI の govulncheck と `pnpm audit` で検出する。
- 大会当日の負荷は、閲覧ページのキャッシュ(終了した大会は長く、進行中の大会は短く)で吸収する([tech-stack.md](tech-stack.md) のランニングコスト)。
