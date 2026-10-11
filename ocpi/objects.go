// Copyright 2026 Abhinav Gautam
// SPDX-License-Identifier: Apache-2.0

package ocpi

// OCPI 2.2.1 objects, classes and enumerations added in v0.1.0 of the
// module, so that every module of the version is complete: the
// remaining Location classes (8.4), charging preferences (9.3.2), signed
// CDR data (10.4.7, 10.4.8), the charging profiles module (14), and the
// enumeration values of every module.

// ModChargingProfiles is the charging profiles module (14).
const ModChargingProfiles = "chargingprofiles"

// Party roles (16.4, Role) beyond CPO, EMSP and HUB.
const (
	RoleNAP   = "NAP"
	RoleNSP   = "NSP"
	RoleOther = "OTHER"
	RoleSCSP  = "SCSP"
)

// Hub status codes (5.4) beyond 4001.
const (
	StatusHubError      = 4000
	StatusHubTimeout    = 4002
	StatusHubConnection = 4003
)

// AdditionalGeoLocation (8.4.1) is a related point with a name.
type AdditionalGeoLocation struct {
	Latitude  string       `json:"latitude"`
	Longitude string       `json:"longitude"`
	Name      *DisplayText `json:"name,omitempty"`
}

// EnergyMix (8.4.6).
type EnergyMix struct {
	IsGreenEnergy     bool                  `json:"is_green_energy"`
	EnergySources     []EnergySource        `json:"energy_sources,omitempty"`
	EnvironImpact     []EnvironmentalImpact `json:"environ_impact,omitempty"`
	SupplierName      string                `json:"supplier_name,omitempty"`
	EnergyProductName string                `json:"energy_product_name,omitempty"`
}

// EnergySource (8.4.7): a source and its share in percent.
type EnergySource struct {
	Source     string  `json:"source"`
	Percentage float64 `json:"percentage"`
}

// EnvironmentalImpact (8.4.9): an impact in g/kWh.
type EnvironmentalImpact struct {
	Category string  `json:"category"`
	Amount   float64 `json:"amount"`
}

// ExceptionalPeriod (8.4.11).
type ExceptionalPeriod struct {
	PeriodBegin string `json:"period_begin"`
	PeriodEnd   string `json:"period_end"`
}

// Image (8.4.15).
type Image struct {
	URL       string `json:"url"`
	Thumbnail string `json:"thumbnail,omitempty"`
	Category  string `json:"category"`
	Type      string `json:"type"`
	Width     int    `json:"width,omitempty"`
	Height    int    `json:"height,omitempty"`
}

// PublishToken is PublishTokenType (8.4.20): who may see an unpublished
// Location.
type PublishToken struct {
	UID          string `json:"uid,omitempty"`
	Type         string `json:"type,omitempty"`
	VisualNumber string `json:"visual_number,omitempty"`
	Issuer       string `json:"issuer,omitempty"`
	GroupID      string `json:"group_id,omitempty"`
}

// RegularHours (8.4.21): weekday 1 (Monday) to 7, times as "HH:MM".
type RegularHours struct {
	Weekday     int    `json:"weekday"`
	PeriodBegin string `json:"period_begin"`
	PeriodEnd   string `json:"period_end"`
}

// StatusSchedule (8.4.23).
type StatusSchedule struct {
	PeriodBegin string `json:"period_begin"`
	PeriodEnd   string `json:"period_end,omitempty"`
	Status      string `json:"status"`
}

// ChargingPreferences (9.3.2), PUT by the eMSP to the CPO's sessions.
type ChargingPreferences struct {
	ProfileType      string   `json:"profile_type"`
	DepartureTime    string   `json:"departure_time,omitempty"`
	EnergyNeed       *float64 `json:"energy_need,omitempty"`
	DischargeAllowed *bool    `json:"discharge_allowed,omitempty"`
}

// SignedData (10.4.7).
type SignedData struct {
	EncodingMethod        string        `json:"encoding_method"`
	EncodingMethodVersion *int          `json:"encoding_method_version,omitempty"`
	PublicKey             string        `json:"public_key,omitempty"`
	SignedValues          []SignedValue `json:"signed_values"`
	URL                   string        `json:"url,omitempty"`
}

// SignedValue (10.4.8).
type SignedValue struct {
	Nature     string `json:"nature"`
	PlainData  string `json:"plain_data"`
	SignedData string `json:"signed_data"`
}

// EnergyContract (12.4.2).
type EnergyContract struct {
	SupplierName string `json:"supplier_name"`
	ContractID   string `json:"contract_id,omitempty"`
}

// ChargingProfileResponse (14.4.1): the CPO's answer to a charging
// profile request.
type ChargingProfileResponse struct {
	Result  string `json:"result"`
	Timeout int    `json:"timeout"`
}

// ActiveChargingProfileResult (14.4.2), POSTed to the response_url of a
// GET.
type ActiveChargingProfileResult struct {
	Result  string                 `json:"result"`
	Profile *ActiveChargingProfile `json:"profile,omitempty"`
}

