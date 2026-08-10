-- Repair bookings left pending after send_invitation job already finished.
-- Caused by setting pending after enqueue (worker could complete first).

UPDATE seat_bookings b
SET
    invitation_email_status = 'sent',
    invitation_email_sent_at = COALESCE(b.invitation_email_sent_at, j.updated_at)
FROM (
    SELECT DISTINCT ON ((data->>'booking_id')::uuid)
        (data->>'booking_id')::uuid AS booking_id,
        updated_at
    FROM jobs
    WHERE type = 'send_invitation'
      AND status = 'done'
    ORDER BY (data->>'booking_id')::uuid, created_at DESC
) j
WHERE b.id = j.booking_id
  AND b.invitation_email_status = 'pending'
  AND b.deleted_at IS NULL;

UPDATE seat_bookings b
SET invitation_email_status = 'failed'
FROM (
    SELECT DISTINCT ON ((data->>'booking_id')::uuid)
        (data->>'booking_id')::uuid AS booking_id
    FROM jobs
    WHERE type = 'send_invitation'
      AND status = 'failed'
    ORDER BY (data->>'booking_id')::uuid, created_at DESC
) j
WHERE b.id = j.booking_id
  AND b.invitation_email_status = 'pending'
  AND b.deleted_at IS NULL;
