-- Migration 000008 DOWN: Remove regions and venue_regions tables.

DROP TABLE venue_regions;
DROP INDEX idx_regions_active;
DROP TABLE regions;
