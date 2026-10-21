# S3 スプレッドシート運用の実態調査(大会運営)

調査日: 2026-10-21。調査者: ユーザーリサーチ担当(WebSearch/WebFetch)。

## 0. 調査の限界(最初に読むこと)

- Reddit(old.reddit.com 含む)は WebFetch が禁止で直接取得できず、WebSearch の `site:reddit.com` でも 1 件も返らなかった。r/excel, r/googlesheets, r/spreadsheets, r/ultimate, r/youthsports, r/soccercoaching は **該当なし(取得不能)**。ここは未調査であり、「存在しない」ではない。
- Microsoft Community / Super User / Web Applications / Google ドキュメント エディタ ヘルプ: 検索では該当スレッドが出なかった(**該当なし**)。代替として ExcelForum と MrExcel を使った。これらの多くは WebFetch が 403 で、検索抜粋のみ。
- 日本語の個人ブログ・note による『大会運営の手順と手間』の体験談は、検索語を変えても直接ヒットしなかった(**該当なし**)。日本側は、テンプレ配布者の解説、競技団体マニュアル、ベンダー資料、行政文書が中心。
- ベンダー記事(Sofascore, Playbook365, Eggbeater 等)は宣伝を含む。需要の裏付けとしては弱い。
- 事例の『読了度』欄で、全文を読めたものと検索抜粋のみのものを区別した。引用が『検索抜粋』と書かれたものは、検索要約に基づく。

## 1. 事例一覧(44 件)

分類記号: A=同率の処理(タイブレーク・直接対決) / B=勝ち上がり・ブラケット自動反映 / C=組み合わせ生成(総当たり・抽選・スイス) / D=日程とコート割り当て / E=スコアの集計と共有(入力・速報・閲覧) / F=当日の変更・遅延対応 / G=規模拡大・保守性・専用ツール移行理由

### S3-01 [A] 汎用(リーグ表)
- 立場: 不明(質問者)
- 要約: 3チーム同率を直接対決で解く方法を質問。同率チーム間の試合だけを抜き出す所が難所で、VLOOKUPで結果を引く案止まり。完成式なし
- 引用: (検索抜粋に基づく要約。直接引用なし)
- 出典: https://www.excelforum.com/excel-general/1434811-how-do-i-resolve-a-3-way-tie-in-a-round-robin-group.html
- 日付: 不明 / 読了度: 抜粋のみ(本文は403)

### S3-02 [A] 汎用(総当たり)
- 立場: 不明(質問者)
- 要約: 総当たりの着順決定と同率判定。RANK+COUNTIFで同率に一意の順位を付ける回答
- 引用: =RANK(Q3,$Q$3:$Q$13,1)+COUNTIF($Q$3:Q3,Q3)-1(検索抜粋)
- 出典: https://www.excelforum.com/excel-formulas-and-functions/1095787-i-need-to-rank-order-of-finish-in-a-round-robin-competition-and-determine-ties.html
- 日付: 不明(古いスレ) / 読了度: 抜粋のみ(403)

### S3-03 [A] 汎用(対戦形式)
- 立場: 不明
- 要約: 直接対決での同率解消を質問(Excel Forum)
- 引用: (検索抜粋のみ。詳細未確認)
- 出典: https://www.excelforum.com/excel-formulas-and-functions/984236-breaking-ties-in-a-head-to-head-contest.html
- 日付: 不明 / 読了度: 抜粋のみ(403)

### S3-04 [A] インドアサッカー
- 立場: 運営者(リーグ管理者)とみられる
- 要約: 順位表で直接対決を3番目の基準にしたい。同じ相手と複数回戦い勝敗が割れる場合に得失点へフォールバックが必要
- 引用: (検索抜粋の要約のみ)
- 出典: https://www.excelforum.com/excel-general/796693-indoor-soccer-league-table-tiebreaker-help.html
- 日付: 不明 / 読了度: 抜粋のみ(403)

