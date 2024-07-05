package firehose

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"strings"
)

// DefaultAddress is the default server address to use for Firehose connections.
const DefaultAddress = "firehose.flightaware.com:1501"

type Event string

const (
	PositionEvent Event = "position"
)

type Rectangle struct {
	LowLat float64
	LowLon float64
	HiLat  float64
	HiLon  float64
}

// InitCommand helps build and serialize an initiation command string which can be provided as the argument to
// Stream.Init.
type InitCommand struct {
	// Live requests data from the present time forward.
	Live bool
	// PITR requests data starting from the specified time, in POSIX epoch format, in the past until the current time,
	// and continue with live behavior.
	PITR string
	// Range requests data between the two specified times, in POSIX epoch format. FlightAware will disconnect the
	// connection when the last message has been sent.
	Range *PITRRange
	// Password supplies the credentials for authentication. In most cases, this should actually be the Firehose API Key
	// and not the password of the account.
	Password string
	// Username supplies the credentials for authentication. This should be the username of the FlightAware account that
	// has been granted access.
	Username string
	// AirportFilter requests information only for flights originating from or destined for airports matching the space
	// separated list of glob patterns provided.
	//
	// For example: "CYUL" or "K??? P* TJSJ"
	AirportFilter []string
	// Events specifies a list of downlink messages which should be sent.
	//
	// If not specified default behavior is to deliver all Airborne Feed messages enabled in the Firehose Subscription.
	// Which event codes are available will depend on which Subscription Layers are enabled.
	Events []Event
	// LatLong specifies that only positions within the specified rectangle should be sent and any others will be
	// ignored, unless the flight has already been matched by other criteria. Once a flight has been matched by a
	// latlong rectangle, it becomes remembered and all subsequent messages until landing for that flight ID will
	// continue to be sent even if the flight no longer matches a specified rectangle.
	LatLong []Rectangle
}

func (i *InitCommand) String() string {
	var parts []string

	if i.Live {
		parts = append(parts, "live")
	}

	if i.PITR != "" {
		parts = append(parts, "pitr", i.PITR)
	}

	if i.Range != nil {
		parts = append(parts, "range", i.Range.Start, i.Range.End)
	}

	parts = append(parts, "username", i.Username)
	parts = append(parts, "password", i.Password)

	if len(i.AirportFilter) > 0 {
		filter := fmt.Sprintf("\"%s\"", strings.Join(i.AirportFilter, " "))
		parts = append(parts, "airport_filter", filter)
	}

	if len(i.Events) > 0 {
		var events []string
		for _, e := range i.Events {
			events = append(events, string(e))
		}
		filter := fmt.Sprintf("\"%s\"", strings.Join(events, " "))
		parts = append(parts, "events", filter)
	}

	for _, rect := range i.LatLong {
		filter := fmt.Sprintf("\"%f %f %f %f\"", rect.LowLat, rect.LowLon, rect.HiLat, rect.HiLon)
		parts = append(parts, "latlong", filter)
	}

	return strings.Join(parts, " ")
}

type PITRRange struct {
	Start string
	End   string
}

// Connect is a simple way to open a Firehose stream using the default configuration.
//
// To customize your connection, use NewStream instead.
func Connect() (*Stream, error) {
	conn, err := tls.Dial("tcp", DefaultAddress, nil)
	if err != nil {
		return nil, err
	}
	return NewStream(conn), nil
}

// NewStream creates a new Firehose Stream over the provided network connection.
func NewStream(conn net.Conn) *Stream {
	return &Stream{
		conn:    conn,
		decoder: json.NewDecoder(conn),
	}
}

type Stream struct {
	conn    net.Conn
	decoder *json.Decoder
}

func (c *Stream) Init(command string) error {
	_, err := fmt.Fprintln(c.conn, command)
	return err
}

type Message struct {
	Type    string
	Payload any
}

func (m *Message) UnmarshalJSON(data []byte) error {
	var stub struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &stub); err != nil {
		return fmt.Errorf("could not determine message type: %w", err)
	}
	m.Type = stub.Type

	switch m.Type {
	case "error":
		var payload ErrorMessage
		err := json.Unmarshal(data, &payload)
		m.Payload = payload
		return err
	case "position":
		var payload PositionMessage
		err := json.Unmarshal(data, &payload)
		m.Payload = payload
		return err
	default:
		return fmt.Errorf("unrecognized message type: %s", m.Type)
	}
}

// ErrorMessage indicates an error condition.
type ErrorMessage struct {
	// Type is always "error".
	Type string `json:"type"`
	// ErrorMessage contains details of the error encountered.
	ErrorMessage string `json:"error_msg"`
}

