package services

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// ============================================================================
// Calling geo-coders
// ============================================================================

// When we have zip-code only (volunteers):

const zippopotamURL = "https://api.zippopotam.us/us/"

type zippopotamResponse struct {
	Places []zippopotamPlace `json:"places"`
}
type zippopotamPlace struct {
	Latitude  string `json:"latitude"`
	Longitude string `json:"longitude"`
}

func GeocodeZip(zip string) (*float64, *float64, error) {

	resp, err := http.Get(zippopotamURL + zip)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("zip %s not found", zip)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	var result zippopotamResponse
	if err = json.Unmarshal(body, &result); err != nil {
		return nil, nil, err
	}
	if len(result.Places) == 0 {
		return nil, nil, fmt.Errorf("no results for zip %s", zip)
	}
	lat, err := strconv.ParseFloat(result.Places[0].Latitude, 64)
	if err != nil {
		return nil, nil, err
	}
	lng, err := strconv.ParseFloat(result.Places[0].Longitude, 64)
	if err != nil {
		return nil, nil, err
	}

	return &lat, &lng, nil
}

// When we have an address (venues):

const censusGeocodeURL = "https://geocoding.geo.census.gov/geocoder/locations/address"

type censusResponse struct {
	Result censusResult `json:"result"`
}

type censusResult struct {
	AddressMatches []censusMatch `json:"addressMatches"`
}

type censusMatch struct {
	Coordinates censusCoordinates `json:"coordinates"`
}

type censusCoordinates struct {
	X float64 `json:"x"` // longitude
	Y float64 `json:"y"` // latitude
}

func GeocodeAddress(street string, city string, state string, zip string) (*float64, *float64, error) {
	params := url.Values{}
	params.Set("street", street)
	params.Set("city", city)
	params.Set("state", state)
	params.Set("zip", zip)
	params.Set("benchmark", "Public_AR_Current")
	params.Set("format", "json")
	return getLatLng(censusGeocodeURL + "?" + params.Encode())
}

func getLatLng(url string) (*float64, *float64, error) {
	response, err := http.Get(url)
	if err != nil {
		return nil, nil, fmt.Errorf("could not get geo info from url %s: %w", url, err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("could not read geocode response: %w", err)
	}

	var result censusResponse
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, nil, fmt.Errorf("could not parse geocode response: %w", err)
	}
	if len(result.Result.AddressMatches) == 0 {
		return nil, nil, fmt.Errorf("the address (%s) was not found.", url)
	}
	coords := result.Result.AddressMatches[0].Coordinates
	lat := coords.Y
	lng := coords.X

	return &lat, &lng, nil

}

// ============================================================================
// Computing distances between between 2 sets of coordinates.
// ============================================================================

// Compute haversine distance :

func fetchDistance(latA float64, lngA float64, latB float64, lngB float64) float64 {
	// Earth's average radius in miles:
	const R = 3958.8

	// Original coordinates to radians:
	rLatA := latA * math.Pi / 180.0
	rLngA := lngA * math.Pi / 180.0
	rLatB := latB * math.Pi / 180.0
	rLngB := lngB * math.Pi / 180.0

	// Half distances between latitudes and longitudes (in radians):
	dLatHalf := (rLatB - rLatA) / 2
	dLngHalf := (rLngB - rLngA) / 2

	a := math.Pow(math.Sin(dLatHalf), 2) + math.Cos(rLatA)*math.Cos(rLatB)*math.Pow(math.Sin(dLngHalf), 2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

// ** Handling datetimes **
// We store all dates and times in the DB as UTC with RFC-3339 format.

const Layout = "2006-01-02 15:04:05" // expected layout of dates in input from client.

func DateTimeToUTC(dateTimeStr string, ianaZone string) (*string, error) {
	loc, err := time.LoadLocation(ianaZone)
	if err != nil {
		return nil, fmt.Errorf("invalid timezone %s: %w", ianaZone, err)
	}

	datetime, err := time.ParseInLocation(Layout, dateTimeStr, loc)
	if err != nil {
		return nil, fmt.Errorf("error parsing %s: %w", dateTimeStr, err)
	}

	rfc := datetime.UTC().Format(time.RFC3339)
	return &rfc, nil
}

func UTCToTimeZone(utcTime string, ianaZone string) (*string, error) {

	dateTime, err := time.Parse(time.RFC3339, utcTime)
	if err != nil {
		return nil, fmt.Errorf("error parsing %s: %w", utcTime, err)
	}

	loc, err := time.LoadLocation(ianaZone)
	if err != nil {
		return nil, fmt.Errorf("invalid timezone %s: %w", ianaZone, err)
	}
	strTime := dateTime.In(loc).Format("01-02-2006 15:04 MST")

	return &strTime, nil
}

func UTCToDateTime(utcTime string) (*string, error) {

	dateTime, err := time.Parse(time.RFC3339, utcTime)
	if err != nil {
		return nil, fmt.Errorf("error parsing %s: %w", utcTime, err)
	}

	strTime := dateTime.Format("01-02-2006 15:04 MST")
	return &strTime, nil
}

// formatStartEnd is used to format start and end times for emails. If we cannot format
// the times (unlikely), use the DB time (formatted in RFC3339).
func formatStartEnd(start string, end string, timezone string) (*string, *string) {
	var fmtStart, fmtEnd *string
	var err error

	fmtStart, err = UTCToTimeZone(start, timezone)
	if err != nil {
		log.Printf("unable to format shift start time: %v", err)
		// An unformatted string is better than nothing.
		fmtStart = &start
	}

	fmtEnd, err = UTCToTimeZone(end, timezone)
	if err != nil {
		log.Printf("unable to format shift end time: %v", err)
		fmtEnd = &end
	}

	return fmtStart, fmtEnd
}
