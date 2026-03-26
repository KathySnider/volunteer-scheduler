-- ============================================================================
-- VOLUNTEER SCHEDULER - FULL SCHEMA INITIALIZATION
-- Combines migrations 000001 through 000009
-- Run this file to create the database from scratch.
-- ============================================================================


-- ============================================================================
-- ENUM TYPES
-- ============================================================================

CREATE TYPE job_type AS ENUM (
    'event_support',
    'advocacy',
    'speaker',
    'volunteer_lead',
    'attendee_only',
    'other'
);

CREATE TYPE volunteer_role AS ENUM (
    'VOLUNTEER',
    'ADMINISTRATOR'
);

CREATE TYPE feedback_type AS ENUM (
    'BUG',
    'ENHANCEMENT',
    'GENERAL'
);

CREATE TYPE feedback_status AS ENUM (
    'OPEN',
    'QUESTION_SENT',
    'RESOLVED_GITHUB',
    'RESOLVED_REJECTED'
);


-- ============================================================================
-- VENUES
-- ============================================================================

CREATE TABLE venues (
    venue_id       SERIAL PRIMARY KEY,
    venue_name     TEXT,
    street_address TEXT NOT NULL,
    city           TEXT NOT NULL,
    state          TEXT NOT NULL,
    zip_code       VARCHAR(10),
    timezone       TEXT NOT NULL DEFAULT 'America/Los_Angeles',
    UNIQUE(street_address, city, state)
);


-- ============================================================================
-- REGIONS (many-to-many with venues)
-- ============================================================================

CREATE TABLE regions (
    region_id SERIAL PRIMARY KEY,
    code      VARCHAR(50) UNIQUE NOT NULL,
    name      TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    CHECK (code = lower(code))
);

CREATE INDEX idx_regions_active ON regions(region_id) WHERE is_active = TRUE;

CREATE TABLE venue_regions (
    venue_id  INT NOT NULL REFERENCES venues(venue_id) ON DELETE CASCADE,
    region_id INT NOT NULL REFERENCES regions(region_id) ON DELETE RESTRICT,
    PRIMARY KEY (venue_id, region_id)
);

CREATE INDEX idx_venue_regions_venue  ON venue_regions(venue_id);
CREATE INDEX idx_venue_regions_region ON venue_regions(region_id);


-- ============================================================================
-- VOLUNTEERS
-- ============================================================================

CREATE TABLE volunteers (
    volunteer_id  SERIAL PRIMARY KEY,
    first_name    TEXT NOT NULL,
    last_name     TEXT NOT NULL,
    email         TEXT UNIQUE,
    phone         VARCHAR(20),
    zip_code      VARCHAR(10),
    role          volunteer_role NOT NULL DEFAULT 'VOLUNTEER',
    is_active     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_login_at TIMESTAMP
);

CREATE INDEX idx_volunteers_email  ON volunteers(email);
CREATE INDEX idx_volunteers_zip    ON volunteers(zip_code);
CREATE INDEX idx_volunteers_role   ON volunteers(role);
CREATE INDEX idx_volunteers_active ON volunteers(volunteer_id) WHERE is_active = TRUE;


-- ============================================================================
-- STAFF
-- ============================================================================

CREATE TABLE staff (
    staff_id   SERIAL PRIMARY KEY,
    first_name TEXT NOT NULL,
    last_name  TEXT NOT NULL,
    email      TEXT UNIQUE NOT NULL,
    phone      VARCHAR(20),
    position   TEXT
);


-- ============================================================================
-- AUTHENTICATION
-- ============================================================================

CREATE TABLE magic_links (
    id         SERIAL PRIMARY KEY,
    email      VARCHAR(255) NOT NULL,
    token      VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    used_at    TIMESTAMP,
    ip_address VARCHAR(45),
    user_agent TEXT
);

CREATE INDEX idx_magic_links_token ON magic_links(token);
CREATE INDEX idx_magic_links_email ON magic_links(email, created_at DESC);

