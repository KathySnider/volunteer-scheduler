-- ============================================================================
-- VOLUNTEER SCHEDULER - SAMPLE DATA
--
-- Run AFTER migrations have been applied (e.g. after docker-compose up --build).
-- Run trunc.sql first if you want a clean reload.
--
-- Lookup tables (roles, job_types, service_types, funding_entities) are seeded
-- by the migration and are NOT re-inserted here.
-- The upserts below are a safety net in case the migration ran incompletely.
-- ============================================================================

-- Safety-net: re-seed roles
INSERT INTO roles (role_name) VALUES ('VOLUNTEER'), ('ADMINISTRATOR')
ON CONFLICT (role_name) DO NOTHING;

-- Safety-net: re-seed service_types
INSERT INTO service_types (code, name) VALUES
    ('outreach',        'Outreach'),
    ('advocacy',        'Advocacy'),
    ('speakers_bureau', 'Speakers Bureau'),
    ('office_support',  'Office Support'),
    ('other',           'Other')
ON CONFLICT (code) DO NOTHING;

-- Safety-net: re-seed funding_entities
INSERT INTO funding_entities (name) VALUES
    ('Eastern Region'),
    ('Central Region'),
    ('Western Region')
ON CONFLICT (name) DO NOTHING;


-- ============================================================================
-- VENUES
-- Note: timezone is no longer on venues — it moved to the events table so
-- that virtual events can carry their own timezone.
-- ============================================================================

INSERT INTO venues (venue_name, street_address, city, state, zip_code) VALUES
    ('Boston Public Library',       '700 Boylston St',              'Boston',    'MA', '02116'),
    ('Atlanta Central Library',     '1 Margaret Mitchell Square',   'Atlanta',   'GA', '30303'),
    ('Chicago Cultural Center',     '78 E Washington St',           'Chicago',   'IL', '60602'),
    ('Houston Public Library',      '500 McKinney St',              'Houston',   'TX', '77002'),
    ('Denver Public Library',       '10 W 14th Ave Pkwy',           'Denver',    'CO', '80204'),
    ('Phoenix Convention Center',   '100 N 3rd St',                 'Phoenix',   'AZ', '85004'),
    ('Portland Community Center',   '1234 SW Morrison St',          'Portland',  'OR', '97205');


-- ============================================================================
-- VOLUNTEERS
--
-- The role column no longer exists on volunteers — roles are stored in the
-- volunteer_roles junction table (see below).
--
-- TODO: Replace the email on the last row ("Test Admin") with your own so
--       you can log in via magic link and test the admin UI.
-- ============================================================================

INSERT INTO volunteers (first_name, last_name, email, phone, zip_code, is_active) VALUES
    ('Alice',   'Hansen',    'alice.hansen@example.com',   '617-555-0101', '02116', TRUE),
    ('Bob',     'Nguyen',    'bob.nguyen@example.com',     '312-555-0102', '60602', TRUE),
    ('Carol',   'Martinez',  'carol.martinez@example.com', '617-555-0103', '02116', TRUE),
    ('David',   'Kim',       'david.kim@example.com',      '713-555-0104', '77002', TRUE),
    ('Ellen',   'Patel',     'ellen.patel@example.com',    '303-555-0105', '80204', TRUE),
    ('Frank',   'Olsen',     'frank.olsen@example.com',    '602-555-0106', '85004', TRUE),
    ('Grace',   'Williams',  'grace.williams@example.com', '503-555-0107', '97205', TRUE),
    ('Henry',   'Thompson',  'henry.thompson@example.com', '404-555-0108', '30303', TRUE),
    ('Isabel',  'Chen',      'isabel.chen@example.com',    '617-555-0109', '02116', TRUE),
    ('James',   'Robinson',  'james.robinson@example.com', '312-555-0110', '60602', TRUE),
    ('Test',    'Admin',     'admin@example.com',          NULL,           NULL,    TRUE);

