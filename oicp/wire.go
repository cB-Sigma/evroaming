package oicp

import (
	"encoding/json"
	"regexp"
	"strings"
)

// The OICP 2.3 messages a CPO sends and takes, as JSON (field
// names as in Hubject's OpenAPI description of the CPO side,
// github.com/hubject/oicp-cpo-2.3-api-doc, src/schemas). Optional fields
// are omitted when empty.

// StatusCode is the status of an answer (03 Data Types, StatusCodeType).
type StatusCode struct {
	Code           string `json:"Code"`
	Description    string `json:"Description,omitempty"`
	AdditionalInfo string `json:"AdditionalInfo,omitempty"`
}

// Status codes (StatusCodeType).
const (
	CodeSuccess          = "000"
	CodeHubjectSystem    = "001"
	CodeHubjectDatabase  = "002"
	CodeDataTransaction  = "009"
	CodeUnauthorized     = "017"
	CodeInconsistentEVSE = "018"
	CodeSystemError      = "021"
	CodeDataError        = "022"
	CodeInvalidUID       = "102"
	CodeNoPositiveAuth   = "106"
	CodeEvcoLocked       = "200"
	CodeNoValidContract  = "210"
	CodePartnerNotFound  = "300"
	CodeSessionInvalid   = "400"
	CodeEVSECommFailed   = "501"
	CodeNoEVConnected    = "510"
	CodeEVSEReserved     = "601"
	CodeEVSEInUse        = "602"
	CodeUnknownEVSE      = "603"
	CodeEVSEOutOfService = "700"

	// The other codes of the StatusCode table (added in v0.1.0 of the
	// module).
	CodeInconsistentEvco     = "019"
	CodeQRAuthFailed         = "101"
	CodeRFIDUnreadable       = "103"
	CodePLCInvalidEvco       = "105"
	CodeQRTimeout            = "110"
	CodePLCInvalidUnderlying = "120"
	CodePLCInvalidCert       = "121"
	CodePLCTimeout           = "122"
	CodePartnerNoResponse    = "310"
	CodeServiceUnavailable   = "320"
	CodeEVSENotCompatible    = "604"
)

// Acknowledgement is eRoamingAcknowledgment.
type Acknowledgement struct {
	Result              bool       `json:"Result"`
	StatusCode          StatusCode `json:"StatusCode"`
	SessionID           string     `json:"SessionID,omitempty"`
	CPOPartnerSessionID string     `json:"CPOPartnerSessionID,omitempty"`
	EMPPartnerSessionID string     `json:"EMPPartnerSessionID,omitempty"`
}

// Identification is IdentificationType: one of its members is set.
type Identification struct {
	RFIDMifareFamilyIdentification *UIDIdentification    `json:"RFIDMifareFamilyIdentification,omitempty"`
	RFIDIdentification             *RFIDIdentification   `json:"RFIDIdentification,omitempty"`
	QRCodeIdentification           *QRCodeIdentification `json:"QRCodeIdentification,omitempty"`
	PlugAndChargeIdentification    *EvcoIdentification   `json:"PlugAndChargeIdentification,omitempty"`
	RemoteIdentification           *EvcoIdentification   `json:"RemoteIdentification,omitempty"`
}

// UIDIdentification is a Mifare card's UID.
type UIDIdentification struct {
	UID string `json:"UID"`
}

// RFIDIdentification is a card with its contract.
type RFIDIdentification struct {
	UID           string `json:"UID"`
	EvcoID        string `json:"EvcoID,omitempty"`
	RFID          string `json:"RFID"`
	PrintedNumber string `json:"PrintedNumber,omitempty"`
	ExpiryDate    string `json:"ExpiryDate,omitempty"`
}

// QRCodeIdentification is a contract with a PIN (only its EvcoID is used).
type QRCodeIdentification struct {
	EvcoID    string          `json:"EvcoID"`
	PIN       string          `json:"PIN,omitempty"`
	HashedPIN json.RawMessage `json:"HashedPIN,omitempty"`
}

