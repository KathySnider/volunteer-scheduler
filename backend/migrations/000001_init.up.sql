-- ============================================================================
-- VOLUNTEER SCHEDULER - FULL SCHEMA INITIALIZATION
-- ============================================================================


-- ============================================================================
-- ENUM TYPES
-- ============================================================================

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

CREATE TYPE feedback_note_type AS ENUM (
    'ADMIN_NOTE',
    'QUESTION',
    'VOLUNTEER_REPLY',
    'EMAIL_TO_VOLUNTEER'
);


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
-- FUNDING ENTITIES
-- ============================================================================

CREATE TABLE funding_entities (
    id          SERIAL PRIMARY KEY,
    name        TEXT NOT NULL UNIQUE,
    description TEXT,
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO funding_entities (name) VALUES
    ('Seattle Area'),
    ('Spokane Area'),
    ('Statewide');


-- ============================================================================
-- EVENTS
-- ============================================================================

CREATE TABLE events (
    event_id          SERIAL PRIMARY KEY,
    event_name        TEXT NOT NULL,
    description       TEXT,
    event_is_virtual  BOOLEAN DEFAULT FALSE,
    venue_id          INT REFERENCES venues(venue_id) ON DELETE RESTRICT,
    funding_entity_id INT NOT NULL REFERENCES funding_entities(id) ON DELETE RESTRICT
);

CREATE INDEX idx_events_venue          ON events(venue_id);
CREATE INDEX idx_events_funding_entity ON events(funding_entity_id);

CREATE TABLE event_dates (
    event_date_id   SERIAL PRIMARY KEY,
    event_id        INT NOT NULL REFERENCES events(event_id) ON DELETE CASCADE,
    start_date_time TIMESTAMP NOT NULL,
    end_date_time   TIMESTAMP NOT NULL,
    CONSTRAINT event_dates_check_end_after_start CHECK (end_date_time > start_date_time)
);

CREATE INDEX idx_event_dates_event ON event_dates(event_id);

CREATE TABLE service_types (
    service_type_id SERIAL PRIMARY KEY,
    code            VARCHAR(50) UNIQUE NOT NULL,
    name            TEXT NOT NULL,
    CHECK (code = lower(code))
);

CREATE TABLE event_service_types (
    event_id        INT REFERENCES events(event_id) ON DELETE CASCADE,
    service_type_id INT REFERENCES service_types(service_type_id) ON DELETE CASCADE,
    PRIMARY KEY (event_id, service_type_id)
);


-- ============================================================================
-- JOB TYPES
-- ============================================================================

CREATE TABLE job_types (
    job_type_id SERIAL PRIMARY KEY,
    code        VARCHAR(50) UNIQUE NOT NULL,
    name        TEXT NOT NULL,
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order  INT NOT NULL DEFAULT 0,
    CHECK (code = lower(code))
);

CREATE INDEX idx_job_types_active ON job_types(job_type_id) WHERE is_active = TRUE;

INSERT INTO job_types (code, name, sort_order) VALUES
    ('event_support',  'Event Support',   10),
    ('advocacy',       'Advocacy',        20),
    ('speaker',        'Speaker',         30),
    ('volunteer_lead', 'Volunteer Lead',  40),
    ('attendee_only',  'Attendee Only',   50),
    ('other',          'Other',           60);


-- ============================================================================
-- OPPORTUNITIES AND SHIFTS
-- ============================================================================

CREATE TABLE opportunities (
    opportunity_id         SERIAL PRIMARY KEY,
    event_id               INT NOT NULL REFERENCES events(event_id) ON DELETE CASCADE,
    job_type_id            INT NOT NULL REFERENCES job_types(job_type_id) ON DELETE RESTRICT,
    opportunity_is_virtual BOOLEAN DEFAULT FALSE,
    pre_event_instructions TEXT
);

CREATE INDEX idx_opportunities_event ON opportunities(event_id);

CREATE TABLE shifts (
    shift_id         SERIAL PRIMARY KEY,
    opportunity_id   INT NOT NULL REFERENCES opportunities(opportunity_id) ON DELETE CASCADE,
    shift_start      TIMESTAMP NOT NULL,
    shift_end        TIMESTAMP NOT NULL,
    staff_contact_id INT REFERENCES staff(staff_id) ON DELETE SET NULL,
    max_volunteers   INT NOT NULL,
    CHECK (shift_end > shift_start),
    CHECK (max_volunteers > 0)
);

CREATE INDEX idx_shifts_opportunity   ON shifts(opportunity_id);
CREATE INDEX idx_shifts_staff_contact ON shifts(staff_contact_id);

CREATE TABLE volunteer_shifts (
    volunteer_id INT REFERENCES volunteers(volunteer_id) ON DELETE CASCADE,
    shift_id     INT REFERENCES shifts(shift_id) ON DELETE CASCADE,
    assigned_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    cancelled_at TIMESTAMP,
    PRIMARY KEY (volunteer_id, shift_id)
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

-- Append-only notes. No edits or deletes permitted (enforced at application layer).
CREATE TABLE feedback_notes (
    note_id      SERIAL PRIMARY KEY,
    feedback_id  INT NOT NULL REFERENCES feedback(feedback_id) ON DELETE CASCADE,
    volunteer_id INT NOT NULL REFERENCES volunteers(volunteer_id) ON DELETE RESTRICT,
    note         TEXT NOT NULL,
    note_type    feedback_note_type NOT NULL DEFAULT 'ADMIN_NOTE',
    created_at   TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_feedback_notes_feedback  ON feedback_notes(feedback_id);
CREATE INDEX idx_feedback_notes_volunteer ON feedback_notes(volunteer_id);
CREATE INDEX idx_feedback_notes_note_type ON feedback_notes(note_type);

CREATE TABLE feedback_attachments (
    attachment_id SERIAL PRIMARY KEY,
    feedback_id   INTEGER NOT NULL REFERENCES feedback(feedback_id),
    filename      TEXT NOT NULL,
    mime_type     TEXT NOT NULL,
    file_data     BYTEA NOT NULL,
    file_size     INTEGER NOT NULL,
    created_at    TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_feedback_attachments_feedback_id ON feedback_attachments(feedback_id);


-- ============================================================================
-- SEED DATA
-- ============================================================================

INSERT INTO service_types (code, name) VALUES
    ('outreach',        'Outreach'),
    ('advocacy',        'Advocacy'),
    ('speakers_bureau', 'Speakers Bureau'),
    ('office_support',  'Office Support'),
    ('other',           'Other');
