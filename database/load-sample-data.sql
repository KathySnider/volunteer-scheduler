-- ============================================
-- SAMPLE DATA - Volunteer Scheduler
-- ============================================

-- ============================================
-- STAFF
-- ============================================
INSERT INTO staff (first_name, last_name, email, phone, position) VALUES
  ('Maria', 'Gonzalez', 'mgonzalez@org.org', '702-555-0101', 'Volunteer Coordinator'),
  ('James', 'Okafor',   'jokafor@org.org',   '702-555-0102', 'Outreach Director'),
  ('Susan', 'Park',     'spark@org.org',      '702-555-0103', 'Advocacy Manager');


-- ============================================
-- SERVICE TYPES
-- ============================================
INSERT INTO service_types (code, name) VALUES
  ('outreach',        'Outreach'),
  ('advocacy',        'Advocacy'),
  ('speakers_bureau', 'Speakers Bureau'),
  ('office_support',  'Office Support'),
  ('other',           'Other');


-- ============================================
-- VENUES
-- Timezones match the city/state of each venue.
-- ============================================
INSERT INTO venues (venue_name, street_address, city, state, zip_code, timezone) VALUES
  ('City Community Center',     '100 Main St',         'Las Vegas',    'NV', '89101', 'America/Los_Angeles'),
  ('Westside Library',          '250 W Sahara Ave',    'Las Vegas',    'NV', '89102', 'America/Los_Angeles'),
  ('Henderson Civic Hall',      '400 N Water St',      'Henderson',    'NV', '89002', 'America/Los_Angeles'),
  ('North Las Vegas Rec Center','1235 N Civic Dr',     'N. Las Vegas', 'NV', '89030', 'America/Los_Angeles'),
  ('Phoenix Convention Center', '100 N 3rd St',        'Phoenix',      'AZ', '85004', 'America/Phoenix'),
  ('Denver Community Hub',      '1400 Glenarm Pl',     'Denver',       'CO', '80202', 'America/Denver'),
  ('Seattle Neighborhood Hall', '810 3rd Ave',         'Seattle',      'WA', '98104', 'America/Los_Angeles');


-- ============================================
-- VOLUNTEERS
-- ============================================
INSERT INTO volunteers (first_name, last_name, email, phone, zip_code) VALUES
  ('Alice',   'Thompson', 'alice.t@email.com',  '702-555-1001', '89101'),
  ('Ben',     'Ruiz',     'ben.ruiz@email.com', '702-555-1002', '89102'),
  ('Carla',   'Nguyen',   'carla.n@email.com',  '702-555-1003', '89002'),
  ('David',   'Kim',      'david.k@email.com',  '702-555-1004', '89030'),
  ('Estrella','Morales',  'estre.m@email.com',  '702-555-1005', '89101'),
  ('Frank',   'Delgado',  'frank.d@email.com',  '602-555-2001', '85004'),
  ('Grace',   'Huang',    'grace.h@email.com',  '303-555-3001', '80202'),
  ('Henry',   'Osei',     'henry.o@email.com',  '206-555-4001', '98104');


