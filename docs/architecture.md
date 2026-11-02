# アーキテクチャ設計書

## 可変点の宣言

フェーズ 2 で自社仕様に作り変えることが決まっているので、変える場所と変えない場所を先に決める。

| 軸 | 差し替え可能にするもの | 置き場所 |
|---|---|---|
| ブラケットの形 | トーナメント表の格子を生成するテンプレート(4、8、16 チーム、順位決定戦つきなど) | `apps/api/internal/bracket/`(テンプレートごとに 1 ファイル、登録は Functional Options) |
| データの入口 | 管理画面からの入力、架空データの生成、フェーズ 3 の LINE からのエントリー | `apps/api/cmd/seed/` と管理用エンドポイント。どれも同じ store を通して書く |
| 画像の置き場 | フェーズ 1 は URL を登録するだけ。フェーズ 2 でアップロード先(Vercel Blob など)を差し込む | `apps/api/internal/media/`(フェーズ 2) |
| ログイン方法 | フェーズ 1 はパスワード。フェーズ 3 で LINE ログインを足す | `apps/api/internal/auth/`。セッションはユーザーに結びつけ、ログイン方法に結びつけない |
| 通知 | フェーズ 3 で LINE 公式アカウントからの配信を足す | 試合の状態を変える store の関数のトランザクション内で通知ジョブを積む([line-integration.md](line-integration.md)) |

変えないものは、データモデル(大会 → ディビジョン → グループ → 試合)と、順位とスコアの計算規則である。
これらは参照サイトの調査で確定した仕様である。
参照サイトを模倣した閲覧ページと取込ツールは 2026 年 11 月に削除し、公開する画面は大会の一覧(`/`)と観戦アプリ(`/live/{大会ID}`)にした。

## 全体構成

```
ブラウザ ──▶ Vercel ──▶ web サービス(Next.js)──binding──▶ api サービス(Go)──▶ PostgreSQL(Neon)
                          │                                   ▲
                          └ /, /live/*, /privacy, /admin      │
                                                              シード(Go コマンド。手元か CI で実行)
```

- 公開するのは web サービスだけである。api サービスには Vercel の公開ルートを付けず、web サービスの binding(内部 URL)からだけ呼ぶ。
- api は全リクエストで `X-Internal-Token` を検証する。binding はネットワーク到達性を与えるだけで、呼び出し元の認証にはならないためである。
- 管理画面の操作は、Next.js の Server Actions が受けて api の管理用エンドポイントへ中継する。ログインのセッショントークンは Next.js が httpOnly Cookie に保存し、api へは `Authorization: Bearer` で渡す。

- web は全ページに Content-Security-Policy を付ける(`apps/web/src/proxy.ts`、`src/security/csp.utils.ts`)。管理画面は要求ごとの nonce でスクリプトを限り、大会の一覧と観戦アプリは画像の読み込み先を自サイトに限る(`REFERENCE_ASSET_MODE=allow` のときだけ参照サイトの画像の置き場も許す)。どちらも埋め込み、`<base>`、他サイトへのフォーム送信を禁じる。Permissions-Policy と HSTS は `next.config.ts` で付ける。

### 環境変数

例はリポジトリ直下の `.env.example`。

| サービス | 名前 | 内容 |
|---|---|---|
| web | `BACKEND_INTERNAL_URL` | api の URL(Vercel では binding が入れる) |
| web, api | `INTERNAL_API_TOKEN` | web と api の共有秘密(32 文字以上。`openssl rand -hex 32`)。`change-me` を含む値では api が起動しない |
| web | `PUBLIC_SITE_URL` | サイトの公開 URL(例 `https://score.example.com`)。記録係の画面の Route Handler が、リクエストの Origin をこの値と比べる(Host ヘッダーは送り手が決めるので使わない)。未設定なら手元の開発用の `http://localhost:3000`。Vercel ではデプロイごとの URL(`VERCEL_URL`)も許す |
| api | `DATABASE_URL` | PostgreSQL の接続文字列 |
| api | `PORT` | 待ち受けるポート(既定 8080。Vercel が入れる) |

