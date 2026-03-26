-- ============================================================================
-- VOLUNTEER SCHEDULER - SAMPLE DATA
-- Washington State AARP Volunteer System
-- Run AFTER init_schema.sql
-- ============================================================================


-- ============================================================================
-- VENUES
-- ============================================================================

INSERT INTO venues (venue_name, street_address, city, state, zip_code, timezone) VALUES
    ('Seattle Central Library',       '1000 4th Ave',              'Seattle',    'WA', '98104', 'America/Los_Angeles'),
    ('Spokane Convention Center',     '334 W Spokane Falls Blvd',  'Spokane',    'WA', '99201', 'America/Los_Angeles'),
    ('Tacoma Convention Center',      '1500 Broadway',             'Tacoma',     'WA', '98402', 'America/Los_Angeles'),
    ('Vancouver Community Library',   '901 C St',                  'Vancouver',  'WA', '98660', 'America/Los_Angeles'),
    ('Bellevue City Hall',            '450 110th Ave NE',          'Bellevue',   'WA', '98004', 'America/Los_Angeles'),
    ('Spokane Valley Library',        '12004 E Main Ave',          'Spokane Valley', 'WA', '99206', 'America/Los_Angeles'),
    ('Olympia Center',                '222 Columbia St NW',        'Olympia',    'WA', '98501', 'America/Los_Angeles');


-- ============================================================================
-- VENUE REGIONS
-- ============================================================================

-- Seattle Metro: Seattle, Tacoma, Bellevue
INSERT INTO venue_regions (venue_id, region_id)
SELECT v.venue_id, r.region_id
FROM venues v, regions r
WHERE v.city IN ('Seattle', 'Tacoma', 'Bellevue')
  AND r.code = 'seattle';

-- Spokane: Spokane, Spokane Valley
INSERT INTO venue_regions (venue_id, region_id)
SELECT v.venue_id, r.region_id
FROM venues v, regions r
WHERE v.city IN ('Spokane', 'Spokane Valley')
  AND r.code = 'spokane';

-- Southwest WA: Vancouver, Olympia
INSERT INTO venue_regions (venue_id, region_id)
SELECT v.venue_id, r.region_id
FROM venues v, regions r
WHERE v.city IN ('Vancouver', 'Olympia')
  AND r.code = 'southwest_wa';


-- ============================================================================
-- VOLUNTEERS
-- ============================================================================

INSERT INTO volunteers (first_name, last_name, email, phone, zip_code, role, is_active) VALUES
    ('Alice',   'Hansen',    'alice.hansen@example.com',    '206-555-0101', '98104', 'ADMINISTRATOR', TRUE),
    ('Bob',     'Nguyen',    'bob.nguyen@example.com',      '509-555-0102', '99201', 'ADMINISTRATOR', TRUE),
    ('Carol',   'Martinez',  'carol.martinez@example.com',  '206-555-0103', '98402', 'VOLUNTEER',     TRUE),
    ('David',   'Kim',       'david.kim@example.com',       '509-555-0104', '99201', 'VOLUNTEER',     TRUE),
    ('Ellen',   'Patel',     'ellen.patel@example.com',     '360-555-0105', '98660', 'VOLUNTEER',     TRUE),
    ('Frank',   'Olsen',     'frank.olsen@example.com',     '206-555-0106', '98004', 'VOLUNTEER',     TRUE),
    ('Grace',   'Williams',  'grace.williams@example.com',  '509-555-0107', '99206', 'VOLUNTEER',     TRUE),
    ('Henry',   'Thompson',  'henry.thompson@example.com',  '360-555-0108', '98501', 'VOLUNTEER',     TRUE),
    ('Isabel',  'Chen',      'isabel.chen@example.com',     '206-555-0109', '98104', 'VOLUNTEER',     TRUE),
    ('James',   'Robinson',  'james.robinson@example.com',  '509-555-0110', '99201', 'VOLUNTEER',     TRUE);


-- ============================================================================
-- STAFF
-- ============================================================================

INSERT INTO staff (first_name, last_name, email, phone, position) VALUES
    ('Margaret', 'Sullivan',  'margaret.sullivan@aarp.org',  '206-555-0201', 'State Coordinator'),
    ('Richard',  'Tanaka',    'richard.tanaka@aarp.org',     '509-555-0202', 'Regional Manager'),
    ('Patricia', 'Flores',    'patricia.flores@aarp.org',    '360-555-0203', 'Event Coordinator');


-- ============================================================================
-- EVENTS
-- ============================================================================

