# B_spec: スケジュール(試合日程・結果一覧)ページ仕様

調査日 2026-10-21。リクエスト総数 94(上限150)。生データは `$S/raw/B/`(tn{大会ID}.html=scv_search_tn GET、en*.html=lang=en、p*.html/s*.html/d*.html=POST結果、common.js、score_app.css 等)。
`[事実]`=実HTMLで確認、`[推測]`=確認不能。

## 0. 調査した大会(18大会以上を解析、うち主要)
| ID | 内容 | 確認できたこと |
|---|---|---|
| 525 / 524 / 522 / 476 | ビーチ・単一ディビジョン(ビーチミックス, 1日) | 66行全て MC_END MC_MIX |
| 440 | 2025 DREAM CUP オープン+ウィメン(2日, 282行) | 未試合+プレースホルダ、MC_END/MC_MAE混在、ソート・POST全種 |
| 452 | DISCRAFT 2025【途中中止】 | 全135行 MC_MAE+未試合+プレースホルダ(中止でも「未試合」のまま) |
| 527 | Fuji 4on4 Vol.14 (2026/11/03, 未来, 受付締切り) | 15行 全て MC_MAE、スコア空欄、状況=「未試合」(プレースホルダ無し) |
| 539,535,534,533,532,531,530,529,528,514,506,523 | 受付中/締切りの未来大会 | 試合0件(ヘッダ行のみ。5節参照) |
| 537, 536, 489, 392 | ディスクゴルフ | 試合0件、ナビにスケジュール項目なし、team選択肢は選手名 |
| 151 (2017), 160, 205 | 旧大会(メン/ウィメン/ミックス) | MC_MEN が出る唯一の例、状況=「終了」が大半 |
| 360, 486 | 3〜4日開催 | ymd選択肢3〜4、既定は最終日 |
| 479,478,455,480,493,446,408,400,373,331,313,289,226,188 ほか | 新旧各種 | 状況文言・クラス全種の網羅用 |
| 249 / 275 / 452 | 【開催中止】/途中中止 | 249,275は行0。タイトルに「【開催中止】」がそのまま入る |

## 1. /js/common.js(全文確認済み: raw/B/common.js)
スケジュールに関係する部分(他は未使用のユーティリティ):
```js
var MyForm = null;
$(function(){ ... MyForm = document.forms[0]; });   // ページ内の最初のform(スケジュールページはformが1つだけ)
function Action(func) {
  MyForm.action = MyController + "/" + func;   // MyController は各ページのインラインJSで定義
  FormLocker();                                  // form内の :button をdisabledにし3秒後に解除(スケジュールページには <button> が無いので実質無効)
  MyForm.submit();                               // method=post, accept-charset=utf-8
}
function AddHiddenVal(name, value) {             // <input type=HIDDEN name=.. value=..> を MyForm に append
  obj = document.createElement("INPUT"); obj.setAttribute("type","HIDDEN"); obj.setAttribute("name",name); obj.setAttribute("value",value); MyForm.appendChild(obj);
}
```
- ActionP1/P2/P3(func,p1,p2,p3): `MyController/func/p1/p2/p3` にPOST(空文字はスキップ)。ActionG1/ActionPG1: `?param=value` を付けてPOST。スケジュールページでは **未使用**。
- ページ内インライン: `var MyController = "https://score.clubjr.com/pscore/scv_search_tn";` (大会全体) / `".../pscore/scv_search_div"` (ディビジョン別)。`$('#score_navi').dropdown();`
- codeigniter profiler 部分は `#codeigniter_profiler` が無いので不発。

### 動作
| 操作 | JS | POST先 | POSTボディ(application/x-www-form-urlencoded, UTF-8) |
|---|---|---|---|
| ymd/team/field の select 変更(onchange) | `Action('search')` | `{MyController}/search` | tn: `tid=525&lang=jp&ymd=20251018&team=&field=` / div: `ediv_id=949&lang=jp&ymd=&team=&field=`(select はフォーム順 ymd, team, field。空選択は空文字で送られる) |
| 見出しリンククリック | `AddHiddenVal('sortkey1','ASC_mat.FIELD_NO'); Action("sort"); return false;` | `{MyController}/sort` | 上記全フィールド + `sortkey1=ASC_mat.FIELD_NO`(末尾に追加) |
| 更新アイコン(Refresh_OK) | `Action('search'); return false;` | `{MyController}/search` | フィルタ現在値を送って再描画 |

