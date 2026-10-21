# D 仕様: チーム / 試合(対戦結果) / 選手 ページ (score.clubjr.com)

調査日 2026-10-21。実HTMLで確認した事実のみ。推測は「(推測)」と明記。個人名は「選手名」等に置換済み。
生HTML: `$S/raw/D/`(約75ファイル、`_count.txt` が取得リクエスト行の記録)。S=/tmp/claude-1000/-home-crackerky/a6257ad0-3ffa-4ea0-952c-bc3db229737b/scratchpad/survey

---------------------------------------------------------------------
## 0. 共通スケルトン (3ページ共通。大会トップ等と同一)

URL: `https://score.clubjr.com/pscore/{scv_team|scv_match|scv_player}/open/{ID}?lang=jp|en`
- lang省略時は jp と同一(team 17322 で lang なし版と lang=jp 版がバイト一致)。
- 存在しないID(例 match/45400, team/99999999, player/99999999)は HTTP 500 + CodeIgniter風エラーページ
  `<title>Error</title><h1>An Error Was Encountered</h1><p>ENTRY_PLAYER_ID [99999999]での操作に失敗しました。<br />情報が既に削除されているか、システムエラーの可能性があります。<br /></p>`
  (チームは ENTRY_TEAM_ID、試合は未確認だが同形式)。
- DOCTYPE: XHTML 1.0 Transitional。`<html xmlns=... xml:lang="ja" lang="ja">`(lang=en でも ja のまま)。
- `<title>` 形式 (全て先頭 `[CJS] ` + 大会名):
  - team:   `[CJS] {大会フル名} -{チームフル名}-`
  - match:  `[CJS] {大会フル名} -[{左チームフル名} VS {右チームフル名}]-`
  - player: `[CJS] {大会フル名} -{選手名}-`
- head: normalize.css / scv_def/score_fonts.css / score_frame.css / score_app.css (全て `?20261021` 付、日付クエリ)、
  `js/footerFixed-modified-for-scv.js`, jquery-1.7.2.min.js, jquery.dropdown.js, scv_def/score_navi.css, js/common.js。
  インライン: `body {background: white url(.../img/ScoreView/def_bgimg.png) repeat top;}`、
  `var MyController = "https://score.clubjr.com/pscore/scv_team"` (match/player も同様に自コントローラ名)、
  `$(function(){ $('#score_navi').dropdown(); });`
- body 構造: `body onload=''` > `#top` > `#wrapper` > `#header`(中に `#outer1 > #header`(ロゴ表+SNSアイコン+`#header_bar`+`#navigation ul li#navi01-04`) と **`ul#score_navi`**) → `#sub`(左サイドバナー枠 table、画像 09_menu_outline_*.jpg) → `#contents > #main > #score_parts > #score_topmenu + (<!-- <hr /> -->) + #score_view` → `#footer_ue` + `#footer > .copyright`「Copyright(c) 2026 CLUB Jr. Inc. All Rights Reserved.」。
- **ul#score_navi は team / match / player 全てに出る**(確認した全ページ)。内容は所属大会で決まる条件分岐(他担当の仕様と同じ):
  - 常に: `大会トップ`(scv_top/open/{大会ID})、`チーム一覧`(scv_team_list/open/{大会ID})
  - `スケジュール`(p.ar) > `全ディビジョン`(scv_search_tn) + ディビジョン別(scv_search_div)
  - `ディビジョン成績`(scv_division)、`グループ成績　`(全角空白付、グループ別 scv_group)、`ギャラリー`(scv_tgallery、li に style="width:172px")
  - 大会により一部だけ出る。実例: 500(生田4on4)=スケジュールのみ / 517,522?,526=スケジュールのみ(team ページで確認:17028,17302,17303) / 525=全部 / 536(ディスクゴルフ団体)=大会トップ・チーム一覧・グループ成績のみ / 537(ディスクゴルフ個人)=大会トップ・「エントリー一覧」(文言が変わる)・「成績」 / 523(フリースタイル)=大会トップ・チーム一覧・グループ成績。
  - en: `Tournament Top` / `Team List` / `Schedule` / `All Division` / ディビジョン名も英語(例 `Beach Mixed`)。
- これら3ページには **パンくず/戻りリンクは無い**。戻り先は score_navi のみ(match ページだけ `#score_topmenu` 内に更新ボタンがある。後述)。
- チーム/選手/試合のリンクは全て `<a href="#" class="..." onclick="document.location = 'https://score.clubjr.com/pscore/scv_xxx/open/ID?lang=jp'; return false;">` 形式(href は `#`、絶対URL、lang を引き継ぐ)。

---------------------------------------------------------------------
## 1. チームページ scv_team/open/{チームID}

実例 (jp): 17322(525 とびたーず), 17388(525), 17498(525), 17031(500 4on4), 16987(508), 17028(517), 17302(526), 17307(522), 17303(524), 17041(500), 17301(526), 17461(527 未開催), 4990(2017), 11615(2021), 17459/17463(536 ディスクゴルフ団体)。

