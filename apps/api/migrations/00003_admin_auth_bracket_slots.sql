-- +goose Up
-- 管理画面(docs/requirements.md A-01〜A-07)のための表。

-- 1. ログイン試行の制限。失敗した試行だけを記録し、同じメールアドレスの直近の失敗数で判定する
--    (internal/auth の定数)。メールアドレスは小文字に正規化した値。users への FK は張らない
--    (存在しないアドレスへの試行も同じように数え、応答で有無を区別しないため)。
CREATE TABLE login_failures (
  email         TEXT        NOT NULL,
  attempted_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX login_failures_email_idx ON login_failures (email, attempted_at);

-- 2. 管理画面で生成したトーナメント表の格子のセルと、そのセルに出す試合の参加チームの対応。
--    格子(groups.bracket_layout)はチーム ID を持つが、どのセルがどの試合の誰を表すかは持たない。
--    試合の勝敗が決まって後続試合のチームが埋まったとき、ここを引いて格子のチームを書き換える。
--    取込ツールの格子には行がない(参照サイトの表を書き換えない)。
--    role: left / right は試合の左右のチーム、winner / loser はその試合の勝者 / 敗者。
CREATE TABLE bracket_slots (
  group_id   BIGINT NOT NULL REFERENCES groups (id) ON DELETE CASCADE,
  row_index  INT    NOT NULL CHECK (row_index >= 0),
  col_index  INT    NOT NULL CHECK (col_index >= 0),
  match_id   BIGINT NOT NULL REFERENCES matches (id) ON DELETE CASCADE,
  role       TEXT   NOT NULL CHECK (role IN ('left', 'right', 'winner', 'loser')),
  PRIMARY KEY (group_id, row_index, col_index)
);
CREATE INDEX bracket_slots_match_idx ON bracket_slots (match_id);

-- +goose Down
DROP TABLE bracket_slots;
DROP TABLE login_failures;
