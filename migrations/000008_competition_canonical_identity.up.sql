
BEGIN;

-- 1. Add canonical identity.
ALTER TABLE competitions
    ADD COLUMN canonical_key TEXT;

-- 2. Generate canonical keys.
-- Example: fpf:pernambucano-a1
UPDATE competitions c
SET canonical_key = (
    COALESCE(
        (
            SELECT MIN(cs.source_key)
            FROM competition_sources cs
            WHERE cs.competition_id = c.id
        ),
        'legacy'
    )
    || ':'
    || CASE
        WHEN EXISTS (
            SELECT 1
            FROM competition_sources cs
            WHERE cs.competition_id = c.id
        )
        THEN TRIM(BOTH '-' FROM REGEXP_REPLACE(
            REGEXP_REPLACE(
                LOWER(TRIM(c.name)),
                '[[:space:]]*/[[:space:]]*[0-9]{4}[[:space:]]*$',
                ''
            ),
            '[^a-z0-9]+',
            '-',
            'g'
        ))
        ELSE c.slug
    END
);

-- 3. Build duplicate competition mapping.
CREATE TEMP TABLE competition_merge_map
ON COMMIT DROP AS
WITH ranked AS (
    SELECT
        id,
        canonical_key,
        FIRST_VALUE(id) OVER (
            PARTITION BY canonical_key
            ORDER BY created_at, id
        ) AS canonical_id,
        ROW_NUMBER() OVER (
            PARTITION BY canonical_key
            ORDER BY created_at, id
        ) AS position
    FROM competitions
)
SELECT
    id AS duplicate_id,
    canonical_id
FROM ranked
WHERE position > 1;

-- 4. Prevent season name conflicts before merging.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM seasons s1
        JOIN competition_merge_map m
            ON m.duplicate_id = s1.competition_id
        JOIN seasons s2
            ON s2.competition_id = m.canonical_id
           AND s2.name = s1.name
    ) THEN
        RAISE EXCEPTION
            'Cannot merge competitions: duplicate season names detected';
    END IF;
END $$;

-- 5. Allow multiple source identities per competition.
ALTER TABLE competition_sources
    DROP CONSTRAINT IF EXISTS
    competition_sources_competition_id_source_key_key;

-- 6. Move seasons.
UPDATE seasons s
SET competition_id = m.canonical_id
FROM competition_merge_map m
WHERE s.competition_id = m.duplicate_id;

-- 7. Move external competition identities.
UPDATE competition_sources cs
SET competition_id = m.canonical_id
FROM competition_merge_map m
WHERE cs.competition_id = m.duplicate_id;

-- 8. Delete duplicate competitions.
DELETE FROM competitions c
USING competition_merge_map m
WHERE c.id = m.duplicate_id;

-- 9. Normalize canonical display names.
UPDATE competitions
SET name = REGEXP_REPLACE(
    TRIM(name),
    '[[:space:]]*/[[:space:]]*[0-9]{4}[[:space:]]*$',
    ''
);

-- 10. Enforce canonical identity.
ALTER TABLE competitions
    ALTER COLUMN canonical_key SET NOT NULL;

CREATE UNIQUE INDEX competitions_canonical_key_uidx
    ON competitions(canonical_key);

COMMIT;