### 1.1 `#score_view` 内の構成 (上から。ultimate 系)
1. `<span class='SCV_MIDA_M'>{チームフル名}<br></span>`(見出し。**フル名**。全角スペース等そのまま)
2. (任意) `<div class='SCV_TEAM_COMM'>{チーム紹介コメント}</div>` — 見出しの直後。チーム自己紹介文(1行テキスト)。
   存在: 17307,17028,17303,17498,17461,17459,17463 / 無し: 17322,17302,17388,17031,17041,16987,17301 → **データがあるときだけ出力**。
   CSS: `.SCV_TEAM_COMM{font-size:100%;color:#483d8b;line-height:140%;margin:5px 30px 10px 30px;padding:6px 10px;border:2px solid lightgray}`
3. 成績表 `div.TBL > table`(幅指定は各 td の width%):
   | 列(jp) | 列(en) |
   |---|---|
   | 試合数 勝ち数 分け数 負け数 得点 失点 得失差 勝率 順位 | Games Wins Draw Losses Goals for Goals against Goals diff Win % Rank |
   - ヘッダ行 `<tr class=''>` の td は `class='MD1 CM' width='9.444%'`(9列)。
   - データ1行 `td.CM`。実例: `4 3 0 1 17␣␣ 21␣␣ -4␣␣ 75% 17`(得点/失点/得失差のセルは値の後ろに空白2つ)。勝率は `75%` `16.67%` `33.33%` `50%`(小数は必要時のみ、末尾ゼロ無し)。
   - 順位は**ディビジョン内の通算順位**(例: 32チーム中17位)。未試合のチーム(17461)は `0 0 0 0 0 0 0 --%` で順位セルは空。
   - 実例: 525/17388 `5 3 0 2 27 28 -1 60% 4`、2021/11615 `6 3 1 2 46 33 13 50% 4`(分けあり)。