- レスポンスは POST への直接 200(リダイレクト無し、PRGなし)。URLは `/pscore/scv_search_tn/search` や `/sort` のまま。クローンは同じ挙動(POSTで同一テンプレート再描画)で良い。
- `lang` は hidden で毎回渡る。POST後のリンクは `?lang={hidden lang}` で生成される。GETで lang 省略時は jp(`nolang525.html` で確認、hidden lang=jp)。

## 2. ソート(sortkey1)の仕様 [事実]
ヘッダ列とキー(JP/EN):
| 列 | ラベル(JP / EN) | 初期の sortkey1 値 | 備考 |
|---|---|---|---|
| 1 | ディビジョン / Division | `ASC_mdv.DIVISION_SORTKEY` | |
| 2 | 日付 / Date | `ASC_mat.MATCH_DATE` | |
| 3 | 時間 / Time | `ASC_mat.START_TIME` | |
| 4 | コート / Field | `ASC_mat.FIELD_NO` | |
| 5 | CD / CD | `ASC_mat.MATCH_CD` | |
| 6 | チーム名 / Team (左) | `ASC_lt_name` | |
| 7-9 | `-` `VS` `-` | リンクなし | |
| 10 | チーム名 / Team (右) | `ASC_rt_name` | |
| 11 | 状況 / Status | リンクなし | |

**ASC/DESC トグルはサーバー側で実装されている**(JSには無い)。ソート後のレスポンスでは、現在のソート列の `<a>` だけが次のように変わる:
- ASC でソート中: `class="MD1_ASC"`、ラベルに `▼` 付加(例 `コート▼` / `Time▼`)、onclick の値が `DSC_mat.FIELD_NO` に変わる。**接頭辞は `DESC_` ではなく `DSC_`**。
- DSC でソート中: `class="MD1_DSC"`、ラベルに `▲` 付加(`コート▲`)、onclick の値は `ASC_mat.FIELD_NO` に戻る。
- 色: `.TBL A.MD1 {color:#fff !important; underline}` / `.MD1_ASC {color:#00ff00 !important}` / `.MD1_DSC {color:#00ffff !important}`。
- 実際の記録: `ASC_mat.FIELD_NO` → `<a href="#" class="MD1_ASC" onclick="AddHiddenVal('sortkey1', 'DSC_mat.FIELD_NO') ;  Action(&quot;sort&quot;)  ; return false; ">コート▼</a>`
- `sortkey1` はソート応答のフォームに hidden として **残らない**(hidden は tid/ediv_id と lang のみ)。よって以後 select 変更(search)をするとソートは既定順に戻る。ymd/team/field の選択状態はソート応答でも保持され、フィルタも継続適用。
- 同一キー内の並び: 副キーは (日付, 時間, コート) の既定順の安定ソート(コートDSCでも同コート内は時間昇順、`s440_fielddsc.html`)。
- チーム名ソート(`ASC_lt_name`)は大文字小文字無視(agora < Arcobaleno < ARROWS < …、日本語は後ろ)。左チーム(lt_name)でのみ並ぶ。プレースホルダ文字列も対象(位置は未確認)[推測: 通常は先頭側]。
- ディビジョンソート(`ASC_mdv.DIVISION_SORTKEY`): オープン→ウィメンの順(DB上のソートキー順、ナビの並びと一致)。

