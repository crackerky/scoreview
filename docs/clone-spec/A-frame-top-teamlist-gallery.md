# A_spec: score.clubjr.com 外枠・大会一覧・大会トップ・チーム一覧・ギャラリー・見た目

調査方法: 実HTML/CSS/JSを取得して確認。リクエスト総数 71(上限150以内、間隔1.2秒、UA `Mozilla/5.0 (survey; low-rate)`)。
生データ: `$S/raw/A/`(HTML、CSS/JS、`img/` に画像、`img/colorbox/` にcolorbox画像)。リクエストログ: `$S/raw/A_reqlog.txt`。
CSV: `$S/A_tournaments.csv`(列 id,name_jp,name_en,section,pos)。
`S=/tmp/claude-1000/-home-crackerky/a6257ad0-3ffa-4ea0-952c-bc3db229737b/scratchpad/survey`
「推測」と書いた箇所以外はすべて実HTMLで確認した事実。

---------------------------------------------------------------------
## 0. 基盤の事実

- 実装: CodeIgniter 系(エラーページが CI 標準、`codeigniter_profiler` 参照が common.js にある)。XHTML 1.0 Transitional, UTF-8, `<html xml:lang="ja" lang="ja">`。
- 言語切替は **URLの `?lang=jp|en` のみ**。Cookie は一切セットされない(cookie jar 空を確認)。`lang` 省略 = jp と同一バイト(scview/525 と scview/525?lang=jp は 22790 B 完全一致)。lang=en の後に lang 省略で取得しても jp に戻る(状態は持たない)。ページ内の全リンクは現在の lang を引き継ぐ。
- **リダイレクトなし**。`/scview/{id}` と `/pscore/scv_top/open/{id}` は **同一内容(バイト単位で同一、22790 B)**。HTTP 200 を直接返す。`MyController` も両者とも `https://score.clubjr.com/pscore/scv_top`。つまり `/scview/{id}` は `/pscore/scv_top/open/{id}` のルート別名(URLを書き換えず同じコントローラが応答)。
- 全リンクは絶対URL(`https://score.clubjr.com/...`)。内部リンクの大半は `<a href="#" onclick="document.location='URL'; return false;">` 形式(大会一覧・大会トップのリンク・チーム一覧のチーム名)。ナビ(`#score_navi`)内は通常の `<a href>`。
- CSS/JS には ?20261021 のキャッシュバスタ(大会ページ側のみ。/sctnmt は付かない)。`<meta http-equiv="PRAGMA" content="NO-CACHE">`, `EXPIRES 0`。viewport `maximum-scale=2.0,user-scalable=yes`。
- head の読み込み順(全ページ共通):
  1. `/css/normalize.css`(2011-06-21 版 necolas normalize の日本語化版、SJIS系の文字化けコメントあり、無害)
  2. `/css/scv_def/score_fonts.css` 3. `score_frame.css` 4. `score_app.css`
  5. JS `/js/footerFixed-modified-for-scv.js`(jQueryより前)
  6. `/js/jquery-1.7.2.min.js` 7. `/js/jquery.dropdown.js`(v1.1.7 ShanaBrian, 2015)
  8. `<link> /css/scv_def/score_navi.css`(body内ではなくhead内、JSの後)
  9. `/js/common.js`
  10. インライン `<style>body {...}</style>`(下記 背景) と `<script>var MyController = ...; $(function(){ $('#score_navi').dropdown(); });</script>`(`dropdown()` 呼び出しは大会ページのみ。/sctnmt では呼ばない)
- `<body onload=''>`。

---------------------------------------------------------------------
## 1. 共通レイアウト

### 1.1 DOM 全体(大会ページ。/sctnmt との差分は 1.6)

```
body
 div#top  (width:980px; margin:0 auto)
  div#wrapper (width:100%)
   div#header  (position:relative; text-align:center; background:url(img/bg-header-white.jpg) no-repeat [1000x80])
    div#outer1
     div#header   ← id重複(入れ子)。同じCSSがもう一度効く
      table[cellpadding=0 cellspacing=0 border=0]
       tr1: td[width=22% rowspan=2] a[href=http://clubjr.com/] img.logo (2012-logo.gif 168x80, alt "クラブジュニア Club Jr. Sport is the best way of communication.")
            td[width=78% align=right valign=top]
               table > tr > td > div#header_social (SNS 5個)
                       td[align=right valign=top] > table
                          tr: td[background=09_header_bar.gif] > img spacer 1x11
                          tr: td#header_bar  "|　会社案内　|　個人情報保護法方針　|　通信販売について　|　お問い合わせ　|　English"
       tr2: td[align=right valign=bottom] > div#navigation > ul > li#navi01..#navi04 > a
       tr3: td[colspan=4] > img spacer 900x14
    ul#score_navi   ← 大会ページのみ(1.3)
   /div#header
   div#sub   (右サイドバー。1.5)
   div#contents > div#main > div#score_parts > div#score_view   (本文)
   <br />
  /div#wrapper
 /div#top
 div#footer_ue (clear:both; height:30px — JSで高さ可変)
 div#footer > div.copyright "Copyright(c) 2026 CLUB Jr. Inc. All Rights Reserved."
```

### 1.2 ヘッダ(全ページ同一、lang で変化しない = en でも日本語のまま)

SNSアイコン(`div#header_social` width:190px; padding:5px 35px 0 0; text-align:center;各 32x32 `border=0`、target=_blank):

| alt | href | 画像 |
|---|---|---|
| facebook icon | https://www.facebook.com/pages/%E3%82%AF%E3%83%A9%E3%83%96%E3%82%B8%E3%83%A5%E3%83%8B%E3%82%A2/241877315923973 | /css/scv_def/img/icon-facebook.jpg |
| twitter icon | https://twitter.com/clubjrjapan | icon-twitter.jpg(実寸 446x430 を width/height=32 で縮小表示) |
| YouTube icon | http://www.youtube.com/user/CLUBJrInc | icon-youtube.jpg |
| Flickr icon | https://www.flickr.com/photos/119452732@N04/albums | icon-flickr.jpg |
| Instagram icon | https://www.instagram.com/clubjrjapan/ | icon-instagram.png |

`td#header_bar`(font-size:12px; padding-right:6px) のリンク(全角スペースで区切り、先頭と末尾に `|`):
`|　会社案内(http://www.clubjr.com/company.html)　|　個人情報保護法方針(…/privacy.html)　|　通信販売について(…/order.html)　|　お問い合わせ(mailto:info@clubjr.com)　|　English(http://www.clubjr.com/english.html)`
注意: この "English" は外部サイトの英語ページへのリンクで、**スコアサイトの lang=en 切替ではない**。スコアサイト内に言語切替UIは存在しない(lang はURLを手で書き換えるのみ)。

`div#navigation`(width:765px)の `li#navi01..04`(各 width:183px; height:32px; float:right → 並び順は右から navi01,02,03,04 で、画面上は左から 04,03,02,01):
背景は `09_header_menu.jpg`(732x64。上段=通常、下段=hover)をスプライト。`a` は `display:block; height:0; padding-top:32px; overflow:hidden`(文字は画像で表現、テキストは隠れる)。
| id | テキスト(非表示) | href | target | background-position(通常 / hover) |
|---|---|---|---|---|
| navi01 | Event イベント | http://www.clubjr.com/eventschedule.html | - | -549px 0 / -549px -32px |
| navi02 | Dodgebee ドッヂビー | http://www.clubjr.com/dodgebee/index.html | _blank | -366px 0 / -366px -32px |
| navi03 | Ultimate gear アルティメットギア | http://www.clubjr.com/uniform.html | - | -183px 0 / -183px -32px |
| navi04 | Flyingdisc フライングディスク | http://www.clubjr.com/u_disc.html | - | 0 0 / 0 -32px |

