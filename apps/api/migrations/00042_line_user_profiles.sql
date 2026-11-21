-- +goose Up
-- 参加者(公式アカウントの友だち)の前回の入力(docs/line-integration.md「参加者の台帳」)。次の申込で、
-- 代表の名前と連絡先を最初から入れておくために持つ。申込とメンバー登録のたびに、そのとき入れた値で
-- 上書きする(メンバー登録は名前だけ)。profile_updated_at が空の人は、まだ何も入れていない。
ALTER TABLE line_users ADD COLUMN name TEXT NOT NULL DEFAULT '';
ALTER TABLE line_users ADD COLUMN phone TEXT NOT NULL DEFAULT '';
ALTER TABLE line_users ADD COLUMN email TEXT NOT NULL DEFAULT '';
ALTER TABLE line_users ADD COLUMN profile_updated_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE line_users DROP COLUMN profile_updated_at;
ALTER TABLE line_users DROP COLUMN email;
ALTER TABLE line_users DROP COLUMN phone;
ALTER TABLE line_users DROP COLUMN name;