## 依存の向き

```
apps/api:  domain ◀── store(GORM) ◀── httpapi ◀── cmd/api
             ▲            ▲
             │            └──── seed(cmd/seed)
             └── bracket(テンプレート)
apps/web:  api クライアント(zod で応答を検証)◀── ページ、管理画面
```

- `domain` は純粋な Go のパッケージで、GORM も HTTP も知らない。順位計算、勝率、得点経過からの統計、未確定チームの表記はここに置き、テーブル駆動テストで固める。
- `store` は GORM のモデルとリポジトリを持ち、`domain` の型へ変換して返す。GORM のモデル型を `httpapi` に漏らさない。
- `httpapi` は薄い層で、リクエストの検証、`store` の呼び出し、応答 DTO への変換だけを行う。エラーから HTTP ステータスへの変換は 1 か所で行う。
- web は api の応答を zod で parse してから使う。Go と TypeScript の間の契約は [api.md](api.md) で、両側の型はこれに合わせる。

## データモデル

```
tournaments 1─* divisions 1─* groups 1─* group_teams *─1 teams
     │              │                                   │
     │              └──────────* matches *──────────────┘ (left/right)
     │                              │
     │                              ├─* match_events(得点、タイムアウト、プル、終了)
     │                              └─* match_player_stats(出場、ブロック、キャラハン、グレイテスト)
     ├─* tournament_banners
     ├─* galleries 1─* gallery_photos
     └─* teams 1─* players(スタッフも同じ表。role が入っていればスタッフ)
users 1─* sessions、users *─* tournaments(tournament_staff)
```

調査で分かった制約をスキーマに反映した。

- チームと選手は大会ごとに別 ID である(同名チームが大会ごとに別 ID。[D 章 6](clone-spec/D-team-match-player.md))。チームを大会横断で名寄せする表は作らない。
- チーム名は「正式名」と「短縮名」を日英で 4 つ持つ。短縮名は機械的な切り詰めでは作れない(「メッターによるお料理向上委員会」の短縮名は「お料理向上委員会」)。画面表示ではさらに表示幅 16 で切り詰める。
- ディビジョンの順位は `teams.final_rank` に保存する値であり、勝率から計算しない。参照サイトでは順位決定戦の結果がそのまま順位になり、同順位(5、5、7、7)も生じる。
- 総当たりグループの順位は計算する。勝ち点(勝ち 3、分け 1)、直接対決、得失差、得点の順で比べる([C 章 2.2](clone-spec/C-standings-groups.md)、41 プールで一致を確認)。計算と異なる値を出す例外があるため、`group_teams.rank_override` で上書きできる。
- トーナメント表は `groups.bracket_layout`(JSONB)に、行と列の格子として保存する。各セルは罫線(上下左右)、種類(チーム、勝者、敗者、スコア、見出し)、文字列、参照する試合とチームを持つ。参照サイトの表は 1 枚の table の罫線だけで描かれており、テンプレートごとに形が違うので、形を計算で再現するより格子を保存するほうが正確である。管理画面からは `bracket` パッケージのテンプレートで格子を生成する。
- 未確定の対戦相手は `matches.left_placeholder_code` と `left_placeholder_outcome`(win または lose)で持つ。表示は「[A1-勝ち]」「[A1-Win]」。
- 引き分けのスコアでもトーナメントでは勝者が決まる例がある(4-4 で勝者あり)。`matches.winner_side` で明示的に持つ。
- 得点経過の各時点のスコアは保存せず、イベントの順番から計算する。オフェンスから始まった得点かどうかもプルの向きから計算できる。ターンオーバーからの得点もディフェンス開始からの得点と同じ数として計算する(参照サイトの実データ 12 試合すべてで一致。migrations/00002)。キャラハンゴールは得点イベントの `callahan` で持ち、選手のキャラハン数はこれを数える。
- コート(`matches.field`)は数字とは限らない("A" など)ので文字列で持ち、自然順(数字を数値順で先に)で並べる。
- ID は自動採番する。取込ツールがあった間は参照サイトの ID を明示して登録し、自動採番のシーケンスを 100,000,000 以上に進めていた(手元の開発用 DB に残っている)。

