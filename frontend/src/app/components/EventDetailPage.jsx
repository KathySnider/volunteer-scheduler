'use client';

import React, { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { Calendar, MapPin, Monitor, Users, ArrowLeft, Clock, CheckCircle } from 'lucide-react';

const EventDetailPage = ({ eventId }) => {
  const router = useRouter();
  const [event, setEvent] = useState(null);
  const [opportunities, setOpportunities] = useState([]);
  const [volunteers, setVolunteers] = useState({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [assignments, setAssignments] = useState({});
  const [searchTerms, setSearchTerms] = useState({});
  const [showDropdown, setShowDropdown] = useState({});
  const [assigning, setAssigning] = useState({});

  useEffect(() => {
    if (eventId) {
      fetchEventDetails();
    }
  }, [eventId]);

  const fetchEventDetails = async () => {
    setLoading(true);
    setError(null);

    const query = `
      query($eventId: ID!) {
        event(id: $eventId) {
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
          opportunities {
            id
            role
            requiresQualifications
            shifts {
              id
              date
              startTime
              endTime
              maxVolunteers
              assignedVolunteers {
                id
                firstName
                lastName
              }
            }
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
          variables: { eventId }
        })
      });

      const result = await response.json();
      
      if (result.errors) {
        setError(result.errors[0].message);
      } else if (result.data?.event) {
        setEvent(result.data.event);
        setOpportunities(result.data.event.opportunities || []);
        
        // Fetch volunteers for each opportunity based on requirements
        result.data.event.opportunities?.forEach(opp => {
          fetchVolunteersForOpportunity(opp.id, opp.requiresQualifications);
        });
      }
    } catch (err) {
      setError('Failed to fetch event details');
    } finally {
      setLoading(false);
    }
  };

  const fetchVolunteersForOpportunity = async (opportunityId, requiredQualifications) => {
    const query = requiredQualifications && requiredQualifications.length > 0
      ? `
        query($qualifications: [String!]!) {
          volunteers(qualifications: $qualifications) {
            id
            firstName
            lastName
          }
        }
      `
      : `
        query {
          volunteers {
            id
            firstName
            lastName
          }
        }
      `;

    try {
      const response = await fetch('http://localhost:8080/query', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          query,
          variables: requiredQualifications && requiredQualifications.length > 0 
            ? { qualifications: requiredQualifications }
            : {}
        })
      });

      const result = await response.json();
      
      if (result.data?.volunteers) {
        setVolunteers(prev => ({
          ...prev,
          [opportunityId]: result.data.volunteers
        }));
      }
    } catch (err) {
      console.error('Failed to fetch volunteers:', err);
    }
  };

  const handleAssignVolunteer = async (shiftId, opportunityId) => {
    const volunteerInput = assignments[shiftId];
    if (!volunteerInput || !volunteerInput.trim()) return;

    setAssigning(prev => ({ ...prev, [shiftId]: true }));

    // Check if this is an existing volunteer
    const oppVolunteers = volunteers[opportunityId] || [];
    
    const existingVolunteer = oppVolunteers.find(v => 
      `${v.firstName} ${v.lastName}`.toLowerCase() === volunteerInput.toLowerCase()
    );

    if (!existingVolunteer) {
      alert('Volunteer not found. Please contact your system administrator to add new volunteers.');
      setAssigning(prev => ({ ...prev, [shiftId]: false }));
      return;
    }

    const volunteerId = existingVolunteer.id;

    // Assign volunteer to shift
    const assignMutation = `
      mutation($shiftId: ID!, $volunteerId: ID!) {
        assignVolunteerToShift(shiftId: $shiftId, volunteerId: $volunteerId) {
          success
          message
        }
      }
    `;

    try {
      const assignResponse = await fetch('http://localhost:8080/query', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          query: assignMutation,
          variables: { shiftId, volunteerId }
        })
      });

      const assignResult = await assignResponse.json();
      
      if (assignResult.errors) {
        alert('Failed to assign volunteer: ' + assignResult.errors[0].message);
      } else {
        // Success! Refresh the event details
        setAssignments(prev => ({ ...prev, [shiftId]: '' }));
        setSearchTerms(prev => ({ ...prev, [shiftId]: '' }));
        fetchEventDetails();
      }
    } catch (err) {
      alert('Failed to assign volunteer');
    } finally {
      setAssigning(prev => ({ ...prev, [shiftId]: false }));
    }
  };

  const handleSearchChange = (shiftId, value) => {
    setSearchTerms(prev => ({ ...prev, [shiftId]: value }));
    setAssignments(prev => ({ ...prev, [shiftId]: value }));
    setShowDropdown(prev => ({ ...prev, [shiftId]: value.length > 0 }));
  };

  const selectVolunteer = (shiftId, volunteer) => {
    const fullName = `${volunteer.firstName} ${volunteer.lastName}`;
    setAssignments(prev => ({ ...prev, [shiftId]: fullName }));
    setSearchTerms(prev => ({ ...prev, [shiftId]: fullName }));
    setShowDropdown(prev => ({ ...prev, [shiftId]: false }));
  };

  const getFilteredVolunteers = (opportunityId, searchTerm) => {
    const oppVolunteers = volunteers[opportunityId] || [];
    if (!searchTerm) return oppVolunteers;
    
    const term = searchTerm.toLowerCase();
    return oppVolunteers.filter(v =>
      `${v.firstName} ${v.lastName}`.toLowerCase().includes(term)
    );
  };

  const isShiftAvailable = (shift) => {
    const assigned = shift.assignedVolunteers?.length || 0;
    return assigned < (shift.maxVolunteers || 1);
  };

  const formatDate = (dateStr) => {
    const date = new Date(dateStr);
    return date.toLocaleDateString('en-US', { 
      weekday: 'long',
      month: 'long', 
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
        return <Monitor className="w-6 h-6 text-blue-600" />;
      case 'IN_PERSON':
        return <MapPin className="w-6 h-6 text-green-600" />;
      case 'HYBRID':
        return <Users className="w-6 h-6 text-purple-600" />;
      default:
        return null;
    }
  };

  const getEventTypeLabel = (type) => {
    return type.replace('_', ' ').toLowerCase().replace(/\b\w/g, l => l.toUpperCase());
  };

  const getRoleLabel = (role) => {
    return role.replace('_', ' ').toLowerCase().replace(/\b\w/g, l => l.toUpperCase());
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="text-center">
          <div className="inline-block animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
          <p className="mt-4 text-gray-600">Loading event details...</p>
        </div>
      </div>
    );
  }

  if (error || !event) {
    return (
      <div className="min-h-screen bg-gray-50 p-8">
        <div className="max-w-4xl mx-auto">
          <div className="bg-red-50 border-l-4 border-red-500 p-4">
            <p className="text-red-800">{error || 'Event not found'}</p>
          </div>
          <button
            onClick={() => router.back()}
            className="mt-4 flex items-center gap-2 text-blue-600 hover:text-blue-700"
          >
            <ArrowLeft className="w-4 h-4" />
            Back to Events
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50 p-8">
      <div className="max-w-6xl mx-auto">
        {/* Back Button */}
        <button
          onClick={() => router.back()}
          className="flex items-center gap-2 text-blue-600 hover:text-blue-700 mb-6"
        >
          <ArrowLeft className="w-4 h-4" />
          Back to Events
        </button>

        {/* Event Header */}
        <div className="bg-white rounded-lg shadow-md p-8 mb-6">
          <div className="flex items-start justify-between mb-4">
            <h1 className="text-3xl font-bold text-gray-800">{event.name}</h1>
            <div className="flex items-center gap-2 px-4 py-2 bg-gray-100 rounded-full">
              {getEventTypeIcon(event.eventType)}
              <span className="font-medium text-gray-700">
                {getEventTypeLabel(event.eventType)}
              </span>
            </div>
          </div>

          <p className="text-gray-600 mb-6">{event.description}</p>

          {event.location && (
            <div className="flex items-start gap-2 text-gray-700 mb-2">
              <MapPin className="w-5 h-5 mt-0.5 flex-shrink-0" />
              <div>
                {event.location.name && <p className="font-medium">{event.location.name}</p>}
                <p>{event.location.address}</p>
                <p>{event.location.city}, {event.location.state} {event.location.zipCode}</p>
              </div>
            </div>
          )}
        </div>

        {/* Opportunities and Shifts */}
        <h2 className="text-2xl font-bold text-gray-800 mb-4">Volunteer Opportunities</h2>

        {opportunities.length === 0 ? (
          <div className="bg-white rounded-lg shadow-md p-8 text-center">
            <p className="text-gray-600">No volunteer opportunities available for this event.</p>
          </div>
        ) : (
          <div className="space-y-6">
            {opportunities.map(opportunity => (
              <div key={opportunity.id} className="bg-white rounded-lg shadow-md overflow-hidden">
                <div className="bg-blue-50 px-6 py-4 border-b border-blue-100">
                  <h3 className="text-xl font-semibold text-gray-800">
                    {getRoleLabel(opportunity.role)}
                  </h3>
                  {opportunity.requiresQualifications && opportunity.requiresQualifications.length > 0 && (
                    <p className="text-sm text-gray-600 mt-1">
                      Requires: {opportunity.requiresQualifications.join(', ')}
                    </p>
                  )}
                </div>

                <div className="p-6">
                  <h4 className="font-semibold text-gray-700 mb-3">Available Shifts:</h4>
                  
                  {opportunity.shifts && opportunity.shifts.length > 0 ? (
                    <div className="space-y-4">
                      {opportunity.shifts.map(shift => {
                        const available = isShiftAvailable(shift);
                        const assigned = shift.assignedVolunteers?.length || 0;
                        
                        return (
                          <div key={shift.id} className={`border rounded-lg p-4 ${available ? 'border-gray-200' : 'border-green-200 bg-green-50'}`}>
                            <div className="flex items-start justify-between">
                              <div className="flex-1">
                                <div className="flex items-center gap-3 mb-2">
                                  <Calendar className="w-4 h-4 text-gray-600" />
                                  <span className="font-medium text-gray-800">
                                    {formatDate(shift.date)}
                                  </span>
                                </div>
                                <div className="flex items-center gap-3 mb-2">
                                  <Clock className="w-4 h-4 text-gray-600" />
                                  <span className="text-gray-700">
                                    {formatTime(shift.startTime)} - {formatTime(shift.endTime)}
                                  </span>
                                </div>
                                <div className="flex items-center gap-3">
                                  <Users className="w-4 h-4 text-gray-600" />
                                  <span className="text-sm text-gray-600">
                                    {assigned} / {shift.maxVolunteers || 1} volunteers assigned
                                  </span>
                                </div>

                                {shift.assignedVolunteers && shift.assignedVolunteers.length > 0 && (
                                  <div className="mt-3 flex items-center gap-2 flex-wrap">
                                    <CheckCircle className="w-4 h-4 text-green-600" />
                                    {shift.assignedVolunteers.map(vol => (
                                      <span key={vol.id} className="px-3 py-1 bg-green-100 text-green-800 rounded-full text-sm">
                                        {vol.firstName} {vol.lastName}
                                      </span>
                                    ))}
                                  </div>
                                )}
                              </div>

                              {available && (
                                <div className="ml-4 min-w-[300px]">
                                  <div className="relative">
                                    <input
                                      type="text"
                                      value={searchTerms[shift.id] || ''}
                                      onChange={(e) => handleSearchChange(shift.id, e.target.value)}
                                      onFocus={() => setShowDropdown(prev => ({ ...prev, [shift.id]: true }))}
                                      placeholder="Type volunteer name..."
                                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white"
                                    />
                                    
                                    {showDropdown[shift.id] && (
                                      <div className="absolute z-50 w-full mt-1 bg-white border border-gray-300 rounded-lg shadow-lg max-h-64 overflow-y-auto">
                                        {getFilteredVolunteers(opportunity.id, searchTerms[shift.id]).map(volunteer => (
                                          <div
                                            key={volunteer.id}
                                            onClick={() => selectVolunteer(shift.id, volunteer)}
                                            className="px-3 py-2 hover:bg-blue-50 cursor-pointer"
                                          >
                                            {volunteer.firstName} {volunteer.lastName}
                                          </div>
                                        ))}
                                        {getFilteredVolunteers(opportunity.id, searchTerms[shift.id]).length === 0 && searchTerms[shift.id] && (
                                          <div className="px-3 py-2 text-red-600 text-sm bg-red-50">
                                            Volunteer not found. Contact your system administrator to add new volunteers.
                                          </div>
                                        )}
                                      </div>
                                    )}
                                  </div>
                                  
                                  <button
                                    onClick={() => handleAssignVolunteer(shift.id, opportunity.id)}
                                    disabled={!assignments[shift.id] || assigning[shift.id]}
                                    className="w-full mt-2 px-4 py-2 bg-blue-600 text-white rounded-lg font-medium hover:bg-blue-700 transition-colors disabled:bg-gray-400 disabled:cursor-not-allowed"
                                  >
                                    {assigning[shift.id] ? 'Assigning...' : 'Assign'}
                                  </button>
                                </div>
                              )}
                            </div>
                          </div>
                        );
                      })}
                    </div>
                  ) : (
                    <p className="text-gray-600">No shifts scheduled for this opportunity.</p>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};

export default EventDetailPage;