// ChargingProfileResult (14.4.3), POSTed to the response_url of a PUT.
type ChargingProfileResult struct {
	Result string `json:"result"`
}

// ClearProfileResult (14.4.4), POSTed to the response_url of a DELETE.
type ClearProfileResult struct {
	Result string `json:"result"`
}

// SetChargingProfile (14.4.5), PUT by the sender of profiles.
type SetChargingProfile struct {
	ChargingProfile ChargingProfile `json:"charging_profile"`
	ResponseURL     string          `json:"response_url"`
}

// ActiveChargingProfile (14.5.1).
type ActiveChargingProfile struct {
	StartDateTime   string          `json:"start_date_time"`
	ChargingProfile ChargingProfile `json:"charging_profile"`
}

// ChargingProfile (14.5.3).
type ChargingProfile struct {
	StartDateTime         string                  `json:"start_date_time,omitempty"`
	Duration              *int                    `json:"duration,omitempty"`
	ChargingRateUnit      string                  `json:"charging_rate_unit"`
	MinChargingRate       *float64                `json:"min_charging_rate,omitempty"`
	ChargingProfilePeriod []ChargingProfilePeriod `json:"charging_profile_period,omitempty"`
}

// ChargingProfilePeriod (14.5.4).
type ChargingProfilePeriod struct {
	StartPeriod int     `json:"start_period"`
	Limit       float64 `json:"limit"`
}

