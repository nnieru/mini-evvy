ALTER TABLE jobs ADD COLUMN type varchar NOT NULL DEFAULT 'unknown';
ALTER TABLE jobs ALTER COLUMN type DROP DEFAULT;

CREATE INDEX idx_jobs_status_created ON jobs(status, created_at);

ALTER TABLE seat_bookings ADD COLUMN barcode varchar;

CREATE UNIQUE INDEX uq_seat_bookings_barcode
    ON seat_bookings(barcode) WHERE barcode IS NOT NULL AND deleted_at IS NULL;
