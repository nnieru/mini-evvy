DROP INDEX IF EXISTS uq_seat_bookings_barcode;
ALTER TABLE seat_bookings DROP COLUMN IF EXISTS barcode;

DROP INDEX IF EXISTS idx_jobs_status_created;
ALTER TABLE jobs DROP COLUMN IF EXISTS type;
