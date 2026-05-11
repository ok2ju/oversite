-- Slice 13: duel-scoped mistakes.
--
-- A `Duel` is a directed attacker→victim engagement reconstructed from
-- the merged weapon_fire + player_hurt + kill stream. Every fire- or
-- kill-anchored mistake (shot_while_moving, no_counter_strafe,
-- missed_first_shot, spray_decay, slow_reaction, caught_reloading,
-- isolated_peek, repeated_death_zone, flash_assist) attaches to the
-- duel it occurred inside via analysis_mistakes.duel_id. Cross-duel
-- patterns (eco_misbuy, he_damage) carry duel_id = NULL and surface in
-- a separate "Patterns & highlights" UI section instead.
--
-- Persisted alongside analysis_mistakes in a single transaction; CASCADE
-- DELETE on demos so re-importing a demo wipes the duel rows.
CREATE TABLE analysis_duels (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    demo_id         INTEGER NOT NULL REFERENCES demos(id) ON DELETE CASCADE,
    round_number    INTEGER NOT NULL,
    round_id        INTEGER REFERENCES rounds(id) ON DELETE CASCADE,
    attacker_steam  TEXT    NOT NULL,
    victim_steam    TEXT    NOT NULL,
    start_tick      INTEGER NOT NULL,
    end_tick        INTEGER NOT NULL,
    outcome         TEXT    NOT NULL,
    end_reason      TEXT    NOT NULL,
    hit_confirmed   INTEGER NOT NULL DEFAULT 0,
    hurt_count      INTEGER NOT NULL DEFAULT 0,
    shot_count      INTEGER NOT NULL DEFAULT 0,
    mutual_duel_id  INTEGER REFERENCES analysis_duels(id) ON DELETE SET NULL,
    extras_json     TEXT    NOT NULL DEFAULT '{}',
    created_at      TEXT    NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_analysis_duels_demo_round
    ON analysis_duels(demo_id, round_number);
CREATE INDEX idx_analysis_duels_demo_attacker
    ON analysis_duels(demo_id, attacker_steam, start_tick);
CREATE INDEX idx_analysis_duels_demo_victim
    ON analysis_duels(demo_id, victim_steam, start_tick);

ALTER TABLE analysis_mistakes
    ADD COLUMN duel_id INTEGER REFERENCES analysis_duels(id) ON DELETE SET NULL;
CREATE INDEX idx_analysis_mistakes_duel ON analysis_mistakes(duel_id);
