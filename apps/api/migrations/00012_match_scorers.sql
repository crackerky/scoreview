-- +goose Up
-- 試合の記録担当(docs/operations/operator-experience.md「記録係:間違えにくく、直しやすい入力」)。
-- 同じ試合を 2 台で記録すると得点が二重に入るので、記録画面を開いた端末が「いま記録している端末」を
-- 1 つだけ持つ。端末が期限(expires_at)まで延長し続け、止まれば他の端末が引き継げる。
-- device_id は端末が作った UUID(同じ人の別の端末を区別する)。試合 1 つに 1 行。
CREATE TABLE match_scorers (
  match_id    BIGINT      PRIMARY KEY REFERENCES matches (id) ON DELETE CASCADE,
  user_id     BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  device_id   UUID        NOT NULL,
  expires_at  TIMESTAMPTZ NOT NULL,
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE match_scorers;