-- Assign roles via junction table.
-- All volunteers get the VOLUNTEER role; admins also get ADMINISTRATOR.
INSERT INTO volunteer_roles (volunteer_id, role_id)
SELECT v.volunteer_id, r.role_id
FROM   volunteers v
CROSS  JOIN roles r
WHERE  r.role_name = 'VOLUNTEER';

INSERT INTO volunteer_roles (volunteer_id, role_id)
SELECT v.volunteer_id, r.role_id
FROM   volunteers v
JOIN   roles r ON r.role_name = 'ADMINISTRATOR'
WHERE  v.first_name IN ('Alice', 'Bob', 'Test');


-- ============================================================================
-- STAFF
-- ============================================================================

INSERT INTO staff (first_name, last_name, email, phone, position) VALUES
    ('Margaret', 'Sullivan', 'margaret.sullivan@example.org', '617-555-0201', 'Regional Coordinator'),
    ('Richard',  'Tanaka',   'richard.tanaka@example.org',    '713-555-0202', 'Regional Manager'),
    ('Patricia', 'Flores',   'patricia.flores@example.org',   '503-555-0203', 'Event Coordinator');


-- ============================================================================
-- EVENTS
--
-- EventType is derived by the application from two columns:
--   event_is_virtual=FALSE, venue_id IS NOT NULL  → IN_PERSON
--   event_is_virtual=TRUE,  venue_id IS NULL       → VIRTUAL
--   event_is_virtual=TRUE,  venue_id IS NOT NULL   → HYBRID
--
-- staff_contact_id moved here from shifts (migration 000004).
--
-- funding_entity_id maps to:
--   Eastern Region  (e.g. Boston, Atlanta)
--   Central Region  (e.g. Chicago, Houston)
--   Western Region  (e.g. Denver, Phoenix, Portland)
-- ============================================================================

-- Eastern Region - IN_PERSON
INSERT INTO events (event_name, description, event_is_virtual, venue_id, funding_entity_id, staff_contact_id) VALUES
    ('Medicare Q&A Workshop',
     'Help seniors navigate Medicare enrollment and plan options. Volunteers assist with one-on-one sessions.',
     FALSE,
     (SELECT venue_id FROM venues WHERE city = 'Boston'),
     (SELECT id FROM funding_entities WHERE name = 'Eastern Region' LIMIT 1),
     (SELECT staff_id FROM staff WHERE last_name = 'Sullivan'));

INSERT INTO events (event_name, description, event_is_virtual, venue_id, funding_entity_id, staff_contact_id) VALUES
    ('Tax Aide Preparation - Summer Session',
     'Free tax preparation assistance for low-to-moderate income seniors. Training provided.',
     FALSE,
     (SELECT venue_id FROM venues WHERE city = 'Chicago'),
     (SELECT id FROM funding_entities WHERE name = 'Central Region' LIMIT 1),
     (SELECT staff_id FROM staff WHERE last_name = 'Sullivan'));

-- Western Region - VIRTUAL (no staff contact)
INSERT INTO events (event_name, description, event_is_virtual, venue_id, funding_entity_id) VALUES
    ('Virtual Fraud Prevention Seminar',
     'Online session covering the latest scams targeting seniors and how to stay safe.',
     TRUE,
     NULL,
     (SELECT id FROM funding_entities WHERE name = 'Western Region' LIMIT 1));

-- Western Region - HYBRID (in-person venue + also streamed online)
INSERT INTO events (event_name, description, event_is_virtual, venue_id, funding_entity_id, staff_contact_id) VALUES
    ('Hybrid Benefits Counseling Day',
     'One-on-one benefits counseling available both in person and via video call. '
     'Volunteers help with check-in and virtual waiting room management.',
     TRUE,
     (SELECT venue_id FROM venues WHERE city = 'Denver'),
     (SELECT id FROM funding_entities WHERE name = 'Western Region' LIMIT 1),
     (SELECT staff_id FROM staff WHERE last_name = 'Sullivan'));

