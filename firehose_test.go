package firehose_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/benburwell/firehose"
)

var (
	username, password string
	runIntegration     bool
)

func TestMain(m *testing.M) {

	flag.BoolVar(&runIntegration, "run-integration-tests", false, "Run integration tests against a real Firehose")
	flag.Parse()

	if runIntegration {
		username = os.Getenv("FIREHOSE_TEST_USERNAME")
		password = os.Getenv("FIREHOSE_TEST_PASSWORD")
		if username == "" || password == "" {
			log.Println("To run integration tests, set your Firehose credentials in the environment variables FIREHOSE_TEST_USERNAME and FIREHOSE_TEST_PASSWORD.")
			os.Exit(1)
		}
	}

	os.Exit(m.Run())
}

func TestConnect(t *testing.T) {
	if !runIntegration {
		t.Skip("Skipping integration tests")
		return
	}

	stream, err := firehose.Connect()
	if err != nil {
		t.Fatalf("could not establish connection: %v", err)
		return
	}
	defer stream.Close()

	init := fmt.Sprintf("live username \"%s\" password \"%s\" events \"position\"", username, password)
	if err := stream.Init(init); err != nil {
		t.Fatalf("could not initialize connection: %v", err)
		return
	}

	msg, err := stream.NextMessage(context.Background())
	if err != nil {
		t.Errorf("NextMessage returned unexpected error: %v", err)
	}
	if msg.Type != "position" {
		t.Errorf("unexpected message type: %s", msg.Type)
	}
	pos, ok := msg.Payload.(firehose.PositionMessage)
	if !ok {
		t.Fatalf("expected a position message but got: %t", msg.Payload)
		return
	}
	t.Logf("received position message: %#v", pos)
	if pos.PITR == "" {
		t.Errorf("expected a PITR, but got nothing")
	}

	if err := stream.Close(); err != nil {
		t.Errorf("unexpected error closing stream: %v", err)
	}
}

func TestUnmarshalError(t *testing.T) {
	data := []byte(`{"type":"error","error_msg":"I am an error"}`)
	var msg firehose.Message
	if err := json.Unmarshal(data, &msg); err != nil {
		t.Errorf("unmarshal error: %v", err)
	}
	if msg.Type != "error" {
		t.Errorf("expected type error, got: %s", msg.Type)
	}
	em, ok := msg.Payload.(firehose.ErrorMessage)
	if !ok {
		t.Errorf("payload is not an error message: %t", msg.Payload)
	}
	if em.Type != "error" {
		t.Errorf("unexpected error message type: %s", em.Type)
	}
	if em.ErrorMessage != "I am an error" {
		t.Errorf("unexpected error message: %s", em.ErrorMessage)
	}
}

func TestUnmarshalPosition(t *testing.T) {
	data := []byte(`{"pitr":"1596067223","type":"position","ident":"WSN145","air_ground":"A","alt":"1550","alt_gnss":"1575","altChange":" ","clock":"1596067217","facility_hash":"152CF652CDC7C81E","facility_name":"FlightAware ADS-B","id":"WSN145-1596063797-adhoc-0","gs":"124","heading":"31","heading_magnetic":"33.6","heading_true":"30.9","hexid":"A15815","lat":"9.01767","lon":"-79.42058","mach":"0.188","orig":"L 9.13179 -81.43443","pressure":"958","reg":"N186MM","speed_ias":"120","speed_tas":"126","squawk":"1261","updateType":"A","vertRate":"-704","vertRate_geom":"-640","wind_dir":"57","wind_speed":"2","wind_quality":"1"}`)
	var msg firehose.Message
	if err := json.Unmarshal(data, &msg); err != nil {
		t.Errorf("unmarshal error: %v", err)
	}
	if msg.Type != "position" {
		t.Errorf("expected type error, got: %s", msg.Type)
	}
	pm, ok := msg.Payload.(firehose.PositionMessage)
	if !ok {
		t.Errorf("payload is not a position message: %t", msg.Payload)
	}
	if pm.Type != "position" {
		t.Errorf("unexpected position message type: %s", pm.Type)
	}
	if pm.Ident != "WSN145" {
		t.Errorf("unexpected ident: %s", pm.Ident)
	}
}

func TestInitCommand(t *testing.T) {
	c := firehose.InitCommand{
		Live: true,
		PITR: "1",
		Range: &firehose.PITRRange{
			Start: "2",
			End:   "3",
		},
		Password:      "pw",
		Username:      "un",
		AirportFilter: []string{"KBOS", "EG??"},
		Events:        []firehose.Event{firehose.PositionEvent},
		LatLong: []firehose.Rectangle{
			{LowLat: 1, LowLon: 2, HiLat: 3, HiLon: 4},
			{LowLat: 5, LowLon: 6, HiLat: 7, HiLon: 8},
		},
	}
	actual := c.String()
	expected := `live pitr 1 range 2 3 username un password pw airport_filter "KBOS EG??" events "position" latlong "1.000000 2.000000 3.000000 4.000000" latlong "5.000000 6.000000 7.000000 8.000000"`
	if actual != expected {
		t.Errorf("unexpected init command: %s", actual)
	}
}
