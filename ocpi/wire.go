package ocpi

import (
	"encoding/json"
	"time"
)

// OCPI 2.2.1 objects as they travel (snake_case members). Unknown members
// are ignored on input (never refuse); objects we build are complete.

// Version is the OCPI version this package speaks.
const Version = "2.2.1"

// Module identifiers (6.2, ModuleID).
const (
	ModCredentials   = "credentials"
	ModLocations     = "locations"
	ModSessions      = "sessions"
	ModCDRs          = "cdrs"
	ModTariffs       = "tariffs"
	ModTokens        = "tokens"
	ModCommands      = "commands"
	ModHubClientInfo = "hubclientinfo"
)

// Interface roles (6.2, InterfaceRole).
const (
	Sender   = "SENDER"
	Receiver = "RECEIVER"
)

// Party roles (16, Role).
const (
	RoleCPO  = "CPO"
	RoleEMSP = "EMSP"
	RoleHub  = "HUB"
)

// Status codes (5).
const (
	StatusOK               = 1000
	StatusClientError      = 2000
	StatusInvalidParams    = 2001
	StatusNotEnoughInfo    = 2002
	StatusUnknownLocation  = 2003
	StatusUnknownToken     = 2004
	StatusServerError      = 3000
	StatusClientAPIError   = 3001
	StatusUnsupportedVer   = 3002
	StatusNoMatchingEndpts = 3003
	StatusHubUnknownTo     = 4001
)

// Envelope is the response format (4.1.7).
type Envelope struct {
	Data          json.RawMessage `json:"data,omitempty"`
	StatusCode    int             `json:"status_code"`
	StatusMessage string          `json:"status_message,omitempty"`
	Timestamp     string          `json:"timestamp"`
}

// VersionInfo is a Version class (6.1).
type VersionInfo struct {
	Version string `json:"version"`
	URL     string `json:"url"`
}

// Endpoint is an Endpoint class (6.2).
type Endpoint struct {
	Identifier string `json:"identifier"`
	Role       string `json:"role"`
	URL        string `json:"url"`
}

// VersionDetails is the version details data (6.2).
type VersionDetails struct {
	Version   string     `json:"version"`
	Endpoints []Endpoint `json:"endpoints"`
}

// BusinessDetails (8.4.3).
type BusinessDetails struct {
	Name    string `json:"name"`
	Website string `json:"website,omitempty"`
	// Logo was added in v0.1.0 of the module.
	Logo *Image `json:"logo,omitempty"`
}

// CredentialsRole (7.4.1).
type CredentialsRole struct {
	Role            string          `json:"role"`
	BusinessDetails BusinessDetails `json:"business_details"`
	PartyID         string          `json:"party_id"`
	CountryCode     string          `json:"country_code"`
}

// Credentials (7.3.1).
type Credentials struct {
	Token string            `json:"token"`
	URL   string            `json:"url"`
	Roles []CredentialsRole `json:"roles"`
}

// Party is a country code and party ID.
type Party struct {
	CountryCode string `json:"country_code"`
	PartyID     string `json:"party_id"`
}

// Key is "CC/PID" upper case (CiString).
func (p Party) Key() string { return Upper(p.CountryCode) + "/" + Upper(p.PartyID) }

// Token (12.3.2).
type Token struct {
	CountryCode        string          `json:"country_code"`
	PartyID            string          `json:"party_id"`
	UID                string          `json:"uid"`
	Type               string          `json:"type"`
	ContractID         string          `json:"contract_id"`
	VisualNumber       string          `json:"visual_number,omitempty"`
	Issuer             string          `json:"issuer"`
	GroupID            string          `json:"group_id,omitempty"`
	Valid              bool            `json:"valid"`
	Whitelist          string          `json:"whitelist"`
	Language           string          `json:"language,omitempty"`
	DefaultProfileType string          `json:"default_profile_type,omitempty"`
	EnergyContract     json.RawMessage `json:"energy_contract,omitempty"`
	LastUpdated        string          `json:"last_updated"`
}

// Whitelist types (12.4.5).
const (
	WhitelistAlways         = "ALWAYS"
	WhitelistAllowed        = "ALLOWED"
	WhitelistAllowedOffline = "ALLOWED_OFFLINE"
	WhitelistNever          = "NEVER"
)

// LocationReferences (12.4.4).
type LocationReferences struct {
	LocationID string   `json:"location_id"`
	EVSEUIDs   []string `json:"evse_uids,omitempty"`
}