### S3-05 [A] 汎用(総当たり)
- 立場: 不明
- 要約: 試合数→ゲーム数→得点の多段順位。RANK+SUMPRODUCTで1式に統合する回答
- 引用: =RANK($N4,$N$4:$N$13,0)+SUMPRODUCT(...)(検索抜粋)
- 出典: https://www.excelforum.com/excel-general/816739-round-robin-tournament-tie-break-via-matches-games-points.html
- 日付: 不明 / 読了度: 抜粋のみ(403)

### S3-06 [A] 汎用(総当たり)
- 立場: 不明
- 要約: 最終順位の式を質問。複数基準を1つの数値に合成して1回で並べ替える案(合計×1000+得失点×100+得点×10)が出た
- 引用: (検索抜粋の要約のみ)
- 出典: https://www.mrexcel.com/board/threads/help-with-formula-to-determine-final-standings-in-a-round-robin-competition.559881/
- 日付: 不明 / 読了度: 抜粋のみ(403)

### S3-07 [A] ハンドボール
- 立場: 不明
- 要約: 3チーム以上の同率で、直接対決の単純勝敗でなく直接対決の得失点差を使いたい
- 引用: (検索抜粋の要約のみ)
- 出典: https://www.mrexcel.com/board/threads/ranking.1256322/
- 日付: 不明 / 読了度: 抜粋のみ

### S3-08 [A] サッカー(W杯方式)
- 立場: チュートリアル筆者
- 要約: 直接対決を含む順位計算の解説。直接対決なしだと式が絡み、9チーム1グループで旧来は1,000超の配列式が必要だったと述べる
- 引用: an impenetrable tangle of formulas / 4チーム超だと head-to-head は大きな労力
- 出典: https://hermann-baum.de/excel/Turnier/en/ranking.php
- 日付: 不明 / 読了度: 全文

### S3-09 [D,F,B] アルティメット
- 立場: TD(ライター)
- 要約: スケジュールは『最適化問題』。コート列×ラウンド行のExcel表で試合を並べる。土曜プール・日曜ブラケット。コンソレで不戦(bye)が増え、途中で帰るチームが出る。1チーム最低6試合の確保
- 引用: Creating the perfect tournament schedule is an optimization problem / An extremely helpful tool ... is an Excel spreadsheet
- 出典: https://skydmagazine.com/2011/12/td-tools-creating-tournament-schedule/
- 日付: 2011-12-23 / 読了度: 全文

### S3-10 [B,A] アルティメット
- 立場: コラムニスト
- 要約: プール順位で次の対戦が決まる方式では順位操作の誘因が生じる。NationalsはプールのAll16進出で最下位狙いも成立。同率・無意味試合が順位を濁す
- 引用: teams can manipulate their results to achieve specific matchups
- 出典: https://skydmagazine.com/2015/03/partial-discussion-tournament-formats-pandoras-box/
- 日付: 2015-03-04 / 読了度: 全文

### S3-11 [A] アルティメット
- 立場: フォーマット規程(UPA)
- 要約: 3チームプールで全員1勝1敗の三すくみ時は標準タイブレーク→得失点差、全部同じなら追加1点。開始前に協議必須
- 引用: in any tournament where a three-team pool exists, this tie-breaker must be discussed before play starts
- 出典: https://www.cs.rochester.edu/u/ferguson/ultimate/upa-formats/upa-formats-3.html
- 日付: 不明 / 読了度: 全文

### S3-12 [A] アルティメット(大学)
- 立場: 大会規程
- 要約: 同率時は(1)直接対決 (2)同率チーム間の得失点差 … (5)コイン投げ。大会ごとに規定が異なる
- 引用: Determining finishes in case of a tie ... 1. Head-to-head 2. Point differential among teams in the tie ... 5. Coin flip
- 出典: https://play.usaultimate.org/events/Santa-Barbara-Invite/Rules
- 日付: 不明 / 読了度: 抜粋のみ

