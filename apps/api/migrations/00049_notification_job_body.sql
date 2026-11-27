-- +goose Up
-- 人ごとに差し込んで送った文(UX の 6 番目。{チーム名} などを、その人のチームの値に置き換えたもの)。
-- 差し込みの無い送信は空で、notification_batches.body をそのまま届けた。送り直しは、ここに残した文を
-- 同じ送り直しのキーで送る(1 回目と違う文にしない)。
ALTER TABLE notification_jobs ADD COLUMN body TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE notification_jobs DROP COLUMN body;
