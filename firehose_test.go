package firehose_test

import (
	"encoding/json"
	"testing"

	"github.com/benburwell/firehose"
)

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