### S3-13 [C,D,E] アルティメット
- 立場: 配布者(スイスドロー用)
- 要約: スイスドロー運営用のExcel(VBA)。結果表から日別スケジュールを自動で埋め、最終順位も表示。無料配布
- 引用: (検索抜粋の要約のみ)
- 出典: https://ultimatecentral.com/fr_ch/p/ultimate-frisbee-tournament-101-swiss-draw-excel-template
- 日付: 不明 / 読了度: 抜粋のみ(403)

### S3-14 [G,E] アルティメット
- 立場: ベンダーの導入者
- 要約: 『28人のリーグをスプレッドシートで回すのは大変だった』という導入事例の声(ベンダー掲載のため宣伝割引)
- 引用: Managing a 28-player league in a spreadsheet would have been a big pain.
- 出典: https://playpass.com/sports-software/ultimate-management
- 日付: 不明 / 読了度: 全文

### S3-15 [B,G] 汎用(トーナメント)
- 立場: アドオン開発者/利用者
- 要約: Sheetsアドオン(7K+)。勝者チェックで次戦へ自動反映。評価1.7(約3件)。要望『32チームに増やして』(上限16)、テンプレ取込エラー。プール戦は対象外
- 引用: can you add 32 teams? The limit is 16.
- 出典: https://workspace.google.com/marketplace/app/tournament_brackets/413708349137
- 日付: 要望: 2026-06-25 / 読了度: 全文

### S3-16 [B,D] 汎用(トーナメント)
- 立場: アドオン開発者
- 要約: Sheetsアドオン(2万+ユーザー)。高スコア側を自動で次戦へ。敗者復活、試合ごとの締切自動計算。レビュー0件。形式はトーナメント系
- 引用: Automatically bumps up the participant with the higher score to the next match.
- 出典: https://workspace.google.com/marketplace/app/tournament_extensions/684126244332?hl=ja
- 日付: 更新 2024-01-01 / 読了度: 全文

### S3-17 [B] 汎用(トーナメント)
- 立場: ガイド/コミュニティ
- 要約: 式による勝ち上がりでも勝者の指定は人手が要る。チェックボックスをラジオボタン化するスクリプトで1クリック化。BYEは名前に『Bye』と入れる運用
- 引用: The user must indicate who won.(検索抜粋)
- 出典: https://groups.google.com/g/siotibandrac/c/oyoGSyyso4E
- 日付: 不明 / 読了度: 抜粋のみ

### S3-18 [C,D] 汎用(リーグ運営)
- 立場: アプリ開発者フォーラム質問者
- 要約: 6チーム・3コート・1日1試合/チームで日程自動生成。『頭が回らない』。総当たりを繰り返し最大75試合。コート数で試合数上限、同日重複禁止
- 引用: I'm having a mental block
- 出典: https://forum.bubble.io/t/how-to-create-a-game-schedule-for-a-sport-league-round-robin-tournament/239337
- 日付: 2023-01-01 / 読了度: 全文

### S3-19 [C,G] ビリヤード
- 立場: リーグ運営者(約20年)
- 要約: 専用ソフトなしでExcelで4〜24チームの総当たり日程を作る手法とXLSを配布。PDF1,208閲覧/XLS669閲覧
- 引用: (本文要約のみ。引用なし)
- 出典: https://forums.azbilliards.com/threads/how-to-create-a-round-robin-schedule.276997/
- 日付: 2012-06-01 / 読了度: 全文

### S3-20 [E,C,F] ビリヤード
- 立場: 月例大会の幹事
- 要約: 結果・通算順位・直接対決成績を集計するシートを共有。全員出席前提の旧方式が破綻し毎晩精算へ変更。待ち時間を均す round-robin 生成を併用
- 引用: in order to keep accurate score, everyone needs to show up for each pool night
- 出典: https://forums.azbilliards.com/threads/tournament-spreadsheet-template.151261/
- 日付: 2009-08-02 / 読了度: 全文

### S3-21 [C,D,E] ピックルボール
- 立場: テンプレ販売者
- 要約: 有料Sheetsテンプレ(6〜16チーム)。総当たり生成、コート・BYE割り当て、リーダーボード
- 引用: (検索抜粋の要約のみ)
- 出典: https://www.etsy.com/listing/1652233546
- 日付: 不明 / 読了度: 抜粋のみ

