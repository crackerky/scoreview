-- +goose Up
-- 公式アカウントから見た LINE の参加者と、本人用のリンク(docs/line-integration.md「本人の確かめ方」)。
-- 参加者は管理画面にログインしないので users とは分け、記録係にするときだけ user_id で結ぶ。

-- line_user_id は Webhook に付いてくる、公式アカウントから見たユーザー ID(U と 16 進 32 桁)。
-- friend はブロックされたら偽(Webhook の unfollow)、友だちに戻ったら真(follow)。
CREATE TABLE line_users (
  id             BIGSERIAL PRIMARY KEY,
  line_user_id   TEXT        NOT NULL UNIQUE,
  friend         BOOLEAN     NOT NULL DEFAULT TRUE,
  followed_at    TIMESTAMPTZ,
  unfollowed_at  TIMESTAMPTZ,
  user_id        BIGINT      REFERENCES users (id) ON DELETE SET NULL,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 同じ Webhook のイベントが 2 回届いたとき(LINE は同じイベントを複数回送ることがある)、2 回目を
-- 処理しないための記録。webhookEventId は ULID。
CREATE TABLE line_webhook_events (
  webhook_event_id  TEXT        PRIMARY KEY,
  received_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 本人用のリンク。合言葉の平文は保存せず SHA-256 だけを持つ(sessions と同じ)。purpose が team なら
-- 大会への代表の申込、member なら代表エントリーへのメンバー登録。
CREATE TABLE entry_links (
  id             BIGSERIAL PRIMARY KEY,
  token_hash     BYTEA       NOT NULL UNIQUE,
  line_user_id   BIGINT      NOT NULL REFERENCES line_users (id) ON DELETE CASCADE,
  purpose        TEXT        NOT NULL CHECK (purpose IN ('team', 'member')),
  tournament_id  BIGINT      REFERENCES tournaments (id) ON DELETE CASCADE,
  team_entry_id  BIGINT      REFERENCES team_entries (id) ON DELETE CASCADE,
  expires_at     TIMESTAMPTZ NOT NULL,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK (
    (purpose = 'team' AND tournament_id IS NOT NULL AND team_entry_id IS NULL)
    OR (purpose = 'member' AND team_entry_id IS NOT NULL AND tournament_id IS NULL)
  )
);

-- 申込の LINE の本人を users ではなく line_users で持つ。00040 の rep_user_id と user_id は、この版の
-- api に入れ替わるまで前の版が読むので、ここでは残し、デプロイの後の別のマイグレーションで消す(先に
-- 当てても前の版が動くように)。
-- 索引は、代表の申込(大会と代表)と、代表が申込を出した大会(代表だけ)の両方の読み方に使う。
ALTER TABLE team_entries ADD COLUMN rep_line_user_id BIGINT REFERENCES line_users (id) ON DELETE SET NULL;
CREATE INDEX team_entries_rep_idx ON team_entries (rep_line_user_id, tournament_id)
  WHERE rep_line_user_id IS NOT NULL;
ALTER TABLE member_entries ADD COLUMN line_user_id BIGINT REFERENCES line_users (id) ON DELETE SET NULL;
-- 同じ人が同じチームに 2 回登録しない
CREATE UNIQUE INDEX member_entries_line_user_uniq ON member_entries (team_entry_id, line_user_id)
  WHERE line_user_id IS NOT NULL;

-- +goose Down
DROP INDEX member_entries_line_user_uniq;
ALTER TABLE member_entries DROP COLUMN line_user_id;
DROP INDEX team_entries_rep_idx;
ALTER TABLE team_entries DROP COLUMN rep_line_user_id;
DROP TABLE entry_links;
DROP TABLE line_webhook_events;
DROP TABLE line_users;
