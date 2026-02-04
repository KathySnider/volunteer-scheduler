import React, { useState, useEffect } from 'react';
import { Calendar, MapPin, Monitor, Users, ChevronDown, UserCircle, LogOut } from 'lucide-react';
import { useRouter } from 'next/navigation';

const VolunteerEventsApp = () => {
  const [events, setEvents] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  const router = useRouter();

  // These are for the showNameModal popup.
  const [currentVolunteer, setCurrentVolunteer] = useState(null);
  const [showNameModal, setShowNameModal] = useState(false);
  const [nameInput, setNameInput] = useState('');
  const [nameError, setNameError] = useState('');
  const [allVolunteers, setAllVolunteers] = useState([]);
  const [showVolunteerDropdown, setShowVolunteerDropdown] = useState(false);
  const [filteredVolunteers, setFilteredVolunteers] = useState([]);

  // Available cities - will be fetched from DB
  const [availableCities, setAvailableCities] = useState([]);
  
  // Available roles (static)
  const availableRoles = [
    { value: 'EVENT_SUPPORT', label: 'Event Support' },
    { value: 'ADVOCACY', label: 'Advocacy' },
    { value: 'SPEAKER', label: 'Speaker' },
    { value: 'VOLUNTEER_LEAD', label: 'Volunteer Lead' },
    { value: 'ATTENDEE_ONLY', label: 'Attendee Only' },
    { value: 'OTHER', label: 'Other' }
  ];

  // Get current date in YYYY-MM-DD format
  const getCurrentDate = () => {
    const today = new Date();
    return today.toISOString().split('T')[0];
  };

  // Filter states with defaults
  const [selectedCities, setSelectedCities] = useState([]);
  const [selectedRoles, setSelectedRoles] = useState(availableRoles.map(r => r.value));
  const [eventType, setEventType] = useState('');
  const [startDate, setStartDate] = useState(getCurrentDate());
  const [endDate, setEndDate] = useState('');
  const [citiesDropdownOpen, setCitiesDropdownOpen] = useState(false);

  const eventTypes = [
    { value: '', label: 'All Events' },
    { value: 'VIRTUAL', label: 'Virtual' },
    { value: 'IN_PERSON', label: 'In Person' },
    { value: 'HYBRID', label: 'Hybrid' }
  ];

  const handleNameSubmit = async () => {
    if (!nameInput.trim()) {
      setNameError('Please enter your name or select from the list.');
      return;
    }

    // Has the user selected an existing volunteer?
    const exactMatch = allVolunteers.find(v => 
      `${v.firstName} ${v.lastName}`.toLowerCase() === nameInput.trim().toLowerCase()
    );

    if (exactMatch) {
      selectExistingVolunteer(exactMatch);
      return;
    }

    // Create new volunteer 
    // TODO: This must move to the admin-only priveleges!
    const nameParts = nameInput.trim().split(/\s+/);
    const firstName = nameParts[0];
    const lastName = nameParts.slice(1).join(' ') || '';

    if (!lastName) {
      setNameError('Please enter both first and last name.');
      return;
    }

    const createMutation = `
      mutation($firstName: String!, $lastName: String!) {
        createVolunteer(firstName: $firstName, lastName: $lastName) {
          id
          firstName
          lastName
        }
      }
    `;

    try {
      const createResponse = await fetch('http://localhost:8080/query', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          query: createMutation,
          variables: { firstName, lastName }
        })
      });

      const createResult = await createResponse.json();
      
      if (createResult.data?.createVolunteer) {
        const volunteer = createResult.data.createVolunteer;
        setCurrentVolunteer(volunteer);
        localStorage.setItem('currentVolunteer', JSON.stringify(volunteer));
        setShowNameModal(false);
      } else {
        setNameError('Failed to create volunteer account');
      }
    } catch (err) {
      setNameError('Failed to connect to server');
    }
  };

  const handleLogout = () => {
    setCurrentVolunteer(null);
    localStorage.removeItem('currentVolunteer');
    setShowNameModal(true);
  };

  const fetchCities = async () => {
    const query = `
      query {
        events {
          venue {
            city
          }
        }
      }
    `;

    try {
      const response = await fetch('http://localhost:8080/query', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ query })
      });

      const result = await response.json();
      
      if (result.data && result.data.events) {
        // Extract unique cities from events, filtering out null/empty values
        const cities = [...new Set(
          result.data.events
            .filter(event => event.venue && event.venue.city && event.venue.city.trim() !== '')
            .map(event => event.venue.city.trim())
        )].sort();
        
        setAvailableCities(cities);
        setSelectedCities(cities); // Select all cities by default
      }
    } catch (err) {
      console.error('Failed to fetch cities:', err);
    }
  };

  const fetchEvents = async () => {
    setLoading(true);
    setError(null);

    const filter = {};
    if (selectedCities.length > 0) filter.cities = selectedCities;
    if (selectedRoles.length > 0) filter.roles = selectedRoles;
    if (eventType) filter.eventType = eventType;
    if (startDate) filter.startDate = startDate;
    if (endDate) filter.endDate = endDate;

    const query = `
      query($filter: EventFilter) {
        events(filter: $filter) {
          id
          name
          description
          eventType
          venue {
            name
            address
            city
            state
            zipCode
          }
          shifts {
            id
            date
            startTime
            endTime
          }
        }
      }
    `;

    try {
      const response = await fetch('http://localhost:8080/query', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          query,
          variables: { filter: Object.keys(filter).length > 0 ? filter : null }
        })
      });

      const result = await response.json();
      
      if (result.errors) {
        setError(result.errors[0].message);
      } else {
        setEvents(result.data.events);
      }
    } catch (err) {
      setError('Failed to fetch events. Make sure the server is running.');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchAllVolunteers();
    // Is volunteer already logged in?
    const storedVolunteer = localStorage.getItem('currentVolunteer');
    if (storedVolunteer) {
      setCurrentVolunteer(JSON.parse(storedVolunteer));
    } else {
      setShowNameModal(true);
    }
    
    fetchCities();
    fetchEvents();
  }, []);

  const fetchAllVolunteers = async () => {


    const query = `
      query {
        allVolunteers {
          id
          firstName
          lastName
          email
          serviceTypes
        }
      }
    `;

    try {
      const response = await fetch('http://localhost:8080/query', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ query })
      });

      const result = await response.json();
      
      if (result.data) {
        if (result.data.allVolunteers) {
          setAllVolunteers(result.data.allVolunteers);
          setFilteredVolunteers(result.data.allVolunteers);
        } else {
          setNameError("No volunteers were returned in the data.")
        }
      } else {
        setNameError("No data was returned from the allVolunteers query.")
      }
    } catch (err) {
      console.error('Failed to fetch volunteers:', err);
    }
  };

  const handleNameInputChange = (value) => {
    setNameInput(value);
    setNameError('');
    setShowVolunteerDropdown(value.length > 0);

    if (value.trim()) {
      const filtered = allVolunteers.filter(v =>
        `${v.firstName} ${v.lastName}`.toLowerCase().includes(value.toLowerCase())
      );
      setFilteredVolunteers(filtered);
    } else {
      setFilteredVolunteers(allVolunteers);
    }
  };

  const selectExistingVolunteer = (volunteer) => {
    setCurrentVolunteer(volunteer);
    localStorage.setItem('currentVolunteer', JSON.stringify(volunteer));
    setShowNameModal(false);
    setShowVolunteerDropdown(false);
  };

  const handleCityToggle = (city) => {
    setSelectedCities(prev =>
      prev.includes(city)
        ? prev.filter(c => c !== city)
        : [...prev, city]
    );
  };

  const handleRoleToggle = (role) => {
    setSelectedRoles(prev =>
      prev.includes(role)
        ? prev.filter(r => r !== role)
        : [...prev, role]
    );
  };

  const formatDate = (dateStr) => {
    const date = new Date(dateStr);
    return date.toLocaleDateString('en-US', { 
      weekday: 'short', 
      month: 'short', 
      day: 'numeric',
      year: 'numeric'
    });
  };

  const formatTime = (timeStr) => {
    const [hours, minutes] = timeStr.split(':');
    const hour = parseInt(hours);
    const ampm = hour >= 12 ? 'p.m.' : 'a.m.';
    const displayHour = hour % 12 || 12;
    return `${displayHour}:${minutes} ${ampm}`;
  };

  const getEventTypeIcon = (type) => {
    switch (type) {
      case 'VIRTUAL':
        return <Monitor className="w-5 h-5 text-blue-600" />;
      case 'IN_PERSON':
        return <MapPin className="w-5 h-5 text-green-600" />;
      case 'HYBRID':
        return <Users className="w-5 h-5 text-purple-600" />;
      default:
        return null;
    }
  };

  const getEventTypeLabel = (type) => {
    return type.replace('_', ' ').toLowerCase().replace(/\b\w/g, l => l.toUpperCase());
  };

  const getEarliestShift = (shifts) => {
    if (!shifts || shifts.length === 0) return null;
    return shifts.sort((a, b) => new Date(a.date) - new Date(b.date))[0];
  };

  return (
    <div className="flex h-screen bg-gray-50">
      {/* Name Input Modal */}
      {showNameModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-8 max-w-md w-full mx-4">
            <h2 className="text-2xl font-bold text-gray-800 mb-4">Welcome!</h2>
            <p className="text-gray-600 mb-6">Select your name or enter it to continue</p>
            
            <div className="relative">
              <input
                type="text"
                value={nameInput}
                onChange={(e) => handleNameInputChange(e.target.value)}
                onKeyUp={(e) => e.key === 'Enter' && handleNameSubmit()}
                onFocus={() => setShowVolunteerDropdown(true)}
                placeholder="Type or select your name..."
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 mb-2 bg-white"
                id="volNameInput"
              />
              
              {showVolunteerDropdown && filteredVolunteers.length > 0 && (
                <div className="absolute z-10 w-full mt-1 bg-white border border-gray-300 rounded-lg shadow-lg max-h-60 overflow-y-auto">
                  {filteredVolunteers.map(volunteer => (
                    <div
                      key={volunteer.id}
                      onClick={() => selectExistingVolunteer(volunteer)}
                      className="px-4 py-2 hover:bg-blue-50 cursor-pointer"
                    >
                      {volunteer.firstName} {volunteer.lastName}
                    </div>
                  ))}
                </div>
              )}
            </div>
            
            {nameError && (
              <p className="text-red-600 text-sm mb-4">{nameError}</p>
            )}
            
            {filteredVolunteers.length === 0 && nameInput.trim() && (
              <p className="text-blue-600 text-sm mb-4">
                Name not found. Click Continue to create a new volunteer account.
              </p>
            )}
            
            <button
              onClick={handleNameSubmit}
              className="w-full bg-blue-600 text-white py-3 rounded-lg font-semibold hover:bg-blue-700 transition-colors"
            >
              Continue
            </button>
          </div>
        </div>
      )}

      {/* Sidebar */}
      <div className="w-80 bg-white shadow-lg p-6 overflow-y-auto">
        {/* User Info */}
        {currentVolunteer && (
          <div className="mb-6 pb-6 border-b border-gray-200">
            <div className="flex items-center gap-3 mb-3">
              <UserCircle className="w-10 h-10 text-blue-600" />
              <div>
                <p className="font-semibold text-gray-800">
                  {currentVolunteer.firstName} {currentVolunteer.lastName}
                </p>
                <p className="text-sm text-gray-500">Volunteer</p>
              </div>
            </div>
            <button
              onClick={handleLogout}
              className="flex items-center gap-2 text-sm text-gray-600 hover:text-red-600 transition-colors"
            >
              <LogOut className="w-4 h-4" />
              Sign Out
            </button>
          </div>
        )}

        <h2 className="text-2xl font-bold text-gray-800 mb-6">Find Events</h2>
        
        {/* Cities Filter */}
        <div className="mb-6">
          <label className="block text-sm font-semibold text-gray-700 mb-2">Cities</label>
          <div className="relative">
            <button
              onClick={() => setCitiesDropdownOpen(!citiesDropdownOpen)}
              className="w-full px-4 py-2 bg-white border border-gray-300 rounded-lg flex items-center justify-between hover:border-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <span className="text-gray-700">
                {selectedCities.length === 0
                  ? 'Select cities...'
                  : `${selectedCities.length} selected`}
              </span>
              <ChevronDown className="w-4 h-4 text-gray-500" />
            </button>
            
            {citiesDropdownOpen && (
              <div className="absolute z-10 w-full mt-1 bg-white border border-gray-300 rounded-lg shadow-lg max-h-60 overflow-y-auto">
                {availableCities.map(city => (
                  <label
                    key={city}
                    className="flex items-center px-4 py-2 hover:bg-gray-50 cursor-pointer"
                  >
                    <input
                      type="checkbox"
                      checked={selectedCities.includes(city)}
                      onChange={() => handleCityToggle(city)}
                      className="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
                      id="cityCheckbox"
                    />
                    <span className="ml-3 text-gray-700">{city}</span>
                  </label>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Event Type Filter */}
        <div className="mb-6">
          <label className="block text-sm font-semibold text-gray-700 mb-2">Event Type</label>
          <select
            value={eventType}
            onChange={(e) => setEventType(e.target.value)}
            className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white"
          >
            {eventTypes.map(type => (
              <option key={type.value} value={type.value}>{type.label}</option>
            ))}
          </select>
        </div>

        {/* Roles Filter */}
        <div className="mb-6">
          <label className="block text-sm font-semibold text-gray-700 mb-2">Roles</label>
          <div className="space-y-2">
            {availableRoles.map(role => (
              <label key={role.value} className="flex items-center cursor-pointer">
                <input
                  type="checkbox"
                  checked={selectedRoles.includes(role.value)}
                  onChange={() => handleRoleToggle(role.value)}
                  className="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
                  id="rolesCheckbox"
                />
                <span className="ml-3 text-gray-700">{role.label}</span>
              </label>
            ))}
          </div>
        </div>

        {/* Date Range */}
        <div className="mb-6">
          <label className="block text-sm font-semibold text-gray-700 mb-2">Start Date</label>
          <input
            type="date"
            value={startDate}
            onChange={(e) => setStartDate(e.target.value)}
            className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white"
            id="startDateInput"
          />
        </div>

        <div className="mb-6">
          <label className="block text-sm font-semibold text-gray-700 mb-2">End Date</label>
          <input
            type="date"
            value={endDate}
            onChange={(e) => setEndDate(e.target.value)}
            className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white"
            id="endDateInput"
          />
        </div>

        {/* Search Button */}
        <button
          onClick={fetchEvents}
          disabled={loading}
          className="w-full bg-blue-600 text-white py-3 rounded-lg font-semibold hover:bg-blue-700 transition-colors disabled:bg-gray-400"
        >
          {loading ? 'Searching...' : 'Search Events'}
        </button>

        {/* Clear Filters */}
        <button
          onClick={() => {
            setSelectedCities(availableCities);
            setSelectedRoles(availableRoles.map(r => r.value));
            setEventType('');
            setStartDate(getCurrentDate());
            setEndDate('');
          }}
          className="w-full mt-3 text-gray-600 py-2 rounded-lg font-medium hover:bg-gray-100 transition-colors"
        >
          Reset to Defaults
        </button>
      </div>

      {/* Main Content */}
      <div className="flex-1 overflow-y-auto p-8">
        <div className="max-w-5xl mx-auto">
          <h1 className="text-3xl font-bold text-gray-800 mb-2">Volunteer Events</h1>
          <p className="text-gray-600 mb-8">
            {events.length} {events.length === 1 ? 'event' : 'events'} found
          </p>

          {error && (
            <div className="bg-red-50 border-l-4 border-red-500 p-4 mb-6">
              <p className="text-red-800">{error}</p>
            </div>
          )}

          {loading ? (
            <div className="text-center py-12">
              <div className="inline-block animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
              <p className="mt-4 text-gray-600">Loading events...</p>
            </div>
          ) : (
            <div className="space-y-6">
              {events.map((event, index) => {
                const earliestShift = getEarliestShift(event.shifts);
                return (
                  <div key={`${event.id}-${index}`} className="bg-white rounded-lg shadow-md overflow-hidden hover:shadow-lg transition-shadow">
                    <div className="flex">
                      {/* Placeholder Image */}
                      <div className="w-64 h-48 bg-gradient-to-br from-blue-400 to-purple-500 flex items-center justify-center flex-shrink-0">
                        <Calendar className="w-16 h-16 text-white opacity-50" />
                      </div>

                      {/* Event Details */}
                      <div className="flex-1 p-6">
                        <div className="flex items-start justify-between mb-3">
                          <h3 className="text-xl font-bold text-gray-800">{event.name}</h3>
                          <div className="flex items-center gap-2 px-3 py-1 bg-gray-100 rounded-full">
                            {getEventTypeIcon(event.eventType)}
                            <span className="text-sm font-medium text-gray-700">
                              {getEventTypeLabel(event.eventType)}
                            </span>
                          </div>
                        </div>

                        <p className="text-gray-600 mb-4 line-clamp-2">{event.description}</p>

                        <div className="space-y-2 mb-4">
                          {earliestShift && (
                            <div className="flex items-center text-gray-700">
                              <Calendar className="w-4 h-4 mr-2" />
                              <span className="text-sm">
                                {formatDate(earliestShift.date)} at {formatTime(earliestShift.startTime)} PT
                              </span>
                            </div>
                          )}

                          {event.venue ? (
                            <div className="flex items-center text-gray-700">
                              <MapPin className="w-4 h-4 mr-2" />
                              <span className="text-sm">
                                {event.venue.name ? `${event.venue.name}, ` : ''}
                                {event.venue.city}, {event.venue.state}
                              </span>
                            </div>
                          ) : (
                            <div className="flex items-center text-gray-700">
                              <Monitor className="w-4 h-4 mr-2" />
                              <span className="text-sm">Online Event</span>
                            </div>
                          )}
                        </div>

                        <button 
                          onClick={() => router.push(`/event/${event.id}?volunteerId=${currentVolunteer?.id}`)}
                          className="px-6 py-2 bg-blue-600 text-white rounded-lg font-medium hover:bg-blue-700 transition-colors"
                        >
                          More Info
                        </button>
                      </div>
                    </div>
                  </div>
                );
              })}

              {!loading && events.length === 0 && (
                <div className="text-center py-12">
                  <Calendar className="w-16 h-16 text-gray-400 mx-auto mb-4" />
                  <h3 className="text-xl font-semibold text-gray-700 mb-2">No events found</h3>
                  <p className="text-gray-600">Try adjusting your filters to see more results</p>
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default VolunteerEventsApp;