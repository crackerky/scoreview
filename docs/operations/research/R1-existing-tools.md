# R1: 既存の大会運営ツール調査

調査日 2026-10-21。WebSearch と WebFetch による。公式ヘルプを直接読めたものは「確認」、検索要約のみ・ベンダー自己申告・リスティング記事のものは弱い根拠として明記した。
**重要な限界**: 利用者の生の不満(Reddit 等)は検索ではほぼ取得できなかった。不満の節は少数の実例と、機能の欠落から読み取れる推測に分けて書く。Exposure Events / Tourneymachine / SportsEngine / Toornament / start.gg の詳細、Leaguevine の終了有無、JFDA の速報システムは未確認。

## 1. ツール比較表

凡例: 確認=公式ヘルプ等を読んだ / 弱=リスティング・要約・ベンダー宣伝のみ / 未確認

| ツール | 対象 | 形式設定・勝ち上がり | 日程・コート自動化 | スコア入力 | 結果の自動反映 | タイブレーク | 観戦者向け | 料金 |
|---|---|---|---|---|---|---|---|---|
| Ultimate Central (TopScore 製。WFDF も採用) | アルティメット運営団体・大会 | 「ブロック」(Pool Play / Bracket Play)を作り、前ブロックの結果を次ブロックの初期シードに連結(確認)。ブラケットは単一/二重敗退/全順位決定戦から選択。プールは「New Pool」へのドラッグで再編 | 手動中心。複数ゲームをチェックして時間・場所を一括変更するスケジュールエディタ(確認)。自動生成は未確認 | ブロックごとに「誰が報告できるか」を設定。既定は登録選手全員。一定時間後に非管理者をロック。Best of 1/3/5。スピリット集計も同一画面(確認) | 連結ブロックに結果が流れる。未消化試合は「Finalize Games」で「Unplayed Tie」にして次へ進める(確認) | ヘルプに記載なし。「Pool Play Scenario」設定の中身も不明(未確認) | 確認は「非公開/公開」設定のみ。ライブ・通知は未確認 | 未確認 |
| Ultie.org | アルティメット大会(欧州中心の新興) | ―(未確認) | 形式・時間・コート数の制約から形式案を提案し、スケジュールを自動生成。64チーム13コートを自動生成し、人手の微調整が数時間(スポンサー記事。弱) | 現場スタッフ向けのクイック入力(弱) | ライブブラケット(弱) | 未確認 | ライブ日程・ブラケット(弱) | 未確認。G2 にレビューなし |
| Tournify (オランダ) | 草の根~中規模の球技・多競技 | ラウンドロビン群・ノックアウト・順位決定戦・親善試合・多段を組み合わせ。「Start」ボタンで直前フェーズの結果から次フェーズへ配置。敗者復活ブラケット可(確認: help.tournifyapp.com) | フィールド追加、試合時間+バッファ、審判割当、自動計画またはドラッグ&ドロップ、休憩挿入(確認) | 主催者・審判(ログインリンク)・チームが提出可。誰が出せるかを制御(確認) | 順位・ブラケットがリアルタイム更新(確認) | 「Tiebreaker configuration」はあるが順序等の詳細は未確認 | 大会ごとの Web サイト、スライドショー(会場モニタ)、アプリ(プッシュ通知)。アプリは有料プラン(確認) | 1大会ごと。無料=8チームまで、€40=60チーム+アプリ、€120=無制限(確認: tournifyapp.com/pricing) |
| Challonge | 主にゲーム/ブラケット中心 | グループステージ→ファイナルステージ。各グループからの進出数を設定。敗退方式では 2 の累乗が必要(確認: kb.challonge.com) | 未確認(スケジュール機能は薄いと推測) | 主催者入力が中心。入力で即ブラケット更新(弱: リスティング記事) | ブラケット自動進行(弱) | ラウンドロビンで最大3つ設定、手動タイブレークも可(確認の一部) | 未確認 | 未確認 |
| Toornament | eスポーツ | 未確認(検索で公式を取得できず) | 「自動スケジュール」を謳う(弱) | 未確認 | 未確認 | 未確認 | 未確認 | 未確認。初心者には機能過多との評(弱) |
| start.gg | 格闘ゲーム等 | プール→アマチュアブラケットへの自動/手動の進出。自動進出の設定は start.gg スタッフ側(確認) | 未確認 | 未確認 | 未確認 | 未確認 | 未確認 | 未確認 |
| Pickleball.com / PickleballBrackets | ピックルボール大会 | ―(未確認) | Live Console の「Fill Empty Court」で空きコートに次の試合を自動割当(確認) | 勝者が「Finish Match」で入力 → 相手が Confirm/Dispute。自動承認までの分数を設定(確認) | 承認後に次の試合を自動割当 | ― | コート割当・スコア入力リンクを SMS で通知(SMS は追加費用)(確認) | 未確認 |
| TeamSnap Tournaments | 米国のユース大会 | スケジュール作成、ブラケット生成(弱) | 未確認 | アプリでスコア更新 | プール順位表示 | 未確認 | 通知、会場案内 | 未確認。App Store 評価 2.9/5(42件) |
| TONAMEL (KAYAC) | 日本のゲーム大会 | 単一/二重敗退、スイス、FFA、総当たり(リーグ戦)(確認) | 日程・コート機能は確認できず | 「主催者のみ」か「主催者+参加者」を選択。勝者が報告、敗者が承認。時間経過で自動承認(確認) | 承認で大会が進行 | 未確認 | 未確認 | 基本無料(KAYAC 英語ページ。弱) |
| USSSA / R2 Sports / Regystra 等 | 米国のユーススポーツ大会 | 「Pool to Bracket」で結果から自動シード、または手動選択(USSSA。検索要約のみ)。R2 は各プールから何チーム進むかを定義し、クロスオーバーと再シードを自動処理(確認: r2sports.com) | Regystra: 自動/手動/CSV 取込(弱) | スコアキーパーを試合・部門ごとに割当(Regystra。確認) | 試合確定でブラケット自動更新 | R2: 直接対決・得失点・失点を自動適用(確認) | 携帯で「ラウンド別/プール別/選手別」日程表示(R2) | 未確認 |
| Zuluru / LeagueApps / Playpass / Spond / SportsEngine / Exposure / Tourneymachine | ― | 詳細未確認。Playpass は登録・決済・Web サイトが主でブラケットは弱め(弱) | ― | ― | ― | ― | ― | ― |
| 日本の汎用アプリ (MatchFlowX, 対戦表-時程表, Tourney, Winner, OFTN.GG) | 日本の小規模大会 | 総当たりとトーナメントの自動作成、日程の自動作成(いずれもストア説明文) | 「対戦表から時程表を自動作成」「コート(会場)管理」(対戦表-時程表。弱) | 結果入力で順位表更新(MatchFlowX。弱)。OFTN.GG は参加者がログイン不要で、閲覧 URL を共有、ページ更新なしで進行が反映(PR Times。弱) | ― | ― | QRコード共有、オフライン(MatchFlowX。弱) | OFTN.GG は基本無料(弱) |