## 3. 既定の並び・フィルタの既定値 [事実]
- 既定順: 日付 → 時間 → コート番号(数値昇順)。ディビジョンは混在(440: オープン10:00のコート1-10、ウィメン10:00のコート11-… がコート順で並ぶ)。
- **scv_search_tn (全ディビジョン)**: GET時 `ymd` の既定は **最後の日付**(440→0316、486の4日開催→20260308、360の3日→最終日)にselected、その日の行のみ表示。ymd選択肢は `['', 日付昇順...]`(空が先頭)。空選択を送ると全日程(440: 282行)。
- **scv_search_div (ディビジョン別)**: GET時 `ymd` は空(selected)= **全日程**(d822: 158行)。ここがtnとの最大差。
- 1日のみの大会は tn でもその日付が selected、先頭に空選択肢あり(525: `<option value=""></option><option value="20251018" selected>`)。ymdが0件の大会は空選択肢のみ。
- 日付形式: option value `YYYYMMDD`、label `YYYY/MM/DD`(en も同じ)。表の日付列は `MM/DD`。
- フィルタは AND(ymd+field で 440 の0316・コート5 → 6行 を確認)。ymd+team(div版: ymd=20250315, team=15634 → 2行)。team はそのチームが **左右どちらでも**該当(Bustar は右側に居る試合も返った、`p440_bustar_en.html`)。
- 該当0件でもテーブルはヘッダ行のみ(メッセージ無し)。

## 4. scv_search_tn と scv_search_div の差 [事実]
| | scv_search_tn/open/{大会ID} | scv_search_div/open/{ディビジョンID} |
|---|---|---|
| title | `[CJS] {大会名} -全スケジュール-` (en: `-All Schedule-`) | `[CJS] {大会名} -ディビジョンスケジュール-` (en: `-Division Schedule-`) |
| MyController | `.../pscore/scv_search_tn` | `.../pscore/scv_search_div` |
| hidden | `tid`, `lang` | `ediv_id`, `lang` |
| ymd 既定 | 最終日 selected | 空(全日程) |
| team の option value | **チーム名そのもの**(`value="雨ニモマケズ"`, label も同じ全名) | **チームID**(`value="17491"`, label=全名) |
| team の option 順 | 大会チーム全体(試合の有無に関わらず。539は試合0件でも4チーム) 順序は非ふりがな順で規則不明 → データそのまま | 対象ディビジョンのチームのみ。ふりがな順っぽい(雨ニモマケズ, いーあーる, EPC, インサイド・スロー, VIVANZ, エイヒレ…) |
| 行 | 全ディビジョン | 当該ディビジョンのみ(`ウィメン`ページは行も `ウィメン` のみ) |
| ディビジョン選択UI | **無し**(select は ymd/team/field の3つだけ、複数ディビジョン大会でも同じ) | 無し |
| 表・列・ソートキー | 同一(「ディビジョン」列は常に出力) | 同一 |
- field の option: 空 + `1..N`(N=その大会(tn)/ディビジョン(div)の最大コート番号。440 tn で 1..22)。試合0件の大会は空のみ。
- ナビ「スケジュール」ドロップダウン: `全ディビジョン`→scv_search_tn、各ディビジョン名→scv_search_div。ナビ順は DIVISION_SORTKEY 順(486: ミックス895, オープン893, ウィメン894)。ディスクゴルフ大会(537,536)はナビに「スケジュール」項目自体が無い(「エントリー一覧 成績 …」)が、URLを直接叩くと空の表を返す(200)。
- 存在しない大会/ディビジョンID(120, 60, 99999)→ **HTTP 500**。