// AuthorizationInfo (12.3.1).
type AuthorizationInfo struct {
	Allowed                string              `json:"allowed"`
	Token                  Token               `json:"token"`
	Location               *LocationReferences `json:"location,omitempty"`
	AuthorizationReference string              `json:"authorization_reference,omitempty"`
	Info                   *DisplayText        `json:"info,omitempty"`
}

// DisplayText (16).
type DisplayText struct {
	Language string `json:"language"`
	Text     string `json:"text"`
}

// GeoLocation (8.4.13).
type GeoLocation struct {
	Latitude  string `json:"latitude"`
	Longitude string `json:"longitude"`
}

// Location (8.3.1).
type Location struct {
	CountryCode        string           `json:"country_code"`
	PartyID            string           `json:"party_id"`
	ID                 string           `json:"id"`
	Publish            bool             `json:"publish"`
	Name               string           `json:"name,omitempty"`
	Address            string           `json:"address"`
	City               string           `json:"city"`
	PostalCode         string           `json:"postal_code,omitempty"`
	State              string           `json:"state,omitempty"`
	Country            string           `json:"country"`
	Coordinates        GeoLocation      `json:"coordinates"`
	ParkingType        string           `json:"parking_type,omitempty"`
	EVSEs              []EVSE           `json:"evses,omitempty"`
	Directions         []DisplayText    `json:"directions,omitempty"`
	Operator           *BusinessDetails `json:"operator,omitempty"`
	Suboperator        *BusinessDetails `json:"suboperator,omitempty"`
	Owner              *BusinessDetails `json:"owner,omitempty"`
	Facilities         []string         `json:"facilities,omitempty"`
	TimeZone           string           `json:"time_zone"`
	OpeningTimes       *Hours           `json:"opening_times,omitempty"`
	ChargingWhenClosed *bool            `json:"charging_when_closed,omitempty"`
	LastUpdated        string           `json:"last_updated"`

	// Added in v0.1.0 of the module.
	PublishAllowedTo []PublishToken          `json:"publish_allowed_to,omitempty"`
	RelatedLocations []AdditionalGeoLocation `json:"related_locations,omitempty"`
	Images           []Image                 `json:"images,omitempty"`
	EnergyMix        *EnergyMix              `json:"energy_mix,omitempty"`
}

// Hours (8.4.14).
type Hours struct {
	TwentyFourSeven bool `json:"twentyfourseven"`
	// Added in v0.1.0 of the module.
	RegularHours        []RegularHours      `json:"regular_hours,omitempty"`
	ExceptionalOpenings []ExceptionalPeriod `json:"exceptional_openings,omitempty"`
	ExceptionalClosings []ExceptionalPeriod `json:"exceptional_closings,omitempty"`
}

// EVSE (8.3.2).
type EVSE struct {
	UID                 string        `json:"uid"`
	EVSEID              string        `json:"evse_id,omitempty"`
	Status              string        `json:"status"`
	Capabilities        []string      `json:"capabilities,omitempty"`
	Connectors          []Connector   `json:"connectors"`
	FloorLevel          string        `json:"floor_level,omitempty"`
	Coordinates         *GeoLocation  `json:"coordinates,omitempty"`
	PhysicalReference   string        `json:"physical_reference,omitempty"`
	Directions          []DisplayText `json:"directions,omitempty"`
	ParkingRestrictions []string      `json:"parking_restrictions,omitempty"`
	LastUpdated         string        `json:"last_updated"`
	// Added in v0.1.0 of the module.
	StatusSchedule []StatusSchedule `json:"status_schedule,omitempty"`
	Images         []Image          `json:"images,omitempty"`
}

// Connector (8.3.3).
type Connector struct {
	ID                 string   `json:"id"`
	Standard           string   `json:"standard"`
	Format             string   `json:"format"`
	PowerType          string   `json:"power_type"`
	MaxVoltage         int      `json:"max_voltage"`
	MaxAmperage        int      `json:"max_amperage"`
	MaxElectricPower   int      `json:"max_electric_power,omitempty"`
	TariffIDs          []string `json:"tariff_ids,omitempty"`
	TermsAndConditions string   `json:"terms_and_conditions,omitempty"`
	LastUpdated        string   `json:"last_updated"`
}

// Price (16).
type Price struct {
	ExclVAT float64  `json:"excl_vat"`
	InclVAT *float64 `json:"incl_vat,omitempty"`
}

// CdrToken (10.4.5).
type CdrToken struct {
	CountryCode string `json:"country_code"`
	PartyID     string `json:"party_id"`
	UID         string `json:"uid"`
	Type        string `json:"type"`
	ContractID  string `json:"contract_id"`
}

// CdrDimension (10.4.2).
type CdrDimension struct {
	Type   string  `json:"type"`
	Volume float64 `json:"volume"`
}

