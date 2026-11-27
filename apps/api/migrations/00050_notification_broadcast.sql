-- +goose Up
-- 公式 LINE の友だち全員に送る宛先(UX の 6 番目)。LINE の一斉送信(broadcast)で送り、宛先の人は
-- 分からないので、送信ごとに「友だち全員」の 1 行を line_user_id を空にして記録する(結果と送り直しの
-- キーと持ち時間は、ほかの宛先の行と同じに扱う)。
ALTER TABLE notification_jobs ALTER COLUMN line_user_id DROP NOT NULL;

-- +goose Down
DELETE FROM notification_jobs WHERE line_user_id IS NULL;
ALTER TABLE notification_jobs ALTER COLUMN line_user_id SET NOT NULL;