-- Central Region - IN_PERSON
INSERT INTO events (event_name, description, event_is_virtual, venue_id, funding_entity_id, staff_contact_id) VALUES
    ('Senior Health Fair',
     'Community health fair with blood pressure checks, medication reviews, and wellness resources.',
     FALSE,
     (SELECT venue_id FROM venues WHERE city = 'Houston'),
     (SELECT id FROM funding_entities WHERE name = 'Central Region' LIMIT 1),
     (SELECT staff_id FROM staff WHERE last_name = 'Tanaka'));

INSERT INTO events (event_name, description, event_is_virtual, venue_id, funding_entity_id, staff_contact_id) VALUES
    ('Driver Safety Course',
     'Smart driver course for seniors. Volunteers help with registration and materials.',
     FALSE,
     (SELECT venue_id FROM venues WHERE city = 'Phoenix'),
     (SELECT id FROM funding_entities WHERE name = 'Western Region' LIMIT 1),
     (SELECT staff_id FROM staff WHERE last_name = 'Tanaka'));

-- Eastern Region - IN_PERSON
INSERT INTO events (event_name, description, event_is_virtual, venue_id, funding_entity_id, staff_contact_id) VALUES
    ('Social Security Benefits Workshop',
     'Informational session on maximizing Social Security benefits. Volunteers greet and assist attendees.',
     FALSE,
     (SELECT venue_id FROM venues WHERE city = 'Atlanta'),
     (SELECT id FROM funding_entities WHERE name = 'Eastern Region' LIMIT 1),
     (SELECT staff_id FROM staff WHERE last_name = 'Flores'));

INSERT INTO events (event_name, description, event_is_virtual, venue_id, funding_entity_id, staff_contact_id) VALUES
    ('Caregiver Support Forum',
     'Forum connecting family caregivers with local resources and support networks.',
     FALSE,
     (SELECT venue_id FROM venues WHERE city = 'Portland'),
     (SELECT id FROM funding_entities WHERE name = 'Western Region' LIMIT 1),
     (SELECT staff_id FROM staff WHERE last_name = 'Flores'));


-- ============================================================================
-- EVENT SERVICE TYPES
-- ============================================================================

INSERT INTO event_service_types (event_id, service_type_id)
SELECT e.event_id, st.service_type_id
FROM events e, service_types st
WHERE e.event_name = 'Medicare Q&A Workshop'
  AND st.code IN ('outreach', 'advocacy');

INSERT INTO event_service_types (event_id, service_type_id)
SELECT e.event_id, st.service_type_id
FROM events e, service_types st
WHERE e.event_name = 'Tax Aide Preparation - Summer Session'
  AND st.code = 'office_support';

INSERT INTO event_service_types (event_id, service_type_id)
SELECT e.event_id, st.service_type_id
FROM events e, service_types st
WHERE e.event_name = 'Virtual Fraud Prevention Seminar'
  AND st.code IN ('outreach', 'speakers_bureau');

INSERT INTO event_service_types (event_id, service_type_id)
SELECT e.event_id, st.service_type_id
FROM events e, service_types st
WHERE e.event_name = 'Hybrid Benefits Counseling Day'
  AND st.code IN ('outreach', 'advocacy');

INSERT INTO event_service_types (event_id, service_type_id)
SELECT e.event_id, st.service_type_id
FROM events e, service_types st
WHERE e.event_name = 'Senior Health Fair'
  AND st.code = 'outreach';

INSERT INTO event_service_types (event_id, service_type_id)
SELECT e.event_id, st.service_type_id
FROM events e, service_types st
WHERE e.event_name = 'Driver Safety Course'
  AND st.code = 'other';

INSERT INTO event_service_types (event_id, service_type_id)
SELECT e.event_id, st.service_type_id
FROM events e, service_types st
WHERE e.event_name = 'Social Security Benefits Workshop'
  AND st.code IN ('outreach', 'advocacy');

INSERT INTO event_service_types (event_id, service_type_id)
SELECT e.event_id, st.service_type_id
FROM events e, service_types st
WHERE e.event_name = 'Caregiver Support Forum'
  AND st.code = 'outreach';


-- ============================================================================
-- EVENT DATES
-- All events are scheduled in the future (relative to mid-2026 baseline).
-- ============================================================================

