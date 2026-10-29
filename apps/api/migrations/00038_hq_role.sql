-- +goose Up
-- 本部スタッフ(hq)の役割を足す(docs/requirements.md「権限モデル」)。大会当日の進行だけを任せる役割で、
-- 担当の大会は記録係と同じ tournament_staff で決める。00001 の列の CHECK は名前を付けていないので、
-- PostgreSQL の既定の名前(表_列_check)で外して付け直す。
ALTER TABLE users DROP CONSTRAINT users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('admin', 'staff', 'hq'));

-- +goose Down
ALTER TABLE users DROP CONSTRAINT users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('admin', 'staff'));
