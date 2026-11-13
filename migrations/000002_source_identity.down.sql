
DROP INDEX IF EXISTS matches_phase_round_idx;

ALTER TABLE matches
    DROP COLUMN IF EXISTS source_time_text,
    DROP COLUMN IF EXISTS source_date_text,
    DROP COLUMN IF EXISTS round_number,
    DROP COLUMN IF EXISTS phase_id;

DROP TABLE IF EXISTS phase_sources;
DROP TABLE IF EXISTS phases;
DROP TABLE IF EXISTS club_sources;
DROP TABLE IF EXISTS season_sources;
DROP TABLE IF EXISTS competition_sources;
