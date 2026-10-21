-- +goose Up
-- 部門の「大会を締める」(docs/api.md「大会の締め」)。残りの試合を未消化にした後、エンジンが順位を
-- 決められなかったチーム(未消化で終わった順位決定戦、未決の同順位)に本部が承認した同順位を
-- 入れる。ranks は [{"team": 12, "rank": 9}, ...](エンジンが決めた順位は含めない)。結果の変更で
-- 最終順位を書き直すとき、エンジンの順位が無いチームにだけこの順位を使う。
CREATE TABLE division_closings (
  division_id BIGINT      PRIMARY KEY REFERENCES divisions (id) ON DELETE CASCADE,
  closed_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  closed_by   BIGINT      REFERENCES users (id) ON DELETE SET NULL,
  ranks       JSONB       NOT NULL DEFAULT '[]'
);

-- +goose Down
DROP TABLE division_closings;
