# C_spec: 成績・順位・グループ(プール/ブラケット) 仕様調査

調査日 2026-10-21。対象 https://score.clubjr.com 。取得は 95 リクエスト(全て 200、間隔 1.2 秒、UA "Mozilla/5.0 (survey; low-rate)")。
生HTML: `$S/raw/C/` (S=/tmp/claude-1000/-home-crackerky/a6257ad0-3ffa-4ea0-952c-bc3db229737b/scratchpad/survey)。解析スクリプト: `$S/scripts/C_*.py`。
表記: 「事実」は実HTMLで確認、「推定」は根拠付きの推測。

---
## 0. 全ページ共通の骨格(本調査対象に関わる部分のみ)

- `<body onload=''>` > `#top > #wrapper > #header`(ロゴ・ul#score_navi) > `#sub`(左メニュー) > `#contents > #main > #score_parts > #score_topmenu(空) > #score_view`。本調査対象の本文はすべて `#score_view` の中。
- CSS: `/css/scv_def/score_app.css`(本文の表・ブラケット)、`score_frame.css`、`score_navi.css`、normalize.css。JS: jquery-1.7.2、common.js、jquery.dropdown.js、footerFixed-modified-for-scv.js。
- リンクは全て `<a href="#" class="" onclick="document.location = 'https://score.clubjr.com/pscore/scv_xxx/open/{id}?lang=jp'; return false;">` 形式(href は常に `#`)。
- 見出しは `<span class='SCV_MIDA_M'>&lt; 見出し &gt;</span>`(JP は `< ` と ` >` の内側に半角スペース、EN は `&lt;Beach Mixed Standings&gt;` でスペース無し)。CSS: font-size 131%, bold, line-height 250%, color #fff, background #FF6600, padding 4px 10px, text-align center, border-radius 4px。
- `<title>`: group ページは `[CJS] {大会名} -{グループ名}-`(例 `[CJS] 2026 Atami Beach Ultimate Beach Line Tournament -Tounament A-`)。EN では大会名も英語名(`[CJS] 2025 Hatsunage 1st -Qualifying pool A-`)。
- 共通表スタイル: `.TBL {margin:0 10px}`, `.TBL table {border-top/left:1px solid #d4d0c8; border-collapse:collapse; background:#fff; empty-cells:show}`, `.TBL td {border-right/bottom:1px solid #d4d0c8; padding:0.1em 0.3em}`, `.TBL TD.MD1 {bold; color:#fff; background:#ffa000}`, `.LM/.RM/.CM` = text-align left/right/center。

---
## 1. ディビジョン成績 `/pscore/scv_division/open/{ediv}?lang=jp|en`

### 1.1 ページ構成(事実)
`#score_view` 直下に、データがある節のみ上から順に:

1. `<span class='SCV_MIDA_M'>&lt; {ディビジョン名} 順位表 &gt;</span>` + `<div class="TBL"><table>…</table></div>` + `<hr />`
2. `< スコアボードリーダー >` + `div.TBL > table` + `<hr />`
3. `< スコアボードリーダー　（ブロック） >`(全角スペース入り) + `div.TBL > table` + `<hr />`

- **グループ別表・試合一覧は置かれない**(949: テーブルは 3 つのみ、行数 32/30/10)。グループ・試合へは score_navi / 左メニューから行く。
- 古い大会(ediv 430 NNFDA2019 など)は順位表のみ(スコアボードリーダー節が無い=データがある節だけ出力)。
- EN 見出し: `<Beach Mixed Standings>` / `<Scoreboard Leaders>` / `<Scoreboard Leaders *Block*>`。JP は「ディビジョン名 + 半角スペース + 順位表」。

### 1.2 順位表 table(事実)
ヘッダ行 `<tr class=''>`、全セル `<td class='MD1 CM' width= '…' >`(`width=` の後ろに空白が入る原文ママ)。

| # | JP | EN | width |
|---|---|---|---|
| 1 | チーム名 | Teams | 24% |
| 2 | 試合数 | Games | 8.444% |
| 3 | 勝ち数 | Wins | 8.444% |
| 4 | 分け数 | Draw | 8.444% |
| 5 | 負け数 | Losses | 8.444% |
| 6 | 得点 | Goals<br />for | 8.444% |
| 7 | 失点 | Goals<br />against | 8.444% |
| 8 | 得失差 | Goals<br />diff | 8.444% |
| 9 | 勝率 | Win % | 8.444% |
| 10 | 順位 | Rank | 8.444% |

データ行 `<tr>`:
```
<td class='MD1' ><span class='teamimg_rnk'><a href="#" class="" onclick="document.location = 'https://score.clubjr.com/pscore/scv_team/open/17396?lang=jp'; return false;">マンマユート</a></span> </td>
<td class='CM HL1' >5</td>      試合数
<td class='CM' >5</td>          勝
<td class='CM' >0</td>          分
<td class='CM' >0</td>          負
<td class='CM' >38 </td>        得点 (末尾に半角スペース)
<td class='CM' >19 </td>        失点 (末尾スペース)
<td class='CM HL1' >19 </td>    得失差 (末尾スペース、負は -1 )
<td class='CM HL1' >100%</td>   勝率
<td class='CM HL2' >1 </td>     順位 (末尾スペース)
```
- チーム名セルは `class='MD1'`(オレンジ背景・白太字、`.teamimg_rnk a {color:#fff !important}`)。チームロゴ画像は無い(`<img` 0 件)。名前は HTML エスケープ済み(`M&amp;M&#039;s`)。
- `.HL1` は CSS 上は空定義、`.HL2` = color #ff0000 / bg #fff0f5(順位列のみピンク地赤字)。
- チーム → `scv_team/open/{teamId}`。

### 1.3 勝率の表記(事実)
- `勝 / 試合数` を百分率、**小数 2 桁で丸め末尾 0 を削る**: `100%`, `83.33%`, `66.67%`, `80%`, `75%`, `16.67%`, `0%`。
- **引き分けは勝ちに数えない**: ediv 844 SeeKers 4勝1分1敗(6 試合) → 66.67%(=4/6)。IKU! 4勝1分0敗(5 試合) → 80%。ediv 815 とっとこハム太郎 1勝1分4敗 → 16.67%。
- 0 試合: `--%`(ediv 954 GAIA 2026 未開催: 全行 `0 | 0 | 0 | 0 | 0 | 0 | 0 | --% | (順位空)`)。

### 1.4 並び順と順位の決まり方(事実+推定)
事実:
- 行は **「順位」列の昇順**(949, 815, 838, 844, 847, 848, 430, 406 全て)。同順位は隣接。
- 949 の順位列: `1,2,3,4,5,5,7,7,9,9,11,11,13,13,15,15,17,17,19,19,21,21,23,23,25,25,27,27,29,29,31,31`。**同順位は 2 チームずつ、次は 2 つ飛ぶ(標準競技順位)**。
- 勝率・得失差・試合数では並んでいない。例 949: 4 試合 3-1(75%) の ジャンジャン=5 位 / MFD=9 位 / Muscle Brain(得失差+19)=17 位。5 試合のチームが 1〜4 位、4 試合のチームが 5 位以下。844 でも IKU!(4-1-0, 得失差+45) が 8 位で UNTRACE(3-0-2, +5) の 6 位より下。

**順位は「順位決定トーナメント(placement)の最終順位」そのもの(強い推定)**。根拠(ediv 949):
- 6627 `1st placement tournament`: Placement 列 1st=マンマユート, 2nd=バトルロイヤル, 3rd=28卒, 4th=HAYAMICHI → 表の 1〜4 位と一致。
- 6628 `5th-9th placement tournament`: `5th(tied)`=SPARKS, ジャンジャン / `7th(tied)`=何かが足りない, すぱーくす / `9th(tied)`=雨ニモマケズ, MFD / `11th(tied)`=EPC, インサイド → 表の 5,5,7,7,9,9,11,11 と一致。
- 6630 `21st-29th placement tournament`: 21th(tied)=SHIMI-ZUSHI, お料理向上委員会 … 31st(tied)=シロコロホルモン, エイヒレ → 表の 21〜31 と一致。
- 「tied」表記がそのまま同順位として出る。placement 未消化の間は順位セルが空になる(954 は全て空)と推定。部分消化の例は未観測。
- **同順位の中の並び**(突き合わせで確認): 勝ち数の降順、次に得失差の降順(代表 16 大会の 11 ページ、同順位 86 組すべて)。勝ち数・得失差とも同じ組は 1 組(338 の 4584 と 4595)だけで、ID 順と失点の少ない順のどちらとも合う。
- 得点ランキングとブロックランキングの「試合数」は選手のチームの試合数(得点 288 行、ブロック 80 行すべて)。得点ランキングで合計とゴールが同じ選手の並びは、背番号・選手 ID・チームの順位のどれとも合わない(参照サイトの DB の返す順と推測。再現しない)。

勝率列は表示用の集計値で、順位計算には使われない。

### 1.5 スコアボードリーダー表(事実)
- 得点表 header: `チーム名(25%) | #(' 5%') | 選手(30%) | 試合数(10%) | アシスト(10%) | ゴール(10%) | トータル(10%)`。EN: `Team | # | Player | Games | Assist | Goal | Tot.`
- 行: `<td class='LM' ><span class='teamimg'>28卒</span> </td>`(チーム名は**リンク無し**) / `<td class='CM' >87</td>` / `<td class='LM' ><a … class="playerLinkM" onclick="…scv_player/open/606311?lang=jp…">(選手名)</a></td>`(女子選手は `playerLinkF`。`.playerLinkM` #000080, `.playerLinkF` #cf143c) / `<td class='CM HL1' >5</td>` 試合数 / `CM` アシスト / `CM` ゴール / `<td class='CM HL1' >17 </td>` トータル。
- 949: 30 行、トータル降順(同点順は未検証)。
- ブロック表 header: `チーム名 | # | 選手 | 試合数 | ブロック | --`(最終列の見出しは文字 `--`、width 20%、行セルは空 `<td class='CM HL1' ></td>`)。949: 10 行。EN: `Block` / `--`。
- 選手名は EN では英字ローマ字(`(Player Name)`)。

---
## 2. グループ成績 `/pscore/scv_group/open/{gid}?lang=jp|en`

2 種類。**種別は HTML で判別可**: プール型=`- MATRIX -` を含み GeneratePtn 無し / トーナメント型=`<!-- GeneratePtn:T8A -->` と `div.TN_TMPL`。
共通の末尾: `<hr>` + 空の `<div class="TBL"> </div>`。

### 2.1 プール(総当たりリーグ)型(事実)
```
<span class='SCV_MIDA_M'>&lt; ４人制部門 - Qualifying pool A &gt;</span><br />
<div class="TBL" ><table> MATRIX </table></div>
<hr>
<span class='SCV_MIDA_M'>&lt; リーグ結果 &gt;</span><br />
<div class="TBL"><table> RESULT </table></div>
<hr>
<span class='SCV_MIDA_M'>&lt; 結果一覧 &gt;</span><br />
<div class="TBL"><table> 試合一覧 </table></div>
<hr>
<div class="TBL"> </div>
```
見出し: `< {ディビジョン名} - {グループ名} >`。EN: `<4 on 4 - Qualifying pool A>` / `<Reague Result>`(原文 typo) / `<GameList for the Group>`。

#### (a) 対戦マトリクス(星取表): あり
- 1 行目 `<tr class=''>`: `<td class='RM_MD1 CM' width='22%' >- MATRIX -</td>` + 各チーム列見出し `<td class='RM_MD1 CM' width='{78/N}%' >`(3 チーム=26%、4 チーム=19.5%、5 チーム=15.6%)。中身 `<span class='teamimg_rnk'><a … scv_team/open/{id}>名前</a></span>`。**5 チーム以上**(2182、2188、3272 の 5 チーム、6631 の 6 チーム)では見出しはリンクのない素の文字で、名前を表示幅 4(全角 2 文字、半角 4 文字)で切り詰める(`慶應` `VORT` `くら`)。
- 未試合の上三角のセルは試合の CD を素の文字で出す(`<td class='RM_ARI  CM'>A1</td>`、527 の 6631)。
- 以降 N 行。第 1 セル `<td class='RM_MD1'>`(チーム名リンク)+ N セル。セルの種別:
  - 対角: `<td class='RM_AIDA CM'>` 空(背景 #ffa000)
  - 下三角: `<td class='RM_NASI CM'>` 空(背景 #fff3b1)
  - 上三角で試合あり: `<td class='RM_ARI CM'><a href="#" class="RM_ARI_Link" onclick="…scv_match/open/40611?lang=jp…">9-3</a> </td>`(背景 #f5f5f5 太字、リンク #0000ff)
  - 試合中: `RM_NOW` / `RM_NOW_Link`(#ff0000 / #fff0f5)。今回データには出現せず CSS 定義のみ。
- **セル表記は `{左}-{右}` 数値スコア。○× ではない**。203 セルで検証: セル文字列は試合一覧の「左チーム得点-右チーム得点」と完全一致(不一致 0)。行チームが左なのは 193/203、10 件は行チームが右 = 試合記録上の左右順で表示し、行視点へ並べ替えない。下三角に鏡像は出ない。引き分けも同形式(`5-5`)。勝敗の色分け無し。マトリクス行数のうち上三角のリンク無しセル(8 件)もあった(未消化か ID 不一致、未精査)。

#### (b) リーグ結果(プール順位表)
header: `<td class='MD1 CM' width= '22%' >- RESULT -</td>` + 8 列 × 8.667%:
JP `試合数 | 勝ち数 | 分け数 | 負け数 | 得点 | 失点 | 得失差 | 順位`、EN `Games | Wins | Draw | Losses | Goals<br />for | Goals<br />against | Goals<br />diff | Rank`。
header 行・各データ行に **HTML コメント `<!-- <td class='MD1 CM' width= '8.667%' >勝ち点</td> -->` / `<!-- <td class='CM HL1' >9 </td> -->` が残存(非表示の勝ち点列)**。値は 3×勝+1×分 で 154 行全て一致。

データ行: `<td class='MD1 LM' >`+チーム名リンク / `<td class='CM HL1' >3</td>` 試合数 / `CM` 勝・分・負 / `<td class='CM' >26 </td>` 得点・失点 / `<td class='CM HL1' >19 </td>` 得失差 / `<td class='CM HL2' >1 </td>` 順位。
- **行順はマトリクスと同じ(チーム登録順)で順位順ではない**(5895: 居酒屋 司=1, Ciel=3, ブルース=2, I Don't Think=4)。
- 勝率列は無い(division 表との違い)。

#### (c) 結果一覧(試合一覧)
header: `CD(10%) | 日付(7%) | 時間(7%) | コート(7%) | チーム名(20%) | -(4%) | VS(3%) | -(4%) | チーム名(20%) | 状況(18%)`(EN: `CD | Date | Time | Field | Team | - | VS | - | Team | Status`)。
行:
```
<td class='MC_END LM' >A3</td>        CD
<td class='MC_END CM' >01/13</td>     日付 mm/dd(年なし)
<td class='MC_END CM' >09:00 </td>    時間(末尾スペース)
<td class='MC_END CM' >1</td>         コート
<td class='MC_END RM' ><span class='teamLink'><a … scv_team/open/15396…>居酒屋　司</a></span></td>  左チーム(右寄せ)
<td class='MC_END CM' >8</td>  <td class='MC_END CM' >Ｘ</td>(全角)  <td class='MC_END CM' >2</td>
<td class='MC_END LM' ><span class='teamLink'>…</span></td>  右チーム(左寄せ)
<td class='MC_END CM' >\n<a href="#" class="Select_Link" onclick="…scv_match/open/40613?lang=jp…">[対戦結果]</a> </td>
```
- 行のクラス接頭辞: 終了 `MC_END`(#0000ff on #F8F8FF)。CSS に未開始 `MC_MAE`(黒/白)、試合中 `MC_NOW`(#ff0000/#fff0f5)、`MC_MEN/MC_WOM/MC_MIX`(#ddfbdd/#fff4e4/#ffffd1)も定義があるが、取得した 3531 行は全て `MC_END`。**未開始・試合中の状況列の文言は未観測**。
- 状況リンク文言: JP `[対戦結果]` / EN `[Result]`。`.Select_Link {color:#008080; font-size:116%; text-decoration:none}`。
- **並び**: 日付・時間昇順(CD 順ではない。5895 は A3, A4, A2, A5, …)。
- CD = `{グループ文字}{連番}`: ブラケットでも使われる(Tournament A=A4〜A12 等、同一 division でプール/ブラケットを跨いで通し番号。1st placement は E2〜)。

### 2.2 プールの順位決定規則(推定。順位が数値の 41 プールで検証し不一致 0)
1. 勝ち点(3×勝+1×分)降順
2. 同点 2 チームなら直接対決の勝者が上(引き分けなら次へ)。例 6511: 和歌山大(得失差+9) < 中京大(+6) だが直接対決で中京勝ち→中京が 1 位。6040: たんきち / いもべいびー は得失差とも +2 同値で、直接対決勝者のいもべいびーが上。
3. 得失差降順。例 6037: IKU!(+36) と SeeKers(+42) は 2勝1分で直接対決 7-7 → SeeKers が 1 位。
4. 得点降順(1〜3 で決まらない例は無く、4 は未検証=推定)。
- 3 チーム以上の同点は 41 件に存在せず未検証。
- 未解明例外: ediv 848 / group 6095 `ウィメン - Qualifying pool K`(3 チーム、全 3 試合 MC_END)で順位が ALBA=1 のみ、他 2 チームは順位セル空。順位セルが空になる条件がある(原因不明)。

### 2.3 トーナメント(ブラケット)型(事実)
```
<span class='SCV_MIDA_M'>&lt; ビーチミックス - Tounament A &gt;</span><br />
 <!-- GeneratePtn:T8A -->
<!-- PtnName :2026 Atami Line =Tournament A= -->
<div class='TN_TMPL'>
<table> … </table>
</div><br />
 <span class='SCV_MIDA_M'>&lt; 結果一覧 &gt;</span><br />
 <div class="TBL"><table> … 2.1(c) と同じ試合一覧 … </table></div>
 <hr> 
<div class="TBL"> </div>
```
- MATRIX / RESULT 節は無い。末尾の試合一覧はそのグループの全試合(ブラケット内の全試合)。
- GeneratePtn コード(取得分): `T4E T4G T7H T8A…T8H T8J T12H T16A W9A`。数字=参加チーム数。末尾英字はディビジョン内グループ順(949 の 8 グループ=A〜H に一致、DREAM CUP の Championship=H=8 番目)と推定。`PtnName` は `{テンプレ名} =グループ名=`。

#### ブラケットは画像・SVG・canvas ではなく **1 枚の `<table>` のセル罫線で描画**
CSS(`score_app.css`):
```
.TN_TMPL table        {border-collapse:collapse; border:0px; width:100%; table-layout:fixed;}
.TN_TMPL table tr td  {padding:2px; word-break:break-all;}
.TN_TMPL .BdLeft  {border-left:   2px solid #ffa000;}
.TN_TMPL .BdRight {border-right:  2px solid #ffa000;}
.TN_TMPL .BdTop   {border-top:    2px solid #ffa000;}
.TN_TMPL .BdBtm   {border-bottom: 2px solid #ffa000;}
.TN_TMPL .txt_yen   {text-align:center; vertical-align:middle; color:#0000ff; font-weight:bold;}  /* ラウンド見出し */
.TN_TMPL .txt_sharp {text-align:center; vertical-align:middle; color:#ff0000; font-weight:bold;}  /* Placement列 */
.TN_TMPL .txt_and   {text-align:center; vertical-align:middle; color:#0000ff; font-weight:bold;}  /* 小見出し/A rank n */
.TN_TMPL .txt_left  {text-align:left;  vertical-align:middle; color:#ff4500; font-weight:bold;}  /* 未使用 */
.TN_TMPL .txt_right {text-align:right; vertical-align:middle; color:#ff4500; font-weight:bold;}  /* 未使用 */
.TN_TMPL .game {text-align:center; vertical-align:middle; color:#000; font-weight:bold;}          /* スコア */
.TN_TMPL .team {text-align:left; vertical-align:middle; color:#008000}                            /* 1回戦チーム(緑) */
.TN_TMPL .win  {text-align:left; vertical-align:middle; color:#ff0000}                            /* 勝者(赤) */
.TN_TMPL .lose {text-align:left; vertical-align:middle; color:#0000cd}                            /* 敗者(青) */
.TN_TMPL .rank {text-align:right; vertical-align:middle; color:#ff4500; font-weight:bold;}        /* 未使用 */
.TN_TMPL .WDiv{N} {width: 100/N %}  (WDiv1=100% … WDiv3=33.333% WDiv4=25% WDiv5=20% … WDiv20=5%)
.teamimg_tn img {margin:0 4px; vertical-align:text-bottom; border:1px solid lightgray}
.teamimg_tn a {color:#0000ff; text-decoration:none}
.tnmt_game_link {color:#008080; font-size:123.1%}   .tnmt_play_now {color:#ff0000; font-size:123.1%}
```
- `td` のクラスは `WDiv{列数} {BdXxx…} {kind}` 例 `<td class='WDiv4 BdRight BdTop game '>`(末尾空白あり)。空セルは `<td class='WDiv4 '>&nbsp;</td>`。全 td が同一 `WDiv{N}`(N=テーブル列数)。table に class なし、`<tr>` に属性なし。
- 列構成: 6623(T8A)は 4 列(1回戦チーム / 2回戦 / 決勝 / Placement)、40 行。6627(T4E)3 列(Semifinals / FINAL / Placement)。T16A(4992)は 5 列。
- ヘッダ行: `txt_yen` セルにラウンド名、`txt_sharp` セルに `Placement`。ヘッダが空行のテンプレもある(6623 は 1 行目の 3 セルが空で最後に Placement)。
- 1 試合の描き方(3 行):
  ```
  行k  : [BdBtm team] 上チーム            | 空 …
  行k+1: [BdRight BdTop game] <a class="tnmt_game_link" …scv_match/open/45280…>2-12</a> | [BdLeft BdBtm win] 勝者名 | …
  行k+2: [BdRight BdBtm team] 下チーム    | [BdLeft BdRight BdTop 空] | [BdLeft 空] …
  ```
  スコアは **上チーム-下チーム順**。罫線(┐┘├)は BdTop/BdBtm/BdLeft/BdRight の組合せだけで表現。勝者は次列に `win`(赤)で名前を再掲、敗者は下位ブラケット側に `lose`(青)で再掲。
- Placement 列(英語固定、JP ページでも同一):
  - 予選型ブラケット(Tounament A〜D)の右列: `A rank 1`(`txt_and`)+ 直下 `1st placement T →`(`txt_sharp`)、`A rank 2`/`5th placement T →`、`A rank 3`/`9th placement T →`、`A rank 4`/`13th placement T →`、`A rank 5`/`17th placement T →`、`A rank 6`/`21st placement T →`、`A rank 7`/`25th placement T →`、`A rank 8`/`29th placement T →`。
  - 最終順位ブラケット: `1st`,`2nd`,`3rd`,`4th`(6627)/ `5th(tied)`,`7th(tied)`,`9th(tied)`,`11th(tied)`(6628)/ `21th(tied)`,`23th(tied)`,`25th(tied)`,`27th(tied)`,`29th(tied)`,`31st(tied)`(6630。`21th` `23th` は原文の誤り)。小見出し `3rd - 4th`,`5th - 7th`,`9th - 11th`,`21th - 23th`,`25th - 27th`,`29th - 31st`(txt_and)。
  - 旧大会: `★1位★`、`Champion`(6042)、`優勝決定トーナメント`(txt_and 見出し)、`Quarterfinals / Semifinals / Finals`(2527)、`Semi final / Final`(4992)、`1st - 9th`(W9A)。
- **引き分けスコアでも win/lose が付く**: 6570 `4-4` で 私大３年 が `win`(スコア文字列からは勝者を決められないので、別データ(サドンデス等)を持っている)。
- バイ: T7H(6042)は空セルで上位列へ直通。棄権表記は無い。

#### ディビジョン 949 の全 8 グループ(全てトーナメント型、プール型 0)
| gid | 名前 | 型 | チーム数 | 試合数 | 備考 |
|---|---|---|---|---|---|
| 6623 | Tounament A | T8A | 8 | 12 | 初戦 A4〜。Placement 列に `A rank 1…8` と `{n}th placement T →`(上表) |
| 6624 | Tounament B | T8B | 8 | 12 | 同形 |
| 6625 | Tounament C | T8C | 8 | 12 | 同形 |
| 6626 | Tounament D | T8D | 8 | 12 | 同形 |
| 6627 | 1st placement tournament | T4E | 4 | 4 | 準決勝2+決勝+3位決定、`1st`〜`4th` |
| 6628 | 5th-9th placement tournament | T8F | 8 | 4 | 2試合×4ブロック(`5th - 7th`×2, `9th - 11th`×2)、`5th(tied)`〜`11th(tied)` |
| 6629 | 13th-17th placement tournament | T8G | 8 | 4 | 同形(13〜19位) |
| 6630 | 21st-29th placement tournament | T12H | 12 | 6 | `21th - 23th`×2, `25th - 27th`×2, `29th - 31st`×2 |

### 2.4 「勝者/敗者が次の試合へ進む」関係は HTML から読めるか(事実)
- group 内: 勝者・敗者名が次列セルに再掲され罫線で結ばれ、視覚的に読める。DOM に試合ID同士の親子参照は無い(同名テキストの重複のみ)。
- group 間: `A rank 1` + `1st placement T →` の **プレーンテキストのみ**(リンク無し)。後続グループとはチーム ID の一致でしか結びつかない。

---
## 3. 他大会との比較(10 大会・約 50 グループ取得)
| 大会(scview) | ediv | 取得 group | 型 |
|---|---|---|---|
| 2025 初投げCUP 1部 (435) | 815 | 5895〜5898 pool A〜D, 5899 1st placement(T8E), 5900 9th placement(T8F) | プール+ブラケット |
| 2025 Invitational (446) | 838/839 | 6006,6007,6010,6011 Qualifying pool, 6008 Championship pool E | プール(決勝もプール) |
| NNFDA 2025 菅平 (451) | 844 | 6034〜6040 pool A〜G, 6042 Championship tournament(T7H), 6045 22nd placement pool | プール+ブラケット+順位決定プール |
| GREATEST CUP 2025 WEST (453) | 847/848 | 6084〜6088, 6092 17th placement pool, 6093〜6095 | プール |
| Ikuta Night 2026 G9 (504) | 924 | 6590〜6592, 6596 9th placement pool | プール(3〜4チーム) |
| Makuhari 2026 Sec7 (511) | 931 | 6564〜6566, 6570 9th placement tournament(T4G) | プール+ブラケット |
| GREATEST CUP 2026 WEST (519) | 940/941 | 6509〜6512, 6518, 6523 9th placement pool | プール |
| DREAM CUP 2025 Mixed (439) | 821 | 5953 Tournament A(T8A), 5960 Championship(T8H), 5962 17th placement(T8J) | ブラケットのみ |
| 2018 NGC 仙台 (186) | 406/407 | 2527(T8A), 2529 予選あリーグ(3チーム) | 日本語グループ名 |
| NNFDA 2019 (203) | 430 | 2876 予選Aリーグ(4チーム、引分3件), 2882 優勝決定トーナメント(T8A) | 同上 |
| 2017 NGC 東日本 (162) | 360/361 | 2182 MAリーグ(5チーム総当たり), 2186 最終順位決定(1位ー8位, T8A), 2190 順位決定トーナメント表(W9A) | 同上 |
| 2022 Fuji 4on4 Vol.7 (342) | 662 | 4992 Championship tournament(T16A) | 16チーム |

比較結果(事実):
- 現行(2022〜)のプールは **3〜4 チームの総当たり**、5 チーム総当たりは 2017 の MAリーグ(2182)。構造(MATRIX/RESULT/結果一覧)は全プール共通。
- プール名: `Qualifying pool A`〜、`Championship pool E`、`9th placement pool`、`17th placement pool`、`22nd placement pool`、旧 `予選Aリーグ` `MAリーグ`。
- プール→プール/ブラケットの繋がりはプールのページには一切書かれない。
- 2027 大会(533, 535, 539)は division があるが group 0 件(score_navi の「グループ成績」見出しは出る)。
- gid 欠番(453 の 6089/6096、451 の 6041)は navi に無い = 非公開グループが存在すると推定。

---
## 4. 順位決定の表記・特殊状況(事実)
- 順位列は数字のみ(`1`, 末尾に半角スペース)。「位」なし。同順位は同じ数字(次は飛ぶ)。
- 順位セルが空: division 未開催(954)・プール 6095 の一部。
- 棄権・不戦・没収: 取得した全 HTML(group/division/match1/team1)に `棄権` `不戦` `没収` `forfeit` 無し。試合一覧は通常の `X-Y` のみ。不戦勝の扱い(0-0/規定スコア)は未検証。
- SOTG/スピリット: division / group / match(45280) / team(17498) ページに該当語なし。CSS に `.SCV_SOTG_COMM {font-size:93%; color:#ff4500}` が残るのみ。
- 引き分け: 分け数列あり。プール勝ち点では 1 点。division 勝率では勝ち扱いにならない。ブラケットの同点にも勝者が付く。

---
## 5. lang=en 版の文言(事実)
| 要素 | JP | EN |
|---|---|---|
| division 見出し | `< ビーチミックス 順位表 >` | `<Beach Mixed Standings>` |
| リーダー見出し | `< スコアボードリーダー >` / `< スコアボードリーダー　（ブロック） >` | `<Scoreboard Leaders>` / `<Scoreboard Leaders *Block*>` |
| 順位表 header | チーム名 試合数 勝ち数 分け数 負け数 得点 失点 得失差 勝率 順位 | Teams Games Wins Draw Losses Goals for / against / diff, Win %, Rank |
| リーダー表 header | チーム名 # 選手 試合数 アシスト ゴール トータル / ブロック / -- | Team # Player Games Assist Goal Tot. / Block / -- |
| group 見出し | `< ビーチミックス - Tounament A >` | `<Beach Mixed - Tounament A>` |
| プール節 | `- MATRIX -` `< リーグ結果 >` `- RESULT -` `< 結果一覧 >` | `- MATRIX -` `<Reague Result>` `- RESULT -` `<GameList for the Group>` |
| 試合一覧 header | CD 日付 時間 コート チーム名 - VS - チーム名 状況 | CD Date Time Field Team - VS - Team Status |
| 状況リンク | `[対戦結果]` | `[Result]` |
| ブラケット内文言 | 英語固定(Placement / Semifinals / FINAL / A rank n / n-th placement T →) | 同左(日本語混じりの旧大会表記 `★1位★` 等を除く) |
| チーム名 | 日本語名 | 英語名(別フィールド): マンマユート→`Mamma Aiuto`, バトルロイヤル→`BRH`, 28卒→`28sotsu`, 居酒屋　司→`izakaya tsukasa`, ジャンジャン→`jang jang` |
| 選手名 | `(選手名)` | `(Player Name)` |
| ナビ | 大会トップ / チーム一覧 / スケジュール / 全ディビジョン / ディビジョン成績 / グループ成績 / ギャラリー | Tournament Top / Team List / Schedule / All Division / Division Score / Group Score / Gallery(左メニューのディビジョン名は `Beach Mixed(Ult)`) |
| title | `[CJS] 2025 初投げCUP 1部 -Qualifying pool A-` | `[CJS] 2025 Hatsunage 1st -Qualifying pool A-` |
- 全リンクの `?lang=` が en に引き継がれる。`Ｘ`(全角)は共通。

---
## 6. データモデル推定
```
tournament (scview/{tid}) 1 ─ * division(ediv: scv_division / scv_search_div)
   division 1 ─ * group (scv_group/open/{gid})  navi に division 単位で列挙、gid 昇順
        type = pool(league) | tournament    ← MATRIX 有無 / GeneratePtn コメント有無で判別
        name = "Qualifying pool A" | "Tounament A" | "1st placement tournament" …
        pattern(tournament のみ) = GeneratePtn:T{チーム数}{グループ順英字}、PtnName
   group 1 ─ * match (scv_match/open/{mid}), CD = "{英字}{連番}"
        match = 日付/時間/コート/左チーム/右チーム/左得点/右得点/状況(MAE|NOW|END)/(同点時の勝者は別保持)
   team (scv_team/open/{id}) = 大会エントリー単位(別大会で別 ID)
   division 順位表 = 全 group の試合の合算(試合数/勝/分/負/得点/失点/得失差/勝率) + 順位 = placement ブラケットの確定順位
   player (scv_player/open/{pid}): division 単位の得点/アシスト/ブロック集計
```
- 順位決定の流れ: 予選(プールまたは Tournament A〜D)→ `A rank n` → `{m}th placement T`(次のグループ)。HTML 上は `rank n` テキスト+行き先テキストのみで、ID 参照なし。
- ブラケットの形は **テンプレート固有の固定グリッド**(同じ T8A でも 6623 と 6628(T8F)は別構造)。再現は(a)取得済みテーブルを静的保存、(b) 行×列と各セルの `{Bd*}{kind}{text/matchId}` を JSON 化して描画、のどちらか。

## 7. 再現時の落とし穴
1. td 末尾の空白(`'38 '`, `'… game '`)、`width= '24%'` の空白は原文ママ。
2. 全角: `Ｘ`、`　`が見出し/チーム名にある。
3. 順位は再計算せずデータとして持つ(placement 由来)。プールの順位は 2.2 の規則で再計算可能(推定)。
4. プール RESULT は登録順、division 表は順位順。
5. マトリクスは上三角のみ、スコアは記録の左右順。
6. ブラケットの Placement 文言は JP でも英語。
7. `Tounament`, `Reague`, `21th`, `23th` の誤綴りを再現するか要判断。
8. 勝ち点列は HTML コメントとして残っているだけ。
9. 未観測: 進行中(MC_NOW/RM_NOW/tnmt_play_now)・未開始(MC_MAE)表示、棄権・没収、3 チーム以上同点時のプール順位、順位が部分的に空の division 表。
