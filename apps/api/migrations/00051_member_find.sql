-- +goose Up
-- メンバーが招待リンクを使わずに、公式 LINE の「メンバー登録」から大会を選び、自分のチームを探して登録する
-- 道(UX、2026-11-27)。
-- entry_links の purpose 'find' は、大会(tournament_id)のチームを探す画面を開く本人用のリンク。
-- entry_links.found は、その画面でチームを選んで発行したメンバー登録のリンク(招待リンクからではない)。
-- member_entries.without_invite は、招待リンクを使わずに登録した人(代表の画面に印を出し、参加が決まる前
-- なら代表が外せる)。member_entries.self_added は、代表の名簿の行ではなく本人の登録で足した行(外すと
-- 行を消す。代表の名簿の行なら、本人の LINE との結びつきだけを外す)。
ALTER TABLE entry_links DROP CONSTRAINT entry_links_purpose_check;
ALTER TABLE entry_links ADD CONSTRAINT entry_links_purpose_check
  CHECK (purpose IN ('team', 'member', 'scorer', 'find'));
ALTER TABLE entry_links DROP CONSTRAINT entry_links_check;
ALTER TABLE entry_links ADD CONSTRAINT entry_links_check CHECK (
  (purpose IN ('team', 'find') AND tournament_id IS NOT NULL AND team_entry_id IS NULL)
  OR (purpose IN ('member', 'scorer') AND team_entry_id IS NOT NULL AND tournament_id IS NULL)
);
ALTER TABLE entry_links ADD COLUMN found BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE member_entries ADD COLUMN without_invite BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE member_entries ADD COLUMN self_added BOOLEAN NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE member_entries DROP COLUMN self_added;
ALTER TABLE member_entries DROP COLUMN without_invite;
ALTER TABLE entry_links DROP COLUMN found;
DELETE FROM entry_links WHERE purpose = 'find';
ALTER TABLE entry_links DROP CONSTRAINT entry_links_check;
ALTER TABLE entry_links ADD CONSTRAINT entry_links_check CHECK (
  (purpose = 'team' AND tournament_id IS NOT NULL AND team_entry_id IS NULL)
  OR (purpose IN ('member', 'scorer') AND team_entry_id IS NOT NULL AND tournament_id IS NULL)
);
ALTER TABLE entry_links DROP CONSTRAINT entry_links_purpose_check;
ALTER TABLE entry_links ADD CONSTRAINT entry_links_purpose_check
  CHECK (purpose IN ('team', 'member', 'scorer'));