**日本のアルティメット固有**: JFDA 等の速報システムは検索で確認できず(結果は後日のプレスリリース中心)。アルティメット向けのコート割りツールも日本語検索では見つからなかった(検索ベース。存在しないとは言えない)。

主な出典
- Ultimate Central ヘルプ: https://help.ultimatecentral.com/support/solutions/articles/4000057205-schedule-admin
- WFDF が Ultimate Central を採用(見出しのみ確認): https://ultiworld.com/livewire/wfdf-joins-ultimate-central-for-event-management/
- Ultie.org(スポンサー記事。本文は 403 で取得できず、検索要約のみ): https://ultiworld.com/2024/06/06/how-three-european-tournaments-saved-loads-of-nerves-and-trees/ , https://www.g2.com/products/ultie-org/reviews
- Tournify: https://help.tournifyapp.com/en/articles/15069302-getting-started-with-tournify , https://tournifyapp.com/features , https://tournifyapp.com/pricing , https://tournifyapp.com/en/customer-stories
- Challonge: https://kb.challonge.com/en/article/learn-about-challonge-competition-formats-1f8j1cf/
- start.gg: https://help.start.gg/en/articles/13813635-running-an-amateur-bracket
- Pickleball.com: https://pickleball.com/docs/en/72000646262-how-to-enable-player-scoring-and-auto-run-your-tournament.html
- TONAMEL: https://tonamel.com/help/create-tournament
- TeamSnap Tournaments: https://apps.apple.com/us/app/-/id1127506805
- R2 Sports: https://www.r2sports.com/site/r2poolPlay.asp
- Regystra: https://regystra.com/features/tournament-scheduling-seeding
- USSSA: https://support.dc.usssa.com/hc/en-us/articles/49339097171731-Event-Scheduling-Moving-Pool-to-Bracket (本文は 403。検索要約のみ)
- 日本のアプリ: https://apps.apple.com/mx/app/matchflowx/id6754811076 , https://apps.apple.com/jp/app/%E5%AF%BE%E6%88%A6%E8%A1%A8-%E6%99%82%E7%A8%8B%E8%A1%A8/id6736855047 , https://prtimes.jp/main/html/rd/p/000000011.000158970.html
- 比較リスティング(アフィリエイト色が強く信頼度低): https://strivecloud.io/?p=320921
- Leaguevine: 創設の経緯(Twitter ベースのスコア速報)のみ検索要約で確認。終了の有無・惜しまれた点は未確認。