INSERT INTO event_dates (event_id, start_date_time, end_date_time)
SELECT event_id, '2026-07-12 09:00:00', '2026-07-12 15:00:00'
FROM events WHERE event_name = 'Medicare Q&A Workshop';

INSERT INTO event_dates (event_id, start_date_time, end_date_time)
SELECT event_id, '2026-07-19 10:00:00', '2026-07-19 16:00:00'
FROM events WHERE event_name = 'Tax Aide Preparation - Summer Session';

INSERT INTO event_dates (event_id, start_date_time, end_date_time)
SELECT event_id, '2026-07-07 13:00:00', '2026-07-07 15:00:00'
FROM events WHERE event_name = 'Virtual Fraud Prevention Seminar';

INSERT INTO event_dates (event_id, start_date_time, end_date_time)
SELECT event_id, '2026-07-26 09:00:00', '2026-07-26 16:00:00'
FROM events WHERE event_name = 'Hybrid Benefits Counseling Day';

INSERT INTO event_dates (event_id, start_date_time, end_date_time)
SELECT event_id, '2026-08-02 09:00:00', '2026-08-02 14:00:00'
FROM events WHERE event_name = 'Senior Health Fair';

INSERT INTO event_dates (event_id, start_date_time, end_date_time)
SELECT event_id, '2026-07-11 08:30:00', '2026-07-11 12:30:00'
FROM events WHERE event_name = 'Driver Safety Course';

INSERT INTO event_dates (event_id, start_date_time, end_date_time)
SELECT event_id, '2026-08-09 10:00:00', '2026-08-09 13:00:00'
FROM events WHERE event_name = 'Social Security Benefits Workshop';

INSERT INTO event_dates (event_id, start_date_time, end_date_time)
SELECT event_id, '2026-08-16 13:00:00', '2026-08-16 16:00:00'
FROM events WHERE event_name = 'Caregiver Support Forum';


-- ============================================================================
-- OPPORTUNITIES AND SHIFTS
-- staff_contact_id is now on events, not shifts.
-- ============================================================================

-- Medicare Q&A Workshop - event_support
INSERT INTO opportunities (event_id, job_type_id, opportunity_is_virtual, pre_event_instructions)
SELECT e.event_id, jt.job_type_id, FALSE,
    'Please arrive 30 minutes early for briefing. Wear your volunteer badge.'
FROM events e, job_types jt
WHERE e.event_name = 'Medicare Q&A Workshop'
  AND jt.code = 'event_support';

INSERT INTO shifts (opportunity_id, shift_start, shift_end, max_volunteers)
SELECT o.opportunity_id, '2026-07-12 08:30:00', '2026-07-12 12:00:00', 4
FROM opportunities o
JOIN events e ON o.event_id = e.event_id
WHERE e.event_name = 'Medicare Q&A Workshop'
  AND o.job_type_id = (SELECT job_type_id FROM job_types WHERE code = 'event_support');

INSERT INTO shifts (opportunity_id, shift_start, shift_end, max_volunteers)
SELECT o.opportunity_id, '2026-07-12 12:00:00', '2026-07-12 15:30:00', 4
FROM opportunities o
JOIN events e ON o.event_id = e.event_id
WHERE e.event_name = 'Medicare Q&A Workshop'
  AND o.job_type_id = (SELECT job_type_id FROM job_types WHERE code = 'event_support');

-- Medicare Q&A Workshop - advocacy (second opportunity at same event)
INSERT INTO opportunities (event_id, job_type_id, opportunity_is_virtual, pre_event_instructions)
SELECT e.event_id, jt.job_type_id, FALSE,
    'Advocates circulate the room and answer policy questions. Talking points will be emailed beforehand.'
FROM events e, job_types jt
WHERE e.event_name = 'Medicare Q&A Workshop'
  AND jt.code = 'advocacy';

INSERT INTO shifts (opportunity_id, shift_start, shift_end, max_volunteers)
SELECT o.opportunity_id, '2026-07-12 09:00:00', '2026-07-12 15:00:00', 2
FROM opportunities o
JOIN events e ON o.event_id = e.event_id
WHERE e.event_name = 'Medicare Q&A Workshop'
  AND o.job_type_id = (SELECT job_type_id FROM job_types WHERE code = 'advocacy');

