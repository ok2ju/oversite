-- Drop tables in reverse order of creation (respect foreign key deps)
DROP TABLE IF EXISTS faceit_matches CASCADE;
DROP TABLE IF EXISTS grenade_lineups CASCADE;
DROP TABLE IF EXISTS strategy_boards CASCADE;
DROP TABLE IF EXISTS game_events CASCADE;
DROP TABLE IF EXISTS tick_data CASCADE;
DROP TABLE IF EXISTS player_rounds CASCADE;
DROP TABLE IF EXISTS rounds CASCADE;
DROP TABLE IF EXISTS demos CASCADE;
DROP TABLE IF EXISTS users CASCADE;

-- Drop extensions
DROP EXTENSION IF EXISTS "timescaledb";
DROP EXTENSION IF EXISTS "uuid-ossp";
