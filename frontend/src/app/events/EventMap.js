"use client";

/**
 * EventMap — renders a Leaflet/OpenStreetMap map with one pin per event
 * that has venue coordinates.  Imported with { ssr: false } from the events
 * page so Leaflet's window references never run on the server.
 *
 * Props:
 *   events       {Array}    — full eventViews list (filtered by the parent)
 *   onEventClick {function} — called with the event id when a pin is clicked
 */

import { useEffect } from "react";
import { MapContainer, TileLayer, Marker, Popup, useMap } from "react-leaflet";
import "leaflet/dist/leaflet.css";
import L from "leaflet";
import styles from "./events.module.css";

// Fix Leaflet's default icon broken by webpack asset hashing.
// Must run once on the client before any map renders.
delete L.Icon.Default.prototype._getIconUrl;
L.Icon.Default.mergeOptions({
  iconRetinaUrl: "/leaflet/marker-icon-2x.png",
  iconUrl:       "/leaflet/marker-icon.png",
  shadowUrl:     "/leaflet/marker-shadow.png",
});

/** Fits the map to the visible pins whenever the events list changes. */
function BoundsUpdater({ points }) {
  const map = useMap();
  useEffect(() => {
    if (points.length === 0) return;
    if (points.length === 1) {
      map.setView(points[0], 11);
      return;
    }
    const bounds = L.latLngBounds(points);
    map.fitBounds(bounds, { padding: [48, 48] });
  }, [map, points]);
  return null;
}

function formatDate(isoString) {
  if (!isoString) return "";
  return new Date(isoString).toLocaleDateString(undefined, {
    month: "short", day: "numeric", year: "numeric",
  });
}

export default function EventMap({ events, onEventClick }) {
  // Only events with a venue that has coordinates
  const mappable = events.filter(
    (e) => e.venue?.latitude != null && e.venue?.longitude != null
  );
  const virtualCount = events.filter((e) => e.eventType === "VIRTUAL").length;
  const noCoordCount = events.filter(
    (e) => e.eventType !== "VIRTUAL" && (!e.venue?.latitude || !e.venue?.longitude)
  ).length;

  const points = mappable.map((e) => [e.venue.latitude, e.venue.longitude]);

  // Default center: Washington state
  const defaultCenter = [47.5, -120.5];
  const defaultZoom   = 7;

  return (
    <div className={styles.mapWrapper}>
      <MapContainer
        center={defaultCenter}
        zoom={defaultZoom}
        className={styles.mapContainer}
        scrollWheelZoom={true}
      >
        <TileLayer
          attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
          url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
        />

        <BoundsUpdater points={points} />

        {mappable.map((event) => (
          <Marker
            key={event.id}
            position={[event.venue.latitude, event.venue.longitude]}
          >
            <Popup>
              <div className={styles.mapPopup}>
                <div className={styles.mapPopupName}>{event.name}</div>
                {event.eventDates?.[0] && (
                  <div className={styles.mapPopupDate}>
                    {formatDate(event.eventDates[0].startDateTime)}
                  </div>
                )}
                {event.venue.city && (
                  <div className={styles.mapPopupCity}>
                    {event.venue.city}, {event.venue.state}
                  </div>
                )}
                <button
                  className={styles.mapPopupLink}
                  onClick={() => onEventClick(event.id)}
                >
                  View event →
                </button>
              </div>
            </Popup>
          </Marker>
        ))}
      </MapContainer>

      {(virtualCount > 0 || noCoordCount > 0) && (
        <p className={styles.mapFootnote}>
          {[
            virtualCount > 0 && `${virtualCount} virtual event${virtualCount !== 1 ? "s" : ""} not shown`,
            noCoordCount > 0 && `${noCoordCount} in-person event${noCoordCount !== 1 ? "s" : ""} missing location data`,
          ].filter(Boolean).join(" · ")}
        </p>
      )}
    </div>
  );
}