// EvcoIdentification is a contract ID (remote and Plug and Charge).
type EvcoIdentification struct {
	EvcoID string `json:"EvcoID"`
}

// Token is the identifier the identification carries: the card's UID or
// the contract ID.
func (i Identification) Token() string {
	switch {
	case i.RFIDMifareFamilyIdentification != nil:
		return i.RFIDMifareFamilyIdentification.UID
	case i.RFIDIdentification != nil:
		return i.RFIDIdentification.UID
	case i.RemoteIdentification != nil:
		return i.RemoteIdentification.EvcoID
	case i.PlugAndChargeIdentification != nil:
		return i.PlugAndChargeIdentification.EvcoID
	case i.QRCodeIdentification != nil:
		return i.QRCodeIdentification.EvcoID
	}
	return ""
}

// Contract is the identification's EvcoID, when it has one.
func (i Identification) Contract() string {
	switch {
	case i.RFIDIdentification != nil:
		return i.RFIDIdentification.EvcoID
	case i.RemoteIdentification != nil:
		return i.RemoteIdentification.EvcoID
	case i.PlugAndChargeIdentification != nil:
		return i.PlugAndChargeIdentification.EvcoID
	case i.QRCodeIdentification != nil:
		return i.QRCodeIdentification.EvcoID
	}
	return ""
}

// AuthorizeStart is eRoamingAuthorizeStart (V2.1).
type AuthorizeStart struct {
	SessionID           string         `json:"SessionID,omitempty"`
	CPOPartnerSessionID string         `json:"CPOPartnerSessionID,omitempty"`
	EMPPartnerSessionID string         `json:"EMPPartnerSessionID,omitempty"`
	OperatorID          string         `json:"OperatorID"`
	EvseID              string         `json:"EvseID,omitempty"`
	Identification      Identification `json:"Identification"`
	PartnerProductID    string         `json:"PartnerProductID,omitempty"`
}

// AuthorizeStop is eRoamingAuthorizeStop (V2.1).
type AuthorizeStop struct {
	SessionID           string         `json:"SessionID"`
	CPOPartnerSessionID string         `json:"CPOPartnerSessionID,omitempty"`
	EMPPartnerSessionID string         `json:"EMPPartnerSessionID,omitempty"`
	OperatorID          string         `json:"OperatorID"`
	EvseID              string         `json:"EvseID,omitempty"`
	Identification      Identification `json:"Identification"`
	// PartnerProductID is not a property of the request's schema, but
	// Hubject's example carries it; it is read and, when set, sent.
	PartnerProductID string `json:"PartnerProductID,omitempty"`
}

// Authorization is eRoamingAuthorizationStart and eRoamingAuthorizationStop.
type Authorization struct {
	SessionID                        string           `json:"SessionID,omitempty"`
	CPOPartnerSessionID              string           `json:"CPOPartnerSessionID,omitempty"`
	EMPPartnerSessionID              string           `json:"EMPPartnerSessionID,omitempty"`
	ProviderID                       string           `json:"ProviderID,omitempty"`
	AuthorizationStatus              string           `json:"AuthorizationStatus"`
	StatusCode                       StatusCode       `json:"StatusCode"`
	AuthorizationStopIdentifications []Identification `json:"AuthorizationStopIdentifications,omitempty"`
}

// Authorization statuses.
const (
	Authorized    = "Authorized"
	NotAuthorized = "NotAuthorized"
)

// RemoteStart is eRoamingAuthorizeRemoteStart (V2.1), which Hubject sends us.
type RemoteStart struct {
	SessionID           string         `json:"SessionID"`
	CPOPartnerSessionID string         `json:"CPOPartnerSessionID,omitempty"`
	EMPPartnerSessionID string         `json:"EMPPartnerSessionID,omitempty"`
	ProviderID          string         `json:"ProviderID"`
	EvseID              string         `json:"EvseID"`
	Identification      Identification `json:"Identification"`
	PartnerProductID    string         `json:"PartnerProductID,omitempty"`
}

