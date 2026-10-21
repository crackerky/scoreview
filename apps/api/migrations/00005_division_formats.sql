-- +goose Up
-- 大会形式エンジン(docs/format-engine.md「データモデルへの追加」)の保存先。既存の表の意味は変えず、
-- 列と表を足すだけ。取込ツールの大会にはここの行がなく、これまでどおり動く。

-- 1. ディビジョンの形の指定。生成物(段、グループ、枠、試合)は template と params から決定的に
--    作り直せるので、試合番号やグループ文字の対応はここから復元する(store/format_state.go)。
--    params の JSON 形は docs/format-engine.md「形の指定の JSON」。
--    tiebreak: current = 現行方式(直接対決あり)/ current_no_h2h = 直接対決なし / jfda = JFDA 競技規則
--    band_rule: 段の全体順位の基準。per_game = 1 試合あたりの勝ち点 → 得失点差 → 得点(推測)
--    schedule: 時間割の設定 {date, start, slotMinutes, courts, teamRestSlots, stageGapSlots, end}
--    version: 「適用」のたびに 1 つ増える(作り直したことを画面で見分けるため)
CREATE TABLE division_formats (
  division_id           BIGINT      PRIMARY KEY REFERENCES divisions (id) ON DELETE CASCADE,
  template              TEXT        NOT NULL CHECK (template IN ('pool_bands', 'blocks', 'round_robin')),
  params                JSONB       NOT NULL,
  start_letter          TEXT        NOT NULL CHECK (start_letter ~ '^[A-Z]+$'),
  tiebreak              TEXT        NOT NULL DEFAULT 'current'
                        CHECK (tiebreak IN ('current', 'current_no_h2h', 'jfda')),
  band_rule             TEXT        NOT NULL DEFAULT 'per_game' CHECK (band_rule IN ('per_game')),
  -- 没収試合の規定スコア(順位計算に使う)。運営会社に未確認なので初期値は設計書の例の 15-0
  forfeit_winner_score  INT         NOT NULL DEFAULT 15,
  forfeit_loser_score   INT         NOT NULL DEFAULT 0,
  schedule              JSONB,
  version               INT         NOT NULL DEFAULT 1 CHECK (version > 0),
  created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK (forfeit_loser_score >= 0 AND forfeit_winner_score > forfeit_loser_score)
);

-- 2. 段。confirmed_at が入ると段の全体順位(band_rank)を出どころにする枠が埋まる。
--    confirmed_unplayed は「未消化のまま確定」したこと(時間切れ)。
CREATE TABLE stages (
  id                  BIGSERIAL   PRIMARY KEY,
  division_id         BIGINT      NOT NULL REFERENCES divisions (id) ON DELETE CASCADE,
  stage_index         INT         NOT NULL CHECK (stage_index >= 0),
  name                TEXT        NOT NULL,
  confirmed_at        TIMESTAMPTZ,
  confirmed_unplayed  BOOLEAN     NOT NULL DEFAULT FALSE,
  UNIQUE (division_id, stage_index)
);

-- 3. グループが属する段と、形の中の序数(Plan.Groups の添字)。プールの確定もここに持つ。
ALTER TABLE groups ADD COLUMN stage_id BIGINT REFERENCES stages (id) ON DELETE SET NULL;
ALTER TABLE groups ADD COLUMN format_index INT CHECK (format_index IS NULL OR format_index >= 0);
ALTER TABLE groups ADD COLUMN confirmed_at TIMESTAMPTZ;
ALTER TABLE groups ADD COLUMN confirmed_unplayed BOOLEAN NOT NULL DEFAULT FALSE;
CREATE INDEX groups_stage_idx ON groups (stage_id);

-- 4. 枠。source は出どころ({"kind":"seed","seed":3}、{"kind":"group_rank","group":0,"rank":1} など。
--    group は形の中のグループの序数)。team_id は確定したチーム(未確定は NULL)。
CREATE TABLE group_slots (
  id        BIGSERIAL PRIMARY KEY,
  group_id  BIGINT    NOT NULL REFERENCES groups (id) ON DELETE CASCADE,
  position  INT       NOT NULL CHECK (position > 0),
  source    JSONB     NOT NULL,
  team_id   BIGINT    REFERENCES teams (id) ON DELETE SET NULL,
  UNIQUE (group_id, position)
);

-- 5. 試合の左右がどの枠か。前の試合の勝者・敗者を待つ側は従来どおり *_placeholder_* で表す。
ALTER TABLE matches ADD COLUMN left_slot_id BIGINT REFERENCES group_slots (id) ON DELETE SET NULL;
ALTER TABLE matches ADD COLUMN right_slot_id BIGINT REFERENCES group_slots (id) ON DELETE SET NULL;
CREATE INDEX matches_left_slot_idx ON matches (left_slot_id);
CREATE INDEX matches_right_slot_idx ON matches (right_slot_id);

-- 6. 規程の基準で決まらない同順位を運営者が決めた記録(ブリックマーク投げなど)。グループの順位か、
--    段の全体順位(stage_id と band_rank)のどちらか。team_ids は上位から並べたチーム ID の配列。
CREATE TABLE group_rank_decisions (
  id          BIGSERIAL   PRIMARY KEY,
  group_id    BIGINT      REFERENCES groups (id) ON DELETE CASCADE,
  stage_id    BIGINT      REFERENCES stages (id) ON DELETE CASCADE,
  band_rank   INT         CHECK (band_rank IS NULL OR band_rank > 0),
  team_ids    JSONB       NOT NULL,
  reason      TEXT        NOT NULL,
  decided_by  BIGINT      REFERENCES users (id) ON DELETE SET NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK ((group_id IS NULL) <> (stage_id IS NULL)),
  CHECK ((stage_id IS NULL) = (band_rank IS NULL))
);
CREATE INDEX group_rank_decisions_group_idx ON group_rank_decisions (group_id, id);
CREATE INDEX group_rank_decisions_stage_idx ON group_rank_decisions (stage_id, id);

-- +goose Down
DROP TABLE group_rank_decisions;
DROP INDEX matches_right_slot_idx;
DROP INDEX matches_left_slot_idx;
ALTER TABLE matches DROP COLUMN right_slot_id;
ALTER TABLE matches DROP COLUMN left_slot_id;
DROP TABLE group_slots;
DROP INDEX groups_stage_idx;
ALTER TABLE groups DROP COLUMN confirmed_unplayed;
ALTER TABLE groups DROP COLUMN confirmed_at;
ALTER TABLE groups DROP COLUMN format_index;
ALTER TABLE groups DROP COLUMN stage_id;
DROP TABLE stages;
DROP TABLE division_formats;