-- Seattle Metro events
INSERT INTO events (event_name, description, event_is_virtual, venue_id) VALUES
    ('Medicare Q&A Workshop',
     'Help seniors navigate Medicare enrollment and plan options. Volunteers assist with one-on-one sessions.',
     FALSE,
     (SELECT venue_id FROM venues WHERE city = 'Seattle')),

    ('Tax Aide Preparation — Spring Session',
     'Free tax preparation assistance for low-to-moderate income seniors. Training provided.',
     FALSE,
     (SELECT venue_id FROM venues WHERE city = 'Bellevue')),

    ('Virtual Fraud Prevention Seminar',
     'Online session covering the latest scams targeting seniors and how to stay safe.',
     TRUE,
     NULL);

-- Spokane events
INSERT INTO events (event_name, description, event_is_virtual, venue_id) VALUES
    ('Spokane Senior Health Fair',
     'Community health fair with blood pressure checks, medication reviews, and wellness resources.',
     FALSE,
     (SELECT venue_id FROM venues WHERE city = 'Spokane')),

    ('Driver Safety Course',
     'AARP Smart Driver course for seniors. Volunteers help with registration and materials.',
     FALSE,
     (SELECT venue_id FROM venues WHERE city = 'Spokane Valley'));

-- Southwest WA events
INSERT INTO events (event_name, description, event_is_virtual, venue_id) VALUES
    ('Social Security Benefits Workshop',
     'Informational session on maximizing Social Security benefits. Volunteers greet and assist attendees.',
     FALSE,
     (SELECT venue_id FROM venues WHERE city = 'Vancouver')),

    ('Caregiver Support Forum',
     'Forum connecting family caregivers with local resources and support networks.',
     FALSE,
     (SELECT venue_id FROM venues WHERE city = 'Olympia'));


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
WHERE e.event_name = 'Tax Aide Preparation — Spring Session'
  AND st.code = 'office_support';

INSERT INTO event_service_types (event_id, service_type_id)
SELECT e.event_id, st.service_type_id
FROM events e, service_types st
WHERE e.event_name = 'Virtual Fraud Prevention Seminar'
  AND st.code IN ('outreach', 'speakers_bureau');

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
FROM events WHERE event_name = 'Tax Aide Preparation — Spring Session';

INSERT INTO event_dates (event_id, start_date_time, end_date_time)
SELECT event_id, '2026-04-30 13:00:00', '2026-04-30 15:00:00'
FROM events WHERE event_name = 'Virtual Fraud Prevention Seminar';

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

-- Medicare Q&A Workshop
INSERT INTO opportunities (event_id, job, opportunity_is_virtual, pre_event_instructions)
SELECT event_id, 'event_support', FALSE,
    'Please arrive 30 minutes early for briefing. Wear your AARP volunteer badge.'
FROM events WHERE event_name = 'Medicare Q&A Workshop';

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

-- Tax Aide
INSERT INTO opportunities (event_id, job, opportunity_is_virtual, pre_event_instructions)
SELECT event_id, 'office_support', FALSE,
    'IRS certification required before volunteering. Contact coordinator for training dates.'
FROM events WHERE event_name = 'Tax Aide Preparation — Spring Session';

INSERT INTO shifts (opportunity_id, shift_start, shift_end, max_volunteers, staff_contact_id)
SELECT o.opportunity_id, '2026-05-17 09:30:00', '2026-05-17 13:00:00', 6,
    (SELECT staff_id FROM staff WHERE last_name = 'Sullivan')
FROM opportunities o
JOIN events e ON o.event_id = e.event_id
WHERE e.event_name = 'Tax Aide Preparation — Spring Session';

INSERT INTO shifts (opportunity_id, shift_start, shift_end, max_volunteers, staff_contact_id)
SELECT o.opportunity_id, '2026-05-17 13:00:00', '2026-05-17 16:30:00', 6,
    (SELECT staff_id FROM staff WHERE last_name = 'Sullivan')
FROM opportunities o
JOIN events e ON o.event_id = e.event_id
WHERE e.event_name = 'Tax Aide Preparation — Spring Session';

-- Virtual Fraud Prevention
INSERT INTO opportunities (event_id, job, opportunity_is_virtual, pre_event_instructions)
SELECT event_id, 'speaker', TRUE,
    'Zoom link will be emailed 24 hours before the event. Test your audio/video beforehand.'
FROM events WHERE event_name = 'Virtual Fraud Prevention Seminar';

INSERT INTO shifts (opportunity_id, shift_start, shift_end, max_volunteers)
SELECT o.opportunity_id, '2026-04-30 12:45:00', '2026-04-30 15:15:00', 3
FROM opportunities o
JOIN events e ON o.event_id = e.event_id
WHERE e.event_name = 'Virtual Fraud Prevention Seminar';

