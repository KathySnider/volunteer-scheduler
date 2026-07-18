-- Move staff_contact_id from shifts to events (one contact per event, not per shift)

ALTER TABLE events
    ADD COLUMN staff_contact_id INT REFERENCES staff(staff_id) ON DELETE SET NULL;

CREATE INDEX idx_events_staff_contact ON events(staff_contact_id);

DROP INDEX IF EXISTS idx_shifts_staff_contact;

ALTER TABLE shifts
    DROP COLUMN staff_contact_id;