## 5. ページ HTML 構造(ほぼ verbatim)
`#score_view` 直下(見出し・タイトルの `<h*>` は無い。大会名は `<title>` のみ):
```html
<script type="text/javascript">
$(function() {
	$("#Refresh_NG").show(500);
	setTimeout( function() {
		$("#Refresh_OK").show(800);
		$("#Refresh_NG").hide(800);
	}, 5000);
});
</script>

<div class="ITBL">
	<form action="" accept-charset="utf-8" method="post">		<input type="hidden" name="ediv_id" value="949" />		<input type="hidden" name="lang" value="jp" />
		<table>
			<tr>
				<td class='NINI CM BOLD'  width='40px'  >日付</td>
				<td class=''         width='120px' ><select name="ymd" class="NWAKU  WMAX"  id='ymd'   onchange="Action('search');"   >
<option value="" selected="selected"></option>
<option value="20251018">2025/10/18</option>
</select></td>
				<td class='NINI'     width='10px'  ></td>
				<td class='NINI CM BOLD'  width='60px'  >チーム名</td>
				<td class=''         width='180px' ><select name="team" class="NWAKU  WMAX"  id='team'    onchange="Action('search');"    >
<option value="" selected="selected"></option>
<option value="17491">雨ニモマケズ</option> ...
</select></td>
				<td class='NINI'     width='10px'  ></td>
				<td class='NINI CM BOLD'  width='60px'  >コート</td>
				<td class=''         width='100px' ><select name="field" class="NWAKU  WMAX"  id='field'    onchange="Action('search');"    >
<option value="" selected="selected"></option> <option value="1">1</option> ...
</select></td>
				<td class='NINI'     width='10px'  ></td>
				<td class='RM'       width='120px' style="height:33px"  >					<a id='Refresh_OK'  style='display: none;'  href="#"  onclick="Action('search'); return false;"
						><img src="https://score.clubjr.com/img/icons/refresh.png" alt="Reflesh_OK" /></a>
					<a id='Refresh_NG'  style='display: none;'  href="#"  onclick='alert("５秒お待ちください"); return false;'
						><img src="https://score.clubjr.com/img/icons/refresh_ng.png" alt="RefleshWait" /></a>
						&nbsp;
				</td>
			</tr>
		</table>
	</form></div>

<div class="TBL">
	<table>
		<tr class='' >
			<td class='MD1 CM' width= '13%' ><a href="#" class="MD1" onclick="AddHiddenVal('sortkey1', 'ASC_mdv.DIVISION_SORTKEY') ;  Action(&quot;sort&quot;)  ; return false; ">ディビジョン</a></td>
			<td class='MD1 CM' width= '7%' > 日付 / 時間 / コート / CD も同形式(各 7%) </td>
			<td class='MD1 RM' width= '17%'><a ... 'ASC_lt_name' ...>チーム名</a> </td>   <!-- 末尾に半角スペース -->
			<td class='MD1 CM' width= '4%' >-</td>
			<td class='MD1 CM' width= '3%' >VS</td>
			<td class='MD1 CM' width= '4%' >-</td>
			<td class='MD1 LM' width= '17%'><a ... 'ASC_rt_name' ...>チーム名</a></td>
			<td class='MD1 CM' width= '14%'>状況</td>
		</tr>
		<tr class='H24'>
			<td class='MC_END MC_MIX CM' >ビーチミッ</td>
			<td class='MC_END CM' >10/18</td>
			<td class='MC_END CM' >09:00 </td>          <!-- 時刻の後ろに半角スペースが1つ -->
			<td class='MC_END CM' >1</td>
			<td class='MC_END CM' >A4</td>
			<td class='MC_END RM' ><span class='teamLink'><a href="#" class="" onclick="document.location = 'https://score.clubjr.com/pscore/scv_team/open/17322?lang=jp'; return false;">とびたーず</a></span></td>
			<td class='MC_END CM' >2</td>
			<td class='MC_END CM' >x</td>
			<td class='MC_END CM' >12</td>
			<td class='MC_END LM' ><span class='teamLink'><a ...scv_team/open/17498?lang=jp...>バトルロイヤル</a></span></td>
			<td class='MC_END CM' >
				<a href="#" class="Select_Link" onclick="document.location = 'https://score.clubjr.com/pscore/scv_match/open/45280?lang=jp'; return false;">[対戦結果]</a>				</td>
		</tr>
		...
	</table>
</div>
```
- 列数は常に11。ヘッダ幅合計 = 13+7*4+17+4+3+4+17+14 = 100%。`.TBL table {table-layout:fixed; width:100%; font-size:93%}` `.TBL td {padding:.1em .3em; border-right/bottom:1px solid #d4d0c8}`、table は border-top/left も同色、背景 #fff。`.TBL {margin:0 10px}`。`.H24 {height:24px}`。`.CM/.RM/.LM` = text-align center/right/left。
- チームセルはチーム名にリンク(`#` + onclick で `document.location = '.../scv_team/open/{チームID}?lang=..'`)。`span.teamLink a {color:inherit}`(下線あり=ブラウザ既定)。右チームセルが LM、左が RM(左チームは右寄せ、スコアを挟んで向かい合う)。
- 日付列 `MM/DD`、時間列 `HH:MM` + 半角スペース1個。コート列は数値、CD列は文字列(`A4`, `E2`, `WAT11`, `MFTc` など。旧大会は英字数字混在)。
- 左スコア / `x` / 右スコア: `x` は常に出る(未試合でも)。未試合は両スコアのセルが空 `<td ...></td>`。
- 状況セル: 内容は常に `\n\t\t\t\t` + 内容 + `\t\t\t\t`。
- 行間に `<tr class='H24'>` 以外は無い。区切り行・日付見出し行は無い。

