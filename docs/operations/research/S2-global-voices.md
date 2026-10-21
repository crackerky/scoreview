# S2: 世界の声(主催者・スコアキーパー・選手・観客)— 既存大会運営ツールへの不満と要望

調査日: 2026-10-21 / 担当: ユーザーリサーチ(S2)

## 0. 調査の限界(先に読むこと)

- **Reddit の検索・直接取得は WebSearch / WebFetch / old.reddit.com / reddit.com の JSON すべてブロックされた**(WebSearch は reddit.com ドメイン不可、WebFetch は old.reddit.com 不可、curl は bot ブロックページが返った)。代替として、公開 Reddit アーカイブ(Arctic Shift API)から投稿・コメント本文を取得した。本文は全文読んだ上で引用している。ユーザー名は記録していない。
- Ultiworld は直接 403。**Internet Archive 経由**で記事本文を読んだ(コメント欄は取得できず、記事本文のみ)。
- Capterra / G2 は直接 403。検索結果のスニペットのみ(「snippet only」と明記)。
- App Store は Apple 公式の公開レビュー RSS(iTunes customer reviews)から取得。**SportsEngine Tourney / Exposure Events / Tournify / start.gg の米国ストア RSS は空または少数**で、前者は justuseapp.com のレビュー転載ページ(WebFetch)で補った。
- **Google Play のレビューは取得できなかった**(パッケージ ID を特定できず)。Twitter/X は検索エンジン経由で該当なし。
- 検索で何も出なかったクエリ: `site:reddit.com` を付けた全クエリ(0件)、「Ultimate Central」の Google Play レビュー、「Tournify / Exposure Events の Capterra レビュー」、日本語「アルティメット 大会運営 スコア 集計」(アルティメット特化ツールの声は0件。他競技の学術論文・アプリ紹介のみ)、Challonge の Trustpilot/Capterra(アグリゲータの定型文のみで、信頼できない)、Hacker News の「ultimate frisbee tournament scheduling」(0件)。
- Leaguevine / UltiAnalytics / PickleballBrackets / 日本のアルティメット運営者の声は、実質的に拾えなかった(Leaguevine 関連 0件、PickleballBrackets 名指しの苦情 0件)。**「世界中の声を網羅した」とは言えない。サンプルは偏っている**(アプリレビューは親・祖父母・観戦者が多く、主催者の声は Reddit と HN に偏る)。
- ベンダー自身の文章(Ultie.org の Ultiworld スポンサー記事、TopScore のレビュー記事)は「声」に数えず、参考扱い(§1-F)。

---

## 1. 声のリスト

凡例: [ID] 製品 / 役割 / 日付 / 引用 / URL。役割は文脈からの推定で、断定できないものは unknown。「snippet only」は検索結果の要約文のみ読めたもの。

### 1-A. Reddit(r/ultimate ほか。Arctic Shift アーカイブ経由で本文確認済み)

**スコア報告・ forfeit(不戦勝)**
- [R1] 大学ナショナル出場資格の騒動 / 選手系(unknown) / 2026-05-17 / "the tournament was using an app to report scores, and the app didnt have forfeit as an option." / https://reddit.com/r/ultimate/comments/1tf96ic/usau_disqualifications_at_sectionalsregionals_are/oman6p8/
- [R2] 同 / unknown / 2026-05-12 / "OV Regionals was using an app to report scores that didnt have a forfeit option (per some posts in the ultiworld discord), which makes reporting it as 13-0 more understandable" / https://reddit.com/r/ultimate/comments/1tb75ah/usa_ultimate_reinstates_franciscan_to_nationals/olg5kl1/
- [R3] 同 / unknown / 2026-05-12 / "What should have happened is the game goes down as a forfeit win ... The 2nd half not being played is not the issue; the score reporting is." / https://reddit.com/r/ultimate/comments/1tancqf/franciscan_disqualified_from_diii_college/olaxcx5/
- [R4] 同 / unknown / 2026-05-12 / "I think a lot of times for forfeits there's still a set score reported, e.g. 3-0 in soccer, not sure if there's a set standard in Ultimate ... would be up to the specific TD." / https://reddit.com/r/ultimate/comments/1tancqf/franciscan_disqualified_from_diii_college/oldppgl/
- [R5] 同 / unknown / 2026-05-17 / "If it's the last game for both teams and neither of them want to play its okay to mark the game as 'cancelled' on score reporter." / https://reddit.com/r/ultimate/comments/1tf96ic/usau_disqualifications_at_sectionalsregionals_are/om80e82/
- [R6] 同 / unknown / 2026-05-12 / "As long as the TD is informed they can make the appropriate adjustments in the schedule and all is fine." / https://reddit.com/r/ultimate/comments/1tb75ah/usa_ultimate_reinstates_franciscan_to_nationals/olfgmbc/
- [R7] Club Regionals 2026 / 選手(unknown) / 2026-09-28 / "30+ mph wind and rain so score reporting was def hard — some teams dropped out on day 2." / https://reddit.com/r/ultimate/comments/1wrhe2j/club_regionals_2026_discussion_thread/pcomir1/

