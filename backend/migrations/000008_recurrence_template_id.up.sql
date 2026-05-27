-- Link opportunities and shifts that were propagated from the same recurring
-- event template so that edits/deletes can be fanned out to future instances.
ALTER TABLE opportunities ADD COLUMN recurrence_template_id UUID NULL;
ALTER TABLE shifts        ADD COLUMN recurrence_template_id UUID NULL;