// RemoteStop is eRoamingAuthorizeRemoteStop (V2.1).
type RemoteStop struct {
	SessionID           string `json:"SessionID"`
	CPOPartnerSessionID string `json:"CPOPartnerSessionID,omitempty"`
	EMPPartnerSessionID string `json:"EMPPartnerSessionID,omitempty"`
	ProviderID          string `json:"ProviderID"`
	EvseID              string `json:"EvseID"`
}

// PushEvseData is eRoamingPushEvseData (V2.3).
type PushEvseData struct {
	ActionType       string           `json:"ActionType"`
	OperatorEvseData OperatorEvseData `json:"OperatorEvseData"`
}

// OperatorEvseData is an operator's EVSE records.
type OperatorEvseData struct {
	OperatorID     string           `json:"OperatorID"`
	OperatorName   string           `json:"OperatorName"`
	EvseDataRecord []EvseDataRecord `json:"EvseDataRecord"`
}

// Push action types.
const (
	ActionFullLoad = "fullLoad"
	ActionUpdate   = "update"
	ActionInsert   = "insert"
	ActionDelete   = "delete"
)

// EvseDataRecord is one EVSE's master data.
type EvseDataRecord struct {
	EvseID                           string             `json:"EvseID"`
	ChargingStationID                string             `json:"ChargingStationId,omitempty"`
	ChargingStationNames             []InfoText         `json:"ChargingStationNames"`
	HardwareManufacturer             string             `json:"HardwareManufacturer,omitempty"`
	SubOperatorName                  string             `json:"SubOperatorName,omitempty"`
	Address                          Address            `json:"Address"`
	GeoCoordinates                   GeoCoordinates     `json:"GeoCoordinates"`
	Plugs                            []string           `json:"Plugs"`
	ChargingFacilities               []ChargingFacility `json:"ChargingFacilities"`
	RenewableEnergy                  bool               `json:"RenewableEnergy"`
	CalibrationLawDataAvailability   string             `json:"CalibrationLawDataAvailability"`
	AuthenticationModes              []string           `json:"AuthenticationModes"`
	PaymentOptions                   []string           `json:"PaymentOptions"`
	ValueAddedServices               []string           `json:"ValueAddedServices"`
	Accessibility                    string             `json:"Accessibility"`
	AccessibilityLocation            string             `json:"AccessibilityLocation,omitempty"`
	HotlinePhoneNumber               string             `json:"HotlinePhoneNumber"`
	AdditionalInfo                   []InfoText         `json:"AdditionalInfo,omitempty"`
	ChargingStationLocationReference []InfoText         `json:"ChargingStationLocationReference,omitempty"`
	IsOpen24Hours                    bool               `json:"IsOpen24Hours"`
	OpeningTimes                     []OpeningTime      `json:"OpeningTimes,omitempty"`
	IsHubjectCompatible              bool               `json:"IsHubjectCompatible"`
	DynamicInfoAvailable             string             `json:"DynamicInfoAvailable"`

	// Added in v0.1.0 of the module (all optional in the OpenAPI).

	// DeltaType (update, insert, delete) and LastUpdate mark a record of
	// a pull answer; a CPO leaves them empty when pushing.
	DeltaType  string `json:"deltaType,omitempty"`
	LastUpdate string `json:"lastUpdate,omitempty"`
	// ChargingPoolID groups the EVSEs of one site (an ID with "P").
	ChargingPoolID       string               `json:"ChargingPoolID,omitempty"`
	ChargingStationImage string               `json:"ChargingStationImage,omitempty"`
	DynamicPowerLevel    *bool                `json:"DynamicPowerLevel,omitempty"`
	EnergySource         []EnergySource       `json:"EnergySource,omitempty"`
	EnvironmentalImpact  *EnvironmentalImpact `json:"EnvironmentalImpact,omitempty"`
	MaxCapacity          *int                 `json:"MaxCapacity,omitempty"`
	// GeoChargingPointEntrance is where the charging point is entered.
	GeoChargingPointEntrance *GeoCoordinates `json:"GeoChargingPointEntrance,omitempty"`
	HubOperatorID            string          `json:"HubOperatorID,omitempty"`
	ClearinghouseID          string          `json:"ClearinghouseID,omitempty"`
}

