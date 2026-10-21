-- +goose Up
-- 大会のコピー(運営者レビュー H9)で作った部門が、どの部門からコピーしたか。新しい部門で形を決めるとき、
-- コピー元の部門の形(プール数、順位帯、タイブレーク、規定スコア、時間割の設定)を初期値にする。
-- コピー元が消えたら NULL(形の初期値が無くなるだけ)。
ALTER TABLE divisions ADD COLUMN copied_from_id BIGINT REFERENCES divisions (id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE divisions DROP COLUMN copied_from_id;