-- ============================================
-- EVENTS
-- ============================================
INSERT INTO events (event_name, description, event_is_virtual, venue_id) VALUES
  -- Las Vegas events
  ('Spring Outreach Fair',
   'Annual community outreach fair connecting residents with local services.',
   FALSE, 1),
  ('Legislative Advocacy Day',
   'Volunteers travel to the state capitol to meet with legislators.',
   FALSE, 3),
  ('Public Speaking Workshop',
   'Training session for volunteers joining the speakers bureau.',
   FALSE, 2),
  ('Neighborhood Canvass - Downtown',
   'Door-to-door outreach in the downtown corridor.',
   FALSE, 1),
  ('Office Volunteer Day',
   'Help staff the office: answer phones, file, and assist walk-ins.',
   FALSE, 1),
  ('North LV Community Forum',
   'Community forum addressing housing and health services in North LV.',
   FALSE, 4),
  ('Speakers Bureau Showcase',
   'Volunteers present personal stories to community groups.',
   FALSE, 2),
  ('Henderson Resource Fair',
   'Multi-agency resource fair serving Henderson residents.',
   FALSE, 3),
  -- Phoenix event
  ('Phoenix Advocacy Summit',
   'Regional summit bringing together advocates from across Arizona.',
   FALSE, 5),
  -- Denver event
  ('Denver Outreach Blitz',
   'Intensive one-day outreach effort across downtown Denver.',
   FALSE, 6),
  -- Seattle event
  ('Seattle Speakers Night',
   'Evening showcase of speakers bureau volunteers in Seattle.',
   FALSE, 7),
  -- Virtual events
  ('Virtual Town Hall',
   'Online town hall discussing upcoming policy changes.',
   TRUE, NULL),
  ('Virtual Advocacy Training',
   'Online training covering advocacy skills and messaging.',
   TRUE, NULL),
  -- Hybrid event
  ('Hybrid Leadership Forum',
   'Leadership forum available in-person in Las Vegas and streamed online.',
   TRUE, 1);


-- ============================================
-- EVENT SERVICE TYPES
-- ============================================
INSERT INTO event_service_types (event_id, service_type_id)
SELECT e.event_id, s.service_type_id
FROM (VALUES
  ('Spring Outreach Fair',            'outreach'),
  ('Spring Outreach Fair',            'advocacy'),
  ('Legislative Advocacy Day',        'advocacy'),
  ('Public Speaking Workshop',        'speakers_bureau'),
  ('Neighborhood Canvass - Downtown', 'outreach'),
  ('Office Volunteer Day',            'office_support'),
  ('North LV Community Forum',        'outreach'),
  ('North LV Community Forum',        'advocacy'),
  ('Speakers Bureau Showcase',        'speakers_bureau'),
  ('Henderson Resource Fair',         'outreach'),
  ('Henderson Resource Fair',         'office_support'),
  ('Phoenix Advocacy Summit',         'advocacy'),
  ('Denver Outreach Blitz',           'outreach'),
  ('Seattle Speakers Night',          'speakers_bureau'),
  ('Virtual Town Hall',               'advocacy'),
  ('Virtual Town Hall',               'outreach'),
  ('Virtual Advocacy Training',       'advocacy'),
  ('Hybrid Leadership Forum',         'advocacy'),
  ('Hybrid Leadership Forum',         'outreach')
) AS mapping(event_name, code)
JOIN events e ON e.event_name = mapping.event_name
JOIN service_types s ON s.code = mapping.code;


-- ============================================
-- EVENT DATES
-- Stored as UTC. Offsets:
--   Las Vegas / Seattle: UTC-7 (PDT) in spring/summer
--   Phoenix:             UTC-7 (MST, no DST)
--   Denver:              UTC-6 (MDT) in spring/summer
-- ============================================
INSERT INTO event_dates (event_id, start_date_time, end_date_time)
SELECT e.event_id, v.start_date_time::TIMESTAMP, v.end_date_time::TIMESTAMP
FROM (VALUES
  -- Las Vegas
  ('Spring Outreach Fair',            '2026-04-11 16:00', '2026-04-12 00:00'),
  ('Spring Outreach Fair',            '2026-04-12 16:00', '2026-04-12 21:00'),
  ('Legislative Advocacy Day',        '2026-04-22 15:00', '2026-04-23 01:00'),
  ('Public Speaking Workshop',        '2026-04-16 01:00', '2026-04-16 03:30'),
  ('Neighborhood Canvass - Downtown', '2026-04-25 17:00', '2026-04-25 21:00'),
  ('Office Volunteer Day',            '2026-05-02 16:00', '2026-05-03 00:00'),
  ('Office Volunteer Day',            '2026-05-09 16:00', '2026-05-10 00:00'),
  ('North LV Community Forum',        '2026-05-07 01:00', '2026-05-07 03:00'),
  ('Speakers Bureau Showcase',        '2026-05-14 00:30', '2026-05-14 03:00'),
  ('Henderson Resource Fair',         '2026-05-16 16:00', '2026-05-16 22:00'),
  ('Henderson Resource Fair',         '2026-05-17 16:00', '2026-05-17 20:00'),
  -- Phoenix (UTC-7)
  ('Phoenix Advocacy Summit',         '2026-04-30 14:00', '2026-04-30 23:00'),
  -- Denver (UTC-6)
  ('Denver Outreach Blitz',           '2026-05-09 14:00', '2026-05-09 22:00'),
  -- Seattle (UTC-7)
  ('Seattle Speakers Night',          '2026-05-21 00:00', '2026-05-21 03:00'),
  -- Virtual
  ('Virtual Town Hall',               '2026-04-19 00:00', '2026-04-19 02:00'),
  ('Virtual Advocacy Training',       '2026-05-07 19:00', '2026-05-07 21:00'),
  ('Virtual Advocacy Training',       '2026-05-14 19:00', '2026-05-14 21:00'),
  -- Hybrid
  ('Hybrid Leadership Forum',         '2026-06-05 17:00', '2026-06-05 23:00')
) AS v(event_name, start_date_time, end_date_time)
JOIN events e ON e.event_name = v.event_name;