// EnergySource is one source of the energy mix and its share in percent.
type EnergySource struct {
	Energy     string `json:"Energy,omitempty"`
	Percentage *int   `json:"Percentage,omitempty"`
}

// EnvironmentalImpact is CO2 emission (g/kWh) and nuclear waste (g/kWh).
type EnvironmentalImpact struct {
	CO2Emission  *float64 `json:"CO2Emission,omitempty"`
	NuclearWaste *float64 `json:"NuclearWaste,omitempty"`
}

// Values of EvseDataRecord fields (OpenAPI enumerations).
const (
	CalibrationLocal        = "Local"
	CalibrationExternal     = "External"
	CalibrationNotAvailable = "Not Available"

	DynamicInfoTrue  = "true"
	DynamicInfoFalse = "false"
	DynamicInfoAuto  = "auto"

	AccessibilityFree       = "Free publicly accessible"
	AccessibilityRestricted = "Restricted access"
	AccessibilityPaying     = "Paying publicly accessible"
	AccessibilityTest       = "Test Station"

	PowerAC1Phase = "AC_1_PHASE"
	PowerAC3Phase = "AC_3_PHASE"
	PowerDC       = "DC"
)

// InfoText is InfoTextType.
type InfoText struct {
	Lang  string `json:"lang"`
	Value string `json:"value"`
}

// Address is AddressIso19773.
type Address struct {
	Country    string `json:"Country"`
	City       string `json:"City"`
	Street     string `json:"Street"`
	PostalCode string `json:"PostalCode"`
	HouseNum   string `json:"HouseNum"`
	Floor      string `json:"Floor,omitempty"`
	Region     string `json:"Region,omitempty"`
	TimeZone   string `json:"TimeZone,omitempty"`
	// ParkingFacility and ParkingSpot were added in v0.1.0 of the module.
	ParkingFacility *bool  `json:"ParkingFacility,omitempty"`
	ParkingSpot     string `json:"ParkingSpot,omitempty"`
}

// GeoCoordinates is one of three forms: decimal degrees, Google's
// "lat lon" string, or degrees, minutes and seconds.
type GeoCoordinates struct {
	DecimalDegree *DecimalDegree `json:"DecimalDegree,omitempty"`
	Google        *struct {
		Coordinates string `json:"Coordinates"`
	} `json:"Google,omitempty"`
	// DegreeMinuteSeconds was added in v0.1.0 of the module.
	DegreeMinuteSeconds *DegreeMinuteSeconds `json:"DegreeMinuteSeconds,omitempty"`
}

// DegreeMinuteSeconds are latitude and longitude as 9°21'39.32” strings.
type DegreeMinuteSeconds struct {
	Longitude string `json:"Longitude"`
	Latitude  string `json:"Latitude"`
}

// DecimalDegree are latitude and longitude as decimal strings.
type DecimalDegree struct {
	Longitude string `json:"Longitude"`
	Latitude  string `json:"Latitude"`
}

// ChargingFacility is one power level an EVSE offers.
type ChargingFacility struct {
	PowerType     string   `json:"PowerType"`
	Voltage       int      `json:"Voltage,omitempty"`
	Amperage      int      `json:"Amperage,omitempty"`
	Power         int      `json:"Power"`
	ChargingModes []string `json:"ChargingModes,omitempty"`
}

// OpeningTime is a period on some days.
type OpeningTime struct {
	Period []Period `json:"Period"`
	On     string   `json:"on"`
}

// Period is a begin and end time ("HH:MM").
type Period struct {
	Begin string `json:"begin"`
	End   string `json:"end"`
}

