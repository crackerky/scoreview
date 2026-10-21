# アーキテクチャ設計書

## 可変点の宣言

フェーズ 2 で自社仕様に作り変えることが決まっているので、変える場所と変えない場所を先に決める。

| 軸 | 差し替え可能にするもの | 置き場所 |
|---|---|---|
| ブランド | サイト名、ロゴ、ヘッダーのリンク、SNS、フッター、配色 | `apps/web/src/brand/`(設定 1 ファイル + CSS 変数) |
| ブラケットの形 | トーナメント表の格子を生成するテンプレート(4、8、16 チーム、順位決定戦つきなど) | `apps/api/internal/bracket/`(テンプレートごとに 1 ファイル、登録は Functional Options) |
| データの入口 | 管理画面からの入力、参照サイトからの取込、架空データの生成 | `apps/api/internal/importer/`、`apps/api/cmd/seed/`。どれも同じ store を通して書く |
| 画像の置き場 | フェーズ 1 は URL を登録するだけ。フェーズ 2 でアップロード先(Vercel Blob など)を差し込む | `apps/api/internal/media/`(フェーズ 2) |

変えないものは、ページの URL 体系、データモデル(大会 → ディビジョン → グループ → 試合)、順位とスコアの計算規則である。
これらは参照サイトの調査で確定した仕様であり、作り変えの段階でも利用者が慣れた情報の並びを保つ。

## 全体構成

```
ブラウザ ──▶ Vercel ──▶ web サービス(Next.js)──binding──▶ api サービス(Go)──▶ PostgreSQL(Neon)
                          │                                   ▲
                          └ /sctnmt, /scview, /pscore/*, /admin │
                                                              取込ツール、シード(Go コマンド。手元か CI で実行)
```

- 公開するのは web サービスだけである。api サービスには Vercel の公開ルートを付けず、web サービスの binding(内部 URL)からだけ呼ぶ。
- api は全リクエストで `X-Internal-Token` を検証する。binding はネットワーク到達性を与えるだけで、呼び出し元の認証にはならないためである。
- 管理画面の操作は、Next.js の Server Actions が受けて api の管理用エンドポイントへ中継する。ログインのセッショントークンは Next.js が httpOnly Cookie に保存し、api へは `Authorization: Bearer` で渡す。

## 依存の向き

```
apps/api:  domain ◀── store(GORM) ◀── httpapi ◀── cmd/api
             ▲            ▲
             │            └──── importer, seed(cmd/importer, cmd/seed)
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
- 得点経過の各時点のスコアは保存せず、イベントの順番から計算する。オフェンスから始まった得点かどうかもプルの向きから計算できる。ターンオーバーからの得点だけは計算できないので、イベントに `from_turnover` を持つ。
- ID は参照サイトの ID と揃えられるように、取込時は明示した ID で登録し、後でシーケンスを進める。

スキーマの正本は `apps/api/migrations/` の SQL である。

## 主要な流れ

### 閲覧ページの表示

1. Next.js のページ(Server Component)が、URL の ID と `lang` を読む。
2. api の閲覧用エンドポイントを 1 回呼ぶ。応答には、そのページの本文データと、大会ナビとサイドバーの描画に要る `frame`(大会の骨格情報)が両方入っている。ページが大会 ID を知らなくても(試合 ID しか知らなくても)1 往復で描ける。
3. zod で応答を検証し、言語に応じて日英の値を選び、参照サイトと同じ DOM 構造で描く。

### スコア入力

1. 記録係が管理画面の試合ページで「得点」を押し、得点者とアシストを選ぶ。
2. Server Action が api の `POST /api/v1/admin/matches/{id}/events` を呼ぶ。
3. api は権限を確かめ、イベントを追加し、同じトランザクションで `matches` のスコアと状態を更新する。スコアはイベントから計算した値で上書きするので、イベントとスコアが食い違わない。

### 参照サイトからの取込

1. `go run ./cmd/importer -tournament 525` を実行する。
2. 取込ツールは大会トップ、チーム一覧、日程、各グループ、各チーム、各試合のページを 1.2 秒間隔で取得し、`.local/import-cache/` に保存する。
3. ページごとのパーサが HTML を `importer` の中間表現に変え、store を通して DB に書く。

## リポジトリ構成

```
scoreview/
├── vercel.json                 Vercel Services の定義(web と api)
├── apps/
│   ├── web/                    Next.js(@scoreview/web)
│   │   ├── src/app/            ページ(参照サイトと同じパス)と /admin
│   │   ├── src/api/            api クライアントと zod スキーマ(*.types.ts)
│   │   ├── src/components/     外枠、大会ナビ、表などの部品
│   │   ├── src/brand/          ブランド設定(可変点)
│   │   ├── src/styles/         参照サイトの見た目を再現する CSS
│   │   ├── src/i18n/           画面文言の日英辞書
│   │   └── test/               *.spec.ts
│   └── api/                    Go モジュール
│       ├── cmd/api/            HTTP サーバー(Vercel のエントリポイント)
│       ├── cmd/migrate/        マイグレーション実行
│       ├── cmd/devdb/          開発用 PostgreSQL の起動
│       ├── cmd/importer/       参照サイトからの取込
│       ├── cmd/seed/           架空データの生成
│       ├── internal/domain/    純粋なドメイン型と計算
│       ├── internal/store/     GORM モデルとリポジトリ
│       ├── internal/httpapi/   ハンドラ、DTO、ミドルウェア
│       ├── internal/auth/      パスワードとセッション
│       ├── internal/bracket/   トーナメント表のテンプレート
│       ├── internal/sanitize/  自由記述 HTML の無害化
│       ├── internal/importer/  取込ツールのパーサ
│       ├── internal/testdb/    テスト用 PostgreSQL
│       └── migrations/         goose の SQL
├── docs/
└── .local/                     調査と取込のキャッシュ(git 管理外。実在の個人名を含む)
```

## テスト戦略

| 対象 | 方法 |
|---|---|
| domain(順位、統計、表記) | テーブル駆動の単体テスト。参照サイトで確認した実例(949 の順位、6511 の直接対決など)をケースにする |
| store と httpapi | embedded-postgres に実マイグレーションを当て、ハンドラを `httptest` で叩く統合テスト |
| importer | 参照サイトの HTML から個人名を除いて縮めたフィクスチャでパーサを検証する |
| web の整形関数(表示幅の切り詰め、勝率、平均) | Node 標準テストランナー |
| web のページ | api をモックした状態でページの HTML を生成し、主要な DOM(クラス名、列数、文言)を検証する |
| 見た目 | Playwright で参照サイトと本サイトを同じ ID で撮り、差分画像を `.local/visual-diff/` に出す(手元で実行。CI では実行しない。参照サイトへ負荷をかけないため) |

## 実装の順序

1. 足場(本書、CLAUDE.md、ツールチェーン、CI、空の web と api)
2. スキーマとドメイン計算(マイグレーション、domain、store)
3. 閲覧用 API と閲覧ページ(api と web を並行。契約は api.md)
4. 取込ツールで代表 14 大会を取り込み、画面比較で差分を潰す
5. 管理画面(認証、登録、スコア入力)
6. Vercel へのデプロイ(架空データ)