## 2. 運営者が「助かる」と言う機能と、不満のパターン

### 助かる(複数ツールに共通)
1. **直前の変更が全員に即反映される**
   - Tournify の導入事例(ベンダー掲載の選別された声): 「Last-minute cancellation or addition? No problem, everyone always has the correct version!」(Arvid Egbertzen, EFC'58)。「When two teams dropped out unexpectedly, we reorganised the group in minutes and stayed right on track.」(Rich Howell, Hardwicke Rangers)。https://tournifyapp.com/en/customer-stories
   - TeamSnap Tournaments も「変更を全チームと観客に即時伝える」と宣伝(弱)。
2. **結果入力が順位・ブラケット・公開ページへ一括で波及する**: Tournify、Regystra、R2、Challonge で共通して謳われる。「ライブ得点で選手・コーチ・応援が全試合を追えた」(Gorillas Rugby UK。Tournify 事例)。
3. **審判・スコアキーパーを分けて権限管理する**: Tournify の審判ログインリンク、Regystra の試合/部門ごとのスコアキーパー割当、Ultimate Central の「誰が報告できるか」設定。
4. **選手が自分で入力し、相手が承認する**: TONAMEL(勝者が報告、敗者が承認、時間で自動承認)、Pickleball.com(勝者入力、相手が Confirm/Dispute)。運営の入力負荷を下げる狙い。
5. **試合が終わると空きコートに次の試合を自動で当てる**: Pickleball.com。運営テーブルが回る(ただし選手入力を有効にした大会のみ)。
6. **前大会・前ステージの設定を再利用する**: TONAMEL の「過去の大会をコピー」。

### 不満・弱点(根拠の強さを併記)
- **古いアプリの質**(実例・出典あり): TeamSnap Tournaments の App Store 評価 2.9/5(42件)。2021 年のレビュー「トップのボタンがステータスバーに重なり動かない」「2 年以上更新されていない」、2024 年のレビュー「試合をしていない年齢区分が表示される」。https://apps.apple.com/us/app/-/id1127506805
- **機能過多で初心者に難しい**(Toornament。リスティング記事の評。弱)。
- **Challonge は大会全体の運営機能が薄い**(順位の通算管理、決済・同意書が無い。リスティング記事。弱)。
- **自動進出の設定を運営会社に頼む必要がある**(start.gg の自動プール進出は start.gg スタッフが設定。公式ヘルプ。運営者の自由度の制約として読める)。
- **Ultimate Central のヘルプは日程編集・スコア訂正・タイブレークを説明していない**: 確認できたのは設定項目のみ。これは「使いにくい」という証拠ではなく、公開情報が不足しているという事実(同ヘルプ)。
- **自動日程でも人手の調整が残る**: Ultie.org の事例で、64 チーム 13 コートの自動生成後に支援側が数時間の手動調整(スポンサー記事。弱)。
- **推測**: アプリ配布が有料プラン限定(Tournify)だと、小規模大会の観戦者はプッシュ通知を使えない。無料枠の線引きは不満になりやすい。ただし不満の声は未確認。
- **未確認**: Reddit/フォーラムの具体的な不満の声は、今回の検索では取得できなかった。主張を補強するには、運営者への直接ヒアリングが必要。

## 3. 画面の流れ・UX の優れた例

| 場面 | 例 | 内容 | 出典 |
|---|---|---|---|
| 大会作成の流れ | Tournify | 1.名称・日程・会場・ディビジョン → 2.参加チーム → 3.形式 → 4.日程 → 5.公開 → 6.結果管理。6 段階で順序が明確 | help.tournifyapp.com の上記記事 |
| 大会作成の流れ | TONAMEL | ゲーム選択 → フォーム → 公開。過去大会のコピー | tonamel.com/help/create-tournament |
| ドラッグ&ドロップ | Ultimate Central | プール間でチームをドラッグ、「New Pool」領域へ落として新プール作成 | ヘルプ |
| ドラッグ&ドロップ | Tournify | 試合をドラッグして日程枠に配置、または自動計画 | features |
| 日程の一括編集 | Ultimate Central | 複数試合にチェックを入れ、時間・場所をまとめて変更 | ヘルプ |
| 当日の進行ダッシュボード | Pickleball.com の Live Console | 「Fill Empty Court」で空きコートに自動割当。選手には SMS でコートと入力リンクを送る | pickleball.com/docs |
| スコア入力と訂正 | Pickleball.com | 勝者のみ入力(重複防止)、相手が Confirm/Dispute。トーナメントは 1 分、リーグは即時承認など、形式ごとに確認時間を変える。紛争は運営デスクへ | 同上 |
| スコア入力と訂正 | TONAMEL | 勝者が報告、敗者が承認、運営はブラケット上で直接修正可 | tonamel.com/help/create-tournament |
| スコア入力の制限 | Ultimate Central | 一定時間後に非管理者の入力をロック | ヘルプ |
| 未消化試合の扱い | Ultimate Central | 「Finalize Games」で未消化を Unplayed Tie にして次ブロックを埋める。時間切れで強制進行できる運営向けの工夫 | ヘルプ |
| 勝ち上がりの見せ方 | Tournify | 「Start」ボタンで次フェーズに配置(運営が意図して押す) | help.tournifyapp.com |
| 勝ち上がりの見せ方 | Ultimate Central | 前ブロックの結果から次ブロックの初期シードを連結 | ヘルプ |
| 勝ち上がりの見せ方 | R2 | 「何チーム進むか」を定義し、クロスオーバーと再シードを自動処理。プール戦とプレーオフの間に時間バッファを置くことを推奨 | r2sports.com |
| 観戦者向け | Tournify | 大会ごとの Web サイト、会場スライドショー、アプリ通知 | features |
| 観戦者向け | OFTN.GG (日本) | ログイン不要で閲覧 URL を共有、更新操作なしで進行が反映(PR Times の説明。弱) | prtimes.jp |

未確認: 各ツールの実画面の質(スクリーンショットは取得していない)。モバイル入力 UI の細部は、ヘルプの文章からは分からない。

## 4. 私たちの製品に取り入れるべきもの / 避けるべきもの

### 取り入れる
1. **ステージ連結モデル(Ultimate Central のブロック連結、Tournify の多段)**。理由: 「予選プール → 決勝トーナメント → 順位決定戦」を同じ仕組みで表せ、勝ち上がりの自動化の土台になる。
2. **勝ち上がり規則を「各プールから何チーム」と宣言する方式(R2)**。理由: 運営者の頭の中の規則に近く、クロスオーバーと再シードを自動化できる。
3. **確定ボタン方式の進行(Tournify の Start)と、未消化試合の強制確定(Ultimate Central の Finalize Games)**。理由: 完全自動で勝手に進むより、運営が最終確認して進める方が当日の事故に耐える。**推測**: 遅延が日常的な大会ほど効く。
4. **スコア入力の権限と確認の設定**: 誰が入力するか(運営/コートの記録係/チーム代表)、確認方法、時間経過による自動承認またはロック。理由: Pickleball.com・TONAMEL・Ultimate Central の 3 製品が同様の設計に収束している。形式(予選は緩く、決勝は厳しく)で確認の厳しさを変える点も採用する。
5. **入力のたびに順位・次の試合・公開ページが同時に更新**されること。理由: 「助かる」の最大公約数。
6. **訂正のしやすさ**: 運営がブラケット上で結果を直接直せること(TONAMEL)と、訂正時に下流(順位・進出先)へ波及する影響を事前に表示すること(後者は他社で確認できていない、当方の提案)。
7. **前回の大会のコピー**(TONAMEL)。理由: 毎回同じ形式の運営会社に刺さる。
8. **ログイン不要の閲覧 URL と QR コード**(OFTN.GG、MatchFlowX)。理由: 観戦者の入口が最短になる。プッシュ通知は後回しでもよい(推測)。
9. **日程の自動生成は「叩き台」として設計し、ドラッグ&ドロップで手直し**(Ultie.org の事例が示すとおり、自動後にも手直しが残る)。コート数・試合時間・休憩・同一チームの連戦間隔を制約として入力する(Tournify の項目が参考)。
10. **コートの空き管理による次試合の自動割当**(Pickleball.com)。ただし日本のアルティメットでは時間固定の運営が多いはずで、必要性は未確認(要ヒアリング)。

### 避ける
1. **全部入りの設定画面**(Toornament が「初心者に難しい」と評される。弱)。理由: アルティメット運営に絞れば必要な形式は少ない。形式テンプレートから始める。
2. **自動進出の設定を運営会社に頼る構造**(start.gg の自動進出設定は同社スタッフ作業)。理由: 当社の売りは運営者自身が自動化できること。
3. **観戦者用アプリを有料プラン限定にする**(Tournify)。理由: 観戦者の接点はアプリより Web の速報ページの方が重要と推測。まず Web を無料で十分に作る。
4. **更新が止まった重いネイティブアプリ**(TeamSnap Tournaments の低評価)。理由: モバイル Web(PWA)の方が更新と運用が軽い。
5. **選手全員に SMS を送る設計**(Pickleball.com は追加費用)。理由: 日本では LINE 等が自然だが、採用判断は未確認。
6. **タイブレーク規則の暗黙化**。Ultimate Central のヘルプでは説明が無く、設定内容が分からない。アルティメットの規則(直接対決、得失点差等)を画面上で順序つきに明示し、同順位が残ったら運営が手動で順位を確定できる逃げ道を用意する(Challonge の手動タイブレークと同じ考え)。

### 次にやるべき調査(今回の未確認事項)
- Ultimate Central / Ultie.org の実画面(デモアカウントまたは動画)。
- 日本のアルティメット運営者 3~5 名へのヒアリング(使っているツール、スプレッドシート運用の実態、日程の制約、記録係の端末)。
- Exposure Events、Tourneymachine、SportsEngine、Toornament、Spond、LeagueApps の公式ヘルプ(今回は検索で取得できなかった)。
- Leaguevine の終了有無と惜しまれた点。
