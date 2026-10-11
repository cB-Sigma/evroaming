package oscp

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"strings"
	"time"

	"github.com/cB-Sigma/jsonschema"
)

// The OSCP 2.0 JSON schemas of the Open Charge Alliance (CC BY-ND 4.0),
// embedded unmodified; spec/data/oscp/2.0 holds the same files with their
// digests and attribution, and TestSchemasMatchSpecData checks both copies.
//
//go:embed schemas/*.json
var schemaFS embed.FS

// Schema file of each message (OSCP 2.0 clause 4; the error body of 4.1.3.2).
const (
	SchemaRegister                     = "register.json"
	SchemaHandshake                    = "handshake.json"
	SchemaHandshakeAcknowledge         = "handshake-acknowledge.json"
	SchemaHeartbeat                    = "heartbeat.json"
	SchemaUpdateGroupCapacityForecast  = "update-group-capacity-forecast.json"
	SchemaAdjustGroupCapacityForecast  = "adjust-group-capacity-forecast.json"
	SchemaGroupCapacityComplianceError = "group-capacity-compliance-error.json"
	SchemaUpdateGroupMeasurements      = "update-group-measurements.json"
	SchemaUpdateAssetMeasurements      = "update-asset-measurements.json"
	SchemaError                        = "error.json"
)

// Corrections applied in memory to schemas that refuse messages the
// specification text requires to be valid (the notes in PROVENANCE.md). The files
// stay unmodified.
var overlays = map[string]func(m map[string]any){
	// OSCP-N1: measurements carries "$ref" next to "type": "array"; under
	// draft-07 the siblings are ignored, so every array fails. The text
	// (5.6, 4.5.2) makes it a list of AssetMeasurement, 1..*.
	SchemaUpdateAssetMeasurements: func(m map[string]any) {
		props := m["properties"].(map[string]any)
		props["measurements"] = map[string]any{"type": "array", "items": map[string]any{"$ref": "asset-measurement.json"}, "minItems": json.Number("1")}
	},
	// OSCP-N2: capacity has minimum 0, but Generation and Fallback
	// Generation capacities are negative (2.3, FR.01.06); the sign is
	// checked per forecast type in code (checkBlocks).
	"forecasted-block.json": func(m map[string]any) {
		props := m["properties"].(map[string]any)
		delete(props["capacity"].(map[string]any), "minimum")
	},
	// OSCP-N3: measurement_configuration is required, while 5.2 makes it
	// 0..* and asks for at least one of the two fields; heartbeat_interval
	// is an integer of 1 or more (5.2), the schema says number.
	"required-behaviour.json": func(m map[string]any) {
		delete(m, "required")
		m["anyOf"] = []any{
			map[string]any{"required": []any{"heartbeat_interval"}},
			map[string]any{"required": []any{"measurement_configuration"}},
		}
		props := m["properties"].(map[string]any)
		props["heartbeat_interval"] = map[string]any{"type": "integer", "minimum": json.Number("1")}
	},
}

var (
	schemas    *jsonschema.Set // with the overlays: what messages are checked against
	rawSchemas *jsonschema.Set // exactly the published files (tests, findings)
)

func init() {
	var err error
	if schemas, err = loadSchemas(true); err != nil {
		panic("oscp: " + err.Error())
	}
	if rawSchemas, err = loadSchemas(false); err != nil {
		panic("oscp: " + err.Error())
	}
}

func loadSchemas(corrected bool) (*jsonschema.Set, error) {
	set := jsonschema.NewSetWith(jsonschema.Options{AssertFormat: true}) // date-time and uri are checked in any draft
	entries, err := fs.ReadDir(schemaFS, "schemas")
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		b, err := schemaFS.ReadFile("schemas/" + e.Name())
		if err != nil {
			return nil, err
		}
		if fix := overlays[e.Name()]; fix != nil && corrected {
			v, err := jsonschema.Decode(b)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", e.Name(), err)
			}
			m, ok := v.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("%s is not an object", e.Name())
			}
			fix(m)
			if b, err = json.Marshal(m); err != nil {
				return nil, err
			}
		}
		if err := set.Add(e.Name(), b); err != nil {
			return nil, err
		}
	}
	for _, name := range set.Names() {
		if _, err := set.Compile(name); err != nil {
			return nil, err
		}
	}
	return set, nil
}

// SchemaFiles returns the embedded schema files by name, unmodified.
func SchemaFiles() (map[string][]byte, error) {
	out := map[string][]byte{}
	entries, err := fs.ReadDir(schemaFS, "schemas")
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		b, err := schemaFS.ReadFile("schemas/" + e.Name())
		if err != nil {
			return nil, err
		}
		out[e.Name()] = b
	}
	return out, nil
}

// ValidateSchema checks a message body against its schema (with the
// corrections OSCP-N1 to OSCP-N3).
func ValidateSchema(schema string, body []byte) error {
	return schemas.ValidateJSON(schema, body)
}

// ValidateRawSchema checks a body against the published schema file as
// it is, without corrections.
func ValidateRawSchema(schema string, body []byte) error {
	return rawSchemas.ValidateJSON(schema, body)
}

