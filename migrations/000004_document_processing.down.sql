DROP INDEX IF EXISTS documents_download_stale_idx;
DROP INDEX IF EXISTS documents_download_queue_idx;

ALTER TABLE documents
DROP COLUMN IF EXISTS last_error,
DROP COLUMN IF EXISTS download_started_at,
DROP COLUMN IF EXISTS download_attempts;