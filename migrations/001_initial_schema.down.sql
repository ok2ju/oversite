-- Reverse of 001_initial_schema.up.sql
-- Drop tables in reverse dependency order.

DROP TABLE IF EXISTS faceit_matches;
DROP TABLE IF EXISTS grenade_lineups;
DROP TABLE IF EXISTS strategy_boards;
DROP TABLE IF EXISTS game_events;
DROP TABLE IF EXISTS tick_data;
DROP TABLE IF EXISTS player_rounds;
DROP TABLE IF EXISTS rounds;
DROP TABLE IF EXISTS demos;
DROP TABLE IF EXISTS users;
