-- Initial SQLite schema for Oversite desktop app.
-- 9 tables: users, demos, rounds, player_rounds, tick_data,
-- game_events, strategy_boards, grenade_lineups, faceit_matches.

-- ─────────────────────────────────────────────
-- Users
-- ─────────────────────────────────────────────
CREATE TABLE users (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    faceit_id       TEXT    NOT NULL UNIQUE,
    nickname        TEXT    NOT NULL,
    avatar_url      TEXT    NOT NULL DEFAULT '',
    faceit_elo      INTEGER NOT NULL DEFAULT 0,
    faceit_level    INTEGER NOT NULL DEFAULT 0,
    country         TEXT    NOT NULL DEFAULT '',
    created_at      TEXT    NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT    NOT NULL DEFAULT (datetime('now'))
);

-- ─────────────────────────────────────────────
-- Demos
-- ─────────────────────────────────────────────
CREATE TABLE demos (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id         INTEGER NOT NULL REFERENCES users(id),
    faceit_match_id TEXT,
    map_name        TEXT    NOT NULL,
    file_path       TEXT    NOT NULL,
    file_size       INTEGER NOT NULL,
    status          TEXT    NOT NULL DEFAULT 'imported',
    total_ticks     INTEGER NOT NULL DEFAULT 0,
    tick_rate       REAL    NOT NULL DEFAULT 0,
    duration_secs   INTEGER NOT NULL DEFAULT 0,
    match_date      TEXT    NOT NULL DEFAULT '',
    created_at      TEXT    NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_demos_user_id ON demos(user_id);
CREATE INDEX idx_demos_status ON demos(status);

-- ─────────────────────────────────────────────
-- Rounds
-- ─────────────────────────────────────────────
CREATE TABLE rounds (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    demo_id         INTEGER NOT NULL REFERENCES demos(id) ON DELETE CASCADE,
    round_number    INTEGER NOT NULL,
    start_tick      INTEGER NOT NULL,
    end_tick        INTEGER NOT NULL,
    winner_side     TEXT    NOT NULL,
    win_reason      TEXT    NOT NULL,
    ct_score        INTEGER NOT NULL DEFAULT 0,
    t_score         INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_rounds_demo_id ON rounds(demo_id);

-- ─────────────────────────────────────────────
-- Player Rounds
-- ─────────────────────────────────────────────
CREATE TABLE player_rounds (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    round_id        INTEGER NOT NULL REFERENCES rounds(id) ON DELETE CASCADE,
    steam_id        TEXT    NOT NULL,
    player_name     TEXT    NOT NULL,
    team_side       TEXT    NOT NULL,
    kills           INTEGER NOT NULL DEFAULT 0,
    deaths          INTEGER NOT NULL DEFAULT 0,
    assists         INTEGER NOT NULL DEFAULT 0,
    damage          INTEGER NOT NULL DEFAULT 0,
    headshot_kills  INTEGER NOT NULL DEFAULT 0,
    first_kill      INTEGER NOT NULL DEFAULT 0,
    first_death     INTEGER NOT NULL DEFAULT 0,
    clutch_kills    INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_player_rounds_round_id ON player_rounds(round_id);
CREATE INDEX idx_player_rounds_steam_id ON player_rounds(steam_id);

-- ─────────────────────────────────────────────
-- Tick Data (largest table; ~1.28M rows per demo)
-- ─────────────────────────────────────────────
CREATE TABLE tick_data (
    demo_id         INTEGER NOT NULL REFERENCES demos(id) ON DELETE CASCADE,
    tick            INTEGER NOT NULL,
    steam_id        TEXT    NOT NULL,
    x               REAL    NOT NULL,
    y               REAL    NOT NULL,
    z               REAL    NOT NULL,
    yaw             REAL    NOT NULL,
    health          INTEGER NOT NULL,
    armor           INTEGER NOT NULL,
    is_alive        INTEGER NOT NULL DEFAULT 1,
    weapon          TEXT    NOT NULL DEFAULT '',
    PRIMARY KEY (demo_id, tick, steam_id)
);

-- ─────────────────────────────────────────────
-- Game Events
-- ─────────────────────────────────────────────
CREATE TABLE game_events (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    demo_id             INTEGER NOT NULL REFERENCES demos(id) ON DELETE CASCADE,
    round_id            INTEGER NOT NULL REFERENCES rounds(id) ON DELETE CASCADE,
    tick                INTEGER NOT NULL,
    event_type          TEXT    NOT NULL,
    attacker_steam_id   TEXT,
    victim_steam_id     TEXT,
    weapon              TEXT,
    x                   REAL    NOT NULL,
    y                   REAL    NOT NULL,
    z                   REAL    NOT NULL,
    extra_data          TEXT    NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_game_events_demo_id ON game_events(demo_id);
CREATE INDEX idx_game_events_round_id ON game_events(round_id);
CREATE INDEX idx_game_events_type ON game_events(demo_id, event_type);

-- ─────────────────────────────────────────────
-- Strategy Boards
-- ─────────────────────────────────────────────
CREATE TABLE strategy_boards (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    title           TEXT    NOT NULL,
    map_name        TEXT    NOT NULL,
    board_state     TEXT    NOT NULL DEFAULT '{}',
    created_at      TEXT    NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT    NOT NULL DEFAULT (datetime('now'))
);

-- ─────────────────────────────────────────────
-- Grenade Lineups
-- ─────────────────────────────────────────────
CREATE TABLE grenade_lineups (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    demo_id         INTEGER REFERENCES demos(id) ON DELETE SET NULL,
    tick            INTEGER NOT NULL DEFAULT 0,
    map_name        TEXT    NOT NULL,
    grenade_type    TEXT    NOT NULL,
    throw_x         REAL    NOT NULL,
    throw_y         REAL    NOT NULL,
    throw_z         REAL    NOT NULL,
    throw_yaw       REAL    NOT NULL,
    throw_pitch     REAL    NOT NULL,
    land_x          REAL    NOT NULL,
    land_y          REAL    NOT NULL,
    land_z          REAL    NOT NULL,
    title           TEXT    NOT NULL DEFAULT '',
    description     TEXT    NOT NULL DEFAULT '',
    tags            TEXT    NOT NULL DEFAULT '[]',
    is_favorite     INTEGER NOT NULL DEFAULT 0,
    created_at      TEXT    NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_grenade_lineups_map ON grenade_lineups(map_name);
CREATE INDEX idx_grenade_lineups_type ON grenade_lineups(map_name, grenade_type);

-- ─────────────────────────────────────────────
-- Faceit Matches
-- ─────────────────────────────────────────────
CREATE TABLE faceit_matches (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id         INTEGER NOT NULL REFERENCES users(id),
    faceit_match_id TEXT    NOT NULL,
    map_name        TEXT    NOT NULL,
    score_team      INTEGER NOT NULL,
    score_opponent  INTEGER NOT NULL,
    result          TEXT    NOT NULL,
    elo_before      INTEGER NOT NULL DEFAULT 0,
    elo_after       INTEGER NOT NULL DEFAULT 0,
    kills           INTEGER NOT NULL DEFAULT 0,
    deaths          INTEGER NOT NULL DEFAULT 0,
    assists         INTEGER NOT NULL DEFAULT 0,
    demo_url        TEXT    NOT NULL DEFAULT '',
    demo_id         INTEGER REFERENCES demos(id) ON DELETE SET NULL,
    played_at       TEXT    NOT NULL,
    created_at      TEXT    NOT NULL DEFAULT (datetime('now')),
    UNIQUE(user_id, faceit_match_id)
);

CREATE INDEX idx_faceit_matches_user_id ON faceit_matches(user_id);
CREATE INDEX idx_faceit_matches_played_at ON faceit_matches(user_id, played_at);
