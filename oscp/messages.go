package oscp

import (
	"errors"
	"fmt"
)

// Message types of OSCP 2.0 clause 4 and datatypes of clause 5: field
// names, cardinalities and enumerations as facts.

// Version is the OSCP version this package implements.
const Version = "2.0"

// Roles (1.6).
const (
	RoleCP = "cp" // Capacity Provider
	RoleFP = "fp" // Flexibility Provider
	RoleCO = "co" // Capacity Optimizer
)

// Endpoint names below a version's base URL (4.2).
const (
	EPRegister                     = "register"
	EPHandshake                    = "handshake"
	EPHandshakeAcknowledge         = "handshake_acknowledge"
	EPHeartbeat                    = "heartbeat"
	EPUpdateGroupCapacityForecast  = "update_group_capacity_forecast"
	EPAdjustGroupCapacityForecast  = "adjust_group_capacity_forecast"
	EPGroupCapacityComplianceError = "group_capacity_compliance_error"
	EPUpdateGroupMeasurements      = "update_group_measurements"
	EPUpdateAssetMeasurements      = "update_asset_measurements"
)

// Endpoint describes one message endpoint: its schema, the role that owns
// (serves) it and the roles allowed to send it (4.2 and the Direction rows
// of 4.3 to 4.5).
type Endpoint struct {
	Name   string
	Schema string
	Owners map[string][]string // owner role → sender roles
}

var endpoints = []Endpoint{
	{EPRegister, SchemaRegister, map[string][]string{RoleFP: {RoleCP, RoleCO}, RoleCP: {RoleFP}, RoleCO: {RoleFP}}},
	{EPHandshake, SchemaHandshake, map[string][]string{RoleFP: {RoleCP}, RoleCO: {RoleFP}}},
	{EPHandshakeAcknowledge, SchemaHandshakeAcknowledge, map[string][]string{RoleCP: {RoleFP}, RoleFP: {RoleCO}}},
	{EPHeartbeat, SchemaHeartbeat, map[string][]string{RoleFP: {RoleCP, RoleCO}, RoleCP: {RoleFP}, RoleCO: {RoleFP}}},
	// OSCP-N5: 4.4.1 lists CP → FP only, but use cases 2 and 4 (3.2.2,
	// 3.3.1) have the CO send its Optimum to the FP with this message.
	{EPUpdateGroupCapacityForecast, SchemaUpdateGroupCapacityForecast, map[string][]string{RoleFP: {RoleCP, RoleCO}, RoleCO: {RoleFP}}},
	{EPAdjustGroupCapacityForecast, SchemaAdjustGroupCapacityForecast, map[string][]string{RoleCP: {RoleFP}}},
	{EPGroupCapacityComplianceError, SchemaGroupCapacityComplianceError, map[string][]string{RoleCP: {RoleFP}}},
	{EPUpdateGroupMeasurements, SchemaUpdateGroupMeasurements, map[string][]string{RoleCP: {RoleFP}}},
	{EPUpdateAssetMeasurements, SchemaUpdateAssetMeasurements, map[string][]string{RoleCO: {RoleFP}}},
}

// EPHandshakeAcknowledgment is a spelling of EPHandshakeAcknowledge that
// some implementations use; CanonicalEndpoint maps it.
const EPHandshakeAcknowledgment = "handshake_acknowledgment"

// CanonicalEndpoint is name with a known variant spelling replaced by
// the endpoint name of 4.2 (handshake_acknowledgment becomes
// handshake_acknowledge); other names are returned as they are.
func CanonicalEndpoint(name string) string {
	if name == EPHandshakeAcknowledgment {
		return EPHandshakeAcknowledge
	}
	return name
}

// Endpoints lists every message endpoint of 4.2 with its owners and
// senders.
func Endpoints() []Endpoint { return append([]Endpoint(nil), endpoints...) }

// EndpointByName is the endpoint called name (EPRegister, ...); the
// variant spellings CanonicalEndpoint knows are accepted.
func EndpointByName(name string) (Endpoint, bool) {
	name = CanonicalEndpoint(name)
	for _, e := range endpoints {
		if e.Name == name {
			return e, true
		}
	}
	return Endpoint{}, false
}

// CanSend reports whether a party of role from may send the message at
// endpoint ep to a party of role to.
func CanSend(from, to, ep string) bool {
	e, ok := EndpointByName(ep)
	if !ok {
		return false
	}
	for _, s := range e.Owners[to] {
		if s == from {
			return true
		}
	}
	return false
}

// VersionURL (5.1).
type VersionURL struct {
	Version string `json:"version"`
	BaseURL string `json:"base_url"`
}

// Register (4.3.1.1).
type Register struct {
	Token      string       `json:"token"`
	VersionURL []VersionURL `json:"version_url"`
}

// RequiredBehaviour (5.2).
type RequiredBehaviour struct {
	HeartbeatInterval *int `json:"heartbeat_interval,omitempty"`
	// Always sent (an empty list says "no configurations"): the published
	// schema requires it (OSCP-N3).
	MeasurementConfiguration []string `json:"measurement_configuration"`
}

// MeasurementConfiguration values (5.2.1).
const (
	MeasurementContinuous   = "CONTINUOUS"
	MeasurementIntermittent = "INTERMITTENT"
)

// Handshake (4.3.2) and HandshakeAcknowledge (4.3.3) share one shape.
type Handshake struct {
	RequiredBehaviour *RequiredBehaviour `json:"required_behaviour,omitempty"`
}

// Heartbeat (4.3.4).
type Heartbeat struct {
	OfflineModeAt string `json:"offline_mode_at"`
}