**順位表・タイブレーク**
- [R8] Easterns 2025 の順位表 / 観客・選手 / 2025-04-04 / "This resulted in weird standings that I cannot explain ... why did Michigan finish above Pittsburgh despite being credited with one more loss than Pitt." / https://reddit.com/r/ultimate/comments/1jr15gs/tiebreaker_in_a_4team_round_robin_with_a_mutual/
- [R9] US Open 2015 / 観客 / 2015-07-03 / "For pool A, I can't even tell for sure who's in good position." / https://reddit.com/r/ultimate/comments/3c0lzd/us_open_mens_tiebreaker_status/
- [R10] バレーボール AES / 選手・親 / 2023-03-16 / "one of our tournaments that we did well in isn't really counting for anything since something was manually edited." / https://reddit.com/r/volleyball/comments/11t8bbg/aes_web_application/

**フォーマット・シード・ブラケット**
- [R11] WUCC 2026 フォーマット議論 / 選手 / 2026-08-20 / "with 48 teams you start with 16 pools of 3. Then each team plays a crossover ... It's 10 games just like the format they used, but a lot fewer blowout games" / https://reddit.com/r/ultimate/comments/1vts2tv/wucc_formats_are_too_dependent_on_initial_seeding/p4ws1a2/
- [R12] 同 / 選手 / 2026-08-21 / "you are pitting top teams against each other way too soon ... those top teams will rest players, shuffle their lines" / https://reddit.com/r/ultimate/comments/1vts2tv/wucc_formats_are_too_dependent_on_initial_seeding/p4zr0u6/
- [R13] 同 / 選手 / 2026-08-20 / "Seedings are little more than guesswork ... they need to do more to prevent teams from the same region being on the same side of the bracket" / https://reddit.com/r/ultimate/comments/1vts2tv/wucc_formats_are_too_dependent_on_initial_seeding/p4waj0t/
- [R14] WUCC 2026 女子決勝 / 観客 / 2026-08-27 / "Brute, Fury, and Scandal all on the same side was crazy" / https://reddit.com/r/ultimate/comments/1vzvikh/scandal_vs_grut_womens_final_2026_world_ultimate/p684vcb/
- [R15] WUCC 2026 総括 / 選手 / 2026-08-28 / "all that depends on seeding. And with so many teams attending ... its really difficult to have accurate seeding ... in mixed, there was a strong side of the bracket and a weak side" / https://reddit.com/r/ultimate/comments/1w0ng5s/wucc_2026_posttournament_discussion/p6ejv49/
- [R16] SFI East 女子(スイス形式)/ 選手 / 2026-08-17 / "How much you won or loss by on Saturday absolutely needed to be weighted by *who* you played" / https://reddit.com/r/ultimate/comments/1v8fuxp/sfi_east_womens_divison/p4af9b6/
- [R17] 同 / unknown / 2026-07-28 / "Can a Swiss format not account for a team ... being seeded 20th? I would expect that correction to be straightforward. Were there not enough rounds of Swiss before brackets?" / https://reddit.com/r/ultimate/comments/1v8fuxp/sfi_east_womens_divison/p06bija/
- [R18] NE Regionals ウェイトリスト / unknown / 2026-09-17 / "brackets can still play out very strangely depending on what kind of pickup team is showing up, and depending on the format" / https://reddit.com/r/ultimate/comments/1witcaf/drama_with_ne_regionals_wait_list_bid/padek0h/
- [R19] 同 / unknown / 2026-09-17 / "the TD can't stop the players from just getting in their cars and leaving." / https://reddit.com/r/ultimate/comments/1witcaf/drama_with_ne_regionals_wait_list_bid/pag3ghb/
- [R20] スイスドロー用ツール探し / 主催者(クラブ)/ 2024-01-29 / "we used [Kickertool] last year but its very blunt (you just get one point for a win and 0 for a loss basicley)" "the xcel sheet ... is kindof clunky to work in my version of office." / https://reddit.com/r/ultimate/comments/1ae0s4n/simple_swiss_draw_tool/

