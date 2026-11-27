-- +goose Up
-- 公式 LINE で送る文の下のボタン(UX の 6 番目)。[{kind, label, url}] の並び。kind は entry(「エントリー」を
-- 送る。押すと本人用の申込のリンクが返る)/ site(スコアサイトの大会のページ)/ url(運営が書いた URL)。
-- url は送るときに決めた実際の行き先(site も大会のページの URL を入れて残す)。ボタンが無ければ文だけで
-- 届ける。
ALTER TABLE notification_batches ADD COLUMN buttons JSONB NOT NULL DEFAULT '[]';

-- +goose Down
ALTER TABLE notification_batches DROP COLUMN buttons;
