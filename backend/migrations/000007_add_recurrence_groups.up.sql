-- Store the recurrence settings that were used to generate a recurring event series.
-- One row per recurrence_group_id; linked to events.recurrence_group_id.
CREATE TABLE recurrence_groups (
    id               UUID NOT NULL PRIMARY KEY,
    pattern          TEXT NOT NULL,
    max_occurrences  INT  NULL,
    weekday_ordinal  TEXT NULL
);
