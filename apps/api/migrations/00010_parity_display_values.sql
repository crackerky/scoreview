-- +goose Up
-- 参照サイトとの突き合わせ(cmd/parity)で見つかった、画面に出るのに保存していなかった値を足す。
-- 根拠は docs/clone-spec の各章。

-- 1. グループとギャラリーの英語名(英語版のナビ、SCORE メニュー、ページの title に出る)。
--    参照サイトでは英語名が空のグループがあり、ギャラリーの英語名は "---" のことが多い。
ALTER TABLE groups ADD COLUMN name_en TEXT NOT NULL DEFAULT '';
ALTER TABLE galleries ADD COLUMN name_en TEXT NOT NULL DEFAULT '';

-- 2. 対戦相手の決まり方の英語表記("[A1-Win]" "[MA-3rd]" "(A10)")。参照サイトの英語版は、
--    英語名のないチームの代わりにこれを出す(clone-spec B 8)。決まったチームにも残る。
ALTER TABLE matches ADD COLUMN left_label_en TEXT NOT NULL DEFAULT '';
ALTER TABLE matches ADD COLUMN right_label_en TEXT NOT NULL DEFAULT '';

-- 3. 試合ページの表示(clone-spec D 2.2、D 2.4、D 2.5)。
--    group_label: グループ欄の文字。NULL ならグループの名前。参照サイトは試合を作ったときの
--      グループ名を試合に持つらしく、後で変えたグループ名と食い違う(推測)。2017 年の試合は空。
--    note: スコアボードの下の行の中央に出る備考("スクラッチのため" など)。
--    hide_blocks / hide_stats: 出場選手表の Block 列と統計表を出さない(試合ごとに違う。2020〜2025 の
--      一部の試合と 2017 の試合)。既定値の false を Go のゼロ値と揃えるため「隠す」向きにする。
ALTER TABLE matches ADD COLUMN group_label TEXT;
ALTER TABLE matches ADD COLUMN note TEXT NOT NULL DEFAULT '';
ALTER TABLE matches ADD COLUMN hide_blocks BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE matches ADD COLUMN hide_stats BOOLEAN NOT NULL DEFAULT FALSE;

-- 4. チームページに SOTG スコア一覧(値は空)を出すか(clone-spec D 1.1 の 6)。
ALTER TABLE divisions ADD COLUMN show_sotg BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE divisions DROP COLUMN show_sotg;
ALTER TABLE matches DROP COLUMN hide_stats;
ALTER TABLE matches DROP COLUMN hide_blocks;
ALTER TABLE matches DROP COLUMN note;
ALTER TABLE matches DROP COLUMN group_label;
ALTER TABLE matches DROP COLUMN right_label_en;
ALTER TABLE matches DROP COLUMN left_label_en;
ALTER TABLE galleries DROP COLUMN name_en;
ALTER TABLE groups DROP COLUMN name_en;
