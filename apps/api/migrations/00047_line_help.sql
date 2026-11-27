-- +goose Up
-- 公式アカウントが知らない言葉を受けたときに返す使い方の案内(UX の 7 番目)。同じ人には 1 日に 1 回だけ
-- 返す(運営とのやりとりの途中で、毎回案内が割り込まないように)。最後に返した時刻を持つ。
ALTER TABLE line_users ADD COLUMN helped_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE line_users DROP COLUMN helped_at;
