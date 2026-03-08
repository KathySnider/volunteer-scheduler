-- Migration 000003: Make venue address, city, and state NOT NULL.
-- name and zip_code remain nullable.

ALTER TABLE venues
    ALTER COLUMN street_address SET NOT NULL,
    ALTER COLUMN city SET NOT NULL,
    ALTER COLUMN state SET NOT NULL;