-- Spokane Health Fair
INSERT INTO opportunities (event_id, job, opportunity_is_virtual, pre_event_instructions)
SELECT event_id, 'event_support', FALSE,
    'Wear comfortable shoes. Setup begins at 8:00 AM.'
FROM events WHERE event_name = 'Spokane Senior Health Fair';

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

-- Driver Safety Course
INSERT INTO opportunities (event_id, job, opportunity_is_virtual, pre_event_instructions)
SELECT event_id, 'volunteer_lead', FALSE,
    'Lead volunteers coordinate check-in and distribute course materials.'
FROM events WHERE event_name = 'Driver Safety Course';

INSERT INTO shifts (opportunity_id, shift_start, shift_end, max_volunteers, staff_contact_id)
SELECT o.opportunity_id, '2026-05-24 08:00:00', '2026-05-24 13:00:00', 2,
    (SELECT staff_id FROM staff WHERE last_name = 'Tanaka')
FROM opportunities o
JOIN events e ON o.event_id = e.event_id
WHERE e.event_name = 'Driver Safety Course';

-- Social Security Workshop
INSERT INTO opportunities (event_id, job, opportunity_is_virtual, pre_event_instructions)
SELECT event_id, 'event_support', FALSE,
    'Greet attendees and help them find seating. Light refreshments provided.'
FROM events WHERE event_name = 'Social Security Benefits Workshop';

INSERT INTO shifts (opportunity_id, shift_start, shift_end, max_volunteers, staff_contact_id)
SELECT o.opportunity_id, '2026-06-14 09:30:00', '2026-06-14 13:30:00', 4,
    (SELECT staff_id FROM staff WHERE last_name = 'Flores')
FROM opportunities o
JOIN events e ON o.event_id = e.event_id
WHERE e.event_name = 'Social Security Benefits Workshop';

-- Caregiver Forum
INSERT INTO opportunities (event_id, job, opportunity_is_virtual, pre_event_instructions)
SELECT event_id, 'event_support', FALSE,
    'Help set up resource tables and guide attendees to breakout sessions.'
FROM events WHERE event_name = 'Caregiver Support Forum';

INSERT INTO shifts (opportunity_id, shift_start, shift_end, max_volunteers, staff_contact_id)
SELECT o.opportunity_id, '2026-06-21 12:30:00', '2026-06-21 16:30:00', 3,
    (SELECT staff_id FROM staff WHERE last_name = 'Flores')
FROM opportunities o
JOIN events e ON o.event_id = e.event_id
WHERE e.event_name = 'Caregiver Support Forum';


-- ============================================================================
-- SAMPLE VOLUNTEER SHIFT ASSIGNMENTS
-- ============================================================================

-- Carol and Frank sign up for Medicare Q&A morning shift
INSERT INTO volunteer_shifts (volunteer_id, shift_id, assigned_at)
SELECT v.volunteer_id, s.shift_id, NOW()
FROM volunteers v, shifts s
JOIN opportunities o ON s.opportunity_id = o.opportunity_id
JOIN events e ON o.event_id = e.event_id
WHERE v.first_name = 'Carol'
  AND e.event_name = 'Medicare Q&A Workshop'
  AND s.shift_start = '2026-05-10 08:30:00';

INSERT INTO volunteer_shifts (volunteer_id, shift_id, assigned_at)
SELECT v.volunteer_id, s.shift_id, NOW()
FROM volunteers v, shifts s
JOIN opportunities o ON s.opportunity_id = o.opportunity_id
JOIN events e ON o.event_id = e.event_id
WHERE v.first_name = 'Frank'
  AND e.event_name = 'Medicare Q&A Workshop'
  AND s.shift_start = '2026-05-10 08:30:00';

-- David and Grace sign up for Spokane Health Fair morning shift
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

-- Ellen signs up for Social Security Workshop (then cancels)
INSERT INTO volunteer_shifts (volunteer_id, shift_id, assigned_at, cancelled_at)
SELECT v.volunteer_id, s.shift_id, NOW() - INTERVAL '5 days', NOW() - INTERVAL '2 days'
FROM volunteers v, shifts s
JOIN opportunities o ON s.opportunity_id = o.opportunity_id
JOIN events e ON o.event_id = e.event_id
WHERE v.first_name = 'Ellen'
  AND e.event_name = 'Social Security Benefits Workshop';

-- Henry signs up for Caregiver Forum
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
INSERT INTO feedback_notes (feedback_id, volunteer_id, note, created_at)
SELECT f.feedback_id, v.volunteer_id,
    'Reproduced the issue. Looks like the event date is being stored correctly in UTC but displaying without timezone conversion. Assigned to dev team.',
    NOW() - INTERVAL '2 days'
FROM feedback f, volunteers v
WHERE f.subject = 'Event date not showing correctly'
  AND v.first_name = 'Alice';
