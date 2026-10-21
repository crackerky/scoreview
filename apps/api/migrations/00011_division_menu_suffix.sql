-- +goose Up
-- サイドバーの SCORE メニューでディビジョン名の後ろに付く競技の略号(clone-spec A 1.5.1)。
-- 参照サイトでは "Ult"(アルティメット)、"Free"(フリースタイル)、"DiscGolfT"(ディスクゴルフ団体戦)を
-- 観測した。空なら最も多い "Ult" として出す(管理画面で作ったディビジョンなど)。
ALTER TABLE divisions ADD COLUMN menu_suffix TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE divisions DROP COLUMN menu_suffix;
