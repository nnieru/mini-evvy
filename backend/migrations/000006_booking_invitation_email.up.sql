ALTER TABLE seat_bookings
    ADD COLUMN invitation_email_status varchar NOT NULL DEFAULT 'not_sent'
        CHECK (invitation_email_status IN ('not_sent', 'pending', 'sent', 'failed')),
    ADD COLUMN invitation_email_sent_at timestamptz;