## 6. 状況列・行クラス [事実]
行クラスは `<td>` 毎に付く(`tr` は常に `H24`)。**全セル共通の状態クラス + ディビジョン列のみ種別クラス**:

状態クラス(全11セルに付く。CSS `.TBL TD.xxx`):
| クラス | 色(文字/背景) | 出現条件 | 状況列の表示 |
|---|---|---|---|
| `MC_END` | #0000ff / #F8F8FF | 試合終了 | `[対戦結果]`(リンク) または `終了`(リンク無しプレーンテキスト) |
| `MC_MAE` | #000000 / #ffffff | 未試合 | `未試合`(プレーンテキスト) |
| `MC_NOW` | #ff0000 / #fff0f5 | 試合中 [推測: CSSのみ定義、今回HTMLでは0件。状況列の文言は不明(JP「試合中」推測)] | 未確認 |

状況列の全パターン(約4,288行集計): `[対戦結果]` 2805 / `未試合` 732 / `終了` 183 / EN: `[Result]` 274 / `Waiting` 133 / `END` 161。
- `[対戦結果]`/`[Result]`: `<a href="#" class="Select_Link" onclick="document.location = 'https://score.clubjr.com/pscore/scv_match/open/{試合ID}?lang=..'; return false;">`。`.Select_Link {color:#008080; font-size:116%; text-decoration:none}`。
- `終了`/`END`: スコアは入っているが試合詳細(scv_match)が無い試合。リンク無し。旧大会(151: 153/156行, 188: 全6行)と、招待大会 446/408/493(最終日 23:00 台の8行)に出現。
- `未試合`/`Waiting`: スコア空欄。大会終了後でも残る(452 途中中止は135行全部が MC_MAE/未試合のまま、行の削除や「中止」表記は無い)。
- 状態は MC_END と MC_MAE の混在あり(440: 114 END / 44 MAE の1ディビジョン内)。
- 「延期」「中止」「試合中」等の文言・専用クラスはサンプル全域で出現せず(`grep` 0件)。

種別クラス(ディビジョン列の先頭td、`.TBL TD.xxx {background-color: ... !important}`):
| クラス | 背景色 | 対応するディビジョン例 |
|---|---|---|
| `MC_MEN` | #ddfbdd(薄緑) | メン / Men's (151, 160, 205, 249 の2017〜2019大会のみ) |
| `MC_WOM` | #fff4e4(薄橙) | ウィメン / Women's |
| `MC_MIX` | #ffffd1(薄黄) | ミックス, ビーチミックス, ミックス STUDENT など |
| (クラス無し) | 状態クラスの背景のまま | オープン, ４人制部門, ５人制部門 |
- クラスは名前ではなくディビジョンの区分属性で決まる(ミックス系は全てMIX)。クラス無しの時は属性値が空文字になり class が `'MC_END  CM'`(MC_END と CM の間に **半角スペース2つ**)、あれば `'MC_END MC_MIX CM'`。MAE も `'MC_MAE  CM'` / `'MC_MAE MC_WOM CM'`。
- 他のセル(日付〜状況)の class は `'MC_END CM'` 等(スペース1つ)で、種別クラスは付かない。

