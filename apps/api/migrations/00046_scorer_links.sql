-- +goose Up
-- チームの記録係が記録の画面を開く本人用のリンク(purpose 'scorer'。team_entry_id は記録係のチームの
-- 申込)と、記録係の利用者(users.team_scorer)。記録係は公式 LINE で「記録」と送って届くリンクから
-- 入り、パスワードは持たない(パスワードのログインの対象にしない)。自分たちの試合だけを記録できる。
ALTER TABLE entry_links DROP CONSTRAINT entry_links_purpose_check;
ALTER TABLE entry_links ADD CONSTRAINT entry_links_purpose_check
  CHECK (purpose IN ('team', 'member', 'scorer'));
ALTER TABLE entry_links DROP CONSTRAINT entry_links_check;
ALTER TABLE entry_links ADD CONSTRAINT entry_links_check CHECK (
  (purpose = 'team' AND tournament_id IS NOT NULL AND team_entry_id IS NULL)
  OR (purpose IN ('member', 'scorer') AND team_entry_id IS NOT NULL AND tournament_id IS NULL)
);
ALTER TABLE users ADD COLUMN team_scorer BOOLEAN NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE users DROP COLUMN team_scorer;
DELETE FROM entry_links WHERE purpose = 'scorer';
ALTER TABLE entry_links DROP CONSTRAINT entry_links_check;
ALTER TABLE entry_links ADD CONSTRAINT entry_links_check CHECK (
  (purpose = 'team' AND tournament_id IS NOT NULL AND team_entry_id IS NULL)
  OR (purpose = 'member' AND team_entry_id IS NOT NULL AND tournament_id IS NULL)
);
ALTER TABLE entry_links DROP CONSTRAINT entry_links_purpose_check;
ALTER TABLE entry_links ADD CONSTRAINT entry_links_purpose_check CHECK (purpose IN ('team', 'member'));
