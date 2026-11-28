-- +goose Up
-- 申込でメンバー本人に聞く性別と生年月日(Lイベントの顧客情報入力項目にあたる。運営の要望 2026-11-28)。
-- 大会ごとに聞くかどうかを決め(tournaments)、本人が登録で答えた値をメンバーエントリーに残し、次の大会で
-- 前もって入れておくために参加者の台帳(line_users)にも残す。gender は選手と同じ M / F、birthdate は日付。
ALTER TABLE tournaments ADD COLUMN entry_ask_gender BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE tournaments ADD COLUMN entry_ask_birthdate BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE member_entries ADD COLUMN gender TEXT CHECK (gender IN ('M', 'F'));
ALTER TABLE member_entries ADD COLUMN birthdate DATE;
ALTER TABLE line_users ADD COLUMN gender TEXT CHECK (gender IN ('M', 'F'));
ALTER TABLE line_users ADD COLUMN birthdate DATE;

-- +goose Down
ALTER TABLE line_users DROP COLUMN birthdate;
ALTER TABLE line_users DROP COLUMN gender;
ALTER TABLE member_entries DROP COLUMN birthdate;
ALTER TABLE member_entries DROP COLUMN gender;
ALTER TABLE tournaments DROP COLUMN entry_ask_birthdate;
ALTER TABLE tournaments DROP COLUMN entry_ask_gender;