## 7. 切り詰め規則 [事実(実データから逆算)]
- **ディビジョン名**: 表示幅10まで(全角=2、半角=1)。JP: `ビーチミックス`→`ビーチミッ`(5字)、`ミックス STUDENT`→`ミックス S`(8+1+1)、`４人制部門`(全角数字含む5字=幅10)は無傷。EN: `Beach Mixed`→`Beach Mixe`、`Mix Student`→`Mix Studen`、`Women's`は7字で無傷。切り詰めマーカー(…)無し。`mb_strimwidth($s,0,10)` 相当。
- **チーム名**: 表示幅16まで。EN名は16文字で切る(`Aichi Syukutoku University Garnet's`→`Aichi Syukutoku `、末尾スペースが残る。`Komazawa University Milagro`→`Komazawa Univers`)。JP: `シロコロホルモンズ`(幅18)→`シロコロホルモン`(幅16)。最大表示幅は全データで16を超えない。
- JPには**登録された別の短縮名**が使われるケースが多数ある [推測: DBに短縮名フィールド]。切り詰めでは説明できない例: `バトルロイヤルホスト`→`バトルロイヤル`(幅14)、`メッターによるお料理向上委員会`→`お料理向上委員会`(接頭辞でない)、`一橋大学フライングディスクサークルUFO`→`一橋大学`、`一橋大学UFO`→`UFO`、`わんこそばすたーず`→`わんこそば`、`インサイド・スロー`→`インサイド`、`上智大学FREAKS`→`FREAKS`、`2025U24オープン部門 日本代表候補`→`JAPAN_U24OPEN`。→ クローンでは **各試合行に表示名(短縮後)をデータとして持つ**。select の option ラベルは常に正式名(全名・切り詰め無し)。
- 切り詰め後の名前はリンクの表示テキスト。リンク先は常に `scv_team/open/{id}`(en も同じID)。
- EN 版のチーム名は英語名フィールド(例 `BunkaShutter Buz`, `NagoyUniversity `、JPの `文化シヤッターBuzzBullets`→`BuzzBullets`)で別データ。
- ディビジョン名・チーム名とも HTML エスケープ出力(`Women&#039;s`, `&amp;`)。

## 8. 未確定チーム(プレースホルダ)[事実]
- 表示形式: **`[{試合CD}-勝ち]` / `[{試合CD}-負け]`**(EN: `[S3-Win]` / `[S3-Lose]`)。例 `[A1-勝ち]` vs `[B1-勝ち]`、`[E9-負け]`、`[S11-勝ち]`。`{試合CD}` は同大会の他試合の「CD」列値(A1, E9, S11, T4, R15 …)。集計: `-勝ち` 272, `-負け` 265, `-Win` 48, `-Lose` 48 のみ。
- DOM: **`<td class='MC_MAE RM' >[A1-勝ち]</td>`** — `<span class="teamLink">`も`<a>`も無い、素のテキスト(左 RM / 右 LM)。リンク無し。
- スコア欄は空、`x` は表示、状況=`未試合`(EN `Waiting`)。行クラスは MC_MAE。
- 「A1位」「勝者」等の順位ベース表記は今回のサンプル(440, 452, 400 など)では**出現せず**。プレースホルダは勝者/敗者参照のみ。
- 片側だけ確定している行もある(`[E11-負け]` vs `Loquitos`:右は通常リンク)。確定済み側は `span.teamLink` + リンク。
- 試合が終わると次の試合の `[CD-勝ち]` は実チーム名リンクに置き換わる(終了大会ではプレースホルダ0件を確認)。
- チーム名ソートの対象にも含まれる(位置は未確認)。team select の option には含まれない。

## 9. 自動リフレッシュ [事実(コードのみ。動的挙動は未実行)]
- `Refresh_OK`(`refresh.png` 32x28) / `Refresh_NG`(`refresh_ng.png` 32x28) は共に `style='display: none;'` で出力。同じ `td.RM`(width 120px, height 33px)内に `&nbsp;` と共に置かれる。
- DOM ready で `#Refresh_NG.show(500)`(jQuery show アニメ 500ms)。5000ms 後に `#Refresh_OK.show(800)` と `#Refresh_NG.hide(800)`(クロスフェード)。
- NG クリック: `alert("５秒お待ちください")`(全角５)/ EN `alert("Please Wait 5 seconds. (Reload)")`。`return false`。`alt="RefleshWait"`。
- OK クリック: `Action('search')` = 現在の select 値で `/search` にPOSTして再描画 → ページ再読込と同じなので5秒カウントが再び始まる。ソート状態は失われる(sortkey1 が残らないため)。
- 自動ポーリングは無い(setTimeout は表示切替のみ)。`<img alt>` は OK=`Reflesh_OK`(スペルミス original)、NG=`RefleshWait`。画像: raw/B/refresh.png, refresh_ng.png。

