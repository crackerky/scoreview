# デプロイ手順

Vercel の 1 プロジェクトに、web(Next.js)と api(Go)を Vercel Services としてまとめてデプロイする。
構成の考え方は [architecture.md](architecture.md) にある。
本書の手順のうち、Vercel プロジェクトと本番 DB の作成は、まだ一度も実行していない(未検証)。初回のデプロイで手順を確かめ、食い違いがあれば本書を直す。

## 前提

- GitHub リポジトリに push してあり、Vercel の Git 連携でデプロイする。CLI(`vercel deploy`)を使う場合も `.vercelignore` が `.local/`(参照サイトの生データと実在の個人名を含む取込データ)を除外する。
- 本番の DB は PostgreSQL 17 相当。Vercel Marketplace の Neon を想定する(料金は契約前に確認する。[tech-stack.md](tech-stack.md))。

## 1. Vercel プロジェクトを作る

1. Vercel でリポジトリを読み込む。ルートディレクトリはリポジトリ直下のままにする(`vercel.json` の `services` が `apps/web` と `apps/api` を指す)。
2. Framework Preset は Services にする(`vercel.json` の `services` を Vercel が読む)。Vercel Services の設定名や画面は変わりうるので、作成時に Vercel のドキュメント(Services)で確かめる。
3. 公開されるのは web だけである。`vercel.json` の rewrites は web にしか向けておらず、api には web の binding(`BACKEND_INTERNAL_URL`)からだけ届く。

## 2. DB を用意する

1. Vercel の Storage(Marketplace)から Neon を追加し、プロジェクトに接続する。`DATABASE_URL` が環境変数に入る。
2. マイグレーションを当てる。手元から本番の接続文字列で次を実行する(接続文字列はシェルの履歴に残さないよう、環境変数ファイルから読む)。

```sh
cd apps/api
DATABASE_URL='<Neon の直結の接続文字列>' go run ./cmd/migrate
```

Neon には接続プール経由(pooled)と直結の 2 つの接続文字列がある。マイグレーションは直結、実行時の api はプール経由を使う。

## 3. 環境変数

| 名前 | 置く場所 | 値 |
|---|---|---|
| `DATABASE_URL` | api | Neon の接続プール経由の接続文字列(Marketplace が入れる) |
| `INTERNAL_API_TOKEN` | web と api(同じ値) | `openssl rand -hex 32` で作る。`change-me` を含む値では api が起動しない |
| `BACKEND_INTERNAL_URL` | web | Vercel の binding が入れる(手で設定しない) |
| `PUBLIC_SITE_URL` | web | 本番の公開 URL(例 `https://score.example.com`)。記録係の画面の送信元の確認に使う |
| `REFERENCE_ASSET_MODE` | web | `block`(既定)。本番に取込データを入れないので変えない |
| `PORT` | api | Vercel が入れる |

Preview 環境では、Production と別の DB(Neon のブランチ)を使い、架空データのシードを入れる。

## 4. 最初の管理者とデータ

```sh
cd apps/api
# 最初の管理者(パスワードは標準入力か環境変数 SCOREVIEW_ADMIN_PASSWORD から読む。引数には書かない)
DATABASE_URL='<直結の接続文字列>' go run ./cmd/createadmin -email <管理者のメールアドレス>

# Preview 環境だけ: 架空データを入れる(本番には入れない)
DATABASE_URL='<Preview の直結の接続文字列>' go run ./cmd/seed
```

参照サイトから取り込んだデータ(`go run ./cmd/importer`)は実在の個人名を含むので、Preview にも本番にも入れない。開発用 DB(`go run ./cmd/devdb`)だけで使う。

## 5. デプロイ後の確認

- `/sctnmt` が開き、大会一覧が出る。
- `/admin` でログインできる。ログイン試行の制限(同じアドレスで 5 回失敗した後は間隔が延びる)が働く。
- api の URL を直接開けない(公開されていない)ことを確かめる。
- 応答ヘッダーに `Content-Security-Policy`、`Strict-Transport-Security`、`Permissions-Policy` が付いている。

## 6. 運用

- マイグレーションは、新しいマイグレーションを含むコミットをデプロイする前に当てる。列の追加だけのマイグレーションは、先に当てても旧版の api が動き続ける。
- 依存の脆弱性は CI の govulncheck と `pnpm audit` で検出する。
- 大会当日の負荷は、閲覧ページのキャッシュ(終了した大会は長く、進行中の大会は短く)で吸収する([tech-stack.md](tech-stack.md) のランニングコスト)。
