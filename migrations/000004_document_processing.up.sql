ALTER TABLE documents
ADD COLUMN download_attempts INTEGER NOT NULL DEFAULT 0,
ADD COLUMN download_started_at TIMESTAMPTZ,
ADD COLUMN last_error TEXT;

CREATE INDEX documents_download_queue_idx
ON documents (discovered_at)
WHERE status IN ('discovered', 'failed')
  AND source_url IS NOT NULL;

CREATE INDEX documents_download_stale_idx
ON documents (download_started_at)
WHERE status = 'downloading';