**データ・成績サイトの分散**
- [R21] 成績参照サイト / 観客・選手 / 2026-05-10 / "it might be better if they could find a way to work together to build one really great website instead of being four separate pretty good websites." / https://reddit.com/r/ultimate/comments/1t8o97c/ulti_reference/okxmbul/

**他競技(ピックルボール / Smash / バレー)**
- [R22] ツール全般 / 主催者(初開催)/ 2026-04-11 / "the free tools I used were an absolute torture to use ... ridiculously expensive. Some even charge 5 bucks per user which is egregious." / https://reddit.com/r/Pickleball/comments/1sib4i7/whats_up_with_tournament_software/
- [R23] ツール全般 / 主催者(NPO)/ 2024-05-21 / "am having a hard time with registration. Can anyone recommend a tournament software that can take registration and payment? I've gone through a couple but they are difficult to set up." / https://reddit.com/r/Pickleball/comments/1cxd3wu/tournament_software_recommendations/
- [R24] start.gg / 運営補助 / 2025-06-24 / "often times it's just one or two people at the venue running the tournament ... you used to be able to have people self report matches, but I for the life of me can't find out how to do that anymore" / https://reddit.com/r/smashbros/comments/1ljn6ze/question_for_tournament_managers_using_startgg/
- [R25] start.gg / 主催者 / 2024-11-26 / "can't seem to find the 'self-report' feature AAAAANYwhere in Start.gg. I've been looking for hours, and I'm getting quite nervous as the bracket starts soon." / https://reddit.com/r/smashbros/comments/1h0fcoc/startgg_online_tourney_help/
- [R26] start.gg / 主催者(小規模)/ 2025-12-12 / "My local is pretty small so pools aren't really useful and would end up taking too long for the time we have. Is there a way to skip them?" / https://reddit.com/r/smashbros/comments/1pl2cb7/pools_in_startgg/
- [R27] 代替サービス開発者(元 TO)/ 2025-10-22 / "the existing platforms are either way too much for my needs, or dont have the right discoverability features. It seems like I have to use multiple platforms" / https://reddit.com/r/smashbros/comments/1odihyh/yet_another_alternative_to_startgg/
- [R28] Challonge / 主催者 / 2020-06-13 / "trying to create a two group bracket on challonge ... there doesn't seem to be any info out there on it" / https://reddit.com/r/smashbros/comments/h85qtp/trying_to_create_a_two_group_bracket_on_challonge/
- [R29] Challonge / 主催者 / 2017-09-13 / 長い名前が入るようブラケット幅を広げる方法を探している(公式サイト上では変更手段が見当たらない、という趣旨。要約であり直接引用ではない) / https://reddit.com/r/smashbros/comments/6zwnzz/is_it_possible_to_increase_the_length_of_the/
- [R30] pocketgg(非公式アプリ)制作者 / 開発者・選手 / 2025-02-05 / "as we all know the mobile web experience can be less than ideal" / https://reddit.com/r/smashbros/comments/1ii04jz/startgg_mobile_app_unofficial_for_ios_android/
- [R31] GameChanger / コーチ / 2025-12-31 / "some parents who would just never use the app (GameChanger) to let us know whether they were planning on attending or not." / https://reddit.com/r/SoccerCoaching/comments/1ppzaje/youth_soccer_coaches_u12u18_whats_your_biggest/nwv13o2/
- [R32] 大会全般 / コーチ / 2026-04-06 / "Tournaments are another story but once again find a league that has their act together" / https://reddit.com/r/SoccerCoaching/comments/1s8rzq1/coaches_is_team_communication_actually_as_bad_as/oeoyss6/