// Enumeration values (facts of 8.4, 9.4, 10.4, 11.4, 12.4, 13.4, 14.5,
// 15.4 and 16).
const (
	// Status (8.4.22) of an EVSE.
	EVSEAvailable   = "AVAILABLE"
	EVSEBlocked     = "BLOCKED"
	EVSECharging    = "CHARGING"
	EVSEInoperative = "INOPERATIVE"
	EVSEOutOfOrder  = "OUTOFORDER"
	EVSEPlanned     = "PLANNED"
	EVSERemoved     = "REMOVED"
	EVSEReserved    = "RESERVED"
	EVSEUnknown     = "UNKNOWN"

	// Capability (8.4.3).
	CapChargingProfileCapable        = "CHARGING_PROFILE_CAPABLE"
	CapChargingPreferencesCapable    = "CHARGING_PREFERENCES_CAPABLE"
	CapChipCardSupport               = "CHIP_CARD_SUPPORT"
	CapContactlessCardSupport        = "CONTACTLESS_CARD_SUPPORT"
	CapCreditCardPayable             = "CREDIT_CARD_PAYABLE"
	CapDebitCardPayable              = "DEBIT_CARD_PAYABLE"
	CapPEDTerminal                   = "PED_TERMINAL"
	CapRemoteStartStopCapable        = "REMOTE_START_STOP_CAPABLE"
	CapReservable                    = "RESERVABLE"
	CapRFIDReader                    = "RFID_READER"
	CapStartSessionConnectorRequired = "START_SESSION_CONNECTOR_REQUIRED"
	CapTokenGroupCapable             = "TOKEN_GROUP_CAPABLE"
	CapUnlockCapable                 = "UNLOCK_CAPABLE"

	// ConnectorFormat (8.4.4).
	FormatSocket = "SOCKET"
	FormatCable  = "CABLE"

	// PowerType (8.4.19).
	PowerAC1Phase      = "AC_1_PHASE"
	PowerAC2Phase      = "AC_2_PHASE"
	PowerAC2PhaseSplit = "AC_2_PHASE_SPLIT"
	PowerAC3Phase      = "AC_3_PHASE"
	PowerDC            = "DC"

	// ParkingType (8.4.18).
	ParkingAlongMotorway     = "ALONG_MOTORWAY"
	ParkingGarage            = "PARKING_GARAGE"
	ParkingLot               = "PARKING_LOT"
	ParkingOnDriveway        = "ON_DRIVEWAY"
	ParkingOnStreet          = "ON_STREET"
	ParkingUndergroundGarage = "UNDERGROUND_GARAGE"

	// EnergySourceCategory (8.4.8).
	SourceNuclear       = "NUCLEAR"
	SourceGeneralFossil = "GENERAL_FOSSIL"
	SourceCoal          = "COAL"
	SourceGas           = "GAS"
	SourceGeneralGreen  = "GENERAL_GREEN"
	SourceSolar         = "SOLAR"
	SourceWind          = "WIND"
	SourceWater         = "WATER"

	// EnvironmentalImpactCategory (8.4.10).
	ImpactNuclearWaste  = "NUCLEAR_WASTE"
	ImpactCarbonDioxide = "CARBON_DIOXIDE"

	// ImageCategory (8.4.16).
	ImageCharger  = "CHARGER"
	ImageEntrance = "ENTRANCE"
	ImageLocation = "LOCATION"
	ImageNetwork  = "NETWORK"
	ImageOperator = "OPERATOR"
	ImageOther    = "OTHER"
	ImageOwner    = "OWNER"

	// SessionStatus (9.4.3).
	SessionActive      = "ACTIVE"
	SessionCompleted   = "COMPLETED"
	SessionInvalid     = "INVALID"
	SessionPending     = "PENDING"
	SessionReservation = "RESERVATION"

	// ProfileType (9.4.2).
	ProfileCheap   = "CHEAP"
	ProfileFast    = "FAST"
	ProfileGreen   = "GREEN"
	ProfileRegular = "REGULAR"

	// ChargingPreferencesResponse (9.4.1).
	PrefsAccepted                = "ACCEPTED"
	PrefsDepartureRequired       = "DEPARTURE_REQUIRED"
	PrefsEnergyNeedRequired      = "ENERGY_NEED_REQUIRED"
	PrefsNotPossible             = "NOT_POSSIBLE"
	PrefsProfileTypeNotSupported = "PROFILE_TYPE_NOT_SUPPORTED"

	// AuthMethod (10.4.1).
	AuthRequest   = "AUTH_REQUEST"
	AuthCommand   = "COMMAND"
	AuthWhitelist = "WHITELIST"

	// CdrDimensionType (10.4.3).
	DimCurrent         = "CURRENT"
	DimEnergy          = "ENERGY"
	DimEnergyExport    = "ENERGY_EXPORT"
	DimEnergyImport    = "ENERGY_IMPORT"
	DimMaxCurrent      = "MAX_CURRENT"
	DimMinCurrent      = "MIN_CURRENT"
	DimMaxPower        = "MAX_POWER"
	DimMinPower        = "MIN_POWER"
	DimParkingTime     = "PARKING_TIME"
	DimPower           = "POWER"
	DimReservationTime = "RESERVATION_TIME"
	DimStateOfCharge   = "STATE_OF_CHARGE"
	DimTime            = "TIME"

	// TariffDimensionType (11.4.5).
	TariffEnergy      = "ENERGY"
	TariffFlat        = "FLAT"
	TariffParkingTime = "PARKING_TIME"
	TariffTime        = "TIME"

	// TariffType (11.4.7).
	TariffAdHocPayment = "AD_HOC_PAYMENT"
	TariffProfileCheap = "PROFILE_CHEAP"
	TariffProfileFast  = "PROFILE_FAST"
	TariffProfileGreen = "PROFILE_GREEN"
	TariffRegular      = "REGULAR"

	// ReservationRestrictionType (11.4.3).
	RestrictionReservation        = "RESERVATION"
	RestrictionReservationExpires = "RESERVATION_EXPIRES"

	// AllowedType (12.4.1).
	AllowedAllowed    = "ALLOWED"
	AllowedBlocked    = "BLOCKED"
	AllowedExpired    = "EXPIRED"
	AllowedNoCredit   = "NO_CREDIT"
	AllowedNotAllowed = "NOT_ALLOWED"

	// TokenType (12.4.4).
	TokenAdHocUser = "AD_HOC_USER"
	TokenAppUser   = "APP_USER"
	TokenOther     = "OTHER"
	TokenRFID      = "RFID"

	// CommandResponseType (13.4.1).
	CmdRespNotSupported   = "NOT_SUPPORTED"
	CmdRespRejected       = "REJECTED"
	CmdRespAccepted       = "ACCEPTED"
	CmdRespUnknownSession = "UNKNOWN_SESSION"

	// CommandResultType (13.4.2).
	CmdResultAccepted            = "ACCEPTED"
	CmdResultCanceledReservation = "CANCELED_RESERVATION"
	CmdResultEVSEOccupied        = "EVSE_OCCUPIED"
	CmdResultEVSEInoperative     = "EVSE_INOPERATIVE"
	CmdResultFailed              = "FAILED"
	CmdResultNotSupported        = "NOT_SUPPORTED"
	CmdResultRejected            = "REJECTED"
	CmdResultTimeout             = "TIMEOUT"
	CmdResultUnknownReservation  = "UNKNOWN_RESERVATION"

	// ChargingRateUnit (14.5.2).
	RateWatts   = "W"
	RateAmperes = "A"

	// ChargingProfileResponseType (14.5.5).
	ProfileRespAccepted       = "ACCEPTED"
	ProfileRespNotSupported   = "NOT_SUPPORTED"
	ProfileRespRejected       = "REJECTED"
	ProfileRespTooOften       = "TOO_OFTEN"
	ProfileRespUnknownSession = "UNKNOWN_SESSION"

	// ChargingProfileResultType (14.5.6).
	ProfileResultAccepted = "ACCEPTED"
	ProfileResultRejected = "REJECTED"
	ProfileResultUnknown  = "UNKNOWN"

	// ConnectionStatus (15.4.2).
	ConnConnected = "CONNECTED"
	ConnOffline   = "OFFLINE"
	ConnPlanned   = "PLANNED"
	ConnSuspended = "SUSPENDED"
)
