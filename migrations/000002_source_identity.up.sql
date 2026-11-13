
-- Competition source identities
CREATE TABLE competition_sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    competition_id UUID NOT NULL REFERENCES competitions(id) ON DELETE CASCADE,
    source_key TEXT NOT NULL,
    external_id TEXT NOT NULL,
    source_url TEXT NOT NULL,
    discovered_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (source_key, external_id),
    UNIQUE (competition_id, source_key)
);

-- Season source identities
CREATE TABLE season_sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    season_id UUID NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    source_key TEXT NOT NULL,
    external_id TEXT NOT NULL,
    source_url TEXT,
    discovered_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (source_key, external_id),
    UNIQUE (season_id, source_key)
);

-- Club source identities
CREATE TABLE club_sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    club_id UUID NOT NULL REFERENCES clubs(id) ON DELETE CASCADE,
    source_key TEXT NOT NULL,
    external_id TEXT NOT NULL,
    source_url TEXT,
    discovered_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (source_key, external_id),
    UNIQUE (club_id, source_key)
);

-- Canonical competition phases
CREATE TABLE phases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    season_id UUID NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    phase_order INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (season_id, name)
);

-- Phase source identities
CREATE TABLE phase_sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phase_id UUID NOT NULL REFERENCES phases(id) ON DELETE CASCADE,
    source_key TEXT NOT NULL,
    external_id TEXT NOT NULL,
    source_url TEXT,
    discovered_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (source_key, external_id),
    UNIQUE (phase_id, source_key)
);

-- Match metadata
ALTER TABLE matches
    ADD COLUMN phase_id UUID REFERENCES phases(id),
    ADD COLUMN round_number INTEGER,
    ADD COLUMN source_date_text TEXT,
    ADD COLUMN source_time_text TEXT;

CREATE INDEX matches_phase_round_idx
    ON matches(phase_id, round_number);