// Validate checks a message: its schema, then the rules of the text the
// schemas do not carry (datetime precision 1.3.3, VersionURL 5.1,
// AssetMeasurement 5.6, capacity signs 2.3 and FR.01.05/FR.01.06).
func Validate(schema string, body []byte) error {
	if err := ValidateSchema(schema, body); err != nil {
		return err
	}
	return validateRules(schema, body)
}

// validateRules checks the rules of the text the schemas do not carry.
func validateRules(schema string, body []byte) error {
	switch schema {
	case SchemaRegister:
		var m Register
		if err := json.Unmarshal(body, &m); err != nil {
			return err
		}
		for _, v := range m.VersionURL {
			if err := CheckBaseURL(v.BaseURL); err != nil {
				return err
			}
		}
	case SchemaHeartbeat:
		var m Heartbeat
		if err := json.Unmarshal(body, &m); err != nil {
			return err
		}
		return checkTime("offline_mode_at", m.OfflineModeAt)
	case SchemaUpdateGroupCapacityForecast, SchemaAdjustGroupCapacityForecast:
		var m GroupCapacityForecast
		if err := json.Unmarshal(body, &m); err != nil {
			return err
		}
		return checkBlocks(m.Type, m.ForecastedBlocks)
	case SchemaGroupCapacityComplianceError:
		var m GroupCapacityComplianceError
		if err := json.Unmarshal(body, &m); err != nil {
			return err
		}
		return checkBlocks("", m.ForecastedBlocks)
	case SchemaUpdateGroupMeasurements:
		var m UpdateGroupMeasurements
		if err := json.Unmarshal(body, &m); err != nil {
			return err
		}
		for i, e := range m.Measurements {
			if err := e.check(); err != nil {
				return fmt.Errorf("measurements/%d: %w", i, err)
			}
		}
	case SchemaUpdateAssetMeasurements:
		var m UpdateAssetMeasurements
		if err := json.Unmarshal(body, &m); err != nil {
			return err
		}
		for i, a := range m.Measurements {
			if (a.EnergyMeasurement == nil) == (a.InstantaneousMeasurement == nil) {
				return fmt.Errorf("measurements/%d: give energy_measurement or instantaneous_measurement, not both or neither (OSCP 2.0 5.6)", i)
			}
			if a.EnergyMeasurement != nil {
				if err := a.EnergyMeasurement.check(); err != nil {
					return fmt.Errorf("measurements/%d/energy_measurement: %w", i, err)
				}
			}
			if a.InstantaneousMeasurement != nil {
				if err := checkTime("measure_time", a.InstantaneousMeasurement.MeasureTime); err != nil {
					return fmt.Errorf("measurements/%d/instantaneous_measurement: %w", i, err)
				}
			}
		}
	}
	return nil
}

var fractionRE = regexp.MustCompile(`\.(\d+)`)

// checkTime applies 1.3.3 and 1.9.1: RFC 3339 with a time zone (the
// schema's date-time format) and at most three decimal places.
func checkTime(field, s string) error {
	t := strings.ToUpper(s)
	if _, err := time.Parse(time.RFC3339Nano, t); err != nil {
		return fmt.Errorf("%s %q is not an RFC 3339 date-time with a time zone", field, s)
	}
	if m := fractionRE.FindStringSubmatch(t); m != nil && len(m[1]) > 3 {
		return fmt.Errorf("%s %q has more than 3 decimal places of seconds (OSCP 2.0 1.3.3)", field, s)
	}
	return nil
}

// ParseTime reads an RFC 3339 date-time (zero when it is not one).
func ParseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339Nano, strings.ToUpper(s))
	return t
}

// CheckBaseURL applies 5.1: a protocol and delimiter first, no trailing
// slash, at most 255 characters (1.3.3 URL).
func CheckBaseURL(u string) error {
	if !strings.HasPrefix(u, "https://") && !strings.HasPrefix(u, "http://") {
		return fmt.Errorf("base_url %q must start with https:// (OSCP 2.0 5.1)", u)
	}
	if strings.HasSuffix(u, "/") {
		return fmt.Errorf("base_url %q must not end with a slash (OSCP 2.0 5.1)", u)
	}
	if len(u) > 255 {
		return errors.New("base_url is longer than 255 characters (OSCP 2.0 1.3.3)")
	}
	return nil
}

// checkBlocks checks forecast blocks: times, end after start, and the sign
// the forecast type demands (2.3; FR.01.05, FR.01.06). typ "" skips the sign.
func checkBlocks(typ string, blocks []ForecastedBlock) error {
	for i, b := range blocks {
		if err := checkTime("start_time", b.StartTime); err != nil {
			return fmt.Errorf("forecasted_blocks/%d: %w", i, err)
		}
		if err := checkTime("end_time", b.EndTime); err != nil {
			return fmt.Errorf("forecasted_blocks/%d: %w", i, err)
		}
		if !ParseTime(b.EndTime).After(ParseTime(b.StartTime)) {
			return fmt.Errorf("forecasted_blocks/%d: end_time is not after start_time", i)
		}
		switch typ {
		case TypeConsumption, TypeFallbackConsumption:
			if b.Capacity < 0 {
				return fmt.Errorf("forecasted_blocks/%d: a %s capacity is 0 or more (OSCP 2.0 2.3, FR.01.05)", i, typ)
			}
		}
	}
	return nil
}
