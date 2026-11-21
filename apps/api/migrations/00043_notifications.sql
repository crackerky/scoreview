-- +goose Up
-- 管理画面から公式アカウントで送った記録(docs/line-integration.md「管理画面から送る仕組み」)。1 回の
-- 「送る」が notification_batches の 1 行、宛先の 1 人が notification_jobs の 1 行。

-- audience は宛先の条件(コードで名前を付けた条件と、その絞り込みの JSON。自由な SQL は持たない)。
-- body は実際に届けた文(大会名の行を含む)。request_key は管理画面の送る画面を開くたびに作る乱数で、
-- 同じ画面から 2 回送られても(押し直し、応答の前の再送)2 回目は記録も送信もしない。
CREATE TABLE notification_batches (
  id             BIGSERIAL PRIMARY KEY,
  tournament_id  BIGINT      NOT NULL REFERENCES tournaments (id) ON DELETE CASCADE,
  request_key    TEXT        NOT NULL UNIQUE,
  audience       JSONB       NOT NULL,
  body           TEXT        NOT NULL,
  automatic      BOOLEAN     NOT NULL DEFAULT FALSE,
  sent_by        BIGINT      REFERENCES users (id) ON DELETE SET NULL,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX notification_batches_tournament_idx ON notification_batches (tournament_id, created_at DESC, id DESC);

-- 宛先ごとの結果。status は pending(送る前か、送っている途中)/ sent(LINE が受け付けた)/ skipped
-- (友だちでないので送らなかった)/ failed(LINE が受け付けなかった。error に理由)。name と teams は
-- 送った時点の宛先の名前と関わるチーム(後で申込が直されても、誰に送ったかが読めるように)。
CREATE TABLE notification_jobs (
  id             BIGSERIAL PRIMARY KEY,
  batch_id       BIGINT      NOT NULL REFERENCES notification_batches (id) ON DELETE CASCADE,
  line_user_id   BIGINT      NOT NULL REFERENCES line_users (id) ON DELETE CASCADE,
  name           TEXT        NOT NULL DEFAULT '',
  teams          TEXT        NOT NULL DEFAULT '',
  status         TEXT        NOT NULL CHECK (status IN ('pending', 'sent', 'skipped', 'failed')),
  error          TEXT        NOT NULL DEFAULT '',
  sent_at        TIMESTAMPTZ,
  UNIQUE (batch_id, line_user_id)
);

-- +goose Down
DROP TABLE notification_jobs;
DROP TABLE notification_batches;