CREATE TABLE sessions (
    id               SERIAL PRIMARY KEY,
    email            VARCHAR(255) NOT NULL UNIQUE,
    token            VARCHAR(255) UNIQUE NOT NULL,
    volunteer_id     INTEGER REFERENCES volunteers(volunteer_id) ON DELETE CASCADE,
    role             volunteer_role NOT NULL DEFAULT 'VOLUNTEER',
    created_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at       TIMESTAMP NOT NULL,
    last_activity_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_sessions_token        ON sessions(token);
CREATE INDEX idx_sessions_email        ON sessions(email);
CREATE INDEX idx_sessions_expires_at   ON sessions(expires_at);
CREATE INDEX idx_sessions_volunteer_id ON sessions(volunteer_id);


-- ============================================================================
-- EVENTS
-- ============================================================================

CREATE TABLE events (
    event_id        SERIAL PRIMARY KEY,
    event_name      TEXT NOT NULL,
    description     TEXT,
    event_is_virtual BOOLEAN DEFAULT FALSE,
    venue_id        INT,
    FOREIGN KEY (venue_id) REFERENCES venues(venue_id) ON DELETE RESTRICT
);

CREATE INDEX idx_events_venue ON events(venue_id);

CREATE TABLE event_dates (
    event_date_id   SERIAL PRIMARY KEY,
    event_id        INT NOT NULL,
    start_date_time TIMESTAMP NOT NULL,
    end_date_time   TIMESTAMP NOT NULL,
    FOREIGN KEY (event_id) REFERENCES events(event_id) ON DELETE CASCADE,
    CONSTRAINT event_dates_check_end_after_start CHECK (end_date_time > start_date_time)
);

CREATE INDEX idx_event_dates_event ON event_dates(event_id);

-- Service types lookup
CREATE TABLE service_types (
    service_type_id SERIAL PRIMARY KEY,
    code            VARCHAR(50) UNIQUE NOT NULL,
    name            TEXT NOT NULL,
    CHECK (code = lower(code))
);

-- Event to service types (many-to-many)
CREATE TABLE event_service_types (
    event_id        INT REFERENCES events(event_id) ON DELETE CASCADE,
    service_type_id INT REFERENCES service_types(service_type_id) ON DELETE CASCADE,
    PRIMARY KEY (event_id, service_type_id)
);


-- ============================================================================
-- OPPORTUNITIES AND SHIFTS
-- ============================================================================

CREATE TABLE opportunities (
    opportunity_id        SERIAL PRIMARY KEY,
    event_id              INT NOT NULL,
    job                   job_type NOT NULL,
    other_job_description TEXT,
    opportunity_is_virtual BOOLEAN DEFAULT FALSE,
    pre_event_instructions TEXT,
    FOREIGN KEY (event_id) REFERENCES events(event_id) ON DELETE CASCADE,
    CHECK (
        (job != 'other' AND other_job_description IS NULL) OR
        (job = 'other'  AND other_job_description IS NOT NULL)
    )
);

CREATE INDEX idx_opportunities_event ON opportunities(event_id);

CREATE TABLE shifts (
    shift_id       SERIAL PRIMARY KEY,
    opportunity_id INT NOT NULL,
    shift_start    TIMESTAMP NOT NULL,
    shift_end      TIMESTAMP NOT NULL,
    staff_contact_id INT,
    max_volunteers INT NOT NULL,
    FOREIGN KEY (opportunity_id) REFERENCES opportunities(opportunity_id) ON DELETE CASCADE,
    FOREIGN KEY (staff_contact_id) REFERENCES staff(staff_id) ON DELETE SET NULL,
    CHECK (shift_end > shift_start),
    CHECK (max_volunteers > 0)
);

CREATE INDEX idx_shifts_opportunity  ON shifts(opportunity_id);
CREATE INDEX idx_shifts_staff_contact ON shifts(staff_contact_id);

-- Volunteer shift assignments
CREATE TABLE volunteer_shifts (
    volunteer_id INT,
    shift_id     INT,
    assigned_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    cancelled_at TIMESTAMP,
    PRIMARY KEY (volunteer_id, shift_id),
    FOREIGN KEY (volunteer_id) REFERENCES volunteers(volunteer_id) ON DELETE CASCADE,
    FOREIGN KEY (shift_id) REFERENCES shifts(shift_id) ON DELETE CASCADE
);

CREATE INDEX idx_volunteer_shifts_volunteer ON volunteer_shifts(volunteer_id);
CREATE INDEX idx_volunteer_shifts_shift     ON volunteer_shifts(shift_id);
CREATE INDEX idx_volunteer_shifts_cancelled ON volunteer_shifts(cancelled_at) WHERE cancelled_at IS NULL;


-- ============================================================================
-- FEEDBACK
-- ============================================================================

CREATE TABLE feedback (
    feedback_id      SERIAL PRIMARY KEY,
    volunteer_id     INT NOT NULL REFERENCES volunteers(volunteer_id) ON DELETE RESTRICT,
    feedback_type    feedback_type NOT NULL DEFAULT 'GENERAL',
    status           feedback_status NOT NULL DEFAULT 'OPEN',
    subject          VARCHAR(100) NOT NULL,
    app_page_name    VARCHAR(100),
    text             TEXT NOT NULL,
    github_issue_url TEXT,
    created_at       TIMESTAMP NOT NULL DEFAULT NOW(),
    last_updated_at  TIMESTAMP,
    resolved_at      TIMESTAMP
);

CREATE INDEX idx_feedback_volunteer  ON feedback(volunteer_id);
CREATE INDEX idx_feedback_status     ON feedback(status);
CREATE INDEX idx_feedback_type       ON feedback(feedback_type);
CREATE INDEX idx_feedback_created_at ON feedback(created_at DESC);

-- Append-only admin notes. No edits or deletes permitted (enforced at application layer).
CREATE TABLE feedback_notes (
    note_id      SERIAL PRIMARY KEY,
    feedback_id  INT NOT NULL REFERENCES feedback(feedback_id) ON DELETE CASCADE,
    volunteer_id INT NOT NULL REFERENCES volunteers(volunteer_id) ON DELETE RESTRICT,
    note         TEXT NOT NULL,
    created_at   TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_feedback_notes_feedback  ON feedback_notes(feedback_id);
CREATE INDEX idx_feedback_notes_volunteer ON feedback_notes(volunteer_id);


-- ============================================================================
-- SEED DATA
-- ============================================================================

-- Service types (AARP-wide convention)
INSERT INTO service_types (code, name) VALUES
    ('outreach',        'Outreach'),
    ('advocacy',        'Advocacy'),
    ('speakers_bureau', 'Speakers Bureau'),
    ('office_support',  'Office Support'),
    ('other',           'Other');

-- Washington state regions
INSERT INTO regions (code, name) VALUES
    ('seattle',      'Seattle Metro'),
    ('spokane',      'Spokane'),
    ('southwest_wa', 'Southwest WA');
