-- +goose Up
-- 大会への申込(docs/line-integration.md「エントリーの流れ」)。申込は teams と別に持ち、運営が承認した
-- ときに初めて teams と players の行を作る。teams は大会の形と組分けが読むので、承認前や取り消した
-- 申込が混じると組の数やシードが狂うため。

-- 大会ごとの質問。scope は team(代表が答える)か member(メンバーが各自答える)。kind は
-- short_text / long_text / choice / number / yes_no。options は choice の選択肢(文字列の配列)。
CREATE TABLE entry_questions (
  id             BIGSERIAL PRIMARY KEY,
  tournament_id  BIGINT      NOT NULL REFERENCES tournaments (id) ON DELETE CASCADE,
  scope          TEXT        NOT NULL CHECK (scope IN ('team', 'member')),
  label          TEXT        NOT NULL,
  kind           TEXT        NOT NULL CHECK (kind IN ('short_text', 'long_text', 'choice', 'number', 'yes_no')),
  required       BOOLEAN     NOT NULL DEFAULT FALSE,
  options        JSONB       NOT NULL DEFAULT '[]',
  sort_order     INT         NOT NULL DEFAULT 0,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX entry_questions_tournament_idx ON entry_questions (tournament_id, sort_order, id);

-- 代表エントリー(チームについての申込)。status は pending(申込中)/ returned(差し戻し)/
-- approved(承認。team_id が入る)/ cancelled(取消)。channel は申込を受けた経路で、line(本人が LINE
-- から)か desk(電話や紙の申込を運営が代わりに入れた)。rep_user_id は LINE で申し込んだ代表で、
-- LINE の本人確認を作るまでは空。
CREATE TABLE team_entries (
  id               BIGSERIAL PRIMARY KEY,
  tournament_id    BIGINT      NOT NULL REFERENCES tournaments (id) ON DELETE CASCADE,
  division_id      BIGINT      NOT NULL REFERENCES divisions (id) ON DELETE CASCADE,
  -- チーム名は加工せずに保存する(後で大会をまたぐ clubs に束ねるため。scoreview-design 8)
  team_name_full   TEXT        NOT NULL,
  team_name_short  TEXT        NOT NULL,
  rep_name         TEXT        NOT NULL,
  rep_phone        TEXT        NOT NULL DEFAULT '',
  rep_email        TEXT        NOT NULL DEFAULT '',
  rep_user_id      BIGINT      REFERENCES users (id) ON DELETE SET NULL,
  channel          TEXT        NOT NULL CHECK (channel IN ('line', 'desk')),
  status           TEXT        NOT NULL DEFAULT 'pending'
                               CHECK (status IN ('pending', 'returned', 'approved', 'cancelled')),
  -- 差し戻しの理由(代表に見せる)。差し戻し以外では空
  return_note      TEXT        NOT NULL DEFAULT '',
  -- メンバーが自分の LINE で登録するための招待コード。推測できない長さの乱数
  invite_code      TEXT        NOT NULL UNIQUE,
  team_id          BIGINT      REFERENCES teams (id) ON DELETE SET NULL,
  submitted_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  decided_at       TIMESTAMPTZ,
  decided_by       BIGINT      REFERENCES users (id) ON DELETE SET NULL,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK (status <> 'approved' OR decided_at IS NOT NULL)
);
CREATE INDEX team_entries_tournament_idx ON team_entries (tournament_id, submitted_at, id);
-- 同じ部門に同じチーム名の申込を二重に入れない(確認の画面から戻って送り直した、など)。取り消した
-- 申込は数えない(取り消してから出し直せるように)。比べるときは前後の空白と大文字小文字を無視する
CREATE UNIQUE INDEX team_entries_name_uniq ON team_entries (division_id, lower(btrim(team_name_full)))
  WHERE status <> 'cancelled';

-- メンバーエントリー。代表が書いた行は user_id が空で、後からメンバー本人が招待リンクで自分の LINE を
-- 結びつける。player_id は承認で作った選手。
CREATE TABLE member_entries (
  id             BIGSERIAL PRIMARY KEY,
  team_entry_id  BIGINT      NOT NULL REFERENCES team_entries (id) ON DELETE CASCADE,
  user_id        BIGINT      REFERENCES users (id) ON DELETE SET NULL,
  name           TEXT        NOT NULL,
  -- "00" のような表記があるので文字列で持つ(players.number と同じ)
  number         TEXT        NOT NULL DEFAULT '',
  player_id      BIGINT      REFERENCES players (id) ON DELETE SET NULL,
  sort_order     INT         NOT NULL DEFAULT 0,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX member_entries_team_entry_idx ON member_entries (team_entry_id, sort_order, id);

-- 質問への答え。代表エントリーかメンバーエントリーのどちらか一方に付く。
CREATE TABLE entry_answers (
  id               BIGSERIAL PRIMARY KEY,
  question_id      BIGINT NOT NULL REFERENCES entry_questions (id) ON DELETE CASCADE,
  team_entry_id    BIGINT REFERENCES team_entries (id) ON DELETE CASCADE,
  member_entry_id  BIGINT REFERENCES member_entries (id) ON DELETE CASCADE,
  value            TEXT   NOT NULL,
  CHECK ((team_entry_id IS NULL) <> (member_entry_id IS NULL)),
  UNIQUE (question_id, team_entry_id),
  UNIQUE (question_id, member_entry_id)
);
CREATE INDEX entry_answers_team_entry_idx ON entry_answers (team_entry_id) WHERE team_entry_id IS NOT NULL;
CREATE INDEX entry_answers_member_entry_idx ON entry_answers (member_entry_id)
  WHERE member_entry_id IS NOT NULL;

-- +goose Down
DROP TABLE entry_answers;
DROP TABLE member_entries;
DROP TABLE team_entries;
DROP TABLE entry_questions;