4. `<hr />`
5. `<span class='SCV_MIDA_M'>&lt; チーム対戦一覧 &gt;</span><br />`(en: `&lt;GameList for the Team&gt;`) + `div.TBL > table`:
   - 列: CD / 日付 / 時間 / コート / チーム名 / - / VS / - / チーム名 / 状況 (width 10,7,7,7,20,4,3,4,20,18 %)。en: CD Date Time Field Team - VS - Team Status。
   - 行 td class は状態で決まる: **`MC_END`**(終了、青文字 #0000ff / 背景 #F8F8FF)、**`MC_MAE`**(未試合、黒/白)、`MC_NOW`(試合中、赤 #ff0000 / 背景 #fff0f5。CSSに定義あり。調査時点で該当試合なし=実HTML未確認)。行内の全 td が同じクラス。CD列は `LM`、チーム名左は `RM`、右は `LM`、他 `CM`。
   - 行順: **時系列昇順**(日付→時間)。チームは左右どちらにも現れる(17028 では対象チームが右側)。= 試合の左右順そのまま。
   - チーム名セル: `<span class='teamLink'><a href="#" class="" onclick="...scv_team/open/{ID}...">{**短縮名**}</a></span>`
   - 得点セル(5,8列目): 終了=整数。未試合=空セル。中央セルは全角 `Ｘ`(U+FF38)。
   - 状況セル: 終了=`<a href="#" class="Select_Link" onclick="...scv_match/open/{試合ID}?lang=jp...">[対戦結果]</a>`(en `[Result]`)、未試合=テキスト `未試合`(en `Waiting`)でリンクなし(= **未試合には試合IDへのリンクが出ない**)。
   - 日付は `MM/DD`、時間は `HH:MM␣`(末尾半角空白)。
   - 試合が 1 つもないチーム(237 の中止大会の 22 チーム)では、この節(見出し、表、後ろの `<hr />`)が丸ごと出ない。
6. (任意・旧大会) `SOTGｽｺｱ一覧` 表 (2017 DISCRAFT=4990 で確認。2021/2026 のチームには無い → データ有無か大会設定で出し分け):
   - 見出し table: `td.LM(26%) > span.SCV_MIDA_M「&lt; SOTGｽｺｱ一覧 &gt;」` + `td.LM(74%) > span.SCV_SOTG_COMM`(※１･･･ルールの理解　※２･･･ファール及び身体接触　※３･･･フェアプレイ　<br />※４･･･ポジティブな姿勢およびセルフマネジメント　※５･･･相手チームのスピリットとの比較)。`.SCV_SOTG_COMM{font-size:93%;color:#ff4500;line-height:120%}`
   - 表: CD / 相手チーム / ※１ ※２ ※３ ※４ ※５ / 合計 (td.MC_END)。値セルは4990では全て空。最終行 `colspan=2 「&lt; SOTG 平均 &gt;」`(td.MD1) + 空6セル。
   - 代表 16 大会では 224(2020 Makuhari 4on4)の 5 チームにだけある。行はチーム対戦一覧と同じ試合・同じ順(未試合を含む状態クラス)、相手チームはリンクなしの短縮名。表の後に `<hr />`、その後に選手一覧。本サイトはディビジョンの設定(divisions.show_sotg)で出す。
   - en: `&lt;SOTG ScoreList&gt;`, 凡例 `* 1 ･･･Rules Knowledge & Use  * 2 ･･･Fouls & Body Contact  * 3 ･･･Fair-Mindedness  * 4 ･･･Positive Attitude & Self-Cont...`、列 `CD / VS Team / * 1..* 5 / Total`、平均 `&lt; SOTG Avg. &gt;`。
7. `<hr />` + `<span class='SCV_MIDA_M'>&lt; 選手一覧 &gt;</span>`(en `&lt;Players List&gt;`) + `table style='width: 100%'`:
   - 列: `#`(5%) / 選手(35%) / 試合数 / アシスト / ゴール / トータル / ブロック (各10%) / `--`(10%、中身は常に空)。
     en: `# Player Games Assist Goal Tot. block --`(**block は小文字**)。
   - 行: `td.CM`(背番号) / `td.LM > a.playerLinkM|playerLinkF`(選手名リンク → scv_player) / 数値は `td.CM`。
     トータルは値の後ろに空白2つ(`9  `)。ブロック0は**空セル**(`  `)、アシスト/ゴールの0は `0`。
   - **試合数は全選手でチームの試合数**(成績表の試合数と同じ値)。出場記録の数ではない(代表 16 大会のチームページ 7,060 行すべてで一致。試合詳細のない 2017 年の大会でも全員 7 など)。
   - **欠場**: 行の全 td に `ABS` が付く(`class='CM ABS'`、`.TBL TD.ABS{#000 on #c0c0c0}`)。該当は 15851 の 3 名だけで、どの試合の出場選手表にも出ていない選手と一致した(同じチームの他の選手は出場記録あり)。出場記録が 1 件もないチーム(試合詳細のない大会)では誰にも付かない。
   - クラス `playerLinkM`=男(色 #000080)/ `playerLinkF`=女(色 #cf143c)(性別データのある大会。2017年の試合ページでは `playerLink` のみ=性別なし)。
   - **並び順**: トータル降順 → ゴール降順 → 背番号昇順(17031 の同点 3名が背番号順、17322 で確認)。0点のみの場合は背番号昇順(17461: #0,#3,#6,#9,#51,#90)。
   - 背番号は文字列として出る(`0`, `00` もあり得る — 選手ページで `#00` を確認)。
8. (任意) `<hr />` + `<span class='SCV_MIDA_M'>&lt; スタッフ一覧 &gt;</span>`(en `&lt;Staffs List&gt;`) + `table style='width: 70%'`: 列 `役割`(30%)/`スタッフ`(70%)(en `Task`/`Staff`)。行: `td.CM`(役割。例 `マネージャ`/en `Manager`) + `td.LM > a.playerLinkM|F`(→ **scv_player/open/{ID}**。スタッフも選手IDと同じID空間)。スタッフ無しの大会(500 の 17031 等)ではセクションごと出ない。並びは役割の順 `監督` → `コーチ` → `トレーナー` → `マネージャ`(五十音順ではない)、同じ役割の中は ID 順(代表 16 大会のスタッフのいるチームページ 101 件すべて)。
9. `<hr />` + `<br /><br /><br />` で終了。

### 1.2 ディスクゴルフ団体 (536) のチームページ — 別レイアウト
実例 17459, 17463。`#score_view` = `SCV_MIDA_M`(チームフル名) → (任意) `SCV_TEAM_COMM` → `SCV_MIDA_M「< 選手一覧 >」` + `table style='width: 70%'` 列 `#`(10%) / `選手`(90%) のみ(成績表・対戦一覧なし)。選手リンクは scv_player。
- 536 の `scv_search_tn` は 0 試合(空の表)。537(個人戦)と 523(フリースタイル)には scv_team/scv_player へのリンク自体が存在しない(537 の `scv_team_list` は選手名が**リンク無しテキスト**の `span.teamimg_rnk`、`... -- 6 Players -- ...`。523 は `0 Teams`)。

### 1.3 チーム名: 短縮名とフル名 (重要)
- 本文テーブルのリンク文字(チーム対戦一覧・試合ページのスコアボード・選手ページの対戦チーム列)は**短縮名**、
  `<title>`・チームページ見出し・ディビジョン日程のチーム絞り込み `<select>`・試合ページ title は**フル名**。
- 短縮は機械的な文字数切りではない(= 別フィールド。推測だが、根拠: 後述の反例)。525 ビーチミックス(949)での全差分(フル → 短縮):
  | ID | フル名 | 短縮 |
  |---|---|---|
  | 17474 | インサイド・スロー | インサイド |
  | 17369 | シロコロホルモンズ | シロコロホルモン |
  | 17498 | バトルロイヤルホスト | バトルロイヤル |
  | 17481 | 一橋大学フライングディスクサークルUFO | 一橋大学 |
  | 17489 | メッターによるお料理向上委員会 | お料理向上委員会(**先頭が落ちる=前方一致切りではない**) |
  | 17493 | わんこそばすたーず | わんこそば |
  32チーム中26は同一。他: 2021(28485) `グランツG　グリーンキウイチーム` → `グリーンキウイ`、526 `HAYAMICHI ブルー`/`ブラック` → `HAYAMICHI-BL`/`-BK`(**短縮の方が英字ハイフン表記で、フルとは別文字列**)、524 `おもちゃのチャチャチャ` → `おもちゃの`、`おかあさんといっしょ` → `おかあさん`、英語 `Something's missing` → `Something&#039;s miss`(HTMLエスケープ)、2017 `横浜国立大学COUGARS` → `横浜国立大学`、選手ページ中央の `[チーム短縮名]` は更に短い場合あり(536: フル `チームWTFRs` → `[WTFR]`)。
- 選手ページ見出し `{短縮} - #背番号 : 名前` の短縮は、チーム対戦一覧と同じ(例 `雨ニモマケズ`=フル同一)。
- 実装メモ: team レコードに `name_full`, `name_short`(＋en用 `name_full_en`, `name_short_en`)を持たせる。

---------------------------------------------------------------------
## 2. 試合ページ scv_match/open/{試合ID}

実例 (jp): 45277(525 A1), 45335(525 G3), 45276(524 H6), 45211(524 A1), 45130(522 A4), 45043,45051(526), 44319(517 H4 50分), 44338,44347(500 4on4、44347は 3-3 引き分け), 28485(2021), 14693(2017 DISCRAFT)。

試合IDは大会をまたいで連番(14693=2017, 28485=2021, 44338=2026春, 45277=2026秋)。**未試合には試合ページへ辿るリンクが無い**(日程に ID が出ない)ため、未開始の試合ページの実HTMLは取得できなかった。

### 2.1 `#score_topmenu`(match のみ中身あり)
```
<div id="score_topmenu"><table style='width:100%; '><tr>
  <!-- <td width="%" ><div id="breadcrumb">&nbsp;&nbsp;<a href=".../scv_top/open/525?lang=jp">2026 Atami-B</a>&nbsp;&nbsp;<a href=".../scv_division/open/949?lang=jp">Div:ビーチミックス</a>&nbsp;&nbsp;<a href=".../scv_group/open/6623?lang=jp">[Grp:Tounament A]</a></div></td> -->
  <td class="RM" width="140px" style="height:33px">
    <a id='Refresh_OK' style='display: none;' href="#" onclick='window.location.replace(".../scv_match/open/45277?lang=jp&amp;refpos=1"); return false;'><img src=".../img/icons/refresh.png" alt="Reflesh_OK" /></a>
    <a id='Refresh_NG' style='display: none;' href="#" onclick='alert("５秒お待ちください"); return false;'><img src=".../img/icons/refresh_ng.png" alt="RefleshWait" /></a>&nbsp;
  </td></tr></table></div>
```
- **パンくずは HTML コメントアウトされており表示されない**(コメント内に大会短縮名 `2026 Atami-B`・ディビジョンID・グループID が入っている=戻り先の対応関係が分かる唯一の箇所)。
- 更新ボタン: head のインラインJS `$("#Refresh_NG").show(500); setTimeout(function(){ $("#Refresh_OK").show(800); $("#Refresh_NG").hide(800); }, 5000);` で、最初の5秒は「待て」アイコン(NG、クリックで alert「５秒お待ちください」)、5秒後に「更新」アイコン(OK)に切替。OK クリックは `?lang=jp&refpos=1` で同ページを `location.replace`。`refpos=1` のときは head JS に `location.href = "#score_navi";` が追加されるだけ(他は同一、差分で確認)。自動リロードは無い(meta refresh / setInterval 無し)。
- team / player ページの `#score_topmenu` は空。

### 2.2 スコアボード (`#score_view` 先頭、`div.TBL` 内の table ×3)
1. 試合情報行 table: `td.LM.TG_MD1(16%)`=試合CD(`A1`) / `td.LM.TG_MD1(48%)`=グループ名(`Tounament A`, `Qualifying pool A`, `9th placement tournament`, `13th-17th placement tournament`。リンクなし。ディビジョン名は出ない。2017の旧データではグループ名セルが空で `td` 2個構成) / `td.CM.TG_MD1(36%)`=`2025/10/18&nbsp;12:20&nbsp; @Field:&nbsp;1`(`YYYY/MM/DD&nbsp;HH:MM&nbsp; @Field:&nbsp;{コート}`、**年は4桁表示。525は大会名が2026でも日付は2025/10/18**)。背景 #FF6600 白文字太字。
2. スコア table `tr style='height:50px'`: `td.CM.TG_VSTITLE(41%)` > `span.teamimg_big_match > a`(左チーム**短縮名**→scv_team、リンク色白) / `td.CM.TG_POINT(7%)` 左得点 / `td.CM.TG_VSTITLE(6%)`=`VS` / `td.CM.TG_POINT(7%)` 右得点 / `td.CM.TG_VSTITLE(41%)` 右チーム。VSTITLE=153.9% 太字白/橙背景、POINT=153.9% 太字青(#0000ff)/白背景。**最終スコアのみで、前半/後半/ハーフタイムの区切りは無い。**
3. フッタ行 table: `td.CM.TG_MD1(16%)`=`-対戦結果-`(en `--Result--`) / `td.LM.TG_MD1(68%)`=空 / `td.CM.TG_MD1(16%)`=`HH:MM:SS JST`(**リクエスト時刻の現在時刻**。試合時刻ではない。取得のたびに変わる → 実装は固定/現在時刻表示)。

### 2.3 得点経過 `<hr/>` + `div.TBL.TG_EventList > table#events`
`.TG_EventList{min-height:400px;border-top/bottom:1px solid #d4d0c8}`
- ヘッダ(8列、td.MD1.CM): `経過`(6%) `アシスト選手`(17%) `ゴール選手`(17%) `点`(3%) `イベント`(14%) `点`(3%) `ゴール選手`(17%) `アシスト選手`(17%)
  en: `Time` `Assist Player` `Goal Player` `P.` `Event` `P.` `Goal Player` `Assist Player`。
- **並び順は新しい順(降順)**。最下行が `00:00 試合開始ｽﾛｰｵﾌ`、最上行が `試合終了`。同時刻(`14:00`が2行)は点数の大きい方が上。
- 行の種類(td class は `TG_EVMOJI` + 位置 `CM`/`LM`):
  - **得点行**(class `TG_EVMOJI CM `): 経過=`MM:SS`(例 `17:00`、**分単位で秒は :00 のものが多い**=入力が分刻み) / 左得点した場合: 左から `アシスト選手`, `ゴール選手`, `点`(=その時点の**左チーム累計**), イベント=`得点`(en `Goal`)、右側3セルは空。右チーム得点時はミラー: 左3セル空、イベント=`得点`、`点`(右チーム累計)、`ゴール選手`、`アシスト選手`。つまり左=「アシスト→ゴール→点」、右=「点→ゴール→アシスト」(中心に向かって点)。
  - 選手セルは `<a class="playerLinkM|F" ...scv_player/open/ID>#{背番号} {選手名}</a>`(背番号と名前は半角空白区切り)。確認した5試合(45277,45043,14693,28485,44319)ではアシスト/ゴール選手が空の得点行は無かった(得点しなかった側のセルが空なだけ)。選手未入力時に空になるかは未確認。
  - **イベント行**(td に追加クラス `EVT`、青文字 #0000ff / 背景 #f5f5f5): `試合開始ｽﾛｰｵﾌ`(en `1st Pull`、半角カナ「ｽﾛｰｵﾌ」。点セルに `★`(en `*`)= **プル(スローオフ)する側のチーム**の点セル側に出る: 左がプルなら左「点」列、右なら右「点」列) / `タイムアウト`(en `Timeout`、★ は呼んだチームの点列) / `試合終了`(en `End`)。
  - 経過時間はイベント種別に関わらず `MM:SS`。`試合終了` の経過は試合全体の経過(20分制=`20:00`、50分=`50:00`、4on4=`05:00`〜`06:00`、2017=`41:00`)。
  - **`試合終了` 行が無い試合もある**(45051, 45130 は最上行が最終得点で終了行なし=終了入力漏れ。日程上は MC_END)。
- 検証した全試合の `イベント` 値集合(jp): `得点`, `タイムアウト`, `試合開始ｽﾛｰｵﾌ`, `試合終了` のみ。**ハーフタイム/前半後半/ピリオド行は存在しない。SOTG・MVP・コメント欄も match ページには無い**(語句 SOTG/MVP/コメント/ハーフ/没収/不戦 はいずれも未出現)。
- 2017/2021 の旧データ: 選手リンクの class が `playerLink`(性別なし)になる試合あり(14693)。

### 2.4 出場選手 `<hr/>` + 2カラム table (`table style='width: 100%'`、td width 50%×2)
各 td に `div.TBL.TG_PlayersList_Ult > table#lplayers`(左チーム)/ `table#rplayers`(右チーム)。`.TG_PlayersList_Ult{overflow-y:scroll;height:150px}`(**固定高150pxで縦スクロール**)。
- 列: `#`(10%) / `Player`(41%) / `Assist`(13%) / `Goal`(13%) / `Block`(13%)。**列見出しは jp でも英語のまま**(`#`,`Player`,`Assist`,`Goal`,`Block`。en も同一)。
- 行: `td.CM`(背番号) / `td.LM > a.playerLinkM|F`(名前のみ。背番号なし) / アシスト数 / ゴール数 / ブロック数(0 は**空セル**、アシスト・ゴールの0は `0`)。
- 並び: **(アシスト+ゴール)降順 → ゴール降順 → 背番号昇順**(5試合10表すべてでこの規則に一致を機械検証。チームページの選手一覧と同じ規則)。
- 試合に出た全員(得点ゼロ含む)を列挙(左11名、右17名の例)。

### 2.5 統計 `<hr/>` + `div.TBL > table style='width: 80%'`
- ヘッダ: `td.MD1.CM(40%)`=`統計情報`(en `Game Statistics`) / `(30%)`=左チーム短縮名 / `(30%)`=右チーム短縮名。
- 行(td.LM、左→右): `得点数`(Goals) → [`オフェンス時間累計`…`失点までの平均時間` の4行は **HTMLコメント `<!-- ... -->` で非表示**(jp/en とも同じ。en 文言は `Time on offence` `Time on defence` `Time on offence/goal` `Time on deffence/goal`)] → `オフェンスからの得点数`(en `Goals from starting on offence`) `5&nbsp;&nbsp;&nbsp;[71.43%]` / `ディフェンスからの得点数`(`Goals from starting on deffence`、綴りミスのまま) `2&nbsp;&nbsp;&nbsp;[28.57%]` / `ターンオーバーからの得点数`(`Goals from turnovers`) `2`(%なし) / `タイムアウト数`(`Time outs`) `0`。
- %の分母が 0 のときは `0&nbsp;&nbsp;&nbsp;[--%]`(45051)。%は小数2桁までで末尾0省略(`50%`, `83.33%`, `100%`, `8.33%`, `66.67%`)。

### 2.6 状態別の差 (実HTMLで確認できた範囲)
| 状態 | 確認 | 内容 |
|---|---|---|
| 終了 | ○(約12試合) | 上記の通り。日程の行は `MC_END` + `[対戦結果]`リンク。 |
| 引き分け | ○ 44347 (3-3) | 特別表示なし。チームページの「分け数」に加算(UCA 例 3勝1分)。 |
| 終了行欠落 | ○ 45051, 45130 | `試合終了` イベントが無いだけで他は同じ。 |
| グループ欄の食い違い | ○ 129 試合(突き合わせ) | 試合情報行のグループ欄は、パンくずのグループの今の名前と違うことがある: 大文字小文字(342 `Championship Tournament` / グループ `Championship tournament`)、別のグループ名(440 `Qualifying Tournament B` / `9th placement tournament`、224 `第2節リーグ` / `第1節リーグ`)、2017 年(151、162)は空。試合を作ったときのグループ名を試合が持つと推測。 |
| 備考 | ○ 15087, 41199, 41279, 41377 | フッター行の中央(68% のセル)に文字が入る(`スクラッチのため`、`NAKEDリップ勝ち`、`フリップ`)。他の試合は空。 |
| Block 列なし | ○ 20 試合(224、342、435、440、451 の一部と 151) | 出場選手表が 4 列 `#`(10%) `Player`(50%) `Assist`(15%) `Goal`(15%)。同じ大会でも試合ごとに違う。Block 列のある試合でも全員 0(空)のことがあり(58 試合)、値からは決まらない。 |
| 統計表なし | ○ 151 の 3 試合 | 出場選手表の後の `<hr/>` で `#score_view` が終わる。 |
| 試合終了の ★ | ○ 44803, 44867 | `試合終了` の行の左の点列に ★。意味は未確認。 |
| 一方的スコア | ○ 45051 (0-12) | 左得点0なので統計の左%は `--%`。 |
| 未試合 | × (ID が見つからない) | 日程/チーム対戦一覧では `MC_MAE`、状況=`未試合`/en `Waiting`、得点セル空、`Ｘ`のみ、**リンクなし**。527(Fuji 4on4 vol.14、11/03予定)で確認。 |
| 試合中 | × (観測不能) | CSS に `.TBL TD.MC_NOW`(赤)と `.TBL TD.TG_NOWPLAY`(赤太字/淡ピンク背景)が定義されているので存在する状態(推測: 日程の行が MC_NOW、状況に別文言+試合ページ有)。調査時点(2026-10-21)で進行中の大会なし。 |
| 没収試合 | × | 該当語句・クラスは全HTML/CSSで未出現。 |
- 未来大会 539, 535, 528 の `scv_search_tn`: 試合ゼロ(表ヘッダのみ、`<select>`も空)。527 は 4人制の未試合行あり(MC_MAE)。536(ディスクゴルフ団体)・523(フリースタイル)も試合ゼロ。→ 試合ページが存在するのは ultimate の実績試合のみ。
- 他スポーツ用CSS(`TG_GUTS_*`, `EVT_GLP/GRP`, `TG_PlayersList_Guts`)が残っている = ガッツ競技用の別レイアウトがコード上にはある(推測。実ページ未確認)。

---------------------------------------------------------------------
## 3. 選手ページ scv_player/open/{選手ID}

実例 (jp): 604138, 605012, 605940(525)、605250(524)、599654(522)、596369(526)、563787(500)、562172(517)、78679(2017)、608227(527 未試合)、608232(527 スタッフ)、620269(536 ディスクゴルフ)。

### 3.1 構成 (`#score_view`)
1. 前後ナビ table `class='WMAX'` `<TR>`(**大文字タグ**)×3セル:
   - 左 `TD width='30%' class='LM'`: `<a class="playerLinkM|F" ...scv_player/open/{前ID}>&lt;=[#{背番号} : {名前}]</a>`
   - 中 `TD width='40%' class='CM'`: `<a class="playerLink" ...scv_team/open/{チームID}>[{チーム短縮名}]</a>`(**チームページへの戻りリンク**)
   - 右 `TD width='30%' class='RM'`: `[#{背番号} : {名前}]=&gt;`(次)
   - 前後の順は**同チーム内のID昇順**(スタッフを含む。例 608231←608232(スタッフ)→608233、604137←604138→604139)で、**末尾↔先頭で循環**(最小ID 608227 の「前」は最大ID 608233)。チームページの並び(成績順)とは別。
2. `<hr />`
3. `<span class='SCV_MIDA_M'>{チーム短縮名} - #{背番号} : {選手名}</span>`(スタッフは `{チーム短縮名} - {役割} : {名前}` 例 `... - マネージャ : ...`。**スタッフページはここで終わり**: 成績表もイベントも無い。前後ナビは出る)。
4. 成績表 `div.TBL > table style='width: 100%'`(ultimate):
   列: 試合数(10%) / アシスト / ゴール / トータル / キャラハン / グレイテスト / ブロック (各15%)。en: `Games Assist Goal Tot. Callahan Greatest Block`。
   - データ行 `td.RM.CM`: `4` / `5 &nbsp;&nbsp;(avg:1.25)` / `0   &nbsp;&nbsp;(avg:0)` / `5  &nbsp;&nbsp;(avg:1.25)` / `0` / `0` / `6`。平均 = 値÷試合数(小数2桁、末尾0省略、`1`,`0.33`,`1.33`,`2.25`,`0.17`,`0.5`)。試合数0なら `(avg:--)`。ブロック0は空セル。キャラハン/グレイテストは 0 表示。
5. (イベントがあるとき) `<hr />` + `<span class='SCV_MIDA_M'>&lt; イベント &gt;</span>`(en `&lt;Game Events&gt;`) + `table`:
   列: CD(10%) / 日付(7%) / 対戦チーム(20%) / 経過(7%) / イベント(16%) / ゴール選手(20%) / アシスト選手(20%)。en: `CD Date VS Team Time Event Goal Player Assist Player`。
   - 行 td class は `TG_EVMOJ`(**EVMOJI ではなく EVMOJ**。CSS未定義の別クラス)+ `LM`/`CM`。対戦チームセル: `span.teamLink > a`(相手の短縮名→scv_team)、**この td は `</td>` が閉じられていない**(HTMLの欠陥。パーサに任せる)。
   - 本人が**ゴールまたはアシスト**した得点のみ(ブロック等は載らない。ブロック6でも行は5)。選手セルは `#{背番号}&nbsp;{名前}`(`&nbsp;`区切り。試合ページの半角空白とは違う)。
   - 行順: 試合の時系列(古い試合が上)→試合内は経過昇順。実例 604138: A3(04:00)が A6(01:00,03:00,04:00)の後に来る=CD順ではなく試合開始時刻順(A6 は A3 より早い時刻の試合と推測)。
   - イベントが0件(試合数0やアシストのみでない)ときは「イベント」節ごと出ない(608227)。
6. ディスクゴルフ団体 (620269) は成績表の列が別: `試合数 / 得点 / キャッチ / スローミス / トータル / ｸﾘｰﾝｷｬｯﾁ / ﾌｫﾛｰｷｬｯﾁ / ﾌｯﾄﾌｧｰﾙ`(width 10,12.8×7%、半角カナ)。値は全て 0、`(avg:--)`。イベント節なし。
- **写真・顔画像は無い**(`#score_view` に `<img>` なし)。

### 3.2 en 版
- 選手名はローマ字(DBの別フィールド。**日本語名と姓が異なるように見えるケースがある**=入力が別系統。例: jp「(選手名)」の en 表記が別姓)。ローマ字未登録の場合は**名前が空文字**(4990 / 17459 の en ではチーム名・選手名が空で見出し/title が `-`)。
- 例: 選手名 `(Player Name)` 形式(姓 名、姓のみ大文字の場合 `(PLAYER Name)`、全大文字 `(PLAYER NAME)` も混在=入力ゆれ)。チーム英語名は `Tobitazu`, `BRH`, `Hitotsubashi Uni`, `wankosobasuta-zu`(伸ばし棒を `-` 表記)。

---------------------------------------------------------------------
## 4. lang=en 版 文言対応表 (確認分)

| jp | en |
|---|---|
| 大会トップ / チーム一覧 / スケジュール / 全ディビジョン | Tournament Top / Team List / Schedule / All Division |
| 試合数 勝ち数 分け数 負け数 得点 失点 得失差 勝率 順位 | Games Wins Draw Losses Goals for Goals against Goals diff Win % Rank |
| &lt; チーム対戦一覧 &gt; | &lt;GameList for the Team&gt; |
| CD 日付 時間 コート チーム名 状況 | CD Date Time Field Team Status |
| [対戦結果] / 未試合 | [Result] / Waiting |
| &lt; 選手一覧 &gt; (# 選手 試合数 アシスト ゴール トータル ブロック) | &lt;Players List&gt; (# Player Games Assist Goal Tot. block) |
| &lt; スタッフ一覧 &gt; (役割 スタッフ) | &lt;Staffs List&gt; (Task Staff) |
| &lt; SOTGｽｺｱ一覧 &gt; / 相手チーム / 合計 / &lt; SOTG 平均 &gt; | &lt;SOTG ScoreList&gt; / VS Team / Total / &lt; SOTG Avg. &gt; |
| -対戦結果- | --Result-- |
| 経過 アシスト選手 ゴール選手 点 イベント | Time Assist Player Goal Player P. Event |
| 得点 / 試合開始ｽﾛｰｵﾌ / タイムアウト / 試合終了 / ★ | Goal / 1st Pull / Timeout / End / * |
| 統計情報 / 得点数 | Game Statistics / Goals |
| オフェンスからの得点数 / ディフェンスからの得点数 / ターンオーバーからの得点数 / タイムアウト数 | Goals from starting on offence / Goals from starting on deffence / Goals from turnovers / Time outs |
| 試合数 アシスト ゴール トータル キャラハン グレイテスト ブロック (選手) | Games Assist Goal Tot. Callahan Greatest Block |
| &lt; イベント &gt; / 対戦チーム / ゴール選手 | &lt;Game Events&gt; / VS Team / Goal Player |
- 試合ページの出場選手表の見出し(`# Player Assist Goal Block`)は jp も英語。
- 日付・数値書式は同じ。サイトヘッダ(会社案内等の日本語リンクとロゴ)は en でも日本語のまま。`<html lang>` も ja のまま。
- ディビジョン名・グループ名(`Beach Mixed`, `Tounament A`)・大会名は en でもDB登録の英語表記(グループ名はそもそも英語)。

---------------------------------------------------------------------
## 5. 各ページからのリンク (戻り先)
| ページ | 出るリンク |
|---|---|
| team | score_navi / 対戦一覧: 相手チーム(scv_team, 短縮名)・[対戦結果](scv_match) / 選手一覧・スタッフ一覧: scv_player |
| match | score_navi / 左右チーム(scv_team)/ 得点経過・出場選手: scv_player(経過ではラベル `#番号 名前`、出場選手表では名前のみ)。**日程・グループ・ディビジョンへの直接リンクなし**(パンくずはコメントアウト) |
| player | score_navi / 前後の選手(scv_player)/ 所属チーム(scv_team)/ イベント表の対戦チーム(scv_team)。**試合ページへのリンクなし** |
- 大会トップへの戻りは score_navi の「大会トップ」のみ。

---------------------------------------------------------------------
## 6. データモデル推定

- **チームは大会ごとに別ID**(根拠: 同名 `HAYAMICHI` が 517=ID 17028、525=ID 17388、526=ID 17302 と別IDで、各ページの `<title>` の大会名・成績・選手が別。`Re:turn` は 500=17041 / 526=17301、`UNTRACE` は 517=17062 / 526=17308)。チームID は大会を跨いで増加(2017: 4990、2021: 11615、2026: 16959〜17579)。→ team = 「大会エントリー」。同名チームの名寄せIDは存在しない。
- **選手IDも大会(エントリー)ごと**。根拠: 同姓同名・同背番号 `#11` の選手が 500(Mirage, ID 563787) と 525(何かが足りない, ID 605940) で別ID(同一人物かは推測)。選手IDはチームの登録順にほぼ連番(例 17031 → 563784〜563789、17461 → 608227〜608233。ただし追加登録で飛ぶ: 17028 は 562172〜563796 に散在)。スタッフもこの連番に含まれる(608232)。大会間の個人成績の通算ページは無い。
- 試合ID: 大会またぎグローバル連番。ディビジョン/グループ/試合CD(A1,B2…)は各大会内で採番。
- 英語名は別カラム(欠損すると空表示)。チームは 短縮名/フル名 × jp/en の4種(§1.3)。選手は 名前 jp/en、背番号(文字列)、性別(M/F → class)。
- 順位は team レコードにはなく、ディビジョン集計で算出されて表示(team ページの 順位 = ディビジョン通算)。
- 取得規模: 全 約77リクエスト、`User-Agent: Mozilla/5.0 (survey; low-rate)`、間隔 1.3秒、セッション cookie 維持(Set-Cookie は特に必要なし。lang は URL クエリのみ)。