## 10. lang=en 差分 [事実]
- hidden `lang=en`、全リンク `?lang=en`。title: `-All Schedule-` / `-Division Schedule-`。大会名が英語名未登録だと `[CJS]  -All Schedule-`(二重スペース、フォールバック無し。151, 226)。
- ラベル: 日付→`Date`、チーム名→`Team`、コート→`Field`(フィルタ・ヘッダとも)、ディビジョン→`Division`、時間→`Time`、CD→`CD`、状況→`Status`。`VS`,`-`,`x` は共通。
- 状況: `[Result]` / `Waiting`(未試合) / `END`(終了)。
- プレースホルダ: `[A1-Win]` / `[A1-Lose]`。
- ディビジョン名(EN): `Open`, `Women's`, `Men's`, `Mixed`, `Beach Mixed`→表示`Beach Mixe`, `Mix Student`→`Mix Studen`, `5 on 5`, `Mix Student`。種別クラスは JP と同じ(MC_WOM など)。
- ナビ: `Tournament Top / Team List / Schedule / All Division / Open / Women's / Division Score / Group Score / Gallery`(参考、他担当領域)。
- alert 文言: `Please Wait 5 seconds. (Reload)`。
- チーム名: 英語名を使用(JPと別)、16文字で切り詰め、末尾スペース保持。日付・時刻書式は同一。

## 11. 未来・未開始・特殊大会の挙動
- 受付中/締切りの大会(539, 535, 534, 533, 532, 531, 530, 529, 528, 514, 506): 試合未登録 → フォーム(ymd/field は空選択肢のみ、team は登録チーム)+ ヘッダ行のみの表。
- 527(2026/11/03、受付締切り): 試合15件登録済み。全 `MC_MAE`、スコア空、`未試合`、ymd=20261103 selected、field 1〜3。ディビジョン名 `４人制部門`、種別クラス無し(`MC_MAE  CM`)。
- 試合中(MC_NOW)の実例は見つからず(公開中の進行中大会が無い)。クローンでは CSS(赤文字 #f00 / 背景 #fff0f5)のみ用意しておく。
- ディスクゴルフ(537: ＤＧシングルス、536: チーム制、489, 392): scv_search_tn は表が空。team select の option は選手名/チーム名。
- 開催中止(249, 275)・途中中止(452)はタイトル(大会名)にそのまま「【開催中止】」「【途中中止】」が付くだけで、画面に専用表示なし。

## 12. 実例URL(確認済み)
- tn GET: `https://score.clubjr.com/pscore/scv_search_tn/open/440?lang=jp` (ymd=0316 のみ134行、未試合44件、プレースホルダ多数)
- div GET: `https://score.clubjr.com/pscore/scv_search_div/open/822?lang=jp` (オープンのみ158行)
- POST search: `POST /pscore/scv_search_tn/search` body `tid=440&lang=jp&ymd=20250315&team=&field=` → 148行
- POST sort: `POST /pscore/scv_search_tn/sort` body `tid=440&lang=jp&ymd=20250316&team=&field=&sortkey1=DSC_mat.FIELD_NO`
- 未来: `.../scv_search_tn/open/527?lang=jp` / 状況混在: `.../open/493?lang=jp`(終了) / 旧3種別: `.../open/151?lang=jp`(MEN/MIX/WOM) / プレースホルダ: `.../open/440?lang=jp`
- EN: `.../scv_search_tn/open/440?lang=en`, `.../scv_search_div/open/823?lang=en`

## 13. クローン実装メモ
1. スケジュール2ルート(tn/div)×(GET `open/{id}`, POST `search`, POST `sort`)。テンプレートは共通。`ediv_id`/`tid` hidden と MyController の値、team option の value(名前 or ID)、ymd 既定(最終日 or 空)だけ差し替える。
2. 行データ: {divisionName(短縮前), divisionKind(MEN/WOM/MIX/none), date, time, field, cd, left{id,name,short}, right, scoreL, scoreR, status(END|MAE|NOW), resultKind(link|none), matchId}。表示は division 幅10 / チーム幅16 切り詰めを適用(JPの登録短縮名はデータ側)。
3. ソートは 3. 参照(DSC_ 接頭辞、▼/▲、MD1_ASC/MD1_DSC、副キー日時コート)。
4. ymd/team/field は AND。team は左右どちらか。空選択=全て。
5. EN は別データ(英語名)+文言辞書。
