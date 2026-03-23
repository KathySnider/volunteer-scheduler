    -- Migration 000008: Add regions lookup table and venue_regions junction table.
-- Regions are many-to-many with venues to support overlapping regions
-- (e.g., a venue in Spokane belongs to both "Spokane" and "Eastern WA").
-- is_active preserves historic event data if a region is retired.
-- Every venue must have at least one region (enforced at application layer).

CREATE TABLE regions (
    region_id   SERIAL PRIMARY KEY,
    code        VARCHAR(50) UNIQUE NOT NULL,
    name        TEXT NOT NULL,
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    CHECK (code = lower(code))
);

CREATE INDEX idx_regions_active ON regions(region_id) WHERE is_active = TRUE;

-- Junction table: a venue can belong to multiple regions.
-- ON DELETE RESTRICT prevents removing a region that still has venues assigned.
-- Reassign venues to another region first, then retire via is_active = FALSE.
CREATE TABLE venue_regions (
    venue_id    INT NOT NULL REFERENCES venues(venue_id) ON DELETE CASCADE,
    region_id   INT NOT NULL REFERENCES regions(region_id) ON DELETE RESTRICT,
    PRIMARY KEY (venue_id, region_id)
);

CREATE INDEX idx_venue_regions_venue   ON venue_regions(venue_id);
CREATE INDEX idx_venue_regions_region  ON venue_regions(region_id);

-- Seed initial Washington state regions.
INSERT INTO regions (code, name) VALUES
    ('seattle',      'Seattle Metro'),
    ('spokane',      'Spokane'),
    ('southwest_wa', 'Southwest WA');
