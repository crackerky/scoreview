-- +goose Up
-- スキーマの正本。データモデルの根拠は docs/architecture.md と docs/clone-spec/ を参照。

CREATE TABLE tournaments (
  id               BIGSERIAL PRIMARY KEY,
  name_ja          TEXT        NOT NULL,
  name_en          TEXT        NOT NULL DEFAULT '',
  -- 試合ページのパンくず(参照サイトではコメントアウト)に出る短い名前。例 "2026 Atami-B"
  short_name       TEXT        NOT NULL DEFAULT '',
  status           TEXT        NOT NULL CHECK (status IN ('entry_open', 'entry_closed', 'finished')),
  -- false の大会は一覧に出さないが、URL を直接開けば表示する(参照サイトの 538, 507 と同じ)
  listed           BOOLEAN     NOT NULL DEFAULT TRUE,
  -- 一覧の並び順(区分内で開催日の降順)に使う
  start_date       DATE,
  logo_url         TEXT,
  top_html_ja      TEXT        NOT NULL DEFAULT '',
  top_html_en      TEXT        NOT NULL DEFAULT '',
  -- ディスクゴルフ個人戦などでナビのラベルが「エントリー一覧」「成績」に変わる
  team_list_label  TEXT        NOT NULL DEFAULT 'teams'  CHECK (team_list_label IN ('teams', 'entries')),
  group_label      TEXT        NOT NULL DEFAULT 'groups' CHECK (group_label IN ('groups', 'results')),
  -- ナビに「スケジュール」と「ディビジョン成績」を出すか(参照サイトでは常に対で出る)
  show_schedule    BOOLEAN     NOT NULL DEFAULT TRUE,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX tournaments_list_idx ON tournaments (status, start_date DESC NULLS FIRST, id DESC) WHERE listed;

CREATE TABLE tournament_banners (
  id             BIGSERIAL PRIMARY KEY,
  tournament_id  BIGINT NOT NULL REFERENCES tournaments (id) ON DELETE CASCADE,
  image_url      TEXT   NOT NULL,
  link_url       TEXT,
  sort_order     INT    NOT NULL DEFAULT 0
);
CREATE INDEX tournament_banners_tournament_idx ON tournament_banners (tournament_id, sort_order);

CREATE TABLE divisions (
  id                   BIGSERIAL PRIMARY KEY,
  tournament_id        BIGINT  NOT NULL REFERENCES tournaments (id) ON DELETE CASCADE,
  name_ja              TEXT    NOT NULL,
  name_en              TEXT    NOT NULL DEFAULT '',
  -- 日程のディビジョン列の色(MC_MEN / MC_WOM / MC_MIX)。オープンや 4 人制は none
  kind                 TEXT    NOT NULL DEFAULT 'none' CHECK (kind IN ('men', 'women', 'mixed', 'none')),
  sort_order           INT     NOT NULL DEFAULT 0,
  -- 大会トップの [スケジュール] をリンクにするか(ディスクゴルフ等は false)
  schedule_enabled     BOOLEAN NOT NULL DEFAULT TRUE,
  -- 大会トップの [最終順位] をリンクにするか(順位確定後に true)
  standings_published  BOOLEAN NOT NULL DEFAULT FALSE,
  -- individual はディスクゴルフ個人戦。チーム一覧で名前をリンクにせず "Players" と数える
  entry_kind           TEXT    NOT NULL DEFAULT 'team' CHECK (entry_kind IN ('team', 'individual')),
  -- 選手ページの成績表の列構成
  stats_kind           TEXT    NOT NULL DEFAULT 'ultimate' CHECK (stats_kind IN ('ultimate', 'discgolf')),
  created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX divisions_tournament_idx ON divisions (tournament_id, sort_order, id);

CREATE TABLE groups (
  id                BIGSERIAL PRIMARY KEY,
  division_id       BIGINT  NOT NULL REFERENCES divisions (id) ON DELETE CASCADE,
  -- 参照サイトの表記ゆれ("Tounament A" 等)を原文のまま保持する
  name              TEXT    NOT NULL,
  kind              TEXT    NOT NULL CHECK (kind IN ('pool', 'bracket', 'image')),
  sort_order        INT     NOT NULL DEFAULT 0,
  -- false ならナビにリンクを出さない(名前だけ表示)。参照サイトの「成績」ラベルの大会と非公開グループ
  linked            BOOLEAN NOT NULL DEFAULT TRUE,
  -- false ならナビに出さない(参照サイトで欠番になっているグループ)
  published         BOOLEAN NOT NULL DEFAULT TRUE,
  bracket_pattern   TEXT,   -- 例 "T8A"
  bracket_ptn_name  TEXT,   -- 例 "2026 Atami Line =Tournament A="
  bracket_layout    JSONB,  -- 行と列の格子。形式は internal/domain/bracket.go
  image_url         TEXT,
  image_width       INT,
  image_height      INT,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK (kind <> 'bracket' OR bracket_layout IS NOT NULL),
  CHECK (kind <> 'image' OR image_url IS NOT NULL)
);
CREATE INDEX groups_division_idx ON groups (division_id, sort_order, id);

CREATE TABLE teams (
  id              BIGSERIAL PRIMARY KEY,
  tournament_id   BIGINT NOT NULL REFERENCES tournaments (id) ON DELETE CASCADE,
  division_id     BIGINT NOT NULL REFERENCES divisions (id) ON DELETE CASCADE,
  -- 短縮名は正式名の機械的な切り詰めでは作れないので別に持つ(docs/clone-spec/D 1.3)
  name_full_ja    TEXT   NOT NULL,
  name_short_ja   TEXT   NOT NULL,
  name_full_en    TEXT   NOT NULL DEFAULT '',
  name_short_en   TEXT   NOT NULL DEFAULT '',
  comment_ja      TEXT   NOT NULL DEFAULT '',
  comment_en      TEXT   NOT NULL DEFAULT '',
  -- チーム一覧の色帯(FTitle / FTitle_Mix / FTitle_Women / FTitle_Univ / FTitle_Waku1 / FTitle_Waku2)
  bar_style       TEXT   NOT NULL DEFAULT 'default'
                  CHECK (bar_style IN ('default', 'mixed', 'women', 'univ', 'waku1', 'waku2')),
  sort_order      INT    NOT NULL DEFAULT 0,
  -- ディビジョンの最終順位。順位決定戦の結果であり計算しない。同順位あり
  final_rank      INT    CHECK (final_rank IS NULL OR final_rank > 0),
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX teams_division_idx ON teams (division_id, sort_order, id);
CREATE INDEX teams_tournament_idx ON teams (tournament_id);

CREATE TABLE group_teams (
  group_id       BIGINT NOT NULL REFERENCES groups (id) ON DELETE CASCADE,
  team_id        BIGINT NOT NULL REFERENCES teams (id) ON DELETE CASCADE,
  sort_order     INT    NOT NULL DEFAULT 0,
  -- 計算した順位と異なる値を出す例外のための上書き。0 は「空欄で表示」
  rank_override  INT    CHECK (rank_override IS NULL OR rank_override >= 0),
  PRIMARY KEY (group_id, team_id)
);

CREATE TABLE players (
  id          BIGSERIAL PRIMARY KEY,
  team_id     BIGINT NOT NULL REFERENCES teams (id) ON DELETE CASCADE,
  -- "00" のような表記があるので文字列で持つ
  number      TEXT   NOT NULL DEFAULT '',
  name_ja     TEXT   NOT NULL,
  name_en     TEXT   NOT NULL DEFAULT '',
  gender      TEXT   CHECK (gender IN ('M', 'F')),
  -- NULL なら選手、値があればスタッフ(例 マネージャ / Manager)
  role_ja     TEXT,
  role_en     TEXT,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX players_team_idx ON players (team_id, id);

CREATE TABLE matches (
  id                         BIGSERIAL PRIMARY KEY,
  division_id                BIGINT NOT NULL REFERENCES divisions (id) ON DELETE CASCADE,
  group_id                   BIGINT REFERENCES groups (id) ON DELETE SET NULL,
  code                       TEXT   NOT NULL DEFAULT '',  -- 試合 CD。例 "A4"
  match_date                 DATE,
  start_time                 TIME,
  field_no                   INT,
  left_team_id               BIGINT REFERENCES teams (id) ON DELETE SET NULL,
  right_team_id              BIGINT REFERENCES teams (id) ON DELETE SET NULL,
  -- 未確定の対戦相手。"[A1-勝ち]" は code='A1', outcome='win'
  left_placeholder_code      TEXT,
  left_placeholder_outcome   TEXT CHECK (left_placeholder_outcome IN ('win', 'lose')),
  right_placeholder_code     TEXT,
  right_placeholder_outcome  TEXT CHECK (right_placeholder_outcome IN ('win', 'lose')),
  left_score                 INT CHECK (left_score IS NULL OR left_score >= 0),
  right_score                INT CHECK (right_score IS NULL OR right_score >= 0),
  status                     TEXT NOT NULL DEFAULT 'scheduled' CHECK (status IN ('scheduled', 'live', 'finished')),
  -- false の終了試合は日程の状況列が [対戦結果] リンクではなく「終了」になる
  has_detail                 BOOLEAN NOT NULL DEFAULT TRUE,
  -- 同点でも勝者が決まる試合(トーナメント)のための明示的な勝者
  winner_side                TEXT CHECK (winner_side IN ('left', 'right')),
  created_at                 TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at                 TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK (left_team_id IS NULL OR left_team_id <> right_team_id),
  CHECK ((left_placeholder_code IS NULL) = (left_placeholder_outcome IS NULL)),
  CHECK ((right_placeholder_code IS NULL) = (right_placeholder_outcome IS NULL))
);
CREATE INDEX matches_division_idx ON matches (division_id, match_date, start_time, field_no);
CREATE INDEX matches_group_idx ON matches (group_id);
CREATE INDEX matches_left_team_idx ON matches (left_team_id);
CREATE INDEX matches_right_team_idx ON matches (right_team_id);

CREATE TABLE match_events (
  id             BIGSERIAL PRIMARY KEY,
  match_id       BIGINT NOT NULL REFERENCES matches (id) ON DELETE CASCADE,
  -- 同じ経過時刻のイベントの順序
  seq            INT    NOT NULL,
  elapsed_sec    INT    NOT NULL CHECK (elapsed_sec >= 0),
  kind           TEXT   NOT NULL CHECK (kind IN ('goal', 'timeout', 'pull', 'end')),
  -- goal は得点した側、timeout は取った側、pull はプルした側。end は NULL
  side           TEXT   CHECK (side IN ('left', 'right')),
  scorer_id      BIGINT REFERENCES players (id) ON DELETE SET NULL,
  assist_id      BIGINT REFERENCES players (id) ON DELETE SET NULL,
  from_turnover  BOOLEAN NOT NULL DEFAULT FALSE,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (match_id, seq),
  CHECK (kind = 'end' OR side IS NOT NULL),
  CHECK (kind = 'goal' OR (scorer_id IS NULL AND assist_id IS NULL AND NOT from_turnover))
);

CREATE TABLE match_player_stats (
  match_id    BIGINT NOT NULL REFERENCES matches (id) ON DELETE CASCADE,
  player_id   BIGINT NOT NULL REFERENCES players (id) ON DELETE CASCADE,
  side        TEXT   NOT NULL CHECK (side IN ('left', 'right')),
  blocks      INT    NOT NULL DEFAULT 0 CHECK (blocks >= 0),
  callahans   INT    NOT NULL DEFAULT 0 CHECK (callahans >= 0),
  greatests   INT    NOT NULL DEFAULT 0 CHECK (greatests >= 0),
  PRIMARY KEY (match_id, player_id)
);
CREATE INDEX match_player_stats_player_idx ON match_player_stats (player_id);

CREATE TABLE galleries (
  id             BIGSERIAL PRIMARY KEY,
  tournament_id  BIGINT NOT NULL REFERENCES tournaments (id) ON DELETE CASCADE,
  name           TEXT   NOT NULL,
  sort_order     INT    NOT NULL DEFAULT 0
);
CREATE INDEX galleries_tournament_idx ON galleries (tournament_id, sort_order, id);

CREATE TABLE gallery_photos (
  id          BIGSERIAL PRIMARY KEY,
  gallery_id  BIGINT NOT NULL REFERENCES galleries (id) ON DELETE CASCADE,
  thumb_url   TEXT   NOT NULL,
  full_url    TEXT   NOT NULL,
  sort_order  INT    NOT NULL DEFAULT 0
);
CREATE INDEX gallery_photos_gallery_idx ON gallery_photos (gallery_id, sort_order, id);

CREATE TABLE users (
  id             BIGSERIAL PRIMARY KEY,
  email          TEXT NOT NULL UNIQUE,
  password_hash  TEXT NOT NULL,  -- argon2id の PHC 文字列
  display_name   TEXT NOT NULL DEFAULT '',
  role           TEXT NOT NULL CHECK (role IN ('admin', 'staff')),
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE sessions (
  id          BIGSERIAL PRIMARY KEY,
  -- トークンそのものは保存せず SHA-256 だけを持つ(DB が漏れてもセッションを乗っ取れない)
  token_hash  BYTEA       NOT NULL UNIQUE,
  user_id     BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  expires_at  TIMESTAMPTZ NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX sessions_user_idx ON sessions (user_id);

CREATE TABLE tournament_staff (
  user_id        BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  tournament_id  BIGINT NOT NULL REFERENCES tournaments (id) ON DELETE CASCADE,
  PRIMARY KEY (user_id, tournament_id)
);

-- +goose Down
DROP TABLE tournament_staff;
DROP TABLE sessions;
DROP TABLE users;
DROP TABLE gallery_photos;
DROP TABLE galleries;
DROP TABLE match_player_stats;
DROP TABLE match_events;
DROP TABLE matches;
DROP TABLE players;
DROP TABLE group_teams;
DROP TABLE teams;
DROP TABLE groups;
DROP TABLE divisions;
DROP TABLE tournament_banners;
DROP TABLE tournaments;