### S3-22 [G,A,D,F,B] ユース全般
- 立場: ベンダーのガイド
- 要約: スプシ+グループLINE相当は『12チーム超か複数年代同時』で破綻。ボランティアが調整業務に潰れる。同率は(1)勝敗(2)直接対決(3)得失点差(±6上限)(4)総得点(5)コイン。10分遅れたらウォームアップ短縮、遅延管理者を一人に
- 引用: work, until it does not / 90分ごとに遅延を確認
- 出典: https://www.eggbeater.app/tournament-guide
- 日付: 不明 / 読了度: 全文

### S3-23 [E,F,G] ユース/リーグ全般
- 立場: ベンダー記事
- 要約: 1つの変更を複数箇所に手で入力する誤り、チームからの進捗問い合わせ対応、シーズン終了時に最終表しか残らない
- 引用: one change no longer means four manual edits in four places
- 出典: https://www.sofascore.com/news/tournament-management-software-four-problems-it-solves-for-organisers
- 日付: 2026-09-15 / 読了度: 全文

### S3-24 [D,E,F,G] ユーススポーツ大会
- 立場: 大会運営会社(複数)
- 要約: スプシ+電話+メールの併用で、日程変更が保護者に届かない。必要機能: ドラッグ&ドロップ日程、二重予約チェック、ブラケットのライブ更新、変更の即時通知
- 引用: schedule changes that never reached parents(要約)
- 出典: https://www.sportsdestinations.com/management/technology/right-tournament-software-can-help-organizers-run-36604
- 日付: 2025-11-01 / 読了度: 全文

### S3-25 [B,E,G] eスポーツ/ブラケット
- 立場: ベンダーのブログ
- 要約: 結果と配置の手入力が時間がかかり、誤りが進出に影響。参加者が増えると維持困難。更新待ち
- 引用: became difficult to maintain as participation increased
- 出典: https://online-brackets.mataroa.blog/blog/the-modern-way-to-manage-online-tournament-bracket-events/
- 日付: 不明 / 読了度: 全文

### S3-26 [G] 汎用(大会運営)
- 立場: ベンダー
- 要約: Excel運用の問題を解く目的で2008年に開発したと説明
- 引用: The idea originated in 2008 to overcome the difficulties posed by existing Excel-based solutions.(検索抜粋)
- 出典: https://tournoiplus.fr/about.php
- 日付: 2008(創業) / 読了度: 抜粋のみ

### S3-27 [A,E,G] ジュニアサッカー
- 立場: コーチ(エンジニア兼)
- 要約: 4チーム総当たり表を関数で半自動化(IF/COUNTIF/RANK.EQ)。当日は電卓で計算し間違いを何度も確認していた。順位式は勝点→得失点→総得点に固定、規定が違うと式の修正、チーム数変更も修正
- 引用: 大会当日に電卓をたたいて結果を計算している(要約)
- 出典: https://soccer.webnote-plus.com/excel-matchtable-semiautomatic/
- 日付: 2020-05-02(更新2022-08-31) / 読了度: 全文

### S3-28 [E,A,G] 学校スポーツ(リーグ表)
- 立場: 教員(20年)
- 要約: 個人戦リーグ表を自作配布。1か所入力で相手側も反映。同勝点は同順位になり、決定は手動。未入力は引き分け扱い。40人超は行列追加では不可
- 引用: 入力作業を1箇所に / 私と同じ悩みの人がいると思い作成
- 出典: https://schoolteacher.kimamam.com/excel-many-people-leaguetable/
- 日付: 不明 / 読了度: 全文

### S3-29 [C,B] 汎用(予選リーグ→決勝T)
- 立場: 配布者(フリーソフト)
- 要約: ExcelVBAで予選リーグ→決勝トーナメントを生成。同一団体を同リーグに入れない抽選、最大約200チーム、再利用可。レビュー0件
- 引用: (本文要約のみ)
- 出典: https://www.vector.co.jp/soft/win95/home/se477598.html
- 日付: 不明 / 読了度: 全文