### 1-B. Hacker News
- [H1] Challonge / 主催者(オンライン大会)/ 2022-05-28 / "I've had problems when organizing online tournaments using Challonge. I've had to switch to Chrome a few times ... their support page requires you to enter a combination of keys to reveal the email address: sadly this didn't work in FF" / https://news.ycombinator.com/item?id=31539365
- [H2] リーグ運営 Web アプリ運営者 / 2010-06-17 / "Scheduling and registration are where the money is at ... Those are by far the two most common and hardest problems for people to solve." / https://news.ycombinator.com/item?id=1440425

### 1-C. Ultiworld(Internet Archive 経由で本文確認、コメント欄は未取得)
- [U1] USAU Score Reporter 刷新 / 記者(観客視点)/ 2014-04-14 / "pool play and bracket play are on separate pages for no apparent reason" "visualizing results is nearly impossible due to the formatting" / https://ultiworld.com/livewire/ongoing-score-reporter-fiasco/
- [U2] 同 / 記者 / 2014-04-14 / "It was never a smooth user experience, it crashed during periods of high usage ... But it worked: tournaments were there, scores were updated." / 同上
- [U3] 同 / 記者 / 2014-04-14 / "Rethink the design from a user perspective and get a bright young software developer to build out a beautiful, mobile-friendly tool ... (reliability, search functionality, a good calendar view, a good map view, etc.)" / 同上
- [U4] Score Reporter 障害 / TD / 2014-02-19 / "I am running a tournament this weekend and had planned on using scores.usaultimate.org to distribute scores/schedules & other valuable information to all the teams involved." / https://ultiworld.com/2014/02/19/usau-tells-td-struggling-score-reporter-figure-another-method/
- [U5] 同 / USAU 担当者(運営側回答)/ 2014-02-19 / "I would figure out another method to share schedules with your teams at this point." / 同上
- [U6] Score Reporter 障害 / 記者 / 2014-02-18 / "USAU had to restart the servers multiple times per day during the 2013 Club Championships to keep Score Reporter up." / https://ultiworld.com/2014/02/18/whats-going-score-reporter/
- [U7] TD 向け解説 / TD(認定 TD) / 日付不明 / "Too many TDs try to do everything themselves — using volunteers to assist lessens the load" / https://ultiworld.com/feature/project-manage-your-tournament-to-success/