ロゴ: `a[href=http://clubjr.com/] > img.logo`(168x80)。

### 1.3 大会内ナビ `ul#score_navi`

出る/出ない:
- 出る: 大会に紐づく全ページ — `/scview/{id}`, `scv_top`, `scv_team_list`, `scv_tgallery`, `scv_search_tn`, `scv_search_div`, `scv_division`(最後3つは取得済みHTMLで `ul id='score_navi'` 有を確認。scv_group/scv_team/scv_match は担当外で未確認、推測: 出る)。
- 出ない: `/sctnmt`(= `/pscore/sct_all` 大会一覧)。エラーページ(500/404)も出ない。
- 配置は `div#header` の中、`div#outer1` の直後(`</div>` の後)。`ul#score_navi{margin:0;padding:0;width:100%;height:50px;list-style:none}`。

大会ナビの標準構造(例: 525, 日本語、`style=''` の空属性まで再現):

```html
<ul id='score_navi'>
 <li style=''><a href=".../pscore/scv_top/open/{T}?lang=jp">大会トップ</a></li>
 <li style=''><a href=".../pscore/scv_team_list/open/{T}?lang=jp">チーム一覧</a></li>
 <li><p class="ar">スケジュール</p>
   <ul style='display: none;'>
     <li><a href=".../scv_search_tn/open/{T}?lang=jp">全ディビジョン</a></li>
     <li><a href=".../scv_search_div/open/{DIV}?lang=jp">{ディビジョン名}</a></li> …(ディビジョン数ぶん)
   </ul></li>
 <li><p class="ar">ディビジョン成績</p>
   <ul style='display: none;'>
     <li><a href=".../scv_division/open/{DIV}?lang=jp">{ディビジョン名}</a></li> …
   </ul></li>
 <li><p class="ar">グループ成績　</p>        ← 末尾に全角スペース(jp)。enは "Group Score"(スペースなし)
   <ul style='display: none;'>
     <li><p>{ディビジョン名}</p>                 ← サブメニュー(右に開く)
        <ul>
          <li><a href=".../scv_group/open/{GROUP}?lang=jp">{グループ名}</a></li> …
        </ul></li> …(ディビジョン数ぶん)
   </ul></li>
 <li style="width:172px"><p class="ar">ギャラリー</p>      ← ギャラリーがある時のみ
   <ul style='display: none;'>
     <li><a href=".../scv_tgallery/open/{GALLERY}?lang=jp">{ギャラリー名 例 Day2-AM}</a></li> …
   </ul></li>
</ul>
```
注: 525 実物ではギャラリー li に `style="width:172px"`。539 など複数日でギャラリー li が最後に来る場合は `<li>` の style 付きのまま(上記)。

lang=en の文言差分(確認済):

| jp | en |
|---|---|
| 大会トップ | Tournament Top |
| チーム一覧 | Team List |
| スケジュール | Schedule |
| 全ディビジョン | All Division |
| ディビジョン成績 | Division Score |
| グループ成績　(全角スペース付き) | Group Score |
| ギャラリー | Gallery |
| ディビジョン名(例 ビーチミックス) | Beach Mixed(DBの英語名) |
| グループ名(例 Tounament A) | そのまま(グループ名は言語非依存、綴り Tounament も原文ママ) |
| ギャラリー名 | そのまま(Day2-AM) |
| (ディスクゴルフ大会)チーム一覧 → エントリー一覧 | Entry List |
| (ディスクゴルフ大会)グループ成績 → 成績 | Result |

条件分岐(実例で確認):
- **標準大会(ウルティメット等)**: 上記 6 項目(ギャラリー無しなら 5 項目)。
- **ギャラリーなし**: ギャラリー li の代わりに フィラー `<li style="width:172px"><p>&nbsp;</p></li>`(151, 172, 300, 250 で確認)。幅の計算: 1項目あたり約 161.67px(160+境界)。5項目→フィラー 172px、3項目→ 495px(980 から逆算で一致)。6項目(ギャラリー有)ではフィラーなし。
- **スケジュールが無い大会**(試合データなし。例 523 GAMAGORI BEACH FREESTYLE チーム制、537/536 ディスクゴルフ): 「スケジュール」「ディビジョン成績」li が **丸ごと出ない**。残り = 大会トップ / チーム一覧(537はエントリー一覧) / グループ成績(537は「成績」)。グループ li の下はディビジョン名 `<li><p>…</p><ul></ul></li>` だけで中身(グループ)が空。末尾に幅 495px のフィラー li。
- **ギャラリーのみ**: 実例 539(受付中)でも既にギャラリー12本(Day1-AM … Day4-PM-13-16)が出る=ギャラリーの有無はデータ次第で、大会状態とは無関係。
- 受付中大会 539 も スケジュール/ディビジョン成績/グループ成績 は出る(ディビジョンが登録済みなら出る)。
- ディビジョンが複数(539 ミックス/オープン/ウィメン、151 メン/ミックス/ウィメン)の場合、スケジュール・ディビジョン成績・グループ成績の各サブメニューにディビジョンが **ディビジョンの並び順(DB順 = 大会トップの表と同じ順)** で並ぶ。グループ成績は「ディビジョン名(右へ開く)→グループ」の2階層。151 は メン: Aリーグ〜Iリーグ、Aトーナメント〜…、ミックス、ウィメン と合計 50 近いグループがある。

dropdown の挙動(`jquery.dropdown.js` v1.1.7、`$('#score_navi').dropdown()` 引数なし=既定値):
- action=`hover`, animate=`slide`, animateDirection=`y`, animateDuration=`100`ms, easing=`linear`, actionElement=`li`, targetElement=`ul`。
- 初期化: `li:has(ul)` を全部(入れ子含む)対象に、子 `ul` を opacity 0 / height 0 に 0ms アニメーション後 `hide()`。
- `mouseenter` で `ul.show().stop().animate({opacity:1, height:初期化時に測った高さ}, 100ms)`、完了で `overflow:visible`。`mouseleave` で `{opacity:0,height:0}` → 完了で子の開いている ul を hide し自身も hide。
- 高さは **初期化時に `target.height()` で測った値**。ただし初期状態が `display:none` なのでjQuery 1.7 は一時表示して測る(通常の挙動)。
- showClass/textClass 等は未設定(クラス付与なし)。`a.open` クラスは使われない(CSSにはあるが付かない)。

`score_navi.css` の見た目:
- `li{float:left; position:relative; width:160px; border-left/top/bottom:1px solid #888}`、最後の li に `border-right:1px solid #888 !important`。サブメニュー内 li は left/right/top を `!important` で 1px #888、bottom なし(最後だけ bottom 1px)。
- `li a, li p{padding:8px 0; margin:0; display:block; color:#008080; text-decoration:none; opacity:.99; background:repeating-linear-gradient(-45deg,#f9f9f9,#f9f9f9 5px,#fdfcff 5px,#fdfcff 10px)}`(斜めストライプ薄灰)。`text-align` は header の `text-align:center` が継承 → 中央揃え。
- `p.ar`(第1階層のサブメニュー持ち): 背景に `navi_ard.png`(11x6、下向き三角)を `140px 50%` no-repeat + ストライプ。
- 第2階層内の `p`(右へ開くディビジョン見出し): `navi_arr.png`(6x11、右向き三角)を `145px 50%` + ストライプ。
- ホバー/open: `a:hover, a.open{background:#ffd973}`(`p` にはホバー色なし。`a` のみ黄)。
- サブメニュー `ul{position:absolute; z-index:999; top:22px; left:-1px; padding:0}`、第2階層 `ul li ul li ul{position:absolute; top:-7px; left:160px}`、第2階層の `a, p` は `padding:5px 0 !important`。
- フォント 13px・line-height 1.5em は body 継承なので、第1階層 li の高さ ≒ 8+19.5+8 = 35.5px(+罫線)。ul 高さ 50px 固定。

