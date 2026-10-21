-- +goose Up
-- 取込パーサが実データ 687 ファイルを解析した結果に合わせる。

-- 1. コートは数字とは限らない(大会 224 / ディビジョン 467 ではコートが "A" "B")。
--    並びは store/domain で自然順(全部数字なら数値順、数字が先)にする。
ALTER TABLE matches ADD COLUMN field TEXT;
UPDATE matches SET field = field_no::TEXT WHERE field_no IS NOT NULL;
-- field_no を含む matches_division_idx は列の削除と一緒に消える
ALTER TABLE matches DROP COLUMN field_no;
CREATE INDEX matches_division_idx ON matches (division_id, match_date, start_time);

-- 2. ターンオーバーからの得点は、プルの向きから数えたディフェンス開始からの得点(ブレイク)と
--    実データ 12 試合すべてで一致したので、保存せず計算する。
--    from_turnover を参照する CHECK は列の削除と一緒に消えるので、残りの条件を付け直す。
ALTER TABLE match_events DROP COLUMN from_turnover;
ALTER TABLE match_events ADD CONSTRAINT match_events_players_only_on_goal
  CHECK (kind = 'goal' OR (scorer_id IS NULL AND assist_id IS NULL));

-- 3. キャラハンゴール(試合 45310)。得点イベントの属性として持ち、選手のキャラハン数は
--    これを数える。match_player_stats.callahans は二重の真実源になるので削除する。
ALTER TABLE match_events ADD COLUMN callahan BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE match_events ADD CONSTRAINT match_events_callahan_only_on_goal
  CHECK (kind = 'goal' OR NOT callahan);
ALTER TABLE match_player_stats DROP COLUMN callahans;

-- +goose Down
ALTER TABLE match_player_stats ADD COLUMN callahans INT NOT NULL DEFAULT 0 CHECK (callahans >= 0);
ALTER TABLE match_events DROP CONSTRAINT match_events_callahan_only_on_goal;
ALTER TABLE match_events DROP COLUMN callahan;
ALTER TABLE match_events DROP CONSTRAINT match_events_players_only_on_goal;
ALTER TABLE match_events ADD COLUMN from_turnover BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE match_events ADD CHECK (kind = 'goal' OR (scorer_id IS NULL AND assist_id IS NULL AND NOT from_turnover));
DROP INDEX matches_division_idx;
-- 数字でないコートは INT に戻せないので NULL にする
ALTER TABLE matches ADD COLUMN field_no INT;
UPDATE matches SET field_no = field::INT WHERE field ~ '^[0-9]+$';
ALTER TABLE matches DROP COLUMN field;
CREATE INDEX matches_division_idx ON matches (division_id, match_date, start_time, field_no);