### 1-D. アプリストアレビュー(Apple 公式 RSS。低評価中心)
- [A1] TeamSnap Tournaments / 親 / 2022-07-27 / "Every time I try to go into a tournament to check something, it says unexpected error ... There's no help button, or contact us for app issues!" / App Store id1127506805(RSS)
- [A2] TeamSnap Tournaments / 親(複数チーム管理)/ 2022-05-24 / "Managing multiple teams in a tournament and can't see everyThing because the fonts are bigger than they should be. No settings either" / 同
- [A3] TeamSnap Tournaments / 親 / 2019-03-21 / "I can't find the schedule of games, I've just wasted 10 minutes ... for a tournament that starts tomorrow." / 同
- [A4] TeamSnap Tournaments / 親 / 2019-01-10 / "I cannot select the event, which means I can't access the schedule or standings." / 同
- [A5] SportsEngine Tourney / 観客・親 / 2023-10-24 / "latest change is causing app to crash as soon as I try and search for a tournament." / id1121895111(RSS)
- [A6] SportsEngine Tourney / 祖父母(観客)/ 2018-02-17 / "It worked GREAT THE 1st game! Now it won't even keep us updated!!" / 同
- [A7] SportsEngine Tourney / 親・観客 / 2023-04-22(5星) / "it would be kind of cool if you could see the progress of the score at any time, just not at the end of the game!"(要望・原文のまま) / 同
- [A8] Exposure Events / 親・観客 / 2026-07-28 / "This is an application that's required to get schedules and scores of youth sporting events. Please get rid of the ads" / id790863924(RSS)
- [A9] GameChanger / 親(スコアラー経験者)/ 2026-10-18(3星) / "I've watched so many arguments over how much time is left in a game. Please please please add a game clock." / id1308415878(RSS)
- [A10] SportsEngine(チーム管理)/ コーチ・親 / 2026-10-04 / "As a coach, Entering all of the information about all the games manually is a nightmare." / id499597400(RSS)
- [A11] SportsEngine / 親 / 2026-10-01 / "Push notifications don't come through, or come through 1-6 hours late." / 同
- [A12] SportsEngine / 親 / 2026-08-30 / "Team manager or coach updates the schedule from time to time but parents don't get any email or notification for those update" / 同
- [A13] SportsEngine / 親 / 2026-09-24 / "The app fritzed out for an entire team, causing parents to all be late and their kids to be stuck in the rain" / 同
- [A14] SportsEngine / 親 / 2025-12-18 / "Updated the app and my schedule got deleted in the middle of the season" / 同
- [A15] Heja / チーム / 2026-08-17 / "Bug on iOS app where an advert appears over the boxes where scores should be input after games" / id1157335714(RSS)
- [A16] iScore / スコアラー / 2020-03-02 / "accidentally tapped the runner icon when no play took place. The app opens a dialogue box ... and does not let you go back or select 'none' to exit out of this screen." / id364364675(RSS)
- [A17] iScore / スコアラー / 2026-08-08 / "the interface is the same as it was 10-15 years ago" / 同
- [A18] Tourney - Tournament Maker / 主催者(64チーム)/ 2026-03-11 / "64 team tournament. Seed 1 plays seed 64 ... but they're placed sequentially in the brackets so the top ranked teams play each other in the same bracket." / id6450659011(RSS)
- [A19] 同 / 主催者 / 2025-06-17 / "Whenever you create a tournament and turn on the 3rd place matchup the app will crash whenever you try to enter matchup results." / 同
- [A20] 同 / 主催者(混成クラブ戦)/ 2024-03-21 / "With mixed clubs participating ... would be helpful to edit with cut and paste capability. This could allow keeping games intraclub from occurring" / 同
- [A21] 同 / 主催者 / 2024-03-02 / "brackets dont show seedings. second you have to pay montly/by year to use double elimination ... u can only have 25 teams/players" / 同
- [A22] My Bracket / 主催者(テニスクラブ)/ 2017-05-03 / テニスクラブの年次トーナメント用に Pro 版を購入したが、共有手段がメールと写真保存のみで広告の説明と違った(要約。直接引用ではない) / id570129169(RSS)
- [A23] BenchApp / 親・選手 / 2026-04-02 / "why does it need to cost $9/month just to get a push notification to my phone on a game day?" / id883504378(RSS)
- [A24] LeagueApps Play / 親 / 2024-07-30 / "my son is on a team that has 125+ calendar events in a 6 month span. Inability to sort by type (practice, game, tourname[nt])" / id1534160239(RSS)
- [A25] pocketgg / 選手 / 2024-08-07 / "start.gg is a literal nightmare to use on mobile web." / id1576064097(RSS)

### 1-E. justuseapp.com 転載(SportsEngine Tourney レビュー。日付が出ているもののみ記載)
- [J1] SportsEngine Tourney / 観客 / 日付なし / "There are no scores, either from yesterday or today, no standings and no brackets!!" / https://justuseapp.com/en/app/1121895111/tourney-machine/reviews
- [J2] 同 / 親 / 2020-12-17 / "posting the schedule with team name and field numbers on same line." / 同
- [J3] 同 / 親 / 日付なし / "SportsEngineTourney is tournament-centric so you can't follow your team across all tournaments and seasons." / 同
- [J4] 同 / 親 / 日付なし / "There is no way to contact SportsEngineTourney support team in app or otherwise." / 同
- [J5] 同 / 親 / 日付なし / "I cannot get rid of it. It's now my default tournament even if it was the wrong one" / 同
- [J6] 同 / 親 / 日付なし / "every time an ad comes on the whole thing crashes." / 同
- [J7] 同 / 親 / 2021-03-10 / "The app is not readable. The words look scrunched." / 同

