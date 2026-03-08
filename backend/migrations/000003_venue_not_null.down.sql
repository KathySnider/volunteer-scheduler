-- Migration 000003 down: Revert venue address, city, and state to nullable.

ALTER TABLE venues
    ALTER COLUMN street_address DROP NOT NULL,
    ALTER COLUMN city DROP NOT NULL,
    ALTER COLUMN state DROP NOT NULL;
