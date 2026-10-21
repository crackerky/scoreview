-- +goose Up
-- ログイン試行の記録に接続元の IP を足す(セキュリティ点検 M-5)。web が渡すクライアントの IP で、
-- 分からなければ空文字。同じ IP から多くのアドレスへ試す総当たりを、関数のインスタンスをまたいで数える。
ALTER TABLE login_failures ADD COLUMN ip TEXT NOT NULL DEFAULT '';
CREATE INDEX login_failures_ip_idx ON login_failures (ip, attempted_at);

-- +goose Down
DROP INDEX login_failures_ip_idx;
ALTER TABLE login_failures DROP COLUMN ip;