### 1-F. snippet only(本文未確認)/ ベンダー文章(声に数えない)
- [S1] Capterra, Tourney Machine(SportsEngine Tourney)/ 主催者 / 2018 / 「複数大会間のコピー&ペーストが面倒」「スケジュール自動生成の支援が足りない」(検索要約の言い換え。直接引用なし。snippet only)/ https://www.capterra.com/p/228573/Tourney-Machine/
- [S2] 同アプリ / 2023 / push 通知が時間通りに届かないことがある(snippet only)
- [S3] GameChanger / ストアレビュー / 日付不明 / 「採点ミスの修正が難しく価格に見合わない」「スコアリングより配信・録画に重心が移った」(検索要約。snippet only)/ https://apps.apple.com/us/app/gamechanger/id1308415878
- [S4] G2, Tourney Machine / 日付不明(2019以降)/ アプリが頻繁にフリーズし真っ白になる(snippet only)/ https://www.g2.com/survey_responses/tourney-machine-review-3731998
- [V1] Ultie.org(ベンダーのスポンサー記事、声ではない)/ 2024-06-06 / "We don't know how much time the Tom's Tourney organizers would have spent compiling the schedule manually in Excel for 64 teams playing on 13 fields." ほか、主催者コメント "we didn't have to build any excel sheets nor make any calculations" / https://ultiworld.com/2024/06/06/how-three-european-tournaments-saved-loads-of-nerves-and-trees/
- [V2] TopScore のイベントサイトレビュー(Skyd 2012、記者による評価)/ "One of the greatest pains for any TD is scheduling and assigning fields, especially when there are last minute changes" "The whole system can be a bit overwhelming at first." / https://skydmagazine.com/2012/05/product-review-topscore-event-site/

声の件数: 一次的な声 73 件(Reddit 32 + HN 2 + Ultiworld 7 + Apple RSS 25 + justuseapp 7)。ほかに snippet only 4 件、ベンダー文章 2 件(件数に含めない)。R29・A22 は要約であり直接引用ではない。

---

## 2. ペインポイント分類(件数は該当 ID の数。1 つの声が複数に属するため合計は一致しない)

分類は私の主観的タグ付けである。サンプルが偏っているため、**件数は重要度ではなく「今回拾えた量」**として読むこと。

| # | ペインポイント | 件数 | 主な ID | 主な役割 |
|---|---|---|---|---|
| P1 | スコア入力のエッジケース未対応(不戦勝・中止・相互不戦勝・半試合・訂正/undo) | 9 | R1-R6, R8, A16, S3 | スコアキーパー、TD |
| P2 | 信頼性(高負荷・悪天候・通信・移行時の停止) | 12 | U1-U6, R7, A1, A4-A6, A14, J1, S4 | TD、観客、親 |
| P3 | フォーマット/シード/ブラケット生成が柔軟でない(power pool、スイス、同地域が同じ側、イントラクラブ回避、2グループ) | 11 | R11-R20, A18-A21, R26, R28 | TD、選手 |
| P4 | 順位表・タイブレークが不透明、手動編集の痕跡 | 4 | R8, R9, R10, R16 | 選手、観客 |
| P5 | スケジュール変更の通知・反映(遅延、通知なし、フィールド番号の表示) | 8 | A11, A12, A13, A23, J2, S2, V2, S1 | 親、観客、TD |
| P6 | 設定・機能の発見しにくさ(self-report、プール省略、登録・決済の設定) | 8 | R23-R28, R29, V2 | 主催者 |
| P7 | 価格・広告・ペイウォール(1人あたり課金、広告がスコア入力欄を覆う) | 9 | R22, A8, A15, A21, A22, A23, J6, R23, 他 | 主催者、親 |
| P8 | UI の可読性・モバイル(文字が重なる、フォントが巨大) | 7 | A2, A3, J2, J7, A25, R30, U1 | 親、観客 |
| P9 | データの分断(大会ごとに別、成績サイトが複数、複数プラットフォーム併用) | 4 | J3, R21, R27, A24 | 観客、選手 |
| P10 | サポート不在 | 3 | J4, A1, H1 | 全般 |
| P11 | 手作業・スプレッドシート依存 | 4 | A10, S1, R20, V1(ベンダー) | 主催者、コーチ |
| P12 | 現地人手不足(1〜2人で運営、セルフレポート無し) | 2 | R24, R25 | 主催者 |

### 役割別の代表的な声