### S3-30 [B,D,C] 汎用(大会形式)
- 立場: ベンダー(日本)
- 要約: 16チーム4組総当たり→決勝T/順位決定T。コート数と試合間隔(分)を入れて日程を試算するUI。1回戦の組合せ変更可
- 引用: スケジュールをシミュレーション / コートの数 / 試合間隔(分)
- 出典: https://matchday-live.com/template/123
- 日付: 不明 / 読了度: 全文

### S3-31 [E,C,D] バドミントン
- 立場: 競技団体(マニュアル)
- 要約: 全国大会の運営手順。開催3年前から準備。組合せ会議、タイムテーブル、審判員配置、プログラム作成、結果速報の段取り、毎日の通信社への結果報告、大会記録結果報告書の作成などの工程が多数
- 引用: ②試合結果速報の段取り / 通信社等への毎日の結果報告 / 組合せ会議の運営
- 出典: https://www.badminton.or.jp/asset/pdf/corporate/concerned/operation/manual_20240331.pdf
- 日付: 2024-04-01 / 読了度: 全文(PDFをpdftotextで確認)

### S3-32 [B,C,E] 高専バドミントン大会
- 立場: 運営教員(論文)
- 要約: トーナメント表を事前にExcelで作成し、抽選結果から選手名を自動配置。進行を半自動で支える。※URL現在404、検索抜粋のみ
- 引用: The sheets realized useful progress of the tournament semi-automatically(検索抜粋)
- 出典: https://kumamoto-nct.ac.jp/file/knct-kiyou-2009/PDF/03.pdf
- 日付: 2009 / 読了度: 抜粋のみ(本文404)

### S3-33 [A] ソフトボール(日本)
- 立場: 大会規程
- 要約: 同勝敗時は(1)該当チーム同士の対戦成績(2)該当チーム同士の得失点差(3)総得失点差(4)総失点(5)前年度順位
- 引用: 勝敗が同じ場合のリーグ順位は、(1)該当チーム同士の対戦成績 (2)該当チーム同士の得失点差 …
- 出典: https://www.hitachi.co.jp/Div/soft/SOFTBALL/result/y2024/league/popup_record.html
- 日付: 2024 / 読了度: 抜粋のみ

### S3-34 [A,B] バレーボール
- 立場: 大会FAQ
- 要約: 1位プールを2位プールより上位。同率は(1)マッチ勝率(2)セット勝率(3)プール得失点差の順で、プール順位からブラケットを決める
- 引用: (検索抜粋の要約のみ)
- 出典: https://carolinaregionvb.org/wp-content/uploads/2018/02/faq-trnplayoffsrev2023.pdf
- 日付: 2018〜2023 / 読了度: 抜粋のみ

### S3-35 [A] バレーボール(国際)
- 立場: 仕様書(FIVB)
- 要約: 直接対決による並べ替えは2チーム間の同率にしか使えず、3チーム以上は別基準
- 引用: (検索抜粋の要約のみ)
- 出典: https://www.fivb.org/VisSDK/Fivb.Vis.DataWPF/Fivb.Vis.DataWPF~Fivb.Vis.Tournament.RoundRankingSortMethod~DirectConfrontation.html
- 日付: 不明 / 読了度: 抜粋のみ

### S3-36 [E,A,G] 汎用(リーグ表)
- 立場: Microsoft公式テンプレ
- 要約: リーグ戦対戦勝敗表。得点入力で勝敗と順位が自動表示、勝点・得失点も自動。最大8チーム
- 引用: (検索抜粋の要約のみ)
- 出典: https://www.microsoft.com/ja-jp/office-pipc-template/result.aspx?id=10254
- 日付: 不明 / 読了度: 抜粋のみ

### S3-37 [A] ラウンドロビン一般
- 立場: ガイド
- 要約: 勝敗のみの競技(バレー・ピックルボール)は勝敗列+タイブレーク。シンプルな2-1-0方式を推奨
- 引用: (検索抜粋の要約のみ)
- 出典: https://joinstriveon.com/coaching-tools/round-robin-tournament
- 日付: 不明 / 読了度: 抜粋のみ

