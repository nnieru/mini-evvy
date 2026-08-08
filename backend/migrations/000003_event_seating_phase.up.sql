ALTER TABLE events
    ADD COLUMN seating_phase varchar NOT NULL DEFAULT 'open'
        CHECK (seating_phase IN ('open', 'preview', 'approved'));
