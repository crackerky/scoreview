-- +goose Up
-- 記録係の画面と本部の記録(docs/operations/operator-experience.md「記録係:間違えにくく、直しやすい入力」
-- 「紙を上回ることが前提」「人手が足りない現場を前提にする」)のための列と表。列と表の追加だけで、
-- 既存の行の意味は変えない。

-- 1. 試合の結果の種類(docs/format-engine.md「試合の結果の種類」)。既存の試合はすべて通常の試合。
--    normal: スコアで勝敗 / forfeit: 不戦勝(規定スコアで記録) / double_forfeit: 両チーム不戦敗
--    no_contest: 不成立(勝敗だけ) / abandoned: 途中打ち切り(その時点のスコアで確定)
--    unplayed: 未消化(集計に入れない。途中打ち切りで未消化にしたものも含む)
ALTER TABLE matches ADD COLUMN result_type TEXT NOT NULL DEFAULT 'normal'
  CHECK (result_type IN ('normal', 'forfeit', 'double_forfeit', 'no_contest', 'abandoned', 'unplayed'));

-- 2. 没収試合(不戦勝)の規定スコア。大会の設定で、運営会社に未確認なので既定値を持たない(NULL は未設定。
--    未設定の大会では不戦勝を記録するときにスコアの入力を求める)。
ALTER TABLE tournaments ADD COLUMN forfeit_winner_score INT;
ALTER TABLE tournaments ADD COLUMN forfeit_loser_score INT;
ALTER TABLE tournaments ADD CONSTRAINT tournaments_forfeit_score_check CHECK (
  (forfeit_winner_score IS NULL AND forfeit_loser_score IS NULL)
  OR (forfeit_loser_score >= 0 AND forfeit_winner_score > forfeit_loser_score)
);

-- 3. 得点経過の、端末が作った ID。通信が切れている間に記録した操作を後から送り直しても二重に
--    登録しないため(冪等)と、まだ api の id を知らない得点経過を端末から指すために使う。
--    端末を通らない行(取込ツール、JavaScript なしの入力)は DB が作る。
ALTER TABLE match_events ADD COLUMN client_id UUID NOT NULL DEFAULT gen_random_uuid();
ALTER TABLE match_events ADD CONSTRAINT match_events_client_id_key UNIQUE (client_id);

-- 4. 監査ログ(docs/format-engine.md「監査ログ」)。誰がいつ何を何から何へ変えたか。
--    1 回の操作を 1 行にし、before / after に変更前後の値(JSON)を持つ。client_id は端末が付けた
--    操作の ID で、同じ ID の操作は 2 回目を適用しない(送り直しの冪等性)。
--    試合、チーム、利用者には外部キーを張らない(削除されても「誰が何を」の ID を消さずに残すため)。
--    大会を消したら一緒に消す。
CREATE TABLE audit_log (
  id             BIGSERIAL PRIMARY KEY,
  tournament_id  BIGINT      NOT NULL REFERENCES tournaments (id) ON DELETE CASCADE,
  match_id       BIGINT,
  team_id        BIGINT,
  user_id        BIGINT,
  action         TEXT        NOT NULL,
  before         JSONB,
  after          JSONB,
  client_id      UUID        UNIQUE,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX audit_log_match_idx ON audit_log (match_id, id);
CREATE INDEX audit_log_tournament_idx ON audit_log (tournament_id, id);

-- 5. 記録係の割り当て。試合を 1 つ指すか、コートと日付と時間帯で範囲を指す(NULL はその条件なし。
--    全部 NULL なら大会の全試合)。時間帯は start_time 以上 end_time 未満で、試合の開始時刻で判定する。
CREATE TABLE staff_assignments (
  id             BIGSERIAL PRIMARY KEY,
  tournament_id  BIGINT      NOT NULL REFERENCES tournaments (id) ON DELETE CASCADE,
  user_id        BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  match_id       BIGINT      REFERENCES matches (id) ON DELETE CASCADE,
  field          TEXT,
  match_date     DATE,
  start_time     TIME,
  end_time       TIME,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK (match_id IS NULL OR (field IS NULL AND match_date IS NULL AND start_time IS NULL AND end_time IS NULL)),
  CHECK (start_time IS NULL OR end_time IS NULL OR start_time < end_time)
);
CREATE INDEX staff_assignments_user_idx ON staff_assignments (user_id, tournament_id);

-- +goose Down
DROP TABLE staff_assignments;
DROP TABLE audit_log;
ALTER TABLE match_events DROP CONSTRAINT match_events_client_id_key;
ALTER TABLE match_events DROP COLUMN client_id;
ALTER TABLE tournaments DROP CONSTRAINT tournaments_forfeit_score_check;
ALTER TABLE tournaments DROP COLUMN forfeit_loser_score;
ALTER TABLE tournaments DROP COLUMN forfeit_winner_score;
ALTER TABLE matches DROP COLUMN result_type;