// PushEvseStatus is eRoamingPushEvseStatus (V2.1).
type PushEvseStatus struct {
	ActionType         string             `json:"ActionType"`
	OperatorEvseStatus OperatorEvseStatus `json:"OperatorEvseStatus"`
}

// OperatorEvseStatus is an operator's EVSE statuses.
type OperatorEvseStatus struct {
	OperatorID       string             `json:"OperatorID"`
	OperatorName     string             `json:"OperatorName,omitempty"`
	EvseStatusRecord []EvseStatusRecord `json:"EvseStatusRecord"`
}

// EvseStatusRecord is one EVSE's status.
type EvseStatusRecord struct {
	EvseID     string `json:"EvseID"`
	EvseStatus string `json:"EvseStatus"`
}

// EVSE statuses (EvseStatusType).
const (
	EvseAvailable    = "Available"
	EvseReserved     = "Reserved"
	EvseOccupied     = "Occupied"
	EvseOutOfService = "OutOfService"
	EvseUnknown      = "Unknown"
	// EvseNotFound is what Hubject reports for an EVSE it does not know.
	EvseNotFound = "EvseNotFound"
)

// ChargeDetailRecord is eRoamingChargeDetailRecord (V2.2). It carries
// no price: the session's cost goes to the EMP through the pricing
// product named by PartnerProductID.
type ChargeDetailRecord struct {
	SessionID           string         `json:"SessionID"`
	CPOPartnerSessionID string         `json:"CPOPartnerSessionID,omitempty"`
	EMPPartnerSessionID string         `json:"EMPPartnerSessionID,omitempty"`
	PartnerProductID    string         `json:"PartnerProductID,omitempty"`
	EvseID              string         `json:"EvseID"`
	Identification      Identification `json:"Identification"`
	ChargingStart       string         `json:"ChargingStart"`
	ChargingEnd         string         `json:"ChargingEnd"`
	SessionStart        string         `json:"SessionStart"`
	SessionEnd          string         `json:"SessionEnd"`
	MeterValueStart     *float64       `json:"MeterValueStart,omitempty"`
	MeterValueEnd       *float64       `json:"MeterValueEnd,omitempty"`
	MeterValueInBetween *MeterValues   `json:"MeterValueInBetween,omitempty"`
	ConsumedEnergy      float64        `json:"ConsumedEnergy"`
	HubOperatorID       string         `json:"HubOperatorID,omitempty"`
	HubProviderID       string         `json:"HubProviderID,omitempty"`
	// SignedMeteringValues and CalibrationLawVerificationInfo carry
	// metering that can be verified under calibration law (added in
	// v0.1.0 of the module).
	SignedMeteringValues           []SignedMeteringValue           `json:"SignedMeteringValues,omitempty"`
	CalibrationLawVerificationInfo *CalibrationLawVerificationInfo `json:"CalibrationLawVerificationInfo,omitempty"`
}

// SignedMeteringValue is one signed meter reading and the status it was
// taken at (Start, Progress or End).
type SignedMeteringValue struct {
	SignedMeteringValue string `json:"SignedMeteringValue,omitempty"`
	MeteringStatus      string `json:"MeteringStatus,omitempty"`
}

// Metering statuses of a SignedMeteringValue.
const (
	MeteringStart    = "Start"
	MeteringProgress = "Progress"
	MeteringEnd      = "End"
)

// CalibrationLawVerificationInfo tells how to verify signed readings.
type CalibrationLawVerificationInfo struct {
	CalibrationLawCertificateID                 string `json:"CalibrationLawCertificateID,omitempty"`
	PublicKey                                   string `json:"PublicKey,omitempty"`
	MeteringSignatureURL                        string `json:"MeteringSignatureUrl,omitempty"`
	MeteringSignatureEncodingFormat             string `json:"MeteringSignatureEncodingFormat,omitempty"`
	SignedMeteringValuesVerificationInstruction string `json:"SignedMeteringValuesVerificationInstruction,omitempty"`
}

