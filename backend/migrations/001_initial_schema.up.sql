-- Extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "timescaledb";

-- ===================
-- Users
-- ===================
CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    faceit_id       VARCHAR(64) NOT NULL UNIQUE,
    nickname        VARCHAR(64) NOT NULL,
    avatar_url      TEXT,
    faceit_elo      INTEGER DEFAULT 0,
    faceit_level    SMALLINT DEFAULT 1,
    country         VARCHAR(2),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_faceit_id ON users (faceit_id);

-- ===================
-- Demos
-- ===================
CREATE TABLE demos (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    faceit_match_id VARCHAR(64),
    map_name        VARCHAR(32) NOT NULL,
    file_path       TEXT NOT NULL,
    file_size       BIGINT NOT NULL,
    status          VARCHAR(16) NOT NULL DEFAULT 'uploaded',
    total_ticks     INTEGER,
    tick_rate       REAL,
    duration_secs   INTEGER,
    match_date      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_demos_user_id ON demos (user_id);
CREATE INDEX idx_demos_status ON demos (status);
CREATE INDEX idx_demos_map ON demos (map_name);

-- ===================
-- Rounds
-- ===================
CREATE TABLE rounds (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    demo_id         UUID NOT NULL REFERENCES demos(id) ON DELETE CASCADE,
    round_number    SMALLINT NOT NULL,
    start_tick      INTEGER NOT NULL,
    end_tick        INTEGER NOT NULL,
    winner_side     VARCHAR(2) NOT NULL,
    win_reason      VARCHAR(32) NOT NULL,
    ct_score        SMALLINT NOT NULL,
    t_score         SMALLINT NOT NULL,

    UNIQUE (demo_id, round_number)
);

CREATE INDEX idx_rounds_demo_id ON rounds (demo_id);

-- ===================
-- Player Rounds
-- ===================
CREATE TABLE player_rounds (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    round_id        UUID NOT NULL REFERENCES rounds(id) ON DELETE CASCADE,
    steam_id        VARCHAR(20) NOT NULL,
    player_name     VARCHAR(64) NOT NULL,
    team_side       VARCHAR(2) NOT NULL,
    kills           SMALLINT NOT NULL DEFAULT 0,
    deaths          SMALLINT NOT NULL DEFAULT 0,
    assists         SMALLINT NOT NULL DEFAULT 0,
    damage          INTEGER NOT NULL DEFAULT 0,
    headshot_kills  SMALLINT NOT NULL DEFAULT 0,
    first_kill      BOOLEAN NOT NULL DEFAULT FALSE,
    first_death     BOOLEAN NOT NULL DEFAULT FALSE,
    clutch_kills    SMALLINT NOT NULL DEFAULT 0,

    UNIQUE (round_id, steam_id)
);

CREATE INDEX idx_player_rounds_round_id ON player_rounds (round_id);
CREATE INDEX idx_player_rounds_steam_id ON player_rounds (steam_id);

-- ===================
-- Tick Data (TimescaleDB Hypertable)
-- ===================
CREATE TABLE tick_data (
    time            TIMESTAMPTZ NOT NULL,
    demo_id         UUID NOT NULL,
    tick            INTEGER NOT NULL,
    steam_id        VARCHAR(20) NOT NULL,
    x               REAL NOT NULL,
    y               REAL NOT NULL,
    z               REAL NOT NULL,
    yaw             REAL NOT NULL,
    health          SMALLINT NOT NULL,
    armor           SMALLINT NOT NULL,
    is_alive        BOOLEAN NOT NULL,
    weapon          VARCHAR(32)
);

SELECT create_hypertable('tick_data', 'time');

CREATE INDEX idx_tick_data_demo_tick ON tick_data (demo_id, tick);
CREATE INDEX idx_tick_data_steam_id ON tick_data (steam_id, time DESC);

-- ===================
-- Game Events
-- ===================
CREATE TABLE game_events (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    demo_id             UUID NOT NULL REFERENCES demos(id) ON DELETE CASCADE,
    round_id            UUID REFERENCES rounds(id) ON DELETE CASCADE,
    tick                INTEGER NOT NULL,
    event_type          VARCHAR(32) NOT NULL,
    attacker_steam_id   VARCHAR(20),
    victim_steam_id     VARCHAR(20),
    weapon              VARCHAR(32),
    x                   REAL,
    y                   REAL,
    z                   REAL,
    extra_data          JSONB
);

CREATE INDEX idx_game_events_demo_id ON game_events (demo_id);
CREATE INDEX idx_game_events_type ON game_events (event_type);
CREATE INDEX idx_game_events_demo_round ON game_events (demo_id, round_id);

-- ===================
-- Strategy Boards
-- ===================
CREATE TABLE strategy_boards (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title           VARCHAR(128) NOT NULL,
    map_name        VARCHAR(32) NOT NULL,
    yjs_state       BYTEA,
    share_mode      VARCHAR(16) NOT NULL DEFAULT 'private',
    share_token     VARCHAR(64) UNIQUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_strat_boards_user_id ON strategy_boards (user_id);
CREATE INDEX idx_strat_boards_share_token ON strategy_boards (share_token);

-- ===================
-- Grenade Lineups
-- ===================
CREATE TABLE grenade_lineups (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    demo_id         UUID REFERENCES demos(id) ON DELETE SET NULL,
    tick            INTEGER,
    map_name        VARCHAR(32) NOT NULL,
    grenade_type    VARCHAR(16) NOT NULL,
    throw_x         REAL NOT NULL,
    throw_y         REAL NOT NULL,
    throw_z         REAL NOT NULL,
    throw_yaw       REAL NOT NULL,
    throw_pitch     REAL NOT NULL,
    land_x          REAL NOT NULL,
    land_y          REAL NOT NULL,
    land_z          REAL NOT NULL,
    title           VARCHAR(128) NOT NULL,
    description     TEXT,
    tags            TEXT[],
    is_favorite     BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_lineups_user_id ON grenade_lineups (user_id);
CREATE INDEX idx_lineups_map_type ON grenade_lineups (map_name, grenade_type);

-- ===================
-- Faceit Matches
-- ===================
CREATE TABLE faceit_matches (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    faceit_match_id VARCHAR(64) NOT NULL,
    map_name        VARCHAR(32) NOT NULL,
    score_team      SMALLINT NOT NULL,
    score_opponent  SMALLINT NOT NULL,
    result          VARCHAR(4) NOT NULL,
    elo_before      INTEGER,
    elo_after       INTEGER,
    kills           SMALLINT,
    deaths          SMALLINT,
    assists         SMALLINT,
    demo_url        TEXT,
    demo_id         UUID REFERENCES demos(id) ON DELETE SET NULL,
    played_at       TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (user_id, faceit_match_id)
);

CREATE INDEX idx_faceit_matches_user_id ON faceit_matches (user_id, played_at DESC);

-- ===================
-- TimescaleDB Configuration
-- ===================
SELECT set_chunk_time_interval('tick_data', INTERVAL '1 day');

ALTER TABLE tick_data SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'demo_id',
    timescaledb.compress_orderby = 'tick ASC, steam_id ASC'
);

SELECT add_compression_policy('tick_data', INTERVAL '7 days');
