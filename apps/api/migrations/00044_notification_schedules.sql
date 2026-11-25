-- +goose Up
-- 時刻を決めて送る(予約)。前日と当日の案内のように、決めた時刻に送る。予約した送信は宛先をまだ
-- 持たず、送る時刻に条件で宛先を数え直す(予約の後に承認したチームにも届くように)。
-- send_at は送る時刻(すぐ送った送信は NULL)。state は scheduled(送る時刻を待っている)/ sent(宛先を
-- 決めて送った。すぐ送った送信は全部これ)/ cancelled(運営が取りやめた)/ skipped(送る時刻に送れ
-- なかった。理由は note に、no_recipients / quota / line_not_configured のどれか)。
ALTER TABLE notification_batches ADD COLUMN send_at TIMESTAMPTZ;
ALTER TABLE notification_batches ADD COLUMN state TEXT NOT NULL DEFAULT 'sent'
  CHECK (state IN ('scheduled', 'sent', 'cancelled', 'skipped'));
ALTER TABLE notification_batches ADD COLUMN note TEXT NOT NULL DEFAULT '';
-- 送る時刻の来た予約を探す(定期の呼び出しのたびに読む)
CREATE INDEX notification_batches_due_idx ON notification_batches (send_at) WHERE state = 'scheduled';

-- +goose Down
DROP INDEX notification_batches_due_idx;
ALTER TABLE notification_batches DROP COLUMN note;
ALTER TABLE notification_batches DROP COLUMN state;
ALTER TABLE notification_batches DROP COLUMN send_at;
