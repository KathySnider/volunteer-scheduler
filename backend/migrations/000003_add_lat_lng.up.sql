ALTER TABLE volunteers
  ADD COLUMN latitude  NUMERIC(9,6),
  ADD COLUMN longitude NUMERIC(9,6);

ALTER TABLE venues
  ADD COLUMN latitude  NUMERIC(9,6),
  ADD COLUMN longitude NUMERIC(9,6);
