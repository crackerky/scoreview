-- +goose Up
-- 部門の組分けの下書き(管理画面「試合を組む」の組分け。2026-11-27)。運営がチームを選んで組に入れるか、
-- Excel の表を貼って決めた組を、形を作る前に残す。pools は組の順に、組ごとのチーム(teams.id)を上から
-- 並べた配列の配列([[12, 15], [13, 14]])。途中の下書き(まだ組に入っていないチームがある)も残せる。
CREATE TABLE pool_drafts (
  division_id BIGINT      PRIMARY KEY REFERENCES divisions (id) ON DELETE CASCADE,
  pools       JSONB       NOT NULL,
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE pool_drafts;
