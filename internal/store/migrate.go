package store

const schemaSQLite = `
CREATE TABLE IF NOT EXISTS events (
  event_id INTEGER PRIMARY KEY,
  title TEXT NOT NULL,
  url TEXT NOT NULL DEFAULT '',
  location TEXT NOT NULL DEFAULT '',
  date_text TEXT NOT NULL DEFAULT '',
  brackets_unavailable INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS matches (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  event_id INTEGER NOT NULL,
  bracket_id INTEGER NOT NULL,
  division TEXT NOT NULL,
  gender TEXT NOT NULL DEFAULT '',
  belt TEXT NOT NULL DEFAULT '',
  style TEXT NOT NULL DEFAULT '',
  match_id INTEGER NOT NULL,
  round_name TEXT NOT NULL DEFAULT '',
  round INTEGER NOT NULL DEFAULT 0,
  mat_name TEXT NOT NULL DEFAULT '',
  mat_match_nr TEXT NOT NULL DEFAULT '',
  state TEXT NOT NULL DEFAULT '',
  is_bye INTEGER NOT NULL DEFAULT 0,
  won_by TEXT NOT NULL DEFAULT '',
  score_text TEXT NOT NULL DEFAULT '',
  penalty_text TEXT NOT NULL DEFAULT '',
  left_name TEXT NOT NULL DEFAULT '',
  left_club TEXT NOT NULL DEFAULT '',
  left_country TEXT NOT NULL DEFAULT '',
  left_user_id INTEGER NOT NULL DEFAULT 0,
  left_result TEXT NOT NULL DEFAULT '',
  right_name TEXT NOT NULL DEFAULT '',
  right_club TEXT NOT NULL DEFAULT '',
  right_country TEXT NOT NULL DEFAULT '',
  right_user_id INTEGER NOT NULL DEFAULT 0,
  right_result TEXT NOT NULL DEFAULT '',
  winner_side TEXT NOT NULL DEFAULT '',
  estimated_start TEXT NOT NULL DEFAULT '',
  registrations_count INTEGER NOT NULL DEFAULT 0,
  opponent_count INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_matches_event ON matches(event_id);
CREATE INDEX IF NOT EXISTS idx_matches_div ON matches(gender, belt, style);

CREATE TABLE IF NOT EXISTS placements (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  event_id INTEGER NOT NULL,
  bracket_id INTEGER NOT NULL,
  division TEXT NOT NULL,
  gender TEXT NOT NULL DEFAULT '',
  belt TEXT NOT NULL DEFAULT '',
  style TEXT NOT NULL DEFAULT '',
  placement INTEGER NOT NULL,
  user_id INTEGER NOT NULL DEFAULT 0,
  name TEXT NOT NULL DEFAULT '',
  club_name TEXT NOT NULL DEFAULT '',
  affiliation_name TEXT NOT NULL DEFAULT '',
  registrations_count INTEGER NOT NULL DEFAULT 0,
  opponent_count INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_placements_event ON placements(event_id);
CREATE INDEX IF NOT EXISTS idx_placements_div ON placements(gender, belt, style);
`

const schemaMySQL = `
CREATE TABLE IF NOT EXISTS events (
  event_id INT PRIMARY KEY,
  title TEXT NOT NULL,
  url TEXT NOT NULL DEFAULT '',
  location TEXT NOT NULL DEFAULT '',
  date_text TEXT NOT NULL DEFAULT '',
  brackets_unavailable INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS matches (
  id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  event_id INTEGER NOT NULL,
  bracket_id INTEGER NOT NULL,
  division TEXT NOT NULL,
  gender TEXT NOT NULL DEFAULT '',
  belt TEXT NOT NULL DEFAULT '',
  style TEXT NOT NULL DEFAULT '',
  match_id INTEGER NOT NULL,
  round_name TEXT NOT NULL DEFAULT '',
  round INTEGER NOT NULL DEFAULT 0,
  mat_name TEXT NOT NULL DEFAULT '',
  mat_match_nr TEXT NOT NULL DEFAULT '',
  state TEXT NOT NULL DEFAULT '',
  is_bye INTEGER NOT NULL DEFAULT 0,
  won_by TEXT NOT NULL DEFAULT '',
  score_text TEXT NOT NULL DEFAULT '',
  penalty_text TEXT NOT NULL DEFAULT '',
  left_name TEXT NOT NULL DEFAULT '',
  left_club TEXT NOT NULL DEFAULT '',
  left_country TEXT NOT NULL DEFAULT '',
  left_user_id INTEGER NOT NULL DEFAULT 0,
  left_result TEXT NOT NULL DEFAULT '',
  right_name TEXT NOT NULL DEFAULT '',
  right_club TEXT NOT NULL DEFAULT '',
  right_country TEXT NOT NULL DEFAULT '',
  right_user_id INTEGER NOT NULL DEFAULT 0,
  right_result TEXT NOT NULL DEFAULT '',
  winner_side TEXT NOT NULL DEFAULT '',
  estimated_start TEXT NOT NULL DEFAULT '',
  registrations_count INTEGER NOT NULL DEFAULT 0,
  opponent_count INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_matches_event ON matches(event_id);
CREATE INDEX IF NOT EXISTS idx_matches_div ON matches(gender, belt, style);

CREATE TABLE IF NOT EXISTS placements (
  id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  event_id INTEGER NOT NULL,
  bracket_id INTEGER NOT NULL,
  division TEXT NOT NULL,
  gender TEXT NOT NULL DEFAULT '',
  belt TEXT NOT NULL DEFAULT '',
  style TEXT NOT NULL DEFAULT '',
  placement INTEGER NOT NULL,
  user_id INTEGER NOT NULL DEFAULT 0,
  name TEXT NOT NULL DEFAULT '',
  club_name TEXT NOT NULL DEFAULT '',
  affiliation_name TEXT NOT NULL DEFAULT '',
  registrations_count INTEGER NOT NULL DEFAULT 0,
  opponent_count INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_placements_event ON placements(event_id);
CREATE INDEX IF NOT EXISTS idx_placements_div ON placements(gender, belt, style);
`