-- Tax Aide - event_support
INSERT INTO opportunities (event_id, job_type_id, opportunity_is_virtual, pre_event_instructions)
SELECT e.event_id, jt.job_type_id, FALSE,
    'IRS certification required before volunteering. Contact coordinator for training dates.'
FROM events e, job_types jt
WHERE e.event_name = 'Tax Aide Preparation - Summer Session'
  AND jt.code = 'event_support';

INSERT INTO shifts (opportunity_id, shift_start, shift_end, max_volunteers)
SELECT o.opportunity_id, '2026-07-19 09:30:00', '2026-07-19 13:00:00', 6
FROM opportunities o
JOIN events e ON o.event_id = e.event_id
WHERE e.event_name = 'Tax Aide Preparation - Summer Session';

INSERT INTO shifts (opportunity_id, shift_start, shift_end, max_volunteers)
SELECT o.opportunity_id, '2026-07-19 13:00:00', '2026-07-19 16:30:00', 6
FROM opportunities o
JOIN events e ON o.event_id = e.event_id
WHERE e.event_name = 'Tax Aide Preparation - Summer Session';

-- Virtual Fraud Prevention - speaker
INSERT INTO opportunities (event_id, job_type_id, opportunity_is_virtual, pre_event_instructions)
SELECT e.event_id, jt.job_type_id, TRUE,
    'Video conference link will be emailed 24 hours before the event. Test your audio/video beforehand.'
FROM events e, job_types jt
WHERE e.event_name = 'Virtual Fraud Prevention Seminar'
  AND jt.code = 'speaker';

INSERT INTO shifts (opportunity_id, shift_start, shift_end, max_volunteers)
SELECT o.opportunity_id, '2026-07-07 12:45:00', '2026-07-07 15:15:00', 3
FROM opportunities o
JOIN events e ON o.event_id = e.event_id
WHERE e.event_name = 'Virtual Fraud Prevention Seminar';

-- Hybrid Benefits Counseling - event_support (in-person)
INSERT INTO opportunities (event_id, job_type_id, opportunity_is_virtual, pre_event_instructions)
SELECT e.event_id, jt.job_type_id, FALSE,
    'Manage in-person check-in table. Comfortable walking shoes recommended.'
FROM events e, job_types jt
WHERE e.event_name = 'Hybrid Benefits Counseling Day'
  AND jt.code = 'event_support';

INSERT INTO shifts (opportunity_id, shift_start, shift_end, max_volunteers)
SELECT o.opportunity_id, '2026-07-26 08:30:00', '2026-07-26 16:30:00', 3
FROM opportunities o
JOIN events e ON o.event_id = e.event_id
WHERE e.event_name = 'Hybrid Benefits Counseling Day'
  AND o.opportunity_is_virtual = FALSE;

-- Hybrid Benefits Counseling - volunteer_lead (virtual waiting room)
INSERT INTO opportunities (event_id, job_type_id, opportunity_is_virtual, pre_event_instructions)
SELECT e.event_id, jt.job_type_id, TRUE,
    'Monitor the video waiting room and admit participants at their scheduled times.'
FROM events e, job_types jt
WHERE e.event_name = 'Hybrid Benefits Counseling Day'
  AND jt.code = 'volunteer_lead';

INSERT INTO shifts (opportunity_id, shift_start, shift_end, max_volunteers)
SELECT o.opportunity_id, '2026-07-26 08:45:00', '2026-07-26 16:15:00', 2
FROM opportunities o
JOIN events e ON o.event_id = e.event_id
WHERE e.event_name = 'Hybrid Benefits Counseling Day'
  AND o.opportunity_is_virtual = TRUE;

-- Senior Health Fair - event_support
INSERT INTO opportunities (event_id, job_type_id, opportunity_is_virtual, pre_event_instructions)
SELECT e.event_id, jt.job_type_id, FALSE,
    'Wear comfortable shoes. Setup begins at 8:00 AM.'
