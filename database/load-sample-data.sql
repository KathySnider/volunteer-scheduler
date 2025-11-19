-- Load sample data from CSV files
-- By default, this block is commented out. 

TRUNCATE event_attendees CASCADE;
TRUNCATE volunteer_shifts CASCADE;
TRUNCATE shifts CASCADE;
TRUNCATE opportunity_requirements CASCADE;
TRUNCATE opportunities CASCADE;
TRUNCATE staff CASCADE;
TRUNCATE volunteer_preferences CASCADE;
TRUNCATE volunteer_qualifications CASCADE;
TRUNCATE volunteers CASCADE;
TRUNCATE event_dates CASCADE;
TRUNCATE events CASCADE;
TRUNCATE locations CASCADE;

\copy locations FROM 'sample-data/01-locations.csv' CSV HEADER;
\copy events FROM 'sample-data/02-events.csv' CSV HEADER;
\copy event_dates FROM 'sample-data/03-event_dates.csv' CSV HEADER;
\copy volunteers FROM 'sample-data/04-volunteers.csv' CSV HEADER;
\copy volunteer_qualifications FROM 'sample-data/05-volunteer_qualifications.csv' CSV HEADER;
\copy volunteer_preferences FROM 'sample-data/06-volunteer_preferences.csv' CSV HEADER;
\copy staff FROM 'sample-data/07-staff.csv' CSV HEADER;
\copy opportunities FROM 'sample-data/08-opportunities.csv' CSV HEADER;
\copy opportunity_requirements FROM 'sample-data/09-opportunity_requirements.csv' CSV HEADER;
\copy shifts FROM 'sample-data/10-shifts.csv' CSV HEADER;
\copy volunteer_shifts FROM 'sample-data/11-volunteer_shifts.csv' CSV HEADER;
\copy event_attendees FROM 'sample-data/12-event_attendees.csv' CSV HEADER;