### 1.4 フッタ

```html
<div id="footer_ue"></div>
<div id="footer">
	 <div class="copyright">Copyright(c) 2026 CLUB Jr. Inc. All Rights Reserved.</div>
</div><!-- /#footer -->
```
- `#footer{background:#f18700}`(オレンジ。幅は画面全幅=#top の外)。`.copyright{padding:8px 0; font-size:85%; color:#fff; text-align:center; font-style:normal}`。
- `#footer_ue{clear:both; height:30px}`。`footerFixed-modified-for-scv.js`: window load/resize で `body.clientHeight` と `window.innerHeight` を比較し、`(フッタのtop+高さ) < ウィンドウ高` の時だけ `#footer_ue.style.height = wh - footerHeight - footer_ue.offsetTop - bodyMarginBottom + 'px'`。→ **ページが短い時にフッタを画面下に押し下げる**(sticky footer)。1秒毎に文字サイズ変更を監視し再計算(setInterval)。
- 年 "2026" はハードコードかサーバ現在年かは不明(推測: 現在年。取得日 2026-10-21)。

### 1.5 サイドバー `div#sub`(大会ページ)

```html
<div id='sub'>
 <table cellpadding=0 cellspacing=0 border=0>
  <tr><td colspan=3><img src=".../09_menu_outline_top.jpg" width=210 height=9></td></tr>
  <tr><td width=4 background=".../09_menu_outline_left.jpg"></td>
      <td width=202 align=center>
        <table …><tr><td><div style='text-align:center; margin:15px 0 15px 0; line-height:40%;'> {バナー} </div></td></tr></table>
      </td>
      <td width=4 background=".../09_menu_outline_right.jpg"></td></tr>
  <tr><td colspan=3><img src=".../09_menu_outline_bottom.jpg" width=210 height=9 class="img_margin"></td></tr>
 </table>
</div>
```
- CSS: `#sub{float:right; width:210px; padding:2px; background:#fff; margin-right:20px}`。
- **常に出力される枠**。バナー無し大会(151, 172, 507, 537 等)では中身が空の `<div …></div>` で、枠線画像だけの空の白いボックスが出る。バナーあり(525, 539, 523, 528 …)は 160x64 の画像が `<br /><br />` 区切りで縦並び(525 は 18枚、539 は 30枚、528 は 25枚、523 は 8枚)。画像は `/zfiles/ScoreViewTop/新Banner(202404以降)/スコアサイト用リンクバナー…jpeg`(日本語ファイル名はURLエンコード)、一部は `<a href=外部>` で囲まれる。バナーセットは大会ごとに違う(525 は熱海関連、523 は愛知FD協会…)=大会ごとのCMS管理HTML。
- `/sctnmt` では `#sub` ブロック全体が `<!-- … -->` でコメントアウトされている(出ない)。
- 画像実寸: top/bottom 199x9(表示は 210x9 に拡大指定)、left/right 4x1。

### 1.5.1 サイドバーの SCORE メニュー(調査後に追記)

調査時には見落としており、ページ担当のスクリーンショット比較で見つけた。生 HTML 271 ページで確認した事実:
- 出るページ: scv_division、scv_group、scv_team、scv_match、scv_player、scv_tgallery。出ないページ: sct_all(大会一覧)、scv_top(大会トップ)、scv_team_list、scv_search_tn、scv_search_div。
- 位置: `#sub` の内側の table の先頭行(バナーの行の上)。`<td align="left">` の中に、見出し画像 `09_menu_score.png`(180x21、橙の帯に「SCORE」)、高さ 7px の空き、区切り線 `09_menu_brake_line.gif`(180x1)、`div.menu_list > ul`、最後にもう一度区切り線。
- `ul > li` ごとにディビジョン 1 つ。`<a href=".../scv_division/open/{DIV}?lang=..">{ディビジョン名}(Ult)</a>` の後に、そのディビジョンのグループを `ul > li > a`(→ scv_group)で並べる。名前の接尾辞 `(Ult)` は日英とも付き、観測した全ディビジョンで同じ値だった。
- スケジュールのないディビジョン(ディスクゴルフ、フリースタイル)は載らない(947、17459 の大会で項目 0)。
- CSS(score_frame.css): `#sub ul{margin:2px 0 0 2px;padding:0}`、`#sub li{padding:2px 0 2px 12px;line-height:1.4;list-style:none}`、`#sub li a{color:#008080;display:block;padding-left:12px;background:url(bg_list.gif) 0 5px no-repeat}`、`#sub li p{color:#696969;...}`。

### 1.6 /sctnmt (大会一覧) と大会ページの外枠差分
- `<style>body {background: black}</style>`(黒背景)。大会ページは `body {background: white url(https://score.clubjr.com/img/ScoreView/def_bgimg.png) repeat top;}`。
- `#score_navi` なし、`#sub` コメントアウト、`dropdown()` 呼び出しなし。ヘッダ・フッタは同一。
- `MyController` = `https://score.clubjr.com/pscore/sct_all`、CSS/JS に ?日付 バスタなし。
- `<title>[CJS] ClubJr大会一覧</title>`(lang=en でもタイトルは日本語のまま)。大会ページの title は `[CJS] {大会名} ` (末尾に半角スペース1個)。

### 1.7 ページ title 規則
- 大会トップ: `[CJS] {name} ` — name は **lang に応じて jp名/en名**(jpは jp名、en は 英語名)。**英語名が空の大会は `[CJS]  `(空)**(537, 172 の lang=en で確認)。
- ただし本文の `div.text_title` は **常に jp名**(lang=en でも日本語名のまま。534 で確認: title "2027 Hatsunage 2nd" / text_title "2027 初投げCUP 2部")。
- スケジュール: `…-ディビジョンスケジュール-` が末尾に付く(949 の例)。
- チーム一覧・ギャラリー: `[CJS] {大会名} `(接尾辞なし)。

---------------------------------------------------------------------
## 2. デザイントークン(CSS を読んだ事実)

### 2.1 基本
| 項目 | 値 | 出典 |
|---|---|---|
| ページ幅 | `#top{width:980px; margin:0 auto}` | score_frame |
| 本文 | `#main{float:left; width:710px; min-height:550px; padding:5px; background:rgba(255,255,255,1); border:1px solid #888}` → 外寸 722px | frame |
| サイド | `#sub{float:right; width:210px; padding:2px; margin-right:20px; background:#fff}` | frame |
| body 文字 | `font:81.25%/1.5em "ＭＳ Ｐゴシック","MS PGothic","ヒラギノ角ゴ Pro W3","Hiragino Kaku Gothic Pro",sans-serif;` (=13px)、`letter-spacing:.05em`、`color:#333`、margin/padding 0 | fonts / frame |
| 入力系 | `select,input,button,textarea{font:100% 同上}`、`table{font-size:inherit;font:100%}`(→ table は body と同じ 13px 相当) | fonts |
| 全体 | `html{overflow-y:scroll}`(縦スクロールバー常時)、score_app で `html{overflow-x:hidden}` | frame/app |
| リンク | `a{color:#333;text-decoration:none}` `a:hover{color:#0172AB}`、`#main a{color:#0172AB}` `#main a:hover{text-decoration:underline}` | frame |
| img | `border:0; vertical-align:bottom` | frame |
| フッタ背景 | `#footer{background:#f18700}` | frame |
| 見出しオレンジ | `#f66013`(h2 の border-top, h3 の left border, h5 色 — スコア画面ではほぼ未使用) | frame |
| 見出し h4 | `#main h4{margin:5px 0 2px 30px; padding:2px; background:#d0dfe9}` | frame |
| dd | `#main dd{background:#f1f1f1; padding:3px}` | frame |
| フォントサイズ補助 | `.px10=77% .px11=85% .px12=93% .px13=100% .px14=108% .px15=116% .px16=123.1% … .px26=197%`(YUI方式)、`.fp20…fp60` は 20〜60% | app |

重要な詳細度の罠: `#main a`(ID詳細度)が `.Select_Link{color:#008080}` に勝つため、`#main` 内の `[スケジュール]` `[最終順位]` のリンク色は実際には **#0172AB**(.Select_Link の color は無効)、サイズは `.Select_Link{font-size:116%; text-decoration:none}`。一方 `<span class='Dummy_Link'>` は `a` ではないので **#ccc**(116%)が効く。`.teamimg_rnk a{color:#fff !important}` は !important で #main a に勝つ → 白。

### 2.2 クラス定義表(score_app.css)
| クラス | 定義 |
|---|---|
| `.text_title` | (frame) `clear:left; width:695px; padding:26px; margin:6px 0 10px 7px; font-weight:bold; font-size:161.6%; background:url(img/09_ti_bg.png) no-repeat`(695x70 の見出し背景画像。width695+padding52=747px で #main(710)からはみ出す。overflowはvisible、これが原寸) |
| `.TeamListMIDA` | `font-size:131%; font-weight:bold; line-height:150%; color:#fff; background-color:#ffa000; padding:4px 10px; text-align:center; border-radius:4px` |
| `.TeamListBLOCK` | `padding:0; float:left; border:1px solid #ccc; white-space:nowrap; overflow:hidden` |
| `.FTitle` | 青グラデ `linear-gradient(to bottom, #75a5e7 0%, #1d4395 64%)` |
| `.FTitle_Univ` | 緑 `linear-gradient(#9fd959 0%, #5cb03e 44%, #22742b 100%)` |
| `.FTitle_Women` | ピンク `linear-gradient(#fac0f2 0%, #f59bd0 31%, #f26a87 100%)` |
| `.FTitle_Mix` | 紫 `linear-gradient(#e084ea 0%, #9f138a 100%)` |
| `.FTitle_Waku1` | 橙黄 `linear-gradient(#edc603 16%, #e27500 78%)` |
| `.FTitle_Waku2` | オリーブ `linear-gradient(#d5d898 0%, #c2c66c 17%, #aeb244 33%, #878c00 70%, #5f6200 100%)` |
| `.AreaSepa` | 灰黒グラデ `linear-gradient(to bottom, #959595 0%, #0d0d0d 46%, #010101 50%, #0a0a0a 53%, #4e4e4e 76%, #383838 87%, #1b1b1b 100%)`(チーム人数 "x11" の黒いセル) |
| `.FNaiyo` | `linear-gradient(135deg,#f6f8f9 0%,#e5ebee 50%,#d7dee3 51%,#f5f7f9 100%)` |
| `.SCV_MIDA_S` | `font-size:93%; color:#00f; bold; line-height:120%` |
| `.SCV_MIDA_M` | `font-size:131%; bold; line-height:250%; color:#fff; background:#FF6600; padding:4px 10px; text-align:center; border-radius:4px`(ギャラリーのタイトル・自由記述内の橙バッジ) |
| `.SCV_MIDA_L` | `font-size:167%; color:#000; bold; line-height:200%` |
| `.SCV_SOTG_COMM` | `93%; color:#ff4500; line-height:120%` |
| `.SCV_TEAM_COMM` | `100%; color:#483d8b; line-height:140%; margin:5px 30px 10px; padding:6px 10px; border:2px solid lightgray` |
| `.Select_Link` | `color:#008080; font-size:116%; text-decoration:none` (実効色 #0172AB、上記罠) |
| `.Dummy_Link` | `color:#ccc; font-size:116%; text-decoration:none` |
| `.playerLink` | `#008080`; `.playerLinkM` `#000080!important`; `.playerLinkF` `#cf143c!important` |
| `.teamLink a` | `color:inherit` |
| `.teamimg img` / `.teamimg_rnk img` | `margin:0 4px 2px 2px; vertical-align:middle; border:1px solid lightgray` |
| `.teamimg a` | `#008080`; `.teamimg_rnk a` `#fff !important`; `.teamimg_tn a` `#00f; no underline`; `.teamimg_big_match a` `#fff !important` |
| `.tnmt_game_link` | `#008080; 123.1%`; `.tnmt_play_now` `#f00; 123.1%` |
| `.TBL` | `margin:0 10px` |
| `.TBL table` | `margin:0; padding:0; border-top/left:1px solid #d4d0c8; border-collapse:collapse; border-spacing:0; background:#fff; empty-cells:show; table-layout:fixed; width:100%; font-size:93%` |
| `.TBL td` | `border-right/bottom:1px solid #d4d0c8; padding:.1em .3em` |
| `.TBL TD.MD1` | `font-weight:bold; color:#fff; background-color:#ffa000`(テーブルヘッダ橙) |
| `.TBL A.MD1` | `bold; #fff !important; underline`; `A.MD1_ASC` `#0f0 !important`; `A.MD1_DSC` `#0ff !important` |
| `.TBL TD.EVT` | `#00f on #f5f5f5`; `EVT_GLP` `#000 on #caecf8`; `EVT_GRP` `#000 on #f6f1c5` |
| `.TBL TD.HL2` | `#f00 on #fff0f5`; `HL3` `#00f on #f5f5f5`; `ABS` `#000 on #c0c0c0` |
| 対戦リスト行状態 | `MC_MAE`(試合前) `#000 on #fff`; `MC_NOW`(試合中) `#f00 on #fff0f5`; `MC_END`(終了) `#00f on #F8F8FF`; `MC_MEN` bg `#ddfbdd !important`; `MC_WOM` bg `#fff4e4 !important`; `MC_MIX` bg `#ffffd1 !important` |
| リーグマトリクス | `RM_MD1` `#fff on #ffa000 bold`; `RM_NASI` `#000 on #fff3b1 bold`; `RM_AIDA` `#000 on #ffa000 bold`; `RM_NOW` `#f00 on #fff0f5 bold`; `RM_ARI` `#00008b on #f5f5f5 bold`; `.RM_NOW_Link` `#f00 bold`; `.RM_ARI_Link` `#00f bold` |
| 対戦画面 | `TG_MD1` `bold #fff on #FF6600`; `TG_VSTITLE` `153.9% bold #fff on #FF6600`; `TG_POINT` `153.9% bold #00f on #fff`; `TG_NOWPLAY` `bold #f00 on #fff0f5`; `TG_EVMOJI` 100%; `.TG_EventList{min-height:400px; border-top/bottom:1px solid #d4d0c8}`; `.TG_PlayersList_Ult/_Guts{overflow-y:scroll; height:150px; border-top/bottom 1px solid #d4d0c8}` |
| 位置揃え汎用 | `.LM` left `.RM` right `.CM` center `.LMB/.LRB/.LCB`(+bold) `.VAT/.VAM/.VAB` `.BOLD` `.NOWRAP` `.CURSOR_P` |
| 色クラス | `.HISU #f00`(必須) `.NINI #0a0`(#00AA00、任意 — 検索フォームのラベル「日付」「チーム名」等に使用) `.MIDA #00f` `.COMM #ff8c00` |
| 入力枠 | `.FWAKU 1px #7f9db9`(普通) `.NWAKU 1px #6495ed`(任意) `.HWAKU 1px #ffa500`(必須) `.LWAKU 1px #fff` `.LWAKU_Y 1px #ffffe0 bg #ffffe0` |
| 幅/高さ汎用 | `.WMAX 100%`, `.W99p…W75p`, `.H0 H5 H10 H18 H24 H30 H40 H50` |
| ボタン | `.BTN_S 50px .BTN_M 100px .BTN_L 140px .BTN_LL 220px`、`text-align:center; color:#000; padding:3px` |
| `.ITBL table` | `margin:10px; border-collapse:collapse; table-layout:fixed; empty-cells:show`(入力画面用、罫線なし) |
| `.TN_TMPL` | トーナメント表: `BdLeft/Right/Top/Btm` = 2px solid #ffa000; `.game` 太字黒中央; `.team` #008000; `.win` #f00; `.lose` #0000cd; `.rank/.txt_left/.txt_right` #ff4500 太字; `.txt_yen/.txt_and` #00f; `.txt_sharp` #f00; `.WDiv1..20` = 100/n % |
| `ul.sstab` | チーム紹介年度タブ。`border-bottom:2px #ddd solid`、li float:left `margin:0 0 0 5px; border:1px #ddd solid; border-top:none; border-bottom:2px #ddd solid; position:relative; bottom:-2px`、`li.none` は `border-bottom:2px #fff solid; bold`、`a` は白→薄灰グラデ、`a.blue/.green/.red/.yellow` の border-top:5px = `#2384dc/#4cbb47/#dd1d25/#ffd242` |
| `#breadcrumb a` | `#0044bb; 108%; bold; underline; padding-left:10px; background:url(bc_arr.png) 0 50% no-repeat` |
| `.Title` | 未使用の可能性大。`border-top:2px solid #f66013; background:url(../../img/bg_h2.gif)` |
| `.NOPRINT` | `@media print` で `visibility:hidden` |

`NINI`/`NWAKU` は入力フォーム(スケジュール検索)用。実例: スケジュール画面 `<td class='NINI CM BOLD' width='40px'>日付</td>` と `<select class="NWAKU WMAX">`。

### 2.3 背景の違い
| ページ | body 背景 |
|---|---|
| /sctnmt(大会一覧) | `black`(黒ベタ)。白い `#main`(710px)が黒の上に浮く |
| 大会ページ全般(scview, scv_top, team_list, tgallery, search_tn/div, division) | `white url(/img/ScoreView/def_bgimg.png) repeat top`(360x245 の薄いパターン、タイル) |
| エラーページ | 別の素朴な CI 標準エラー(下記 7) — 外枠なし |
大会ごとに背景が違うか: 今回確認した全 20 大会のインライン style は全て def_bgimg(例外なし)。大会別カスタム背景の機構(`body{background}` を DB で差し替え)は存在し得るが未確認(推測)。

### 2.4 使用画像一覧(すべて取得済み → `$S/raw/A/img/`)
| URL | 実寸 | 用途 |
|---|---|---|
| /css/scv_def/img/2012-logo.gif | 168x80 | ロゴ |
| /css/scv_def/img/icon-facebook.jpg | 32x32 | SNS |
| /css/scv_def/img/icon-twitter.jpg | 446x430(表示32) | SNS |
| /css/scv_def/img/icon-youtube.jpg | 32x32 | SNS |
| /css/scv_def/img/icon-flickr.jpg | 32x32 | SNS |
| /css/scv_def/img/icon-instagram.png | 32x32 | SNS |
| /css/scv_def/img/09_header_bar.gif | 1x11 | header_bar上部の背景 |
| /css/scv_def/img/spacer.gif | 1x1 | スペーサー(1x11, 900x14) |
| /css/scv_def/img/09_header_menu.jpg | 732x64 | #navigation スプライト |
| /css/scv_def/img/bg-header-white.jpg | 1000x80 | #header 背景 |
| /css/scv_def/img/09_ti_bg.png | 695x70 | .text_title 背景 |
| /css/scv_def/img/09_menu_outline_top.jpg / _bottom.jpg | 199x9 | サイドバー枠上下 |
| /css/scv_def/img/09_menu_outline_left.jpg / _right.jpg | 4x1 | サイドバー枠左右 |
| /css/scv_def/navi_ard.png | 11x6 | ナビ第1階層 ▼ |
| /css/scv_def/navi_arr.png | 6x11 | ナビ第2階層 ▶ |
| /css/scv_def/bc_arr.png | 4x7 | パンくず矢印(担当ページでは未使用) |
| /img/ScoreView/def_bgimg.png | 360x245 | 大会ページ body 背景 |
| /img/ClubJr/logo_scv_default.png | 701x96 | ロゴ未設定大会(151)の既定ロゴ |
| /js/colorbox/images/{border.png, controls.png, loading.gif, loading_background.png, overlay.png} | - | ギャラリーのライトボックス |
| /js/colorbox/colorbox.css, jquery.colorbox-min.js | v1.3.20.1 (2012 Jack Moore) | 同上 |
未取得(参照されるがスコア画面では実使用されない、またはIE6用): `/img/bg_h2.gif, bg_h3.gif, bg_sub_h2.gif, bg_list.gif, bg_pagetop.gif, bg_footmenu.gif, bg_dotline.gif, bg_menu.gif, env_*.png, nav_*_darker.jpg`、colorbox の `images/ie6/*`。
大会ロゴ: `/zfiles/ScoreViewLogo/{ファイル}`(525=atami_2025.png, 539=DreamCup.jpg, 528=GaiaUltimate.jpg, 523=gamagori_2025.png, 172=2015ung-banner.jpg など)。サイズは大会ごとに異なり `width/height` 属性なし(原寸表示、`alt="ロゴ"`)。
自由記述HTML内画像: `/zfiles/ScoreViewTop/…`(525: event_2026atami_photo_B_1st.jpg 他, 600x400指定)。

### 2.5 common.js / 他JS
- `common.js` は汎用(フォーム送信 `Action()/ActionP1..3/ActionG1/ActionPG1`、`FormLocker`(ボタンを3秒無効化)、日付ピッカー、画像アップロード等)。**閲覧ページで実際に使われるのは スケジュール検索の `Action('search')` 系と `MyForm = document.forms[0]` 設定のみ**。大会トップ・チーム一覧・一覧ページではフォーム自体がなく実質未使用(`$(function(){ MyForm = document.forms[0]; })` は null になるだけ)。`#codeigniter_profiler` がある場合のみプロファイラUIを追加(本番では無い)。
- 大会トップ・チーム一覧・一覧ページは **JSなしの純HTML + onclick遷移**。必要なJS挙動は (1) `$('#score_navi').dropdown()` (2) footerFixed (3) ギャラリーの colorbox のみ。

---------------------------------------------------------------------
## 3. 大会一覧 /sctnmt (= /pscore/sct_all)

構造:
```html
<div id="score_view">
 <div style='clear:both; height:10px'></div>
 <div style="margin:2px; padding:1px; border-color:#cccccc;border-width:0px;border-style:solid; ">
   <div class='TeamListMIDA'>{区分見出し}</div>
 </div>
 <!-- 大会ごと -->
 <div style="margin-left:14px;margin-top:10px;">
   <div class='TeamListBLOCK' style="margin:2px 0px 2px 0px;  width:98%; ">
     <div style='padding:4px; color: white;' class='px12 FTitle  '>
       <span class='teamimg_rnk'><a href="#" class="" onclick="document.location = 'https://score.clubjr.com/scview/{ID}?lang=jp'; return false;">{大会名}</a></span>
     </div>
   </div>
 </div>
 …
</div>
```
- 1大会=横幅いっぱい(98%)の青グラデ(`.FTitle`)バー1行。白文字(`!important`)、12px(px12)、`a`にホバー下線(`#main a:hover`)。バー間は margin-top:10px。左 14px インデント。
- `.TeamListMIDA`(橙 #ffa000 見出し)が区分見出し。**区分は 3 種のみ**(全348件で他に無い): `エントリー受付中` (12件) / `エントリー締切り` (2件) / `大会終了` (334件)。この順で縦に並ぶ。
- 英語表示(`?lang=en`): **区分見出しは日本語のまま**(翻訳されない)。大会名だけ英語名に差し替わり(例 534 `2027 初投げCUP 2部` → `2027 Hatsunage 2nd`)、**英語名が未登録の大会は `<a>` の中身が空**(リンク自体は出るが文字なし。47件: 537, 536, 495, 489, 488, 445, 444, 443, 406, 391, 392, 393, 359, 344, 347, 326, 321, 322, 317, 315, 312, 298, 291, 276, 259, 237, 226, 210, 198, 200, 197, 194, 192, 191, 190, 189, 179, 181, 177, 176, 175, 172, 165, 162, 160, 158, 151)。リンク先の `?lang=en` は付く。
- 件数: 348件(ID重複なし)。ID範囲 151〜539。欠番(一覧に出ない)41個: 152〜157, 159, 161, 163, 164, 166〜171, 173, 193, 215, 220, 221, 223, 234, 236, 254, 257, 262, 273, 282, 296, 297, 308, 309, 332, 401, 405, 442, 507, 515, 516, 538。**欠番のうち 538 と 507 は実在する(HTTP200で大会トップが表示される = 一覧に載せない「非公開/未掲載」)。152, 1, 0 は存在しない(500)**。つまり一覧に出ない=存在しない、ではない。
- 並び順: **ID順ではない**。区分(受付中 → 締切り → 終了)内で **開催日の新しい順(降順)** と推定(推測だが根拠は強い)。
  - 根拠(各大会のスケジュール日付 `scv_search_tn` の最初の日 / 一覧上の順): 527=2026-11-03, 525=2026-10-18(写真EXIF日時 2026-10-18; スケジュール選択肢の表示は 2025/10/18 と年が誤入力)、524=2026-10-17, 522=2026-10-10, 505=2026-10-05, 513=2026-09-28, 504=2026-09-07, 526=2026-08-30, 512=2026-08-24, 521=2026-07-18, 520=2026-07-11, 502=2026-07-06, 500=2026-05-11, 494=2026-04-29 — 一覧上の順(527, 525, 524, 522, 505, 513, 504, 526, 512, 521, 520, 502, 500, 494)と完全に日付降順で一致。一方 ID の隣接逆転(次が大きい)は終了区分で117箇所(526 が 504 の後ろ、など)。
  - 受付中の並び: 539 535 534 533 532 531 530 529 537 536 506 514(537/536 は ID が大きいが開催日が早い=ディスクゴルフ、506/514 は 4on4 シリーズ)。日付降順と整合するが全件の日付は未確認。
  - 厳密には「大会に紐づく日付フィールド(開始日?)の降順」。開催日が未入力の大会がどこに入るかは不明。
  - ID のみで再現する必要がある場合は CSV の `pos` 列(取得時の並び順)をそのまま使うこと。
- 区分の判定規則は不明(推測: 大会ごとの「状態」フラグ。DBの手動設定)。528(締切り)の日付は未入力(スケジュール未公開)、539(受付中)も 2027 年表記。

CSV: `$S/A_tournaments.csv`(348行、jp名・en名・区分・pos)。

---------------------------------------------------------------------
## 4. 大会トップ /scview/{ID} = /pscore/scv_top/open/{ID}

- 両URLは同一内容(完全一致)。リダイレクトなし。
- `div#score_view` の中身:
```html
<div class="text_title">{jp大会名}</div>                    ← lang非依存
<div style='text-align: center; margin: 10px 10px 15px 10px'>
	<img src="/zfiles/ScoreViewLogo/xxx.png" alt="ロゴ" />   ← ロゴ(無い時は /img/ClubJr/logo_scv_default.png を絶対URLで)
</div>
<div class="TBL"  style='margin-left: 80px; margin-right: 100px;'>
	<table style='width: 100%;'>
		<tr class=''>
			<td class='MD1 CM' width= '40%' >開催ディビジョン</td>     ← en: Division
			<td class='MD1 CM' width= '40%' >大会情報</td>             ← en: Information
		</tr>
		<!-- ディビジョンごと -->
		<tr>
			<td class=''  >&nbsp;{ディビジョン名}</td>
			<td class='CM' >
				<a href="#" class="Select_Link" onclick="document.location = '…/pscore/scv_search_div/open/{DIV}?lang=jp'; return false;">[スケジュール]</a>&nbsp;&nbsp;&nbsp;
				<a href="#" class="Select_Link" onclick="document.location = '…/pscore/scv_division/open/{DIV}?lang=jp'; return false;">[最終順位]</a>
			</td>
		</tr>
	</table>
</div>
<br /><hr /><br />
<div id="SCV_TopPage" style="margin-left: 10px">  {自由記述HTML(CMSのWYSIWYG出力)}  </div>
```
- ロゴ画像: 大会ごとに `/zfiles/ScoreViewLogo/…`(DB登録)。未登録(151)は `https://score.clubjr.com/img/ClubJr/logo_scv_default.png`(絶対URL。ここだけ絶対、他は相対)。幅/高さ指定なし(原寸)。
- 表は幅 100% だが `.TBL`(margin 0 10px)+インライン margin-left:80px/right:100px → 表幅は 710-… ≒ 710−(10+80)−(10+100)。
- 列幅 40%/40%(合計80%。table-layout:fixed なので右20%は空白で、2列が80%に収まる)。
- lang=en: `Division`/`Information`、`[Schedule]`/`[Standings]`、ディビジョン名は英語名(Beach Mixed)。**`[最終順位]` ↔ `[Standings]`**。ナビの語は別(上記表)。
- 条件分岐(`[スケジュール]`/`[最終順位]` の出方、実測):

| 状態 | 例 | `[スケジュール]` | `[最終順位]` |
|---|---|---|---|
| スケジュールあり+最終順位あり(終了) | 525(949), 524, 522, 521, 505, 494, 300, 250, 151, 172 | リンク(`Select_Link`) | リンク(`Select_Link`) |
| スケジュールあり+最終順位なし(受付中/締切り/未確定) | 539(3ディビジョン全て), 538(非公開), 528, 527, 507 | リンク | `<span class='Dummy_Link'>[最終順位]</span>`(灰色#ccc、リンクなし) |
| スケジュールなし+最終順位なし | 523(フリースタイル), 537/536(ディスクゴルフ) | `<span class='Dummy_Link'>[スケジュール]</span>&nbsp;&nbsp;&nbsp;` | `<span class='Dummy_Link'>[最終順位]</span>` |
  マークアップの違い: リンク行は `<a …>[スケジュール]</a>&nbsp;&nbsp;&nbsp;` の後に最終順位 `<a>`。ダミー時は span。
  判定の根拠(推測): スケジュール=ディビジョンに試合が1件以上登録、最終順位=順位確定フラグ。大会の区分(受付中/終了)とは独立 — 大会終了でも 523/537 はダミー、締切り/受付中でも539等のスケジュールはリンク。

- 複数ディビジョンの並び: DBの順序(ID昇順ではない): 539 は ミックス(971)→オープン(974)→ウィメン(973)、151 は メン(338)→ミックス(340)→ウィメン(339)。ナビのサブメニューも同順。ディビジョン名に全角英数字を含む場合がある(526 `Ｕ－１２`、537 `ＤＧシングルス（土曜日）`)。
- 自由記述 `#SCV_TopPage`: 管理画面のリッチテキストをそのまま出力(インラインstyle満載: `font-family: &quot;ヒラギノ角ゴ Pro W3&quot;…`、橙バッジ `span.SCV_MIDA_M` + `background-color: rgb(255,102,0)`、`<img style="width:600px;height:400px">` 等)。525 の構成: 【Web Site】バッジ+URL → 【Result】バッジ+「■1st place：マンマユート」+写真…(2nd,3rd,MVP,Fighting-spirit award)。空の大会は `<div id="SCV_TopPage" style="margin-left: 10px">\n\t</div>`(507, 537, 534 の en 等)。
  - **自由記述は言語別フィールド**: 525 は jp/en 同一(英語併記で書かれている)だが、172 は lang=en にすると中身が空になる(日本語のみ入力)。つまり `SCV_TopPage` は jp用 と en用 の2フィールド。en が空なら空 div。
- 状態を比較(今回取得した大会):

| 大会 | 区分 | ディビジョン | 最終順位 | ギャラリー | #sub バナー | ロゴ | 備考 |
|---|---|---|---|---|---|---|---|
| 539 DREAM CUP | 受付中 | 3(ミックス/オープン/ウィメン) | ダミー×3 | 12 (Day1-AM〜Day4-PM-13-16) | 30枚 | DreamCup.jpg | チーム一覧は 3/1/0 teams |
| 535 Boon Young Star Cup | 受付中 | (未取得) | - | - | - | - | 一覧のみ確認 |
| 538 (非公開) | 一覧外 | 2 | ダミー | あり | 13枚 | - | 一覧未掲載でも到達可 |
| 528 GAIA ULTIMATE CUP | 締切り | 1(ミックス) | ダミー | 8 | 25枚 | GaiaUltimate.jpg | スケジュール日付未登録 |
| 527 Fuji 4on4 Vol.14 | 締切り | 1(４人制部門) | ダミー | あり | あり | F4 バナー.jpg | 日付 2026-11-03 |
| 525 Atami Beach Line | 終了 | 1(ビーチミックス) | リンク | 2(Day2-AM/PM) | 18枚 | atami_2025.png | 自由記述あり(Result+写真) |
| 523 GAMAGORI FREESTYLE | 終了 | 1 | ダミー(スケジュール無し) | なし | あり | gamagori_2025.png | ナビ縮退 |
| 537 DG 美浦 | 受付中 | 2(ＤＧシングルス土/日) | 全ダミー | なし | 空 | Discgolf_cjcup.jpg | ナビ「エントリー一覧」「成績」 |
| 526 Kunugidaira | 終了 | 2(オープン/Ｕ－１２) | リンク | あり | - | 2024 kunugidaira.jpg | |
| 521 GREATEST CUP CHUBU | 終了 | 2(オープン/ウィメン) | リンク | あり | - | GREATEST CUP 2025(600×213).png | |
| 300 | 終了 | 1(４人制部門) | リンク | なし | - | F4 バナー.jpg | |
| 172 2017 UNG 岡崎 | 終了 | 2(メン/ウィメン) | リンク | なし(フィラーli) | 空枠 | 2015ung-banner.jpg | 自由記述 jp のみ |
| 151 2017 DREAM CUP | 終了 | 3(メン/ミックス/ウィメン) | リンク | なし | 空枠 | logo_scv_default.png(既定) | 自由記述あり(短) |

---------------------------------------------------------------------
## 5. チーム一覧 /pscore/scv_team_list/open/{大会ID}

- 標準の大会外枠(ナビ・サイドバー付き)。`MyController=…/scv_team_list`。title `[CJS] {大会名} `。
- `div#score_view` 内: ディビジョンごとに「見出し」+「チーム群」を繰り返す:
```html
<div style="margin:2px; padding:1px; border-color:#cccccc;border-width:0px;border-style:solid; text-align:center;">
  <div class='TeamListMIDA'>ビーチミックス&nbsp;&nbsp;&nbsp;--&nbsp;32 Teams&nbsp;--&nbsp;&nbsp;&nbsp;</div>
</div>
<div style="margin-left:14px;margin-top:10px;">
   <!-- チームごとに2ブロック(float:left)を繰り返す -->
   <div class='TeamListBLOCK'   style="margin:2px 0px 2px 0px;  width:187px; ">
     <div style='padding:4px; color: white;' class='px12 FTitle_Mix  '>
        <span class='teamimg_rnk'><a href="#" class="" onclick="document.location = '…/pscore/scv_team/open/{チームID}?lang=jp'; return false;">{チーム名}</a></span>
     </div>
   </div>
   <div class='TeamListBLOCK' style="margin:2px 8px 2px 0px;  width:30px">
     <div style='padding:4px; color: white;' class='px12 AreaSepa RM'>
        x11
     </div>
   </div>
   …
</div>
<div style='clear:both; height:10px'></div>
```
- 見出し文言: `{ディビジョン名}   -- {N} Teams --`(**"Teams" は jp でも英語固定**。lang=en では名前だけ英語名に)。`&nbsp;` 3個+`--&nbsp;`N` Teams`&nbsp;--&nbsp;&nbsp;&nbsp;`。ディスクゴルフ(537)は `-- 6 Players --` で、リンクなし(`<span class='teamimg_rnk'>(選手名)</span>`)、人数セルは `&nbsp;`。単位の Players はナビが「エントリー一覧」の大会で全ディビジョンに付き、237(ビーチ、中止)ではチーム名がリンクで人数セルもあるチーム戦なのに `-- 22 Players --`。
- チーム1件 = 187px の色バー(チーム名リンク、白文字、12px)+ 30px の黒バー(右寄せ、`x{人数}` = 登録選手数と推定、539 のように選手未登録だと `x0`)。1行あたり 187+2+30+8+2(各1px枠) ≒ 229px で、#main 内側(710−14 margin = 696) に **3組/行**(float:left で折り返し)。
- バーの色(`class='px12 FTitle*'`)は **チームの属する部門種別**: メン/オープン等=`FTitle`(青)、ミックス=`FTitle_Mix`(紫)、ウィメン=`FTitle_Women`(ピンク)、大学など特定チーム=`FTitle_Univ`(緑。151 のメンで "2018U-24日本代表候補メン部門" の1チームだけ — ディビジョンではなくチーム属性、規則は推測)。ディビジョン名ではなく種別で決まる(539 のオープンは `FTitle`、525 のビーチミックスは `FTitle_Mix`)。
- チームの並び: 525 は ID 順ではない(17491, 17409, 17400, 17474, 17456…)。172/539/151ミックス/ウィメンは ID 昇順。順序規則は不明(推測: ディビジョン内の登録順 / エントリー番号)。
- ディビジョンの並び: 大会トップの表と同順(DB順)。選手0チームのディビジョンも見出し(`-- 0 Teams --`)と空の `<div style="margin-left:14px;margin-top:10px;"></div>` が出る(539 ウィメン)。
- リンク先: `/pscore/scv_team/open/{チームID}?lang=jp|en`。lang=en ではチーム名が英語名(雨ニモマケズ → AmenimoMakezu, いーあーる → ER)。
- 件数の例: 525=32(ビーチミックス), 151=72/32/48(計152), 172=14/9(計23), 539=3/1/0。
- 5項目ナビのラベルが標準大会と違う大会がある(537: エントリー一覧)。

---------------------------------------------------------------------
## 6. ギャラリー /pscore/scv_tgallery/open/{ギャラリーID}

- ギャラリーIDは大会IDとは別体系(525 → 362 (Day2-AM), 363 (Day2-PM)、539 → 407〜414)。ナビの「ギャラリー」メニューからのみ入る。
- 標準の大会外枠(ナビ付き)+ 本文:
```html
<div id="score_view">
<link rel="stylesheet" type="text/css" href="https://score.clubjr.com/js/colorbox/colorbox.css">
<script type="text/javascript" src="https://score.clubjr.com/js/colorbox/jquery.colorbox-min.js"></script>
<script>
$(document).ready(function(){ $("#photo a").colorbox(); });
</script>
<span class='SCV_MIDA_M'>&lt; Day2-AM &gt;</span><br />
<div class='CM' id='photo'>
<a  href="/zfiles/TGalleryImg/2026 Atami-B/Day2-AM/event_2026atami_photo_day2_am_00001.jpg" rel="photos"><img src="/zfiles/TGalleryImg/2026 Atami-B/Day2-AM/mini/event_2026atami_photo_day2_am_00001.jpg" border="0"  style="margin: 2px;" /></a><a …(連続・間に改行/空白なし)…
</div>
```
- タイトル: `<span class='SCV_MIDA_M'>&lt; {ギャラリー名} &gt;</span>`(橙バッジ。例 `< Day2-AM >`)。lang 非依存。
- 写真グリッド: `#photo`(text-align:center)に `<a><img></a>` が連続して並ぶ → インライン要素として折り返す(サムネ 120x80 + margin 2px)。**ページングなし**(362 は189枚が1ページ、サーバがディレクトリを全列挙)。
- 画像URL: サムネ `/zfiles/TGalleryImg/{大会フォルダ}/{セッション}/mini/{ファイル名}.jpg`(120x80、gd-jpeg 生成)、原寸 `/zfiles/TGalleryImg/{大会フォルダ}/{セッション}/{ファイル名}.jpg`(945x630, 162KB)。フォルダ名にスペース・日本語の可能性があり、HTML内の `href`/`src` は **生のスペースのまま**(例 `2026 Atami-B`。エンコード済みはリクエスト時 `%20`)。ファイル名は連番5桁 `_00001.jpg`〜。
- ライトボックス: `rel="photos"` で colorbox v1.3.20.1 のグループ化、既定オプション(transition elastic, speed 300ms, opacity .9, 文言 "image {current} of {total}" / previous / next / close、slideshow off)。lang=en でも文言は変わらない(colorbox既定の英語のまま。jp でも英語表示)。
- lang=en: ナビの文言のみ変化(Schedule…)。ギャラリーの本文は同一(65958 B vs 65862 B の差はナビ分)。
- `MyController=…/scv_tgallery`。
- `body` 背景は他の大会ページと同じ(def_bgimg)。

---------------------------------------------------------------------
## 7. エラー / 存在しないID

| URL | HTTP | 内容 |
|---|---|---|
| /scview/999999 | **500** | CodeIgniter 標準エラー画面 |
| /pscore/scv_top/open/999999 | 500 | 同(同一メッセージ) |
| /pscore/scv_team_list/open/999999 | 500 | 同(`TOURNAMENT_ID`) |
| /pscore/scv_tgallery/open/999999 | 500 | 同(`TGALLERY_ID`) |
| /scview/152, /scview/1, /scview/0 | 500 | 同(存在しないID/削除済み) |
| /scview/abc | **404** | CI 標準 `404 Page Not Found` / `The page you requested was not found.`(IDが数値でない=ルートに合致しない) |

500 ページ全文の骨格(外枠なし、サイトのCSS/ロゴも無し):
```html
<!DOCTYPE html><html lang="en"><head><title>Error</title><style>…(CI標準 body{margin:40px; font:13px/20px normal Helvetica,Arial,sans-serif; color:#4F5155} h1{font-size:19px;border-bottom:1px solid #D0D0D0} #container{margin:10px;border:1px solid #D0D0D0;box-shadow:0 0 8px #D0D0D0} p{margin:12px 15px} …)</style></head>
<body><div id="container">
 <h1>An Error Was Encountered</h1>
 <p>TOURNAMENT_ID [999999]での操作に失敗しました。<br />情報が既に削除されているか、システムエラーの可能性があります。<br />メニュー画面より、再操作し確認して下さい。<br /><br /><a href="https://score.clubjr.com/auth/login">←メニュー画面に戻る</a></p>
</div></body></html>
```
- 大会トップ/チーム一覧: 上記 `TOURNAMENT_ID [{id}]`、ギャラリー: `TGALLERY_ID [{id}]` で、「メニュー画面より、再操作…」の文と戻るリンクは大会側のみ(ギャラリーは `…可能性があります。<br />` で終わる)。
- `lang` を付けても同じ(日本語固定)。戻るリンクは管理画面ログイン `/auth/login`。

---------------------------------------------------------------------
## 8. 実装メモ(模倣側向け)

1. 外枠(ヘッダ・フッタ・ナビCSS)は全ページ共通なので共通コンポーネント化。`ul#score_navi` はデータ駆動: {divisions[], groups[div][], galleries[]} から生成。リンクは全て lang 引き継ぎ。
2. 「同一ID入れ子 `#header`」「`style=''` 空属性」「`</li>` 直前のタブ/空白」などは見た目に影響しないが、完全再現するなら維持。
3. 見た目に効く落とし穴: `.text_title` のはみ出し(747px)、`#main a` 詳細度による `.Select_Link` の色、`html{overflow-y:scroll}`、`.TeamListBLOCK{white-space:nowrap; overflow:hidden}`、フッタは `footerFixed` で最下部固定。
4. /sctnmt の `lang=en` は 区分見出し=日本語 / 大会名のみ英語(空なら空リンク)。
5. 非公開大会(538 など)は一覧に載らず URL 直打ちで到達可能。実在判定=DB に行があるか。

取得ファイル一覧(`$S/raw/A/`): normalize.css, score_fonts.css, score_frame.css, score_app.css, score_navi.css, common.js, jquery.dropdown.js, footerFixed.js, colorbox.css, jquery.colorbox-min.js, sctnmt_nolang.html, sctnmt_en.html, scview_{0,1,151,152,172,172_en,250,300,494,505,507,521,522,523,524,525*,526,527,528,534_en,537,537_en,538,539,abc,999999}.html, scvtop_525_jp.html, tl_{525,525_en,151,172,537,539,999999}.html, tg_{362,362_en,999999}.html, top_999999.html, stn_{…}.html, g_mini1.jpg, g_full1.jpg, img/*, img/colorbox/*。