// CapacityForecastType values (5.3).
const (
	TypeConsumption         = "CONSUMPTION"
	TypeGeneration          = "GENERATION"
	TypeFallbackConsumption = "FALLBACK_CONSUMPTION"
	TypeFallbackGeneration  = "FALLBACK_GENERATION"
	TypeOptimum             = "OPTIMUM"
)

// ForecastTypes lists 5.3 in order.
var ForecastTypes = []string{TypeConsumption, TypeGeneration, TypeFallbackConsumption, TypeFallbackGeneration, TypeOptimum}

// ForecastedBlock (5.4).
type ForecastedBlock struct {
	Capacity  float64 `json:"capacity"`
	Phase     string  `json:"phase"`
	Unit      string  `json:"unit"`
	StartTime string  `json:"start_time"`
	EndTime   string  `json:"end_time"`
}

// Units of ForecastedBlock (5.4.1) and InstantaneousMeasurement (5.6.4.1).
const (
	UnitA   = "A"
	UnitW   = "W"
	UnitKW  = "KW"
	UnitWH  = "WH"
	UnitKWH = "KWH"
)

// PhaseIndicator values (5.5).
var Phases = []string{"UNKNOWN", "ONE", "TWO", "THREE", "ALL"}

// GroupCapacityForecast is UpdateGroupCapacityForecast (4.4.1) and
// AdjustGroupCapacityForecast (4.4.2), which share their fields.
type GroupCapacityForecast struct {
	GroupID          string            `json:"group_id"`
	Type             string            `json:"type"`
	ForecastedBlocks []ForecastedBlock `json:"forecasted_blocks"`
}

// GroupCapacityComplianceError (4.4.3).
type GroupCapacityComplianceError struct {
	Message          string            `json:"message"`
	ForecastedBlocks []ForecastedBlock `json:"forecasted_blocks,omitempty"`
}

// EnergyMeasurement (5.6.3).
type EnergyMeasurement struct {
	Value              float64 `json:"value"`
	Phase              string  `json:"phase"`
	Unit               string  `json:"unit"`
	Direction          string  `json:"direction"`
	EnergyType         string  `json:"energy_type,omitempty"`
	MeasureTime        string  `json:"measure_time"`
	InitialMeasureTime string  `json:"initial_measure_time,omitempty"`
}

func (e EnergyMeasurement) check() error {
	if err := checkTime("measure_time", e.MeasureTime); err != nil {
		return err
	}
	if e.InitialMeasureTime != "" {
		return checkTime("initial_measure_time", e.InitialMeasureTime)
	}
	return nil
}

// InstantaneousMeasurement (5.6.4).
type InstantaneousMeasurement struct {
	Value       float64 `json:"value"`
	Phase       string  `json:"phase"`
	Unit        string  `json:"unit"`
	MeasureTime string  `json:"measure_time"`
}

// AssetMeasurement (5.6).
type AssetMeasurement struct {
	AssetID                  string                    `json:"asset_id"`
	AssetCategory            string                    `json:"asset_category"`
	EnergyMeasurement        *EnergyMeasurement        `json:"energy_measurement,omitempty"`
	InstantaneousMeasurement *InstantaneousMeasurement `json:"instantaneous_measurement,omitempty"`
}

// UpdateGroupMeasurements (4.5.1).
type UpdateGroupMeasurements struct {
	GroupID      string              `json:"group_id"`
	Measurements []EnergyMeasurement `json:"measurements"`
}

// UpdateAssetMeasurements (4.5.2).
type UpdateAssetMeasurements struct {
	GroupID      string             `json:"group_id"`
	Measurements []AssetMeasurement `json:"measurements"`
}

// ErrorBody is the optional error body (4.1.3.2).
type ErrorBody struct {
	Message string `json:"message,omitempty"`
}

// CheckForecastSet applies the rules between the forecast types of one
// group and one block time (FR.01.05 to FR.01.09, FR.02.07): capacities
// are signed (generation negative). A missing type is not checked.
func CheckForecastSet(cc, gc, fcc, fgc, o *float64) error {
	var errs []error
	if cc != nil && *cc < 0 {
		errs = append(errs, errors.New("the consumption capacity is below 0 (FR.01.05)"))
	}
	if gc != nil && *gc > 0 {
		errs = append(errs, errors.New("the generation capacity is above 0 (FR.01.06)"))
	}
	if fcc != nil && cc != nil && *fcc > *cc {
		errs = append(errs, errors.New("the fallback consumption capacity exceeds the consumption capacity (FR.01.07)"))
	}
	// OSCP-N4: FR.01.08 and FR.02.07 disagree; the table 7 example
	// (GC -10, FGC -5) follows FR.02.07.
	if fgc != nil && gc != nil && *fgc < *gc {
		errs = append(errs, errors.New("the fallback generation capacity is below the generation capacity (FR.02.07)"))
	}
	if o != nil && cc != nil && *o > *cc {
		errs = append(errs, fmt.Errorf("the optimum %v exceeds the consumption capacity %v (FR.01.09)", *o, *cc))
	}
	if o != nil && gc != nil && *o < *gc {
		errs = append(errs, fmt.Errorf("the optimum %v is below the generation capacity %v (FR.01.09)", *o, *gc))
	}
	return errors.Join(errs...)
}

// Signed returns a block's capacity with the sign of 2.3: generation
// forecasts negative whichever sign they arrived with (OSCP-N2).
func Signed(typ string, capacity float64) float64 {
	if (typ == TypeGeneration || typ == TypeFallbackGeneration) && capacity > 0 {
		return -capacity
	}
	return capacity
}
