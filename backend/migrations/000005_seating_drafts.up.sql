ALTER TABLE seats DROP CONSTRAINT IF EXISTS seats_status_check;
ALTER TABLE seats ADD CONSTRAINT seats_status_check
    CHECK (status IN ('available', 'held', 'reserved', 'occupied', 'blocked'));

CREATE TABLE seating_drafts (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id    uuid NOT NULL REFERENCES events(id),
    status      varchar NOT NULL DEFAULT 'open'
                CHECK (status IN ('open', 'approved', 'rejected')),
    created_by  uuid NOT NULL REFERENCES users(id),
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX uq_seating_drafts_event_open
    ON seating_drafts(event_id)
    WHERE status = 'open';

CREATE INDEX idx_seating_drafts_event_id ON seating_drafts(event_id);

CREATE TABLE seating_draft_items (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    draft_id    uuid NOT NULL REFERENCES seating_drafts(id) ON DELETE CASCADE,
    guest_id    uuid NOT NULL REFERENCES guests(id),
    seat_id     uuid NOT NULL REFERENCES seats(id),
    category_id uuid NOT NULL REFERENCES seat_categories(id),
    created_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_seating_draft_items_draft_seat UNIQUE (draft_id, seat_id)
);

CREATE INDEX idx_seating_draft_items_draft_id ON seating_draft_items(draft_id);
CREATE INDEX idx_seating_draft_items_guest_id ON seating_draft_items(guest_id);
CREATE INDEX idx_seating_draft_items_seat_id ON seating_draft_items(seat_id);