### S3-38 [B] 汎用(トーナメント)
- 立場: 記事(マイナビ)
- 要約: Excelでのトーナメント表はセル結合と罫線で『描く』解説が中心。勝者の自動反映は含まれない
- 引用: (検索抜粋の要約のみ)
- 出典: https://news.mynavi.jp/techplus/article/Excelgraphzissentech-72/
- 日付: 不明 / 読了度: 抜粋のみ

### S3-39 [B] 汎用(トーナメント)
- 立場: Google公式ドキュメント
- 要約: Apps Scriptで最大64人/チームのブラケットを作る公式解説。Sheets自動化の需要を示す
- 引用: (検索抜粋の要約のみ)
- 出典: https://developers.google.com/apps-script/articles/bracket_maker.html?hl=ja
- 日付: 不明 / 読了度: 抜粋のみ

### S3-40 [E] グラウンド・ゴルフ
- 立場: 大会運営者(自治体系とみられる)
- 要約: スマホ入力のスコア集計アプリで、紙のスコアシートの手集計に要していた約1時間を削減
- 引用: (検索抜粋の要約のみ。出典URL未保持のため確認不足)
- 出典: (出典URLは検索結果に不明確。確認不足)
- 日付: 不明 / 読了度: 抜粋のみ

### S3-41 [E,G] ソフトテニス
- 立場: 大会運営(報道)
- 要約: 進行管理と記録はシステム、審判のスコア記録は紙のスコアシートという併用方式
- 引用: 基本的な大会進行管理と記録は「raquty Pro」で行い、審判によるスコア記録は紙のスコアシート(検索抜粋)
- 出典: https://www.NTT-west.co.jp/newscms/oita/13450/oita_n20231127a.pdf
- 日付: 2023-11-27(推定) / 読了度: 抜粋のみ

### S3-42 [D,C,G] 日本・各種アプリ
- 立場: アプリ紹介(App Store)
- 要約: 対戦表からタイムスケジュールを自動作成、会場(コート)管理、PDF/CSV出力。学年別など複数カテゴリ大会向け
- 引用: リーグやトーナメントの試合日程やタイムテーブルを自動作成(検索抜粋)
- 出典: https://apps.apple.com/jp/app/%E5%AF%BE%E6%88%A6%E8%A1%A8-%E6%99%82%E7%A8%8B%E8%A1%A8/id6736855047
- 日付: 不明 / 読了度: 抜粋のみ

### S3-43 [E,F] ラグビー
- 立場: 協会(導入事例)
- 要約: 試合ごとの運営管理シートを印刷して出張していたが、スマホ閲覧と更新版配信に。スコア速報などの共有が円滑になった(ベンダー掲載)
- 引用: (検索抜粋の要約のみ)
- 出典: https://line-works.com/cases/rugby-japan/
- 日付: 不明 / 読了度: 抜粋のみ

### S3-44 [G] 部活全般(行政文書)
- 立場: 行政/教委
- 要約: 大会の運営が特定校や顧問の負担になっている、大会の統廃合や外部人材活用で負担軽減を求める
- 引用: 生徒や運動部顧問の過度な負担とならないよう、大会の統廃合等を要請(検索抜粋)
- 出典: https://www.pref.oita.jp:443/uploaded/attachment/2032682.pdf
- 日付: 不明 / 読了度: 抜粋のみ

## 2. 自動化したいこと・詰まりどころの分類と件数

1 事例が複数分類に該当するため合計は事例数を超える。件数は上の一覧のタグを機械集計した値。