FROM events e, job_types jt
WHERE e.event_name = 'Senior Health Fair'
  AND jt.code = 'event_support';

INSERT INTO shifts (opportunity_id, shift_start, shift_end, max_volunteers)
SELECT o.opportunity_id, '2026-08-02 08:00:00', '2026-08-02 11:30:00', 5
FROM opportunities o
JOIN events e ON o.event_id = e.event_id
WHERE e.event_name = 'Senior Health Fair';

INSERT INTO shifts (opportunity_id, shift_start, shift_end, max_volunteers)
SELECT o.opportunity_id, '2026-08-02 11:30:00', '2026-08-02 14:30:00', 5
FROM opportunities o
JOIN events e ON o.event_id = e.event_id
WHERE e.event_name = 'Senior Health Fair';

-- Driver Safety Course - volunteer_lead
INSERT INTO opportunities (event_id, job_type_id, opportunity_is_virtual, pre_event_instructions)
SELECT e.event_id, jt.job_type_id, FALSE,
    'Lead volunteers coordinate check-in and distribute course materials.'
FROM events e, job_types jt
WHERE e.event_name = 'Driver Safety Course'
  AND jt.code = 'volunteer_lead';

INSERT INTO shifts (opportunity_id, shift_start, shift_end, max_volunteers)
SELECT o.opportunity_id, '2026-07-11 08:00:00', '2026-07-11 13:00:00', 2
FROM opportunities o
JOIN events e ON o.event_id = e.event_id
WHERE e.event_name = 'Driver Safety Course';

-- Social Security Workshop - event_support
INSERT INTO opportunities (event_id, job_type_id, opportunity_is_virtual, pre_event_instructions)
SELECT e.event_id, jt.job_type_id, FALSE,
    'Greet attendees and help them find seating. Light refreshments provided.'
FROM events e, job_types jt
WHERE e.event_name = 'Social Security Benefits Workshop'
  AND jt.code = 'event_support';

INSERT INTO shifts (opportunity_id, shift_start, shift_end, max_volunteers)
SELECT o.opportunity_id, '2026-08-09 09:30:00', '2026-08-09 13:30:00', 4
FROM opportunities o
JOIN events e ON o.event_id = e.event_id
WHERE e.event_name = 'Social Security Benefits Workshop';

-- Caregiver Forum - event_support
INSERT INTO opportunities (event_id, job_type_id, opportunity_is_virtual, pre_event_instructions)
SELECT e.event_id, jt.job_type_id, FALSE,
    'Help set up resource tables and guide attendees to breakout sessions.'
FROM events e, job_types jt
WHERE e.event_name = 'Caregiver Support Forum'
  AND jt.code = 'event_support';

INSERT INTO shifts (opportunity_id, shift_start, shift_end, max_volunteers)
SELECT o.opportunity_id, '2026-08-16 12:30:00', '2026-08-16 16:30:00', 3
FROM opportunities o
JOIN events e ON o.event_id = e.event_id
WHERE e.event_name = 'Caregiver Support Forum';


-- ============================================================================
-- SAMPLE VOLUNTEER SHIFT ASSIGNMENTS
-- ============================================================================

-- Carol and Frank: Medicare Q&A morning shift (event_support)
INSERT INTO volunteer_shifts (volunteer_id, shift_id, assigned_at)
SELECT v.volunteer_id, s.shift_id, NOW()
FROM volunteers v, shifts s
JOIN opportunities o ON s.opportunity_id = o.opportunity_id
JOIN events e ON o.event_id = e.event_id
JOIN job_types jt ON o.job_type_id = jt.job_type_id
WHERE v.first_name = 'Carol'
  AND e.event_name = 'Medicare Q&A Workshop'
  AND jt.code = 'event_support'
  AND s.shift_start = '2026-07-12 08:30:00';

