-- Prevent destructive rollback when pending documents exist.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM documents
        WHERE source_url IS NULL
    ) THEN
        RAISE EXCEPTION
            'Cannot rollback: pending documents with NULL source_url exist';
    END IF;
END $$;

DROP INDEX IF EXISTS documents_pending_type_idx;

ALTER TABLE documents
    ALTER COLUMN source_url SET NOT NULL;

ALTER TABLE documents
    DROP COLUMN document_type;