ALTER TABLE seat_bookings
    DROP COLUMN IF EXISTS invitation_email_sent_at,
    DROP COLUMN IF EXISTS invitation_email_status;