INSERT INTO volunteer_shifts (volunteer_id, shift_id, assigned_at)
SELECT v.volunteer_id, s.shift_id, NOW()
FROM volunteers v, shifts s
JOIN opportunities o ON s.opportunity_id = o.opportunity_id
JOIN events e ON o.event_id = e.event_id
JOIN job_types jt ON o.job_type_id = jt.job_type_id
WHERE v.first_name = 'Frank'
  AND e.event_name = 'Medicare Q&A Workshop'
  AND jt.code = 'event_support'
  AND s.shift_start = '2026-07-12 08:30:00';

-- Isabel and James: Medicare Q&A advocacy slot (fully fills that 2-person opportunity)
INSERT INTO volunteer_shifts (volunteer_id, shift_id, assigned_at)
SELECT v.volunteer_id, s.shift_id, NOW()
FROM volunteers v, shifts s
JOIN opportunities o ON s.opportunity_id = o.opportunity_id
JOIN events e ON o.event_id = e.event_id
JOIN job_types jt ON o.job_type_id = jt.job_type_id
WHERE v.first_name = 'Isabel'
  AND e.event_name = 'Medicare Q&A Workshop'
  AND jt.code = 'advocacy';

INSERT INTO volunteer_shifts (volunteer_id, shift_id, assigned_at)
SELECT v.volunteer_id, s.shift_id, NOW()
FROM volunteers v, shifts s
JOIN opportunities o ON s.opportunity_id = o.opportunity_id
JOIN events e ON o.event_id = e.event_id
JOIN job_types jt ON o.job_type_id = jt.job_type_id
WHERE v.first_name = 'James'
  AND e.event_name = 'Medicare Q&A Workshop'
  AND jt.code = 'advocacy';

-- David and Grace: Senior Health Fair morning shift
INSERT INTO volunteer_shifts (volunteer_id, shift_id, assigned_at)
SELECT v.volunteer_id, s.shift_id, NOW()
FROM volunteers v, shifts s
JOIN opportunities o ON s.opportunity_id = o.opportunity_id
JOIN events e ON o.event_id = e.event_id
WHERE v.first_name = 'David'
  AND e.event_name = 'Senior Health Fair'
  AND s.shift_start = '2026-08-02 08:00:00';

INSERT INTO volunteer_shifts (volunteer_id, shift_id, assigned_at)
SELECT v.volunteer_id, s.shift_id, NOW()
FROM volunteers v, shifts s
JOIN opportunities o ON s.opportunity_id = o.opportunity_id
JOIN events e ON o.event_id = e.event_id
WHERE v.first_name = 'Grace'
  AND e.event_name = 'Senior Health Fair'
  AND s.shift_start = '2026-08-02 08:00:00';

-- Ellen: Social Security Workshop (signs up then cancels)
INSERT INTO volunteer_shifts (volunteer_id, shift_id, assigned_at, cancelled_at)
SELECT v.volunteer_id, s.shift_id, NOW() - INTERVAL '5 days', NOW() - INTERVAL '2 days'
FROM volunteers v, shifts s
JOIN opportunities o ON s.opportunity_id = o.opportunity_id
JOIN events e ON o.event_id = e.event_id
WHERE v.first_name = 'Ellen'
  AND e.event_name = 'Social Security Benefits Workshop';

-- Henry: Caregiver Forum
INSERT INTO volunteer_shifts (volunteer_id, shift_id, assigned_at)
SELECT v.volunteer_id, s.shift_id, NOW()
FROM volunteers v, shifts s
JOIN opportunities o ON s.opportunity_id = o.opportunity_id
JOIN events e ON o.event_id = e.event_id
WHERE v.first_name = 'Henry'
  AND e.event_name = 'Caregiver Support Forum';


-- ============================================================================
-- SAMPLE RECURRING EVENT (Monthly Volunteer Orientation — 3 occurrences)
-- Uses a DO block so we can share a single recurrence_group_id UUID across
-- all three event rows without hard-coding a literal UUID.
-- ============================================================================

DO $$
DECLARE
  grp_id   UUID;
  evt1_id  INT;
  evt2_id  INT;
  evt3_id  INT;