// ChargingPeriod (10.4.6).
type ChargingPeriod struct {
	StartDateTime string         `json:"start_date_time"`
	Dimensions    []CdrDimension `json:"dimensions"`
	TariffID      string         `json:"tariff_id,omitempty"`
}

// Session (9.3.1).
type Session struct {
	CountryCode            string           `json:"country_code"`
	PartyID                string           `json:"party_id"`
	ID                     string           `json:"id"`
	StartDateTime          string           `json:"start_date_time"`
	EndDateTime            string           `json:"end_date_time,omitempty"`
	KWh                    float64          `json:"kwh"`
	CdrToken               CdrToken         `json:"cdr_token"`
	AuthMethod             string           `json:"auth_method"`
	AuthorizationReference string           `json:"authorization_reference,omitempty"`
	LocationID             string           `json:"location_id"`
	EVSEUID                string           `json:"evse_uid"`
	ConnectorID            string           `json:"connector_id"`
	MeterID                string           `json:"meter_id,omitempty"`
	Currency               string           `json:"currency"`
	ChargingPeriods        []ChargingPeriod `json:"charging_periods,omitempty"`
	TotalCost              *Price           `json:"total_cost,omitempty"`
	Status                 string           `json:"status"`
	LastUpdated            string           `json:"last_updated"`
}

// CdrLocation (10.4.4).
type CdrLocation struct {
	ID                 string      `json:"id"`
	Name               string      `json:"name,omitempty"`
	Address            string      `json:"address"`
	City               string      `json:"city"`
	PostalCode         string      `json:"postal_code,omitempty"`
	State              string      `json:"state,omitempty"`
	Country            string      `json:"country"`
	Coordinates        GeoLocation `json:"coordinates"`
	EVSEUID            string      `json:"evse_uid"`
	EVSEID             string      `json:"evse_id"`
	ConnectorID        string      `json:"connector_id"`
	ConnectorStandard  string      `json:"connector_standard"`
	ConnectorFormat    string      `json:"connector_format"`
	ConnectorPowerType string      `json:"connector_power_type"`
}

// CDR (10.3.1).
type CDR struct {
	CountryCode            string           `json:"country_code"`
	PartyID                string           `json:"party_id"`
	ID                     string           `json:"id"`
	StartDateTime          string           `json:"start_date_time"`
	EndDateTime            string           `json:"end_date_time"`
	SessionID              string           `json:"session_id,omitempty"`
	CdrToken               CdrToken         `json:"cdr_token"`
	AuthMethod             string           `json:"auth_method"`
	AuthorizationReference string           `json:"authorization_reference,omitempty"`
	CdrLocation            CdrLocation      `json:"cdr_location"`
	MeterID                string           `json:"meter_id,omitempty"`
	Currency               string           `json:"currency"`
	Tariffs                []Tariff         `json:"tariffs,omitempty"`
	ChargingPeriods        []ChargingPeriod `json:"charging_periods"`
	TotalCost              Price            `json:"total_cost"`
	TotalFixedCost         *Price           `json:"total_fixed_cost,omitempty"`
	TotalEnergy            float64          `json:"total_energy"`
	TotalEnergyCost        *Price           `json:"total_energy_cost,omitempty"`
	TotalTime              float64          `json:"total_time"`
	TotalTimeCost          *Price           `json:"total_time_cost,omitempty"`
	TotalParkingTime       *float64         `json:"total_parking_time,omitempty"`
	TotalParkingCost       *Price           `json:"total_parking_cost,omitempty"`
	Remark                 string           `json:"remark,omitempty"`
	LastUpdated            string           `json:"last_updated"`

	// Added in v0.1.0 of the module.
	SignedData               *SignedData `json:"signed_data,omitempty"`
	TotalReservationCost     *Price      `json:"total_reservation_cost,omitempty"`
	InvoiceReferenceID       string      `json:"invoice_reference_id,omitempty"`
	Credit                   *bool       `json:"credit,omitempty"`
	CreditReferenceID        string      `json:"credit_reference_id,omitempty"`
	HomeChargingCompensation *bool       `json:"home_charging_compensation,omitempty"`
}

// Tariff (11.3.1).
type Tariff struct {
	CountryCode   string          `json:"country_code"`
	PartyID       string          `json:"party_id"`
	ID            string          `json:"id"`
	Currency      string          `json:"currency"`
	Type          string          `json:"type,omitempty"`
	TariffAltText []DisplayText   `json:"tariff_alt_text,omitempty"`
	TariffAltURL  string          `json:"tariff_alt_url,omitempty"`
	MinPrice      *Price          `json:"min_price,omitempty"`
	MaxPrice      *Price          `json:"max_price,omitempty"`
	Elements      []TariffElement `json:"elements"`
	LastUpdated   string          `json:"last_updated"`
	// Added in v0.1.0 of the module.
	StartDateTime string     `json:"start_date_time,omitempty"`
	EndDateTime   string     `json:"end_date_time,omitempty"`
	EnergyMix     *EnergyMix `json:"energy_mix,omitempty"`
}