// Waypoint contains position data
type Waypoint struct {
	// Latitude in decimal degrees.
	Lat float64 `json:"lat"`
	// Longitude in decimal degrees.
	Lon float64 `json:"lon"`
	// Clock is the time in POSIX epoch format.
	Clock *string `json:"clock"`
	// Name is the airport, navaid, waypoint, intersection, or other identifier.
	Name *string `json:"name"`
	// Alt is the altitude in feet (MSL).
	Alt *string `json:"alt"`
	// GS is the ground speed in knots.
	GS *string `json:"gs"`
}

// PositionMessage includes a position report.
type PositionMessage struct {
	// Type is always "position".
	Type string `json:"type"`
	// Ident is the callsign identifying the flight. Typically, ICAO airline code plus IATA/ticketing flight number, or the aircraft registration.
	Ident string `json:"ident"`
	// Latitude in decimal degrees, rounded to 5 decimal points.
	Lat string `json:"lat"`
	// Longitude in decimal degrees, rounded to 5 decimal points.
	Lon string `json:"lon"`
	// Clock is the report time in POSIX epoch format. Time should be the time generated by flight hardware if possible.
	Clock string `json:"clock"`
	// ID is the FlightAware Flight ID, a unique identifier associated with each flight.
	ID string `json:"id"`
	// UpdateType specifies the source of the message.
	//
	// - A for ADS-B
	// - Z for radar
	// - O for transoceanic
	// - P for estimated
	// - D for datalink
	// - M for multilateration (MLAT)
	// - X for ASDE-X
	// - S for space-based ADS-B
	UpdateType string `json:"updateType"`
	// AirGround indicates whether the aircraft is on the ground.
	//
	// - A for Air
	// - G for Ground
	// - WOW for Weight-on-Wheels
	AirGround string `json:"air_ground"`
	// FacilityHash is a consistent and unique obfuscated identifier string for each source reporting positions to
	// FlightAware.
	FacilityHash string `json:"facility_hash"`
	// FacilityName is a description of the reporting facility intended for end-user consumption. May be a blank string
	// if undefined.
	FacilityName string `json:"facility_name"`
	// PITR is the point-in-time-recovery timestamp value that should be supplied to the "pitr" connection initiation
	// command when reconnecting and you wish to resume firehose playback at that approximate position.
	PITR string `json:"pitr"`
	// Alt is altitude in feet (MSL).
	Alt *string `json:"alt"`
	// AltChange describes change in altitude.
	//
	// - C for climbing
	// - D for descending
	// - " " when undetermined
	AltChange *string `json:"alt_change"`
	// GS is ground speed in knots.
	GS *string `json:"gs"`
	// Heading indicates the course in degrees.
	Heading *string `json:"heading"`
	// Squawk is the transponder squawk code, a 4 digit octal transponder beacon code assigned by ATC.
	Squawk *string `json:"squawk"`
	// Hexid is the transponder Mode S code, a 24-bit transponder code assigned by aircraft registrar. Formatted in
	// upper case hexadecimal.
	Hexid *string `json:"hexid"`
	// ATCIdent is an identifier used for ATC, if that differs from the flight identifier.
	ATCIdent *string `json:"atcident"`
	// AircraftType is the ICAO aircraft type code.
	AircraftType *string `json:"aircrafttype"`
	// Orig is the origin ICAO airport code, waypoint, or latitude/longitude pair.
	Orig *string `json:"orig"`
	// Dest is the destination ICAO airport code, waypoint, or latitude/longitude pair. May be missing if not known.
	Dest *string `json:"dest"`
	// Reg is the tail number or registration of the aircraft, if known and it differs from the ident.
	Reg *string `json:"reg"`
	// ETA is the estimated time of arrival in POSIX epoch format.
	ETA *string `json:"eta"`
	// EDT is the revised timestamp of when the flight is expected to depart in POSIX epoch format.
	EDT *string `json:"edt"`
	// ETE is the en route time in seconds. May be missing if not known.
	ETE *string `json:"ete"`
	// Speed is the filed cruising speed in knots.
	Speed *string `json:"speed"`
	// Waypoints is an array of 2D, 3D, or 4D objects of locations, times, and altitudes.
	Waypoints []Waypoint `json:"waypoints"`
	// Route is a textual route string.
	Route *string `json:"route"`
	// ADSBVersion is the ADS-B version used by the transmitter responsible for position, when known/applicable.
	ADSBVersion *string `json:"adsb_version"`
	// NACp is the ADS-B Navigational Accuracy Category for Position.
	NACp *int `json:"nac_p"`
	// NACv is the ADS-B Navigational Accuracy Category for Velocity.
	NACv *int `json:"nac_v"`
	// NIC is the ADS-B Navigational Integrity Category.
	NIC *int `json:"nic"`
	// NICBaro is the ADS-B Navigational Integrity Category for Barometer.
	NICBaro *int `json:"nic_baro"`
	// SIL is the ADS-B Source Integrity Level
	SIL *int `json:"sil"`
	// SILType is the ADS-B Source Integrity Level type (applies per-hour or per-sample).
	//
	// Possible values are "perhour", "persample", and "unknown".
	SILType *string `json:"sil_type"`
	// PosRC is the ADS-B Radius of Containment, in meters.
	PosRC *float64 `json:"pos_rc"`
	// HeadingMagnetic is the aircraft's heading, in degrees, relative to magnetic North.
	HeadingMagnetic *string `json:"heading_magnetic"`
	// HeadingTrue is the aircraft's heading, in degrees, relative to true North.
	HeadingTrue *string `json:"heading_true"`
	// Mach is the mach number of the aircraft.
	Mach *string `json:"mach"`
	// SpeedTAS is the true airspeed of the aircraft in knots.
	SpeedTAS *string `json:"speed_tas"`
	// SpeedIAS is the indicated airspeed of the aircraft in knots.
	SpeedIAS *string `json:"speed_ias"`
	// Pressure is the computed static air pressure in hPa.
	Pressure *string `json:"pressure"`
	// WindQuality is set to 1 if the aircraft is stable (not maneuvering) and 0 if the aircraft is maneuvering.
	//
	// Derived wind data is less reliable if the aircraft is maneuvering.
	WindQuality *string `json:"wind_quality"`
	// WindDir is the computed wind direction, in degrees, relative to true North.
	//
	// The value uses the normal convention where the direction is opposite the wind vector (i.e. wind_dir = 0 means
	// wind from the North).
	WindDir *string `json:"wind_dir"`
	// WindSpeed is the computed wind speed in knots.
	WindSpeed *string `json:"wind_speed"`
	// TemperatureQuality is set to 0 if the derived temperature is likely to be inaccurate due to quantization errors,
	// 0 otherwise.
	TemperatureQuality *string `json:"temperature_quality"`
	// Temperature is the computed outside air temperature in degrees Celsius.
	Temperature *string `json:"temperature"`
	// NavHeading is the heading in degrees from the navigation equipment.
	NavHeading *string `json:"nav_heading"`
	// NavAltitude is the altitude setting in feet from tne navigation equipment.
	NavAltitude *string `json:"nav_altitude"`
	// NavQNH is the altimeter setting in hPa that has been set.
	NavQNH *string `json:"nav_qnh"`
	// NavModes is the list of active modes from the navigation equipment.
	//
	// Possible values are autopilot, vnav, althold, approach, lnav, tcas.
	NavModes *string `json:"nav_modes"`
	// AltGNSS is the reported GNSS altitude (feet above WGS84 ellipsoid).
	AltGNSS *string `json:"alt_gnss"`
	// VertRate is the aircraft's vertical rate of climb/descent derived from pressure altitude, reported in feet per
	// minute.
	VertRate *string `json:"vertRate"`
	// VertRateGeom is the aircraft's vertical rate of climb/descent derived from GNSS altitude, reported in feet per
	// minute.
	VertRateGeom *string `json:"vertRate_geom"`
	// FuelOnBoard is the amount of fuel in the tank.
	//
	// The units are reported in the FuelOnBoardUnit field. This data is available for specifically authorized customers
	// only.
	FuelOnBoard *string `json:"fuel_on_board"`
	// FuelOnBoardUnit is the unit for FuelOnBoard.
	//
	// Possible values are LITERS, GALLONS, POUNDS, KILOGRAMS, or UNKNOWN. This data is available for specifically
	// authorized customers only.
	FuelOnBoardUnit *string `json:"fuel_on_board_unit"`
}

func (c *Stream) NextMessage(ctx context.Context) (*Message, error) {
	// If our context has a deadline, set the read deadline on our underlying connection accordingly.
	if deadline, hasDeadline := ctx.Deadline(); hasDeadline {
		if err := c.conn.SetReadDeadline(deadline); err != nil {
			return nil, fmt.Errorf("could not set read deadline: %w", err)
		}
	}

	var msg Message
	errc := make(chan error)
	go func() {
		errc <- c.decoder.Decode(&msg)
	}()

	select {
	case <-ctx.Done():
		c.Close()
		return nil, ctx.Err()
	case err := <-errc:
		return &msg, err
	}
}

func (c *Stream) Close() error {
	return c.conn.Close()
}
