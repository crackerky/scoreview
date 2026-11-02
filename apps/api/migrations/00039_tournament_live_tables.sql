-- +goose Up
-- 観戦アプリの「表で見る」(星取表とトーナメント表)の見せ方(docs/api.md「観戦アプリの表」)。管理者が
-- 大会ごとに決める。行が無い大会は既定(全部出す、列は全部、まる・さんかく・ばつ、登録順、試合中は
-- いまの点、進出圏に色)。本サイト独自の追加(docs/requirements.md「観戦アプリ」)。
-- marks は maru(○△×)/ wdl(W D L)/ kanji(勝 分 敗)、order_by は entry(登録順)/ rank(順位順)、
-- live は score(試合中のマスにいまの点)/ blank(終わるまで空ける)。
CREATE TABLE tournament_live_tables (
  tournament_id BIGINT      PRIMARY KEY REFERENCES tournaments (id) ON DELETE CASCADE,
  shown         BOOLEAN     NOT NULL DEFAULT TRUE,
  pool_shown    BOOLEAN     NOT NULL DEFAULT TRUE,
  bracket_shown BOOLEAN     NOT NULL DEFAULT TRUE,
  col_wdl       BOOLEAN     NOT NULL DEFAULT TRUE,
  col_goals     BOOLEAN     NOT NULL DEFAULT TRUE,
  col_diff      BOOLEAN     NOT NULL DEFAULT TRUE,
  col_points    BOOLEAN     NOT NULL DEFAULT TRUE,
  col_rank      BOOLEAN     NOT NULL DEFAULT TRUE,
  marks         TEXT        NOT NULL DEFAULT 'maru' CHECK (marks IN ('maru', 'wdl', 'kanji')),
  order_by      TEXT        NOT NULL DEFAULT 'entry' CHECK (order_by IN ('entry', 'rank')),
  live          TEXT        NOT NULL DEFAULT 'score' CHECK (live IN ('score', 'blank')),
  highlight     BOOLEAN     NOT NULL DEFAULT TRUE,
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_by    BIGINT      REFERENCES users (id) ON DELETE SET NULL
);

-- +goose Down
DROP TABLE tournament_live_tables;