| 記号 | 分類 | 件数 | 代表的な詰まり |
|---|---|---|---|
| A | 同率の処理(タイブレーク・直接対決) | 19 | 3チーム以上の同率で『同率チーム間だけ』を再集計する処理が式で書けない。直接対決が循環(三すくみ)すると決まらない。基準の順序が大会ごとに違い、式がハードコードされる |
| B | 勝ち上がり・ブラケット自動反映 | 13 | 勝者の入力が手動のまま。プール順位→ブラケット枠への流し込み(1A vs 2B 等)とBYEの扱い。参加数が変わるとテンプレが使えない(16上限など) |
| C | 組み合わせ生成(総当たり・抽選・スイス) | 10 | 総当たり生成(奇数チームのBYE含む)、抽選で同一団体を分ける、スイスドロー。チーム数が変わると式を作り直し |
| D | 日程とコート割り当て | 10 | コート数と時間枠の制約、同一チームの同時刻重複禁止、連戦の回避、休憩20分程度の確保、プールを隣接コートに置く配慮 |
| E | スコアの集計と共有(入力・速報・閲覧) | 15 | 結果の二重入力、電卓での検算、結果を見たい人への共有。印刷物・LINE・メールに分散し更新が届かない |
| F | 当日の変更・遅延対応 | 6 | 遅延で後続がずれる、コート閉鎖、棄権・没収試合、チームの途中帰り。変更が一箇所で済まず複数表を手で直す |
| G | 規模拡大・保守性・専用ツール移行理由 | 14 | 12チーム超や複数カテゴリでスプシが破綻。式の保守者が一人しかいない。ボランティア頼みの運営負荷 |

所見: 件数だけならA(同率)が最多。ただし収集元に ExcelForum/MrExcel の『数式質問』が多く入っており、数式で解きたい人に偏る。実際の運営者の痛みはD・E・Fに寄る可能性があるが、この調査ではその裏付けはベンダー記事と少数の事例のみ(要追加調査)。

## 3. 運営の手順で手間のかかる工程(大会前・当日・大会後)

根拠: 日本バドミントン協会マニュアル(S3-31)、TD Tools(S3-09)、Eggbeater(S3-22)、SoccerNote+(S3-27)、Playbook365/SportsDestinations(S3-24)。競技はアルティメットに限らない。

**大会前**
- 参加数の確定とフォーマット選定(プール数・サイズ、パワープール、進出枠)。1チーム最低試合数と使えるコート・日照からの逆算(TD Tools)。
- 総当たり・抽選の組み合わせ作成、シード、同一団体の分離。
- 日程とコート割り当て(コート×ラウンドの表を手で埋める)。休憩・BYE・連戦の調整。
- 同率規定の事前決定と周知(3チームプールは開始前に協議必須)。
- 組合せ会議・代表者会議の資料、プログラム冊子、審判員配置の作成(バドミントン協会の例)。
- 参加者への日程・組合せ連絡(PDF、メール、LINE)。
**当日**
- スコア報告の回収と本部での入力、順位表の更新(電卓で検算という記述あり)。
- プール終了時の順位確定(同率処理)と、その結果に基づくブラケット枠の確定・掲示。
- 遅延・コート閉鎖・棄権への対応と、変更の周知。遅延監視は約90分ごとの確認、遅延判断の窓口は一人に集約という推奨(Eggbeater)。
- 参加者・保護者・観戦者からの『結果はどうなったか』への対応。
**大会後**
- 最終順位の確定と結果報告書・通信社への報告(バドミントン協会の例)。
- 記録の保存。スプシ運用ではシーズン終了時に最終表しか残らないという指摘(Sofascore)。
- 会計・反省・次回への引き継ぎ(式の保守者がいなくなる問題)。

## 4. 専用ツールに求められている条件(スプレッドシートより良くないと移らない理由)

以下は一次の利用者発言ではなく、ベンダー記事・既存ツールの仕様・テンプレの限界からの推定が多い。直接の乗り換え体験談は見つからなかった。

