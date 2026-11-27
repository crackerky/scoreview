-- +goose Up
-- チームの記録係(UX の 8 番目)。参加が決まったチームの代表が、LINE で登録したメンバーから 1 人を
-- 選ぶ。記録係は自分たちの試合のスコアを、公式 LINE から開く本人用のリンクで入れる(両方のチームに
-- 記録係がいれば、時間割で上に書いてあるチームが記録する)。メンバーの行を消したら選び直してもらう。
ALTER TABLE team_entries ADD COLUMN scorer_member_id BIGINT REFERENCES member_entries (id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE team_entries DROP COLUMN scorer_member_id;
