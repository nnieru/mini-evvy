DROP TABLE IF EXISTS seating_draft_items;
DROP TABLE IF EXISTS seating_drafts;

ALTER TABLE seats DROP CONSTRAINT IF EXISTS seats_status_check;
ALTER TABLE seats ADD CONSTRAINT seats_status_check
    CHECK (status IN ('available', 'reserved', 'occupied', 'blocked'));
