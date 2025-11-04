-- ============================================
-- VOLUNTEER SCHEDULER DATABASE SCHEMA
-- Initial Migration - Creates all tables and types
-- ============================================

-- Create ENUM types
CREATE TYPE qualification_type AS ENUM (
    'outreach', 
    'advocacy', 
    'speakers_bureau', 
    'office_support', 
    'other'
);

CREATE TYPE role_type AS ENUM (
    'event_support', 
    'advocacy', 
    'speaker', 
    'volunteer_lead', 
    'other'
);

-- ============================================
-- CORE TABLES
-- ============================================

-- Locations table (normalized address information)
CREATE TABLE locations (
    location_id SERIAL PRIMARY KEY,
    location_name TEXT,
    street_address TEXT,
    city TEXT,
    state TEXT,
    zip_code VARCHAR(10),
    UNIQUE(street_address, city, state)
);

-- Events table
CREATE TABLE events (
    event_id SERIAL PRIMARY KEY,
    event_name TEXT NOT NULL,
    description TEXT,
    event_is_virtual BOOLEAN DEFAULT FALSE,
    location_id INT,
    FOREIGN KEY (location_id) REFERENCES locations(location_id)
);

-- Event dates table (handles multi-day events)
CREATE TABLE event_dates (
    event_date_id SERIAL PRIMARY KEY,
    event_id INT NOT NULL,
    event_date DATE NOT NULL,
    start_time TIME,
    end_time TIME,
    FOREIGN KEY (event_id) REFERENCES events(event_id) ON DELETE CASCADE,
    UNIQUE(event_id, event_date)
);

-- Volunteers table
CREATE TABLE volunteers (
    volunteer_id SERIAL PRIMARY KEY,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    email TEXT UNIQUE,
    phone VARCHAR(20),
    zip_code VARCHAR(10),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Volunteer qualifications (normalized many-to-many)
CREATE TABLE volunteer_qualifications (
    volunteer_id INT,
    qualification qualification_type,
    other_description TEXT,
    acquired_date DATE,
    PRIMARY KEY (volunteer_id, qualification),
    FOREIGN KEY (volunteer_id) REFERENCES volunteers(volunteer_id) ON DELETE CASCADE,
    CHECK (
        (qualification != 'other' AND other_description IS NULL) OR
        (qualification = 'other' AND other_description IS NOT NULL)
    )
);

-- Staff table (separate from volunteers)
CREATE TABLE staff (
    staff_id SERIAL PRIMARY KEY,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    phone VARCHAR(20),
    position TEXT
);

-- Opportunities table
CREATE TABLE opportunities (
    opportunity_id SERIAL PRIMARY KEY,
    event_id INT NOT NULL,
    role role_type NOT NULL,
    other_role_description TEXT,
    opportunity_is_virtual BOOLEAN DEFAULT FALSE,
    pre_event_instructions TEXT,
    FOREIGN KEY (event_id) REFERENCES events(event_id) ON DELETE CASCADE,
    CHECK (
        (role != 'other' AND other_role_description IS NULL) OR
        (role = 'other' AND other_role_description IS NOT NULL)
    )
);

-- Opportunity requirements (normalized many-to-many)
CREATE TABLE opportunity_requirements (
    opportunity_id INT,
    required_qualification qualification_type,
    PRIMARY KEY (opportunity_id, required_qualification),
    FOREIGN KEY (opportunity_id) REFERENCES opportunities(opportunity_id) ON DELETE CASCADE
);

-- Shifts table
CREATE TABLE shifts (
    shift_id SERIAL PRIMARY KEY,
    opportunity_id INT NOT NULL,
    shift_start TIMESTAMP NOT NULL,
    shift_end TIMESTAMP NOT NULL,
    staff_lead_id INT,
    max_volunteers INT,
    FOREIGN KEY (opportunity_id) REFERENCES opportunities(opportunity_id) ON DELETE CASCADE,
    FOREIGN KEY (staff_lead_id) REFERENCES staff(staff_id),
    CHECK (shift_end > shift_start)
);

-- Volunteer shift assignments (junction table)
CREATE TABLE volunteer_shifts (
    volunteer_id INT,
    shift_id INT,
    assigned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    status TEXT DEFAULT 'confirmed',
    notes TEXT,
    PRIMARY KEY (volunteer_id, shift_id),
    FOREIGN KEY (volunteer_id) REFERENCES volunteers(volunteer_id) ON DELETE CASCADE,
    FOREIGN KEY (shift_id) REFERENCES shifts(shift_id) ON DELETE CASCADE
);

-- ============================================
-- OPTIONAL TABLES
-- ============================================

-- Volunteer preferences
CREATE TABLE volunteer_preferences (
    volunteer_id INT PRIMARY KEY,
    preferred_roles role_type[],
    max_distance_miles INT,
    availability_notes TEXT,
    FOREIGN KEY (volunteer_id) REFERENCES volunteers(volunteer_id) ON DELETE CASCADE
);

-- Event attendees (separate from volunteers)
CREATE TABLE event_attendees (
    attendee_id SERIAL PRIMARY KEY,
    event_id INT NOT NULL,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    email TEXT NOT NULL,
    registered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (event_id) REFERENCES events(event_id) ON DELETE CASCADE
);

-- ============================================
-- INDEXES FOR PERFORMANCE
-- ============================================

CREATE INDEX idx_events_location ON events(location_id);
CREATE INDEX idx_event_dates_event ON event_dates(event_id);
CREATE INDEX idx_event_dates_date ON event_dates(event_date);
CREATE INDEX idx_volunteers_email ON volunteers(email);
CREATE INDEX idx_volunteers_zip ON volunteers(zip_code);
CREATE INDEX idx_opportunities_event ON opportunities(event_id);
CREATE INDEX idx_shifts_opportunity ON shifts(opportunity_id);
CREATE INDEX idx_shifts_staff_lead ON shifts(staff_lead_id);
CREATE INDEX idx_volunteer_shifts_volunteer ON volunteer_shifts(volunteer_id);
CREATE INDEX idx_volunteer_shifts_shift ON volunteer_shifts(shift_id);