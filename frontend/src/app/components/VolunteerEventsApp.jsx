'use client';

import React, { useState, useEffect } from 'react';
import { Calendar, MapPin, Monitor, Users, ChevronDown } from 'lucide-react';
import { useRouter } from 'next/navigation';

const VolunteerEventsApp = () => {
  const router = useRouter();
  const [events, setEvents] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  
  // Available cities - will be fetched from DB
  const [availableCities, setAvailableCities] = useState([]);
  
  // Available roles (static)
  const availableRoles = [
    { value: 'EVENT_SUPPORT', label: 'Event Support' },
    { value: 'ADVOCACY', label: 'Advocacy' },
    { value: 'SPEAKER', label: 'Speaker' },
    { value: 'VOLUNTEER_LEAD', label: 'Volunteer Lead' },
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

  const fetchCities = async () => {
    const query = `
      query {
        events {
          location {
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
            .filter(event => event.location && event.location.city && event.location.city.trim() !== '')
            .map(event => event.location.city.trim())
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
          location {
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
    fetchCities();
    fetchEvents();
  }, []);

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
      {/* Sidebar */}
      <div className="w-80 bg-white shadow-lg p-6 overflow-y-auto">
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
          />
        </div>

        <div className="mb-6">
          <label className="block text-sm font-semibold text-gray-700 mb-2">End Date</label>
          <input
            type="date"
            value={endDate}
            onChange={(e) => setEndDate(e.target.value)}
            className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white"
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

                          {event.location ? (
                            <div className="flex items-center text-gray-700">
                              <MapPin className="w-4 h-4 mr-2" />
                              <span className="text-sm">
                                {event.location.name ? `${event.location.name}, ` : ''}
                                {event.location.city}, {event.location.state}
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
                          onClick={() => {
                            console.log('Event ID:', event.id);
                            router.push(`/event/${event.id}`);

                          }}
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