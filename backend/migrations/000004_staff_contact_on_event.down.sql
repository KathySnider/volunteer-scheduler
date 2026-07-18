-- Revert: move staff_contact_id back from events to shifts

ALTER TABLE shifts
    ADD COLUMN staff_contact_id INT REFERENCES staff(staff_id) ON DELETE SET NULL;

CREATE INDEX idx_shifts_staff_contact ON shifts(staff_contact_id);

DROP INDEX IF EXISTS idx_events_staff_contact;

ALTER TABLE events
    DROP COLUMN staff_contact_id;
