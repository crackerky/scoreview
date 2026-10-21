-- +goose Up
-- 未決の同順位の運営者の指定を、指定した時点の成績に結び付ける(docs/format-engine.md「プール順位」)。
-- basis は並んだチームと各基準の値の JSON 配列 [{"team": 12, "values": ["3/1", "0/1", ...]}, ...]
-- (format.TeamValues。values はタイブレーク設定の基準の順)。訂正で結果が変わり、並んだチームか値が
-- 変われば、その指定は使わずに未決へ戻す(試合前の指定が後で黙って使われないように)。
-- basis が NULL の行(この列より前の指定)は、どの成績の指定か分からないので使わない。
ALTER TABLE group_rank_decisions ADD COLUMN basis JSONB;

-- +goose Down
ALTER TABLE group_rank_decisions DROP COLUMN basis;
