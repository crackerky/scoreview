-- +goose Up
-- 「トーナメントだけ」の形(knockout。全チームで 1 つの勝ち抜きを組む。2026-11-28)を、形の指定に使える
-- テンプレートに足す。
ALTER TABLE division_formats DROP CONSTRAINT division_formats_template_check;
ALTER TABLE division_formats ADD CONSTRAINT division_formats_template_check
  CHECK (template IN ('pool_bands', 'blocks', 'round_robin', 'knockout'));

-- +goose Down
-- knockout の形が残っていると戻せない(先にその部門の形を作り直す)
ALTER TABLE division_formats DROP CONSTRAINT division_formats_template_check;
ALTER TABLE division_formats ADD CONSTRAINT division_formats_template_check
  CHECK (template IN ('pool_bands', 'blocks', 'round_robin'));
