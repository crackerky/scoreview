-- +goose Up
-- 大会の途中で棄権したチーム(本部の「棄権にする」。docs/api.md「チームの棄権」)。
-- 棄権したチームの残りの試合は、本部が確かめたうえで相手の不戦勝として一括で記録する。
-- 後で対戦相手が決まる試合(順位決定戦など)に入ったときに本部の「いまやること」に出すため、
-- 棄権の事実をチームごとに残す。チームを消したら一緒に消す。
CREATE TABLE team_withdrawals (
  team_id      BIGINT      PRIMARY KEY REFERENCES teams (id) ON DELETE CASCADE,
  withdrawn_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  withdrawn_by BIGINT      REFERENCES users (id) ON DELETE SET NULL
);

-- +goose Down
DROP TABLE team_withdrawals;
