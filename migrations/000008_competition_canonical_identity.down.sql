
DO $$
BEGIN
    RAISE EXCEPTION
        'Migration 000008 is irreversible automatically: competition records were consolidated.';
END $$;