**TD / 主催者**
- P1(競技固有・今回最も具体的): "the app didnt have forfeit as an option" [R1]。結果として別のスコア(13-0)で報告され、チームが失格・再審の騒ぎになった(2026 年 5 月、複数スレッド)。ツールの欠落が競技上の紛争の起点になった実例。
- P3: スイスドロー用に Kickertool と Excel を使い、どちらも不満 [R20]。
- P6/P12: 現地 1〜2 人で運営する中、"self-report" を探して "getting quite nervous as the bracket starts soon" [R25]。
- P7: "ridiculously expensive. Some even charge 5 bucks per user" [R22]。
- P2: TD は "figure out another method to share schedules" と運営側に言われた [U4, U5]。大会 3 日前にスコア基盤が使えなくなる事態。

**スコアキーパー(スコア入力者)**
- 直接の声は少ない(Ultimate 専用アプリのレビューが拾えず)。近い他競技の声: 訂正できない/戻れない [A16, S3]、広告が入力欄を覆う [A15]、試合時計が無く残り時間で揉める [A9]。
- Ultimate 固有の運用(ソフトキャップ、タイムアウト、ハーフ)に関する苦情は今回の調査では 0 件。**これは「問題が無い」ではなく「声が拾えなかった」**。

**選手**
- P3/P4: シードが "little more than guesswork" [R13]、強豪が同じ側に集中 [R14, R15]、パワープールの是非 [R11, R12]。
- P4: 順位がなぜそうなったか説明がない [R8, R9]。

**観客・親**
- P2/P5: 通知遅延と画面不具合 [A5, A6, A11, A13]。大会当日の「次の試合はどこ」が最大の関心。
- P8: 読めない画面 [J2, J7, A2]。

---

## 3. 要望リスト("I wish it did X")

明示的な要望と、不満から読み取れる暗黙の要望を区別する。

**明示的(原文に wish / please / would be helpful がある)**
1. 不戦勝・中止・相互不戦勝を選べるスコア入力 [R1-R5]
2. 試合時計(残り時間の共通表示)[A9]
3. 進行中のスコアを見られる(終了時だけでなく)[A7]
4. セルフレポート(選手自身が結果を入力)[R24, R25]
5. 小規模大会ではプールを省略できる [R26]
6. 同じクラブ同士の対戦を避けるブラケット編集(コピー&ペースト)[A20]
7. 第 3 位決定戦を含むブラケットで落ちない [A19]
8. 大会横断で自チームを追える [J3]
9. 複数の成績サイトを 1 つに [R21]
10. スイスでタイブレークを細かく設定できる [R20]
11. シード補正がスイスの途中で簡単にできる [R17]
12. スコア入力画面に広告を出さない [A8, A15]
13. 登録と決済を簡単に設定 [R23]

**暗黙的(不満からの推定。私の解釈)**
14. 順位決定の理由を 1 クリックで見られる(どのタイブレークで決まったか)[R8, R9]
15. 同地域チームを自動的に別の山へ振るシード補助 [R13, R14]
16. 悪天候時に全試合の時刻を一括で後ろへずらし、通知を一斉送信する(他競技ツールでは LeagueLobster / USTA が実装済み。検索要約のみ)
17. 通信不安定でもスコア入力でき、後で同期される [R7, A13]
18. 1〜2 人でも回せる運営支援 [R24, R25]
19. 主催者への連絡窓口(問い合わせ導線)[J4, A1]

---

## 4. 好評なツールが「ユーザーの言葉で」うまくやっていること

ここでの根拠は 5 星レビュー(Apple RSS)と少数の Reddit コメントで、**Ultimate の大会運営ツールに対する好評はほぼ拾えなかった**。以下は隣接ツール(チーム管理)の声で、大会運営に直接一般化はできない。

- **「全部が一か所にある」**: Heja "all your training, matches and player availability in one place" / LeagueApps Play "keeps all of your teams organized and easy to read, you can get reminders and updates for games and location changes" (2024-10-04)
- **フィールド/場所の具体性**: LeagueApps Play(2024-10-16)"they post exactly where in the field. The children will be playing. No more roaming around looking for the fields."(観客・親が最もうれしい情報は「どのフィールドか」)
- **出欠の可視化と催促**: Spond(2026-02-26)"can see who has accepted and send reminders to the parents who haven't confirmed attendance"
- **広告が少なく清潔**: Spond(2026-04-20)"Clean and easy to use design. Minimal adverts (given it's free to use)"
- **試合の事後情報**: GameChanger(2026-10-19)"Being able to look at the video, recap, box score, etc is so very helpful. I gladly pay"(価値のある情報には課金してよいという声)
- **簡単さ**: GameChanger(2026-10-11)"Easy to operate, highly recommend for score keeping"
- **満足の典型は「必要な情報にすぐ着く」**であり、機能数ではない。逆に、低評価はほぼすべて「必要な情報(スケジュール/スコア/順位)に着けない」「通知が遅い」「落ちる」。
- ベンダー側の主張(参考のみ、未検証): Ultie.org は、スケジュール自動生成・クリック数回のスコア入力・SOTG の自動集計・ライブスコアを訴求しており、複数の主催者コメントが好意的 [V1]。ただしスポンサー記事のため、独立した評価ではない。