1. **無料で、登録なしで即使える**。Sheetsアドオン(2万+, 7K+)やVectorのフリーソフトが使われる事実から。配布テンプレは無料が基準線。
2. **大会規程どおりの同率ルールを設定できる**。SoccerNote+は基準固定、ソフトボール・USAU・FIVB・バレー協会で順序が全て違う。固定だと使えない。3チーム以上の同率で『同率チーム内で再集計』『循環時の追加基準』に対応が必要。
3. **入力は1回**。二重入力(対戦表の両側)を嫌う声(教員のシート)と、1変更=複数箇所の手作業が誤りの原因という指摘。
4. **プール順位から決勝・順位決定の枠へ自動で流れる**。Sheetsアドオンの利用者が最も使うのはこの勝ち上がり。ただしプール戦非対応で限界。
5. **数に制限がない**。『32チームに増やして』の要望、Microsoftテンプレは最大8チーム、教員シートは40人上限。
6. **日程とコートの制約を満たす生成と、当日の再割り当て**。コート×時間の表を手で埋める作業が最大の手作業(TD Tools, Bubble質問)。
7. **変更が一箇所で済み、見る側にも即反映**。LINE/PDF/メールの分散で保護者に届かない問題。
8. **紙・オフライン併用を許す**。ソフトテニスは審判の記録は紙のまま、本部だけシステム。会場の通信が不安定でも動くことが前提になりやすい(推定)。
9. **スプシの出力を取り込める/出せる**。Eggbeaterは Google Sheet からの日程取り込みを選定条件に挙げる。既存の参加者名簿・運用との接続が移行障壁を下げる。
10. **式の保守者に依存しない**。教員・コーチ個人の自作シートが多く、保守は属人的(推定)。

## 5. 私たちの製品への示唆(優先順位つき)

前提: 根拠は上の事例。S3 だけでは運営者の優先度を測れないため、順位は『事例の頻度+実装リスク』による仮説。

1. **規程どおりに設定できるタイブレークエンジン(最優先)**。直接対決・同率チーム内得失点差・総得失点差・コイン投げ等を順序つき設定にし、3チーム以上の同率は『同率チームだけで再集計→まだ同率なら残りに再適用』を実装。三すくみと全指標同一は『未決』として人に明示し、黙って順位を付けない。根拠: S3-01〜08, 11, 12, 27, 33〜35, 37。
2. **プール順位→決勝・順位決定トーナメントへの自動反映**。順位が確定するまで『暫定』表示し、確定で枠に流す。BYE、クロスシード(1A vs 2B)、人数が2の冪乗でない場合に対応。根拠: S3-10, 15, 16, 22, 30。
3. **スコア1回入力→順位・ブラケット・速報へ即反映**。入力は現場スマホ/本部端末のどちらからも。参加者向けの閲覧URLを標準にし、LINE/紙の併用は共有リンクで補う。根拠: S3-23, 24, 27, 28, 41。
4. **日程とコート割り当ての自動生成と当日の再割り当て**。制約: コート数、同一チームの重複禁止、最低休憩、BYEの位置。遅延時は後続を自動でずらす提案を出す。根拠: S3-09, 18, 22, 30, 42。
5. **組み合わせ生成**(総当たり、抽選、同一団体の分離、必要ならスイス)。単体では需要が高い(配布物が多い)が、競合の無料ツールも多く差別化は弱い。1〜4 の土台として提供。根拠: S3-13, 18, 19, 21, 29。
6. **移行障壁の低さ**。CSV/スプレッドシートからの名簿・日程取り込み、紙の帳票出力(PDF/CSV)、無料枠、登録なし閲覧。根拠: S3-16, 22, 29, 42。
7. **後回しでよいもの**: 登録・決済・会員管理(Playpass 等が強い領域)、保護者向けの通知基盤。事例で多いのは順位・ブラケット・日程の計算であり、周辺機能ではない。

### 追加調査が必要な点
- Reddit と海外のアルティメット運営者コミュニティ(Reddit r/ultimate、Facebook グループ等)の生の声。この環境では取得できなかった。
- 日本のアルティメット運営者(JFDA 加盟、学連、社会人チーム)への直接ヒアリング。日本語の運営体験談は文書として見つからなかった。
- 『乗り換えた/乗り換えなかった』理由の一次情報。今回はベンダー発言と仕様からの推定が中心。