-- ============================================
-- OPPORTUNITIES
-- ============================================
INSERT INTO opportunities (event_id, job, opportunity_is_virtual, pre_event_instructions)
SELECT e.event_id, v.job::job_type, v.is_virtual, v.instructions
FROM (VALUES
  ('Spring Outreach Fair',            'event_support',  FALSE, 'Wear org t-shirt. Arrive 30 min early for setup.'),
  ('Spring Outreach Fair',            'volunteer_lead', FALSE, 'Lead a team of 4 event support volunteers.'),
  ('Legislative Advocacy Day',        'advocacy',       FALSE, 'Review talking points sent via email before the event.'),
  ('Public Speaking Workshop',        'speaker',        FALSE, 'Prepare a 5-minute personal story to share.'),
  ('Neighborhood Canvass - Downtown', 'event_support',  FALSE, 'Bring comfortable shoes. Materials provided on site.'),
  ('Office Volunteer Day',            'event_support',  FALSE, 'Check in with Maria at the front desk upon arrival.'),
  ('North LV Community Forum',        'event_support',  FALSE, 'Help with setup, registration, and breakdown.'),
  ('Speakers Bureau Showcase',        'speaker',        FALSE, 'Prepare a 10-minute story. Run-through at 5pm before event.'),
  ('Henderson Resource Fair',         'event_support',  FALSE, 'Arrive 45 min early. Wear org t-shirt.'),
  ('Phoenix Advocacy Summit',         'advocacy',       FALSE, 'Review the summit agenda and prepare questions for legislators.'),
  ('Phoenix Advocacy Summit',         'volunteer_lead', FALSE, 'Coordinate check-in and manage breakout room logistics.'),
  ('Denver Outreach Blitz',           'event_support',  FALSE, 'Teams of 3. Meet at venue at 7:45am for assignment.'),
  ('Seattle Speakers Night',          'speaker',        FALSE, 'Prepare an 8-minute story. Arrive by 5:30pm for sound check.'),
  ('Virtual Town Hall',               'advocacy',       TRUE,  'Log in 10 minutes early to test audio/video.'),
  ('Virtual Advocacy Training',       'attendee_only',  TRUE,  'No prep needed. Just bring your questions!'),
  ('Hybrid Leadership Forum',         'event_support',  FALSE, 'Assist with in-person registration and AV setup.'),
  ('Hybrid Leadership Forum',         'advocacy',       TRUE,  'Moderate the online chat during the forum.')
) AS v(event_name, job, is_virtual, instructions)
JOIN events e ON e.event_name = v.event_name;


-- ============================================
-- SHIFTS
-- Stored as UTC.
-- ============================================
INSERT INTO shifts (opportunity_id, shift_start, shift_end, staff_contact_id, max_volunteers)
SELECT o.opportunity_id, v.shift_start::TIMESTAMP, v.shift_end::TIMESTAMP, s.staff_id, v.max_volunteers
FROM (VALUES
  -- Spring Outreach Fair
  ('Spring Outreach Fair', 'event_support',  '2026-04-11 15:30', '2026-04-11 21:00', 'mgonzalez@org.org', 6),
  ('Spring Outreach Fair', 'event_support',  '2026-04-12 15:30', '2026-04-12 21:30', 'mgonzalez@org.org', 6),
  ('Spring Outreach Fair', 'volunteer_lead', '2026-04-11 15:00', '2026-04-12 00:30', 'mgonzalez@org.org', 1),
  -- Legislative Advocacy Day
  ('Legislative Advocacy Day', 'advocacy',   '2026-04-22 15:00', '2026-04-23 01:00', 'jokafor@org.org',   10),
  -- Public Speaking Workshop
  ('Public Speaking Workshop', 'speaker',    '2026-04-16 00:45', '2026-04-16 03:30', 'spark@org.org',     8),
  -- Neighborhood Canvass
  ('Neighborhood Canvass - Downtown', 'event_support', '2026-04-25 16:45', '2026-04-25 21:00', 'mgonzalez@org.org', 12),
  -- Office Volunteer Day
  ('Office Volunteer Day', 'event_support',  '2026-05-02 16:00', '2026-05-02 20:00', 'mgonzalez@org.org', 3),
  ('Office Volunteer Day', 'event_support',  '2026-05-02 20:00', '2026-05-03 00:00', 'mgonzalez@org.org', 3),
  ('Office Volunteer Day', 'event_support',  '2026-05-09 16:00', '2026-05-09 20:00', 'mgonzalez@org.org', 3),
  ('Office Volunteer Day', 'event_support',  '2026-05-09 20:00', '2026-05-10 00:00', 'mgonzalez@org.org', 3),
  -- North LV Community Forum
  ('North LV Community Forum', 'event_support', '2026-05-07 00:30', '2026-05-07 03:30', 'mgonzalez@org.org', 5),
  -- Speakers Bureau Showcase
  ('Speakers Bureau Showcase', 'speaker',    '2026-05-14 00:00', '2026-05-14 03:30', 'spark@org.org',     6),
  -- Henderson Resource Fair
  ('Henderson Resource Fair', 'event_support', '2026-05-16 15:15', '2026-05-16 22:00', 'mgonzalez@org.org', 8),
  ('Henderson Resource Fair', 'event_support', '2026-05-17 15:15', '2026-05-17 20:00', 'mgonzalez@org.org', 5),
  -- Phoenix Advocacy Summit
  ('Phoenix Advocacy Summit', 'advocacy',      '2026-04-30 14:00', '2026-04-30 20:00', 'jokafor@org.org',  15),
  ('Phoenix Advocacy Summit', 'volunteer_lead','2026-04-30 13:00', '2026-04-30 23:00', 'mgonzalez@org.org', 2),
  -- Denver Outreach Blitz
  ('Denver Outreach Blitz', 'event_support',   '2026-05-09 14:00', '2026-05-09 18:00', 'mgonzalez@org.org', 10),
  ('Denver Outreach Blitz', 'event_support',   '2026-05-09 18:00', '2026-05-09 22:00', 'mgonzalez@org.org', 10),
  -- Seattle Speakers Night
  ('Seattle Speakers Night', 'speaker',        '2026-05-21 00:00', '2026-05-21 03:00', 'spark@org.org',     8),
  -- Virtual Town Hall
  ('Virtual Town Hall', 'advocacy',            '2026-04-18 23:50', '2026-04-19 02:00', 'jokafor@org.org',  20),
  -- Virtual Advocacy Training
  ('Virtual Advocacy Training', 'attendee_only', '2026-05-07 19:00', '2026-05-07 21:00', 'jokafor@org.org', 30),
  ('Virtual Advocacy Training', 'attendee_only', '2026-05-14 19:00', '2026-05-14 21:00', 'jokafor@org.org', 30),
  -- Hybrid Leadership Forum
  ('Hybrid Leadership Forum', 'event_support', '2026-06-05 16:30', '2026-06-05 23:00', 'mgonzalez@org.org', 6),
  ('Hybrid Leadership Forum', 'advocacy',      '2026-06-05 17:00', '2026-06-05 23:00', 'jokafor@org.org',   5)
) AS v(event_name, job, shift_start, shift_end, staff_email, max_volunteers)
JOIN events e ON e.event_name = v.event_name
JOIN opportunities o ON o.event_id = e.event_id AND o.job = v.job::job_type
JOIN staff s ON s.email = v.staff_email;


-- ============================================
-- VOLUNTEER SHIFT ASSIGNMENTS
-- ============================================
INSERT INTO volunteer_shifts (volunteer_id, shift_id, assigned_at)
SELECT vol.volunteer_id, sh.shift_id, NOW()
FROM (VALUES
  ('alice.t@email.com',  'Spring Outreach Fair',            'event_support',  '2026-04-11 15:30'),
  ('alice.t@email.com',  'Virtual Advocacy Training',       'attendee_only',  '2026-05-07 19:00'),
  ('alice.t@email.com',  'Henderson Resource Fair',         'event_support',  '2026-05-16 15:15'),
  ('ben.ruiz@email.com', 'Spring Outreach Fair',            'event_support',  '2026-04-11 15:30'),
  ('ben.ruiz@email.com', 'Office Volunteer Day',            'event_support',  '2026-05-02 16:00'),
  ('ben.ruiz@email.com', 'Henderson Resource Fair',         'event_support',  '2026-05-16 15:15'),
  ('carla.n@email.com',  'Legislative Advocacy Day',        'advocacy',       '2026-04-22 15:00'),
  ('carla.n@email.com',  'Virtual Town Hall',               'advocacy',       '2026-04-18 23:50'),
  ('carla.n@email.com',  'Hybrid Leadership Forum',         'advocacy',       '2026-06-05 17:00'),
  ('david.k@email.com',  'Neighborhood Canvass - Downtown', 'event_support',  '2026-04-25 16:45'),
  ('david.k@email.com',  'North LV Community Forum',        'event_support',  '2026-05-07 00:30'),
  ('estre.m@email.com',  'Public Speaking Workshop',        'speaker',        '2026-04-16 00:45'),
  ('estre.m@email.com',  'Speakers Bureau Showcase',        'speaker',        '2026-05-14 00:00'),
  ('estre.m@email.com',  'Seattle Speakers Night',          'speaker',        '2026-05-21 00:00'),
  ('frank.d@email.com',  'Phoenix Advocacy Summit',         'advocacy',       '2026-04-30 14:00'),
  ('frank.d@email.com',  'Phoenix Advocacy Summit',         'volunteer_lead', '2026-04-30 13:00'),
  ('grace.h@email.com',  'Denver Outreach Blitz',           'event_support',  '2026-05-09 14:00'),
  ('grace.h@email.com',  'Virtual Advocacy Training',       'attendee_only',  '2026-05-14 19:00'),
  ('henry.o@email.com',  'Seattle Speakers Night',          'speaker',        '2026-05-21 00:00'),
  ('henry.o@email.com',  'Virtual Town Hall',               'advocacy',       '2026-04-18 23:50')
) AS v(vol_email, event_name, job, shift_start)
JOIN volunteers vol ON vol.email = v.vol_email
JOIN events e ON e.event_name = v.event_name
JOIN opportunities o ON o.event_id = e.event_id AND o.job = v.job::job_type
JOIN shifts sh ON sh.opportunity_id = o.opportunity_id
              AND sh.shift_start = v.shift_start::TIMESTAMP;