BEGIN
  grp_id := gen_random_uuid();

  INSERT INTO recurrence_groups (id, pattern, max_occurrences, weekday_ordinal)
  VALUES (grp_id, 'MONTHLY', 3, NULL);

  -- Occurrence 1 — first Tuesday of August 2026
  INSERT INTO events (event_name, description, event_is_virtual, venue_id, funding_entity_id,
                      recurrence_group_id, recurrence_order, timezone)
  VALUES (
    'Monthly Volunteer Orientation',
    'Introduction for new volunteers. Covers mission, policies, and upcoming events.',
    FALSE,
    (SELECT venue_id FROM venues WHERE city = 'Chicago'),
    (SELECT id FROM funding_entities WHERE name = 'Central Region' LIMIT 1),
    grp_id, 1, 'America/Chicago'
  )
  RETURNING event_id INTO evt1_id;

  INSERT INTO event_dates (event_id, start_date_time, end_date_time)
  VALUES (evt1_id, '2026-08-04 09:00:00', '2026-08-04 11:00:00');

  -- Occurrence 2 — first Tuesday of September 2026
  INSERT INTO events (event_name, description, event_is_virtual, venue_id, funding_entity_id,
                      recurrence_group_id, recurrence_order, timezone)
  VALUES (
    'Monthly Volunteer Orientation',
    'Introduction for new volunteers. Covers mission, policies, and upcoming events.',
    FALSE,
    (SELECT venue_id FROM venues WHERE city = 'Chicago'),
    (SELECT id FROM funding_entities WHERE name = 'Central Region' LIMIT 1),
    grp_id, 2, 'America/Chicago'
  )
  RETURNING event_id INTO evt2_id;

  INSERT INTO event_dates (event_id, start_date_time, end_date_time)
  VALUES (evt2_id, '2026-09-01 09:00:00', '2026-09-01 11:00:00');

  -- Occurrence 3 — first Tuesday of October 2026
  INSERT INTO events (event_name, description, event_is_virtual, venue_id, funding_entity_id,
                      recurrence_group_id, recurrence_order, timezone)
  VALUES (
    'Monthly Volunteer Orientation',
    'Introduction for new volunteers. Covers mission, policies, and upcoming events.',
    FALSE,
    (SELECT venue_id FROM venues WHERE city = 'Chicago'),
    (SELECT id FROM funding_entities WHERE name = 'Central Region' LIMIT 1),
    grp_id, 3, 'America/Chicago'
  )
  RETURNING event_id INTO evt3_id;

  INSERT INTO event_dates (event_id, start_date_time, end_date_time)
  VALUES (evt3_id, '2026-10-06 09:00:00', '2026-10-06 11:00:00');
END;
$$;


-- ============================================================================
-- SAMPLE FEEDBACK
-- ============================================================================

INSERT INTO feedback (volunteer_id, feedback_type, status, subject, app_page_name, text, created_at)
SELECT volunteer_id, 'BUG', 'OPEN',
    'Event date not showing correctly',
    'Event Detail',
    'When I click on the Medicare workshop, the date shows as January instead of July. Might be a timezone issue.',
    NOW() - INTERVAL '3 days'
FROM volunteers WHERE first_name = 'Carol';

INSERT INTO feedback (volunteer_id, feedback_type, status, subject, app_page_name, text, created_at)
SELECT volunteer_id, 'ENHANCEMENT', 'OPEN',
    'Add email reminders for upcoming shifts',
    'My Signups',
    'It would be really helpful to get an email reminder 24 hours before a shift. I almost forgot about my last one!',
    NOW() - INTERVAL '1 day'
FROM volunteers WHERE first_name = 'David';

-- Admin note on the bug report
INSERT INTO feedback_notes (feedback_id, volunteer_id, note, note_type, created_at)
SELECT f.feedback_id, v.volunteer_id,
    'Reproduced the issue. Looks like the event date is being stored correctly in UTC but displaying without timezone conversion. Assigned to dev team.',
    'ADMIN_NOTE',
    NOW() - INTERVAL '2 days'
FROM feedback f, volunteers v
WHERE f.subject = 'Event date not showing correctly'
  AND v.first_name = 'Alice';