// TariffElement (11.4.4).
type TariffElement struct {
	PriceComponents []PriceComponent    `json:"price_components"`
	Restrictions    *TariffRestrictions `json:"restrictions,omitempty"`
}

// PriceComponent (11.4.2).
type PriceComponent struct {
	Type     string   `json:"type"`
	Price    float64  `json:"price"`
	VAT      *float64 `json:"vat,omitempty"`
	StepSize int      `json:"step_size"`
}

// TariffRestrictions (11.4.6).
type TariffRestrictions struct {
	StartTime   string   `json:"start_time,omitempty"`
	EndTime     string   `json:"end_time,omitempty"`
	StartDate   string   `json:"start_date,omitempty"`
	EndDate     string   `json:"end_date,omitempty"`
	MinKWh      *float64 `json:"min_kwh,omitempty"`
	MaxKWh      *float64 `json:"max_kwh,omitempty"`
	MinCurrent  *float64 `json:"min_current,omitempty"`
	MaxCurrent  *float64 `json:"max_current,omitempty"`
	MinPower    *float64 `json:"min_power,omitempty"`
	MaxPower    *float64 `json:"max_power,omitempty"`
	MinDuration *int     `json:"min_duration,omitempty"`
	MaxDuration *int     `json:"max_duration,omitempty"`
	DayOfWeek   []string `json:"day_of_week,omitempty"`
	Reservation string   `json:"reservation,omitempty"`
}

// Commands (13.3).
type StartSession struct {
	ResponseURL            string `json:"response_url"`
	Token                  Token  `json:"token"`
	LocationID             string `json:"location_id"`
	EVSEUID                string `json:"evse_uid,omitempty"`
	ConnectorID            string `json:"connector_id,omitempty"`
	AuthorizationReference string `json:"authorization_reference,omitempty"`
}

type StopSession struct {
	ResponseURL string `json:"response_url"`
	SessionID   string `json:"session_id"`
}

type UnlockConnector struct {
	ResponseURL string `json:"response_url"`
	LocationID  string `json:"location_id"`
	EVSEUID     string `json:"evse_uid"`
	ConnectorID string `json:"connector_id"`
}

type ReserveNow struct {
	ResponseURL            string `json:"response_url"`
	Token                  Token  `json:"token"`
	ExpiryDate             string `json:"expiry_date"`
	ReservationID          string `json:"reservation_id"`
	LocationID             string `json:"location_id"`
	EVSEUID                string `json:"evse_uid,omitempty"`
	AuthorizationReference string `json:"authorization_reference,omitempty"`
}

type CancelReservation struct {
	ResponseURL   string `json:"response_url"`
	ReservationID string `json:"reservation_id"`
}

// CommandResponse (13.3.2).
type CommandResponse struct {
	Result  string        `json:"result"`
	Timeout int           `json:"timeout"`
	Message []DisplayText `json:"message,omitempty"`
}

// CommandResult (13.3.3).
type CommandResult struct {
	Result  string        `json:"result"`
	Message []DisplayText `json:"message,omitempty"`
}

// Command types (13.4.3).
const (
	CmdCancelReservation = "CANCEL_RESERVATION"
	CmdReserveNow        = "RESERVE_NOW"
	CmdStartSession      = "START_SESSION"
	CmdStopSession       = "STOP_SESSION"
	CmdUnlockConnector   = "UNLOCK_CONNECTOR"
)

// ClientInfo (15.4.1).
type ClientInfo struct {
	PartyID     string `json:"party_id"`
	CountryCode string `json:"country_code"`
	Role        string `json:"role"`
	Status      string `json:"status"`
	LastUpdated string `json:"last_updated"`
}

// FormatTime formats a DateTime (16: UTC, seconds).
func FormatTime(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05Z") }

// ParseTime reads a DateTime; OCPI allows a missing zone (UTC is meant).
func ParseTime(s string) (time.Time, bool) {
	for _, l := range []string{time.RFC3339Nano, "2006-01-02T15:04:05", "2006-01-02T15:04:05.999"} {
		if t, err := time.Parse(l, s); err == nil {
			return t.UTC(), true
		}
	}
	return time.Time{}, false
}