// MeterValues are meter readings in kWh.
type MeterValues struct {
	MeterValues []float64 `json:"meterValues"`
}

// ChargingNotification is eRoamingChargingNotification (V1.1) of type
// Start, Progress, End or Error.
type ChargingNotification struct {
	Type                   string          `json:"Type"`
	SessionID              string          `json:"SessionID"`
	CPOPartnerSessionID    string          `json:"CPOPartnerSessionID,omitempty"`
	EMPPartnerSessionID    string          `json:"EMPPartnerSessionID,omitempty"`
	Identification         *Identification `json:"Identification,omitempty"`
	EvseID                 string          `json:"EvseID"`
	ChargingStart          string          `json:"ChargingStart,omitempty"`
	ChargingEnd            string          `json:"ChargingEnd,omitempty"`
	SessionStart           string          `json:"SessionStart,omitempty"`
	SessionEnd             string          `json:"SessionEnd,omitempty"`
	EventOccurred          string          `json:"EventOccurred,omitempty"`
	ChargingDuration       *int64          `json:"ChargingDuration,omitempty"`
	ConsumedEnergyProgress *float64        `json:"ConsumedEnergyProgress,omitempty"`
	ConsumedEnergy         *float64        `json:"ConsumedEnergy,omitempty"`
	MeterValueStart        *float64        `json:"MeterValueStart,omitempty"`
	MeterValueEnd          *float64        `json:"MeterValueEnd,omitempty"`
	OperatorID             string          `json:"OperatorID"`
	PartnerProductID       string          `json:"PartnerProductID,omitempty"`
	ErrorType              string          `json:"ErrorType,omitempty"`
	ErrorAdditionalInfo    string          `json:"ErrorAdditionalInfo,omitempty"`
	// MeterValueInBetween (Progress, End) and PenaltyTimeStart (End) were
	// added in v0.1.0 of the module.
	MeterValueInBetween *MeterValues `json:"MeterValueInBetween,omitempty"`
	PenaltyTimeStart    string       `json:"PenaltyTimeStart,omitempty"`
}

// Charging notification types and error types.
const (
	NotificationStart    = "Start"
	NotificationProgress = "Progress"
	NotificationEnd      = "End"
	NotificationError    = "Error"

	ErrorConnector = "ConnectorError"
	ErrorCritical  = "CriticalError"
)

// RemoteReservationStart is eRoamingAuthorizeRemoteReservationStart
// (V1.1), which Hubject sends a CPO for an EMP. Duration is in minutes.
type RemoteReservationStart struct {
	SessionID           string          `json:"SessionID"`
	CPOPartnerSessionID string          `json:"CPOPartnerSessionID,omitempty"`
	EMPPartnerSessionID string          `json:"EMPPartnerSessionID,omitempty"`
	ProviderID          string          `json:"ProviderID"`
	EvseID              string          `json:"EvseID"`
	Identification      *Identification `json:"Identification,omitempty"`
	PartnerProductID    string          `json:"PartnerProductID,omitempty"`
	Duration            int             `json:"Duration,omitempty"`
}

// RemoteReservationStop is eRoamingAuthorizeRemoteReservationStop (V1.1).
type RemoteReservationStop struct {
	SessionID           string `json:"SessionID"`
	CPOPartnerSessionID string `json:"CPOPartnerSessionID,omitempty"`
	EMPPartnerSessionID string `json:"EMPPartnerSessionID,omitempty"`
	ProviderID          string `json:"ProviderID"`
	EvseID              string `json:"EvseID"`
}

// PushPricingProductData is eRoamingPushPricingProductData (V1.0).
type PushPricingProductData struct {
	ActionType         string             `json:"ActionType"`
	PricingProductData PricingProductData `json:"PricingProductData"`
}

