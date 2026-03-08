-- ============================================
-- MIGRATION 000001 - Initial Schema
-- ============================================

-- Create ENUM types
CREATE TYPE job_type AS ENUM (
    'event_support', 
    'advocacy', 
    'speaker', 
    'volunteer_lead', 
    'attendee_only',
    'other'
);

-- Venues
CREATE TABLE venues (
    venue_id SERIAL PRIMARY KEY,
    venue_name TEXT,
    street_address TEXT,
    city TEXT,
    state TEXT,
    zip_code VARCHAR(10),
    UNIQUE(street_address, city, state)
);

-- Events
CREATE TABLE events (
    event_id SERIAL PRIMARY KEY,
    event_name TEXT NOT NULL,
    description TEXT,
    event_is_virtual BOOLEAN DEFAULT FALSE,
    venue_id INT,
    FOREIGN KEY (venue_id) REFERENCES venues(venue_id) ON DELETE RESTRICT
);

-- Event dates (handles multi-day events)
CREATE TABLE event_dates (
    event_date_id SERIAL PRIMARY KEY,
    event_id INT NOT NULL,
    event_date DATE NOT NULL,
    start_time TIME,
    end_time TIME,
    FOREIGN KEY (event_id) REFERENCES events(event_id) ON DELETE CASCADE,
    UNIQUE(event_id, event_date)
);

-- Volunteers
CREATE TABLE volunteers (
    volunteer_id SERIAL PRIMARY KEY,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    email TEXT UNIQUE,
    phone VARCHAR(20),
    zip_code VARCHAR(10),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Staff
CREATE TABLE staff (
    staff_id SERIAL PRIMARY KEY,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    phone VARCHAR(20),
    position TEXT
);

-- Opportunities
CREATE TABLE opportunities (
    opportunity_id SERIAL PRIMARY KEY,
    event_id INT NOT NULL,
    job job_type NOT NULL,
    other_job_description TEXT,
    opportunity_is_virtual BOOLEAN DEFAULT FALSE,
    pre_event_instructions TEXT,
    FOREIGN KEY (event_id) REFERENCES events(event_id) ON DELETE CASCADE,
    CHECK (
        (job != 'other' AND other_job_description IS NULL) OR
        (job = 'other' AND other_job_description IS NOT NULL)
    )
);

-- Shifts
CREATE TABLE shifts (
    shift_id SERIAL PRIMARY KEY,
    opportunity_id INT NOT NULL,
    shift_start TIMESTAMP NOT NULL,
    shift_end TIMESTAMP NOT NULL,
    staff_contact_id INT,
    max_volunteers INT NOT NULL,
    FOREIGN KEY (opportunity_id) REFERENCES opportunities(opportunity_id) ON DELETE CASCADE,
    FOREIGN KEY (staff_contact_id) REFERENCES staff(staff_id) ON DELETE SET NULL,
    CHECK (shift_end > shift_start),
    CHECK (max_volunteers > 0)
);

-- Volunteer shift assignments
CREATE TABLE volunteer_shifts (
    volunteer_id INT,
    shift_id INT,
    assigned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (volunteer_id, shift_id),
    FOREIGN KEY (volunteer_id) REFERENCES volunteers(volunteer_id) ON DELETE CASCADE,
    FOREIGN KEY (shift_id) REFERENCES shifts(shift_id) ON DELETE CASCADE
);

-- Service types lookup
CREATE TABLE service_types (
    service_type_id SERIAL PRIMARY KEY,
    code VARCHAR(50) UNIQUE NOT NULL,
    name TEXT NOT NULL,
    CHECK (code = lower(code))
);

-- Event to service types (many-to-many)
CREATE TABLE event_service_types (
    event_id INT REFERENCES events(event_id) ON DELETE CASCADE,
    service_type_id INT REFERENCES service_types(service_type_id) ON DELETE CASCADE,
    PRIMARY KEY (event_id, service_type_id)
);

-- ============================================
-- INDEXES
-- ============================================
CREATE INDEX idx_events_venue ON events(venue_id);
CREATE INDEX idx_event_dates_event ON event_dates(event_id);
CREATE INDEX idx_event_dates_date ON event_dates(event_date);
CREATE INDEX idx_volunteers_email ON volunteers(email);
CREATE INDEX idx_volunteers_zip ON volunteers(zip_code);
CREATE INDEX idx_opportunities_event ON opportunities(event_id);
CREATE INDEX idx_shifts_opportunity ON shifts(opportunity_id);
CREATE INDEX idx_shifts_staff_contact ON shifts(staff_contact_id);
CREATE INDEX idx_volunteer_shifts_volunteer ON volunteer_shifts(volunteer_id);
CREATE INDEX idx_volunteer_shifts_shift ON volunteer_shifts(shift_id);