---

## 5. 自社プロダクトへの示唆(優先度順)

前提: 想定は「日本の大会主催者向けの、プール戦→勝ち上がり規則→順位決定ブラケット自動生成・スケジュール/フィールド割当・スコア入力・順位表のライブスコアサイト」。今回の声は海外中心で、日本の運用差(言語、現地人数、規模)は検証できていない。

**優先 1(必須。競技的な事故を防ぐ)**
1. **スコア入力に「不戦勝 / 中止 / 相互不戦勝 / 半試合打切り」を最初から持たせる**。R1-R6 は 2026 年に実際に失格騒動を起こした。順位表への反映ルール(固定スコアか、順位計算から除外か)を設定可能にし、TD が決めた理由を記録する。
2. **順位表に「なぜこの順位か」を表示する**(タイブレークの段階別の根拠)。R8, R9 は観客・選手が自力で理解できていない。勝ち上がり規則と連動させるのが差別化になりうる。
3. **スコア入力は訂正・取消が簡単で、オフラインでも入力でき、広告を一切出さない**。A15, A16, R7。

**優先 2(自動生成の核。ここで他ツールとの差が出る)**
4. **フォーマットの選択肢を広く**: パワープール、クロスオーバー、スイス、小規模でのプール省略 [R11, R18, R20, R26]。出場チーム数が 2 のべき乗でない場合の扱いも。
5. **シード補助**: 同地域/同クラブを別の山に分ける制約、スイス途中でのシード補正 [R13, R14, R17, A20]。生成後に手で動かせ、動かすと影響する試合を表示する。
6. **悪天候・遅延のとき、全試合の時刻を一括で後ろへずらし、通知する** [S1-S2 の言及、R7]。第一版の簡易版でよい(再生成より「ずらす」)。

**優先 3(観客・選手の体験)**
7. **「次の試合の時刻と場所(フィールド番号)」を最短で見られる**。登録なしで開けるページ、フィールド番号を別行で大きく表示 [J2, LeagueApps の好評]。
8. **ライブスコア(進行中)と試合時計/キャップ表示**: A7, A9。
9. **大会をまたいだチーム追跡は後回し**(J3, R21 は要望だが、第一版の範囲を超える。まず 1 大会で完結させる)。

**優先 4(主催者の運営負担)**
10. **1〜2 人で運営できる**: セルフレポート(両チーム入力→一致で確定、食い違いで TD 通知)[R24, R25]。ただし R1 の件が示すように、**自己申告には不正・誤報告のリスクがあり**、TD の承認フローを入れる。
11. **ペイウォール/課金を主催者側に寄せ、観客には無料・広告なし**。R22, A8, A23 の不満は「利用者に課金/広告」に集中。

**やらないほうがよいこと(声から)**
- 機能を詰め込み、設定が見つからない UI にしない [R25, R26, V2]。
- 複数のページに結果を分けない(プール戦とブラケットを別ページにして批判された [U1])。

**未検証・追加調査が必要**
- 日本のアルティメット主催者・スコアキーパーの生の声(今回 0 件)。学連・協会の運営者への直接ヒアリングが必要。
- Ultimate Central / Leaguevine / UltiAnalytics への直接の苦情(取得 0 件)。Google Play レビュー(未取得)。
- Ultiworld のコメント欄、Facebook グループ、Discord(未取得)。
- ソフトキャップ・タイムアウト・ハーフの運用に関する要望(0 件)。

生データ(スクレイプ結果)は .local/research-global/(git 管理外) にある(reddit.json, reddit2.json, apple_reviews.json)。