// PricingProductData is the operator's pricing products for a provider.
type PricingProductData struct {
	OperatorID                  string                     `json:"OperatorID"`
	OperatorName                string                     `json:"OperatorName,omitempty"`
	ProviderID                  string                     `json:"ProviderID"`
	PricingDefaultPrice         float64                    `json:"PricingDefaultPrice"`
	PricingDefaultPriceCurrency string                     `json:"PricingDefaultPriceCurrency"`
	PricingDefaultReferenceUnit string                     `json:"PricingDefaultReferenceUnit"`
	PricingProductDataRecords   []PricingProductDataRecord `json:"PricingProductDataRecords"`
}

// PricingProductDataRecord is one pricing product.
type PricingProductDataRecord struct {
	ProductID                   string                `json:"ProductID"`
	ReferenceUnit               string                `json:"ReferenceUnit"`
	ProductPriceCurrency        string                `json:"ProductPriceCurrency"`
	PricePerReferenceUnit       float64               `json:"PricePerReferenceUnit"`
	MaximumProductChargingPower float64               `json:"MaximumProductChargingPower"`
	IsValid24hours              bool                  `json:"IsValid24hours"`
	ProductAvailabilityTimes    []AvailabilityTime    `json:"ProductAvailabilityTimes"`
	AdditionalReferences        []AdditionalReference `json:"AdditionalReferences,omitempty"`
}

// AvailabilityTime is when a product applies.
type AvailabilityTime struct {
	Periods []Period `json:"Periods"`
	On      string   `json:"on"`
}

// AdditionalReference is a fee on top of the base price.
type AdditionalReference struct {
	AdditionalReference             string  `json:"AdditionalReference"`
	AdditionalReferenceUnit         string  `json:"AdditionalReferenceUnit"`
	PricePerAdditionalReferenceUnit float64 `json:"PricePerAdditionalReferenceUnit"`
}

// Reference units.
const (
	UnitHour   = "HOUR"
	UnitKWh    = "KILOWATT_HOUR"
	UnitMinute = "MINUTE"
)

// PushEVSEPricing is eRoamingPushEVSEPricing (V1.0).
type PushEVSEPricing struct {
	ActionType  string        `json:"ActionType"`
	EVSEPricing []EVSEPricing `json:"EVSEPricing"`
}

// EVSEPricing assigns pricing products to an EVSE for a provider.
type EVSEPricing struct {
	EvseID            string   `json:"EvseID"`
	ProviderID        string   `json:"ProviderID"`
	EvseIDProductList []string `json:"EvseIDProductList"`
}

// Identifier patterns (03 Data Types: OperatorIDType, EvseIDType,
// ProviderIDType).
var (
	operatorIDRe = regexp.MustCompile(`^(([A-Za-z]{2}\*?[A-Za-z0-9]{3})|(\+?[0-9]{1,3}\*[0-9]{3}))$`)
	evseIDRe     = regexp.MustCompile(`^(([A-Za-z]{2}\*?[A-Za-z0-9]{3}\*?E[A-Za-z0-9\*]{1,30})|(\+?[0-9]{1,3}\*[0-9]{3}\*[0-9\*]{1,32}))$`)
	providerIDRe = regexp.MustCompile(`^([A-Za-z]{2}\-?[A-Za-z0-9]{3}|[A-Za-z]{2}[\*|-]?[A-Za-z0-9]{3})$`)
	hotlineRe    = regexp.MustCompile(`^\+[0-9]{5,15}$`)
)

// ValidOperatorID, ValidEvseID and ValidProviderID check an identifier's
// form.
func ValidOperatorID(s string) bool { return operatorIDRe.MatchString(s) }
func ValidEvseID(s string) bool     { return evseIDRe.MatchString(s) }
func ValidProviderID(s string) bool { return providerIDRe.MatchString(s) }

// ValidHotline checks a hotline phone number's form (+ and 5 to 15
// digits).
func ValidHotline(s string) bool { return hotlineRe.MatchString(s) }

// NormID is an identifier without its "*" and "-" separators, in upper
// case, so identifiers written with and without them compare equal.
func NormID(s string) string {
	return strings.ToUpper(strings.NewReplacer("*", "", "-", "", " ", "").Replace(s))
}