スキーマの正本は `apps/api/migrations/` の SQL である。

## 主要な流れ

### 観戦アプリの表示

1. 観戦アプリのページ(Server Component)が、大会のスケジュール、組、「表で見る」の形を api の閲覧用エンドポイントから読む。
2. zod で応答を検証し、観戦アプリのデータ(`apps/web/src/live/`)に組み立てて描く。
3. 開いたあとは、ブラウザが `/live/{id}/data` を 15 秒ごとに取り直す。

### スコア入力

1. 記録係が管理画面の試合ページで「得点」を押し、得点者とアシストを選ぶ。
2. Server Action が api の `POST /api/v1/admin/matches/{id}/events` を呼ぶ。
3. api は権限を確かめ、イベントを追加し、同じトランザクションで `matches` のスコアと状態を更新する。スコアはイベントから計算した値で上書きするので、イベントとスコアが食い違わない。

## リポジトリ構成

```
scoreview/
├── vercel.json                 Vercel Services の定義(web と api)
├── apps/
│   ├── web/                    Next.js(@scoreview/web)
│   │   ├── src/app/            大会の一覧、観戦アプリ(/live)、管理画面(/admin)
│   │   ├── src/api/            api クライアントと zod スキーマ(*.types.ts)
│   │   ├── src/components/     観戦アプリ、管理画面、サイトの入口の部品
│   │   ├── src/styles/         画面ごとの CSS(site、live、admin)
│   │   ├── src/i18n/           画面文言の日英辞書
│   │   └── test/               *.spec.ts
│   └── api/                    Go モジュール
│       ├── cmd/api/            HTTP サーバー(Vercel のエントリポイント)
│       ├── cmd/migrate/        マイグレーション実行
│       ├── cmd/devdb/          開発用 PostgreSQL の起動
│       ├── cmd/seed/           架空データの生成
│       ├── internal/domain/    純粋なドメイン型と計算
│       ├── internal/store/     GORM モデルとリポジトリ
│       ├── internal/httpapi/   ハンドラ、DTO、ミドルウェア
│       ├── internal/auth/      パスワードとセッション
│       ├── internal/bracket/   トーナメント表のテンプレート
│       ├── internal/sanitize/  自由記述 HTML の無害化
│       ├── internal/testdb/    テスト用 PostgreSQL
│       └── migrations/         goose の SQL
├── docs/
└── .local/                     参照サイトの調査のキャッシュ(git 管理外。実在の個人名を含む)
```

## テスト戦略

| 対象 | 方法 |
|---|---|
| domain(順位、統計、表記) | テーブル駆動の単体テスト。参照サイトで確認した実例(949 の順位、6511 の直接対決など)をケースにする |
| store と httpapi | embedded-postgres に実マイグレーションを当て、ハンドラを `httptest` で叩く統合テスト |
| web の整形関数(表示幅の切り詰め、勝率、平均) | Node 標準テストランナー |
| web の観戦アプリと管理画面 | api の応答の fixture からデータを組み立てる関数を検証する。画面は手元のヘッドレスブラウザで操作して確かめる |

## 実装の順序

1. 足場(本書、CLAUDE.md、ツールチェーン、CI、空の web と api)
2. スキーマとドメイン計算(マイグレーション、domain、store)
3. 閲覧用 API と閲覧ページ(api と web を並行。契約は api.md)
4. 取込ツールで代表 14 大会を取り込み、画面比較で差分を潰す
5. 管理画面(認証、登録、スコア入力)
6. Vercel へのデプロイ(架空データ)
