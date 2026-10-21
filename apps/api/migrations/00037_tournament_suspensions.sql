-- +goose Up
-- 雷・雨・暑さによる大会の中断(docs/api.md「大会の中断」)。行がある間は中断中で、閲覧ページの
-- 大会の上部に帯を出す(本サイト独自の追加。docs/requirements.md「意図的な差分」)。再開すると行を
-- 消し、中断した分だけ未開始の試合の時刻をずらす。
-- reason は thunder(雷)/ rain(雨)/ heat(暑さ)/ other(その他)。resume_at は再開予定 "HH:MM"。
CREATE TABLE tournament_suspensions (
  tournament_id BIGINT      PRIMARY KEY REFERENCES tournaments (id) ON DELETE CASCADE,
  reason        TEXT        NOT NULL CHECK (reason IN ('thunder', 'rain', 'heat', 'other')),
  note          TEXT        NOT NULL DEFAULT '',
  resume_at     TEXT,
  suspended_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  suspended_by  BIGINT      REFERENCES users (id) ON DELETE SET NULL
);

-- +goose Down
DROP TABLE tournament_suspensions;
