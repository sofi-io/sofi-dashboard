CREATE UNIQUE INDEX extractions_active_document_provider_uidx
ON extractions (document_id, provider)
WHERE status IN (
    'pending',
    'submitting',
    'submitted',
    'processing',
    'submission_unknown'
);