
ALTER TABLE documents
    ADD COLUMN document_type TEXT NOT NULL DEFAULT 'unknown';

ALTER TABLE documents
    ALTER COLUMN source_url DROP NOT NULL;

CREATE UNIQUE INDEX documents_pending_type_idx
    ON documents (match_id, source_key, document_type)
    WHERE source_url IS NULL;
