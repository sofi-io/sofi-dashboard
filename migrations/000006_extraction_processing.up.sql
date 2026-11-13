ALTER TABLE extractions
    ADD COLUMN submission_attempts INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN submission_started_at TIMESTAMPTZ,
    ADD COLUMN submission_token UUID;

ALTER TABLE extractions
    ADD COLUMN next_poll_at TIMESTAMPTZ,
    ADD COLUMN poll_started_at TIMESTAMPTZ,
    ADD COLUMN poll_token UUID,
    ADD COLUMN last_polled_at TIMESTAMPTZ;

ALTER TABLE extractions
    ADD CONSTRAINT extractions_submission_attempts_check
    CHECK (submission_attempts >= 0);