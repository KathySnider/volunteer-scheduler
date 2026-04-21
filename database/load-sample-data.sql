-- ============================================================================
-- VOLUNTEER SCHEDULER - SAMPLE DATA
-- Washington State AARP Volunteer System
--
-- Run AFTER migrations have been applied (e.g. after docker-compose up --build).
-- Run trunc.sql first if you want a clean reload.
--
-- Lookup tables (job_types, service_types, funding_entities) are seeded by
-- the migration and are NOT re-inserted here.
-- The upserts below are a safety net in case the migration ran incompletely.
-- ============================================================================

-- Safety-net: re-seed service_types (ON CONFLICT = no-op if already there)
INSERT INTO service_types (code, name) VALUES
    ('outreach',        'Outreach'),
    ('advocacy',        'Advocacy'),
    ('speakers_bureau', 'Speakers Bureau'),
    ('office_support',  'Office Support'),
    ('other',           'Other')
ON CONFLICT (code) DO NOTHING;

-- Safety-net: re-seed funding_entities (ON CONFLICT = no-op if already there)
INSERT INTO funding_entities (name) VALUES
    ('Seattle Area'),
    ('Spokane Area'),
    ('Statewide')
ON CONFLICT (name) DO NOTHING;


-- ============================================================================
-- VENUES
-- ============================================================================

INSERT INTO venues (venue_name, street_address, city, state, zip_code, timezone) VALUES
    ('Seattle Central Library',     '1000 4th Ave',             'Seattle',        'WA', '98104', 'America/Los_Angeles'),
    ('Spokane Convention Center',   '334 W Spokane Falls Blvd', 'Spokane',        'WA', '99201', 'America/Los_Angeles'),
    ('Tacoma Convention Center',    '1500 Broadway',            'Tacoma',         'WA', '98402', 'America/Los_Angeles'),
    ('Vancouver Community Library', '901 C St',                 'Vancouver',      'WA', '98660', 'America/Los_Angeles'),
    ('Bellevue City Hall',          '450 110th Ave NE',         'Bellevue',       'WA', '98004', 'America/Los_Angeles'),
    ('Spokane Valley Library',      '12004 E Main Ave',         'Spokane Valley', 'WA', '99206', 'America/Los_Angeles'),
    ('Olympia Center',              '222 Columbia St NW',       'Olympia',        'WA', '98501', 'America/Los_Angeles');


-- ============================================================================
-- VOLUNTEERS
--
-- TODO: Replace the email on the last row ("Test Admin") with your own so
--       you can log in via magic link and test the admin UI.
-- ============================================================================

INSERT INTO volunteers (first_name, last_name, email, phone, zip_code, role, is_active) VALUES
    ('Alice',   'Hansen',    'alice.hansen@example.com',   '206-555-0101', '98104', 'ADMINISTRATOR', TRUE),
    ('Bob',     'Nguyen',    'bob.nguyen@example.com',     '509-555-0102', '99201', 'ADMINISTRATOR', TRUE),
    ('Carol',   'Martinez',  'carol.martinez@example.com', '206-555-0103', '98402', 'VOLUNTEER',     TRUE),
    ('David',   'Kim',       'david.kim@example.com',      '509-555-0104', '99201', 'VOLUNTEER',     TRUE),
    ('Ellen',   'Patel',     'ellen.patel@example.com',    '360-555-0105', '98660', 'VOLUNTEER',     TRUE),
    ('Frank',   'Olsen',     'frank.olsen@example.com',    '206-555-0106', '98004', 'VOLUNTEER',     TRUE),
    ('Grace',   'Williams',  'grace.williams@example.com', '509-555-0107', '99206', 'VOLUNTEER',     TRUE),
    ('Henry',   'Thompson',  'henry.thompson@example.com', '360-555-0108', '98501', 'VOLUNTEER',     TRUE),
    ('Isabel',  'Chen',      'isabel.chen@example.com',    '206-555-0109', '98104', 'VOLUNTEER',     TRUE),
    ('James',   'Robinson',  'james.robinson@example.com', '509-555-0110', '99201', 'VOLUNTEER',     TRUE),
    -- *** Replace this email with your own to log in as admin during testing ***
    ('Test',    'Admin',     'admin@example.com',          NULL,           NULL,    'ADMINISTRATOR', TRUE);


-- ============================================================================
-- STAFF
-- ============================================================================

INSERT INTO staff (first_name, last_name, email, phone, position) VALUES
    ('Margaret', 'Sullivan', 'margaret.sullivan@aarp.org', '206-555-0201', 'State Coordinator'),
    ('Richard',  'Tanaka',   'richard.tanaka@aarp.org',    '509-555-0202', 'Regional Manager'),
    ('Patricia', 'Flores',   'patricia.flores@aarp.org',   '360-555-0203', 'Event Coordinator');


-- ============================================================================
-- EVENTS
--
-- EventType is derived by the application from two columns:
--   event_is_virtual=FALSE, venue_id IS NOT NULL  → IN_PERSON
--   event_is_virtual=TRUE,  venue_id IS NULL       → VIRTUAL
--   event_is_virtual=TRUE,  venue_id IS NOT NULL   → HYBRID
--
-- funding_entity_id maps to the funding_entities table seeded by the migration:
--   1 = Seattle Area  (western WA: Seattle, Bellevue, Tacoma, Vancouver, Olympia)
--   2 = Spokane Area  (eastern WA: Spokane, Spokane Valley)
--   3 = Statewide     (virtual or state-wide events)
-- ============================================================================

-- Seattle Area - IN_PERSON
INSERT INTO events (event_name, description, event_is_virtual, venue_id, funding_entity_id) VALUES
    ('Medicare Q&A Workshop',
     'Help seniors navigate Medicare enrollment and plan options. Volunteers assist with one-on-one sessions.',
     FALSE,
     (SELECT venue_id FROM venues WHERE city = 'Seattle'),
     (SELECT id FROM funding_entities WHERE name = 'Seattle Area' LIMIT 1));

INSERT INTO events (event_name, description, event_is_virtual, venue_id, funding_entity_id) VALUES
    ('Tax Aide Preparation - Spring Session',
     'Free tax preparation assistance for low-to-moderate income seniors. Training provided.',
     FALSE,
     (SELECT venue_id FROM venues WHERE city = 'Bellevue'),
     (SELECT id FROM funding_entities WHERE name = 'Seattle Area' LIMIT 1));

-- Statewide - VIRTUAL
INSERT INTO events (event_name, description, event_is_virtual, venue_id, funding_entity_id) VALUES
    ('Virtual Fraud Prevention Seminar',
     'Online session covering the latest scams targeting seniors and how to stay safe.',
     TRUE,
     NULL,
     (SELECT id FROM funding_entities WHERE name = 'Statewide' LIMIT 1));

-- Seattle Area - HYBRID (in-person venue + also streamed online)
INSERT INTO events (event_name, description, event_is_virtual, venue_id, funding_entity_id) VALUES
    ('Hybrid Benefits Counseling Day',
     'One-on-one benefits counseling available both in person and via video call. '
     'Volunteers help with check-in and virtual waiting room management.',
     TRUE,
     (SELECT venue_id FROM venues WHERE city = 'Tacoma'),
     (SELECT id FROM funding_entities WHERE name = 'Seattle Area' LIMIT 1));

-- Spokane Area - IN_PERSON
INSERT INTO events (event_name, description, event_is_virtual, venue_id, funding_entity_id) VALUES
    ('Spokane Senior Health Fair',
     'Community health fair with blood pressure checks, medication reviews, and wellness resources.',
     FALSE,
     (SELECT venue_id FROM venues WHERE city = 'Spokane'),
     (SELECT id FROM funding_entities WHERE name = 'Spokane Area' LIMIT 1));

INSERT INTO events (event_name, description, event_is_virtual, venue_id, funding_entity_id) VALUES
    ('Driver Safety Course',
     'AARP Smart Driver course for seniors. Volunteers help with registration and materials.',
     FALSE,
     (SELECT venue_id FROM venues WHERE city = 'Spokane Valley'),
     (SELECT id FROM funding_entities WHERE name = 'Spokane Area' LIMIT 1));

-- Seattle Area - IN_PERSON (western WA)
INSERT INTO events (event_name, description, event_is_virtual, venue_id, funding_entity_id) VALUES
    ('Social Security Benefits Workshop',
     'Informational session on maximizing Social Security benefits. Volunteers greet and assist attendees.',
     FALSE,
     (SELECT venue_id FROM venues WHERE city = 'Vancouver'),
     (SELECT id FROM funding_entities WHERE name = 'Seattle Area' LIMIT 1));

INSERT INTO events (event_name, description, event_is_virtual, venue_id, funding_entity_id) VALUES
    ('Caregiver Support Forum',
     'Forum connecting family caregivers with local resources and support networks.',
     FALSE,
     (SELECT venue_id FROM venues WHERE city = 'Olympia'),
     (SELECT id FROM funding_entities WHERE name = 'Seattle Area' LIMIT 1));


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
WHERE e.event_name = 'Tax Aide Preparation - Spring Session'
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
WHERE e.event_name = 'Spokane Senior Health Fair'
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
-- ============================================================================

INSERT INTO event_dates (event_id, start_date_time, end_date_time)
SELECT event_id, '2026-05-10 09:00:00', '2026-05-10 15:00:00'
FROM events WHERE event_name = 'Medicare Q&A Workshop';

INSERT INTO event_dates (event_id, start_date_time, end_date_time)
SELECT event_id, '2026-05-17 10:00:00', '2026-05-17 16:00:00'
FROM events WHERE event_name = 'Tax Aide Preparation - Spring Session';

INSERT INTO event_dates (event_id, start_date_time, end_date_time)
SELECT event_id, '2026-04-30 13:00:00', '2026-04-30 15:00:00'
FROM events WHERE event_name = 'Virtual Fraud Prevention Seminar';

INSERT INTO event_dates (event_id, start_date_time, end_date_time)
SELECT event_id, '2026-05-03 09:00:00', '2026-05-03 16:00:00'
FROM events WHERE event_name = 'Hybrid Benefits Counseling Day';

INSERT INTO event_dates (event_id, start_date_time, end_date_time)
SELECT event_id, '2026-06-07 09:00:00', '2026-06-07 14:00:00'
FROM events WHERE event_name = 'Spokane Senior Health Fair';

INSERT INTO event_dates (event_id, start_date_time, end_date_time)
SELECT event_id, '2026-05-24 08:30:00', '2026-05-24 12:30:00'
FROM events WHERE event_name = 'Driver Safety Course';

INSERT INTO event_dates (event_id, start_date_time, end_date_time)
SELECT event_id, '2026-06-14 10:00:00', '2026-06-14 13:00:00'
FROM events WHERE event_name = 'Social Security Benefits Workshop';

INSERT INTO event_dates (event_id, start_date_time, end_date_time)
SELECT event_id, '2026-06-21 13:00:00', '2026-06-21 16:00:00'
FROM events WHERE event_name = 'Caregiver Support Forum';


-- ============================================================================
-- OPPORTUNITIES AND SHIFTS
-- ============================================================================

-- Medicare Q&A Workshop - event_support
INSERT INTO opportunities (event_id, job_type_id, opportunity_is_virtual, pre_event_instructions)
SELECT e.event_id, jt.job_type_id, FALSE,
    'Please arrive 30 minutes early for briefing. Wear your AARP volunteer badge.'
FROM events e, job_types jt
WHERE e.event_name = 'Medicare Q&A Workshop'
  AND jt.code = 'event_support';

INSERT INTO shifts (opportunity_id, shift_start, shift_end, max_volunteers, staff_contact_id)
SELECT o.opportunity_id, '2026-05-10 08:30:00', '2026-05-10 12:00:00', 4,
    (SELECT staff_id FROM staff WHERE last_name = 'Sullivan')
FROM opportunities o
JOIN events e ON o.event_id = e.event_id
WHERE e.event_name = 'Medicare Q&A Workshop';

INSERT INTO shifts (opportunity_id, shift_start, shift_end, max_volunteers, staff_contact_id)
SELECT o.opportunity_id, '2026-05-10 12:00:00', '2026-05-10 15:30:00', 4,
    (SELECT staff_id FROM staff WHERE last_name = 'Sullivan')
FROM opportunities o
JOIN events e ON o.event_id = e.event_id
WHERE e.event_name = 'Medicare Q&A Workshop';

-- Medicare Q&A Workshop - advocacy (second opportunity at same event)
INSERT INTO opportunities (event_id, job_type_id, opportunity_is_virtual, pre_event_instructions)
SELECT e.event_id, jt.job_type_id, FALSE,
    'Advocates circulate the room and answer policy questions. Talking points will be emailed beforehand.'
FROM events e, job_types jt
WHERE e.event_name = 'Medicare Q&A Workshop'
  AND jt.code = 'advocacy';

INSERT INTO shifts (opportunity_id, shift_start, shift_end, max_volunteers, staff_contact_id)
SELECT o.opportunity_id, '2026-05-10 09:00:00', '2026-05-10 15:00:00', 2,
    (SELECT staff_id FROM staff WHERE last_name = 'Sullivan')
FROM opportunities o
JOIN events e ON o.event_id = e.event_id
WHERE e.event_name = 'Medicare Q&A Workshop'
  AND o.job_type_id = (SELECT job_type_id FROM job_types WHERE code = 'advocacy');

-- Tax Aide - event_support
INSERT INTO opportunities (event_id, job_type_id, opportunity_is_virtual, pre_event_instructions)
SELECT e.event_id, jt.job_type_id, FALSE,
    'IRS certification required before volunteering. Contact coordinator for training dates.'
FROM events e, job_types jt
WHERE e.event_name = 'Tax Aide Preparation - Spring Session'
  AND jt.code = 'event_support';

INSERT INTO shifts (opportunity_id, shift_start, shift_end, max_volunteers, staff_contact_id)
SELECT o.opportunity_id, '2026-05-17 09:30:00', '2026-05-17 13:00:00', 6,
    (SELECT staff_id FROM staff WHERE last_name = 'Sullivan')
FROM opportunities o
JOIN events e ON o.event_id = e.event_id
WHERE e.event_name = 'Tax Aide Preparation - Spring Session';

INSERT INTO shifts (opportunity_id, shift_start, shift_end, max_volunteers, staff_contact_id)
SELECT o.opportunity_id, '2026-05-17 13:00:00', '2026-05-17 16:30:00', 6,
    (SELECT staff_id FROM staff WHERE last_name = 'Sullivan')
FROM opportunities o
JOIN events e ON o.event_id = e.event_id
WHERE e.event_name = 'Tax Aide Preparation - Spring Session';

-- Virtual Fraud Prevention - speaker
INSERT INTO opportunities (event_id, job_type_id, opportunity_is_virtual, pre_event_instructions)
SELECT e.event_id, jt.job_type_id, TRUE,
    'Zoom link will be emailed 24 hours before the event. Test your audio/video beforehand.'
FROM events e, job_types jt
WHERE e.event_name = 'Virtual Fraud Prevention Seminar'
  AND jt.code = 'speaker';

INSERT INTO shifts (opportunity_id, shift_start, shift_end, max_volunteers)
SELECT o.opportunity_id, '2026-04-30 12:45:00', '2026-04-30 15:15:00', 3
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

INSERT INTO shifts (opportunity_id, shift_start, shift_end, max_volunteers, staff_contact_id)
SELECT o.opportunity_id, '2026-05-03 08:30:00', '2026-05-03 16:30:00', 3,
    (SELECT staff_id FROM staff WHERE last_name = 'Sullivan')
FROM opportunities o
JOIN events e ON o.event_id = e.event_id
WHERE e.event_name = 'Hybrid Benefits Counseling Day'
  AND o.opportunity_is_virtual = FALSE;

-- Hybrid Benefits Counseling - volunteer_lead (virtual waiting room)
INSERT INTO opportunities (event_id, job_type_id, opportunity_is_virtual, pre_event_instructions)
SELECT e.event_id, jt.job_type_id, TRUE,
    'Monitor the Zoom waiting room and admit participants at their scheduled times.'
FROM events e, job_types jt
WHERE e.event_name = 'Hybrid Benefits Counseling Day'
  AND jt.code = 'volunteer_lead';

INSERT INTO shifts (opportunity_id, shift_start, shift_end, max_volunteers, staff_contact_id)
SELECT o.opportunity_id, '2026-05-03 08:45:00', '2026-05-03 16:15:00', 2,
    (SELECT staff_id FROM staff WHERE last_name = 'Sullivan')
FROM opportunities o
JOIN events e ON o.event_id = e.event_id
WHERE e.event_name = 'Hybrid Benefits Counseling Day'
  AND o.opportunity_is_virtual = TRUE;

-- Spokane Senior Health Fair - event_support
INSERT INTO opportunities (event_id, job_type_id, opportunity_is_virtual, pre_event_instructions)
SELECT e.event_id, jt.job_type_id, FALSE,
    'Wear comfortable shoes. Setup begins at 8:00 AM.'
FROM events e, job_types jt
WHERE e.event_name = 'Spokane Senior Health Fair'
  AND jt.code = 'event_support';

INSERT INTO shifts (opportunity_id, shift_start, shift_end, max_volunteers, staff_contact_id)
SELECT o.opportunity_id, '2026-06-07 08:00:00', '2026-06-07 11:30:00', 5,
    (SELECT staff_id FROM staff WHERE last_name = 'Tanaka')
FROM opportunities o
JOIN events e ON o.event_id = e.event_id
WHERE e.event_name = 'Spokane Senior Health Fair';

INSERT INTO shifts (opportunity_id, shift_start, shift_end, max_volunteers, staff_contact_id)
SELECT o.opportunity_id, '2026-06-07 11:30:00', '2026-06-07 14:30:00', 5,
    (SELECT staff_id FROM staff WHERE last_name = 'Tanaka')
FROM opportunities o
JOIN events e ON o.event_id = e.event_id
WHERE e.event_name = 'Spokane Senior Health Fair';

-- Driver Safety Course - volunteer_lead
INSERT INTO opportunities (event_id, job_type_id, opportunity_is_virtual, pre_event_instructions)
SELECT e.event_id, jt.job_type_id, FALSE,
    'Lead volunteers coordinate check-in and distribute course materials.'
FROM events e, job_types jt
WHERE e.event_name = 'Driver Safety Course'
  AND jt.code = 'volunteer_lead';

INSERT INTO shifts (opportunity_id, shift_start, shift_end, max_volunteers, staff_contact_id)
SELECT o.opportunity_id, '2026-05-24 08:00:00', '2026-05-24 13:00:00', 2,
    (SELECT staff_id FROM staff WHERE last_name = 'Tanaka')
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

INSERT INTO shifts (opportunity_id, shift_start, shift_end, max_volunteers, staff_contact_id)
SELECT o.opportunity_id, '2026-06-14 09:30:00', '2026-06-14 13:30:00', 4,
    (SELECT staff_id FROM staff WHERE last_name = 'Flores')
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

INSERT INTO shifts (opportunity_id, shift_start, shift_end, max_volunteers, staff_contact_id)
SELECT o.opportunity_id, '2026-06-21 12:30:00', '2026-06-21 16:30:00', 3,
    (SELECT staff_id FROM staff WHERE last_name = 'Flores')
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
  AND s.shift_start = '2026-05-10 08:30:00';

INSERT INTO volunteer_shifts (volunteer_id, shift_id, assigned_at)
SELECT v.volunteer_id, s.shift_id, NOW()
FROM volunteers v, shifts s
JOIN opportunities o ON s.opportunity_id = o.opportunity_id
JOIN events e ON o.event_id = e.event_id
JOIN job_types jt ON o.job_type_id = jt.job_type_id
WHERE v.first_name = 'Frank'
  AND e.event_name = 'Medicare Q&A Workshop'
  AND jt.code = 'event_support'
  AND s.shift_start = '2026-05-10 08:30:00';

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

-- David and Grace: Spokane Health Fair morning shift
INSERT INTO volunteer_shifts (volunteer_id, shift_id, assigned_at)
SELECT v.volunteer_id, s.shift_id, NOW()
FROM volunteers v, shifts s
JOIN opportunities o ON s.opportunity_id = o.opportunity_id
JOIN events e ON o.event_id = e.event_id
WHERE v.first_name = 'David'
  AND e.event_name = 'Spokane Senior Health Fair'
  AND s.shift_start = '2026-06-07 08:00:00';

INSERT INTO volunteer_shifts (volunteer_id, shift_id, assigned_at)
SELECT v.volunteer_id, s.shift_id, NOW()
FROM volunteers v, shifts s
JOIN opportunities o ON s.opportunity_id = o.opportunity_id
JOIN events e ON o.event_id = e.event_id
WHERE v.first_name = 'Grace'
  AND e.event_name = 'Spokane Senior Health Fair'
  AND s.shift_start = '2026-06-07 08:00:00';

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
-- SAMPLE FEEDBACK
-- ============================================================================

INSERT INTO feedback (volunteer_id, feedback_type, status, subject, app_page_name, text, created_at)
SELECT volunteer_id, 'BUG', 'OPEN',
    'Event date not showing correctly',
    'Event Detail',
    'When I click on the Medicare workshop, the date shows as January instead of May. Might be a timezone issue.',
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
