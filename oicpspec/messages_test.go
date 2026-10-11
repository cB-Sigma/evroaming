// Copyright 2026 Abhinav Gautam
// SPDX-License-Identifier: Apache-2.0

package oicpspec

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/cB-Sigma/evroaming/oicp"
)

// Demo values in the identifier forms the description asks for.
const (
	op       = "DE*ABC"
	evse     = "DE*ABC*E1001"
	provider = "DE-DCB"
	session  = "b2688855-7f00-0002-6d8e-48d883f6abb6"
	uid      = "1234ABCD"
	evco     = "DE-DCB-C12345678-X"
	ts       = "2026-10-11T08:30:00Z"
	ts2      = "2026-10-11T09:45:00Z"
)

func ptr[T any](v T) *T { return &v }

func idents() map[string]oicp.Identification {
	return map[string]oicp.Identification{
		"mifare": {RFIDMifareFamilyIdentification: &oicp.UIDIdentification{UID: uid}},
		"rfid": {RFIDIdentification: &oicp.RFIDIdentification{UID: uid, EvcoID: evco, RFID: "mifareCls",
			PrintedNumber: "9876655", ExpiryDate: "2027-01-01T00:00:00Z"}},
		"qr": {QRCodeIdentification: &oicp.QRCodeIdentification{EvcoID: evco,
			HashedPIN: json.RawMessage(`{"Value":"a5ghdhf73hkk","Function":"Bcrypt","LegacyHashData":{"Function":"MD5","Salt":"s4lt","Value":"v4lue"}}`)}},
		"pnc":    {PlugAndChargeIdentification: &oicp.EvcoIdentification{EvcoID: evco}},
		"remote": {RemoteIdentification: &oicp.EvcoIdentification{EvcoID: evco}},
	}
}

func evseRecord() oicp.EvseDataRecord {
	return oicp.EvseDataRecord{
		EvseID: evse, ChargingPoolID: "DE*ABC*P1001", ChargingStationID: "station-1",
		ChargingStationNames: []oicp.InfoText{{Lang: "en", Value: "Demo Street 1"}},
		HardwareManufacturer: "Demo", ChargingStationImage: "https://example.com/station.png", SubOperatorName: "Demo Sub",
		Address: oicp.Address{Country: "DEU", City: "Berlin", Street: "Demo Street", PostalCode: "10115", HouseNum: "1",
			Floor: "0", Region: "BE", TimeZone: "UTC+01:00", ParkingFacility: ptr(true), ParkingSpot: "12"},
		GeoCoordinates:                   oicp.GeoCoordinates{DecimalDegree: &oicp.DecimalDegree{Longitude: "13.404954", Latitude: "52.520008"}},
		Plugs:                            []string{"Type 2 Outlet", "CCS Combo 2 Plug (Cable Attached)"},
		DynamicPowerLevel:                ptr(true),
		ChargingFacilities:               []oicp.ChargingFacility{{PowerType: oicp.PowerAC3Phase, Voltage: 400, Amperage: 32, Power: 22, ChargingModes: []string{"Mode_3"}}},
		RenewableEnergy:                  true,
		EnergySource:                     []oicp.EnergySource{{Energy: "Solar", Percentage: ptr(60)}, {Energy: "Wind", Percentage: ptr(40)}},
		EnvironmentalImpact:              &oicp.EnvironmentalImpact{CO2Emission: ptr(0.0), NuclearWaste: ptr(0.0)},
		CalibrationLawDataAvailability:   oicp.CalibrationLocal,
		AuthenticationModes:              []string{"NFC RFID Classic", "REMOTE"},
		MaxCapacity:                      ptr(22),
		PaymentOptions:                   []string{"Contract"},
		ValueAddedServices:               []string{"Reservation"},
		Accessibility:                    oicp.AccessibilityFree,
		AccessibilityLocation:            "ParkingLot",
		HotlinePhoneNumber:               "+4930123456",
		AdditionalInfo:                   []oicp.InfoText{{Lang: "en", Value: "Demo"}},
		ChargingStationLocationReference: []oicp.InfoText{{Lang: "en", Value: "Level 0"}},
		GeoChargingPointEntrance:         &oicp.GeoCoordinates{DegreeMinuteSeconds: &oicp.DegreeMinuteSeconds{Longitude: "13°24'17.83''", Latitude: "52°31'12.03''"}},
		IsOpen24Hours:                    false,
		OpeningTimes:                     []oicp.OpeningTime{{Period: []oicp.Period{{Begin: "08:00", End: "20:00"}}, On: "Workdays"}},
		HubOperatorID:                    "DE*HUB",
		ClearinghouseID:                  "DE*CH1",
		IsHubjectCompatible:              true,
		DynamicInfoAvailable:             oicp.DynamicInfoAuto,
	}
}

// requests are the messages the package builds, by the schema file of
// the operation's request body.
func requests() map[string][]any {
	m := map[string][]any{}
	add := func(s string, v any) { m[s] = append(m[s], v) }
	add("eRoamingPushEvseData", oicp.PushEvseData{ActionType: oicp.ActionFullLoad,
		OperatorEvseData: oicp.OperatorEvseData{OperatorID: op, OperatorName: "Demo CPO", EvseDataRecord: []oicp.EvseDataRecord{evseRecord()}}})
	g := evseRecord()
	g.GeoCoordinates = oicp.GeoCoordinates{Google: &struct {
		Coordinates string `json:"Coordinates"`
	}{Coordinates: "52.520008 13.404954"}}
	add("eRoamingPushEvseData", oicp.PushEvseData{ActionType: oicp.ActionInsert,
		OperatorEvseData: oicp.OperatorEvseData{OperatorID: op, OperatorName: "Demo CPO", EvseDataRecord: []oicp.EvseDataRecord{g}}})
	for _, st := range []string{oicp.EvseAvailable, oicp.EvseReserved, oicp.EvseOccupied, oicp.EvseOutOfService, oicp.EvseNotFound, oicp.EvseUnknown} {
		add("eRoamingPushEvseStatus", oicp.PushEvseStatus{ActionType: oicp.ActionUpdate,
			OperatorEvseStatus: oicp.OperatorEvseStatus{OperatorID: op, OperatorName: "Demo CPO", EvseStatusRecord: []oicp.EvseStatusRecord{{EvseID: evse, EvseStatus: st}}}})
	}
	add("eRoamingPushPricingProductData", oicp.PushPricingProductData{ActionType: oicp.ActionFullLoad, PricingProductData: oicp.PricingProductData{
		OperatorID: op, OperatorName: "Demo CPO", ProviderID: provider, PricingDefaultPrice: 0.39, PricingDefaultPriceCurrency: "EUR",
		PricingDefaultReferenceUnit: oicp.UnitKWh,
		PricingProductDataRecords: []oicp.PricingProductDataRecord{{ProductID: "AC 1", ReferenceUnit: oicp.UnitKWh, ProductPriceCurrency: "EUR",
			PricePerReferenceUnit: 0.39, MaximumProductChargingPower: 22, IsValid24hours: true,
			ProductAvailabilityTimes: []oicp.AvailabilityTime{{Periods: []oicp.Period{{Begin: "00:00", End: "23:59"}}, On: "Everyday"}},
			AdditionalReferences:     []oicp.AdditionalReference{{AdditionalReference: "START FEE", AdditionalReferenceUnit: oicp.UnitHour, PricePerAdditionalReferenceUnit: 1}}}}}})
	add("eRoamingPushEVSEPricing", oicp.PushEVSEPricing{ActionType: oicp.ActionFullLoad,
		EVSEPricing: []oicp.EVSEPricing{{EvseID: evse, ProviderID: provider, EvseIDProductList: []string{"AC 1"}}, {EvseID: evse, ProviderID: "*", EvseIDProductList: []string{"Standard"}}}})
	for _, id := range idents() {
		add("eRoamingAuthorizeStart", oicp.AuthorizeStart{SessionID: session, CPOPartnerSessionID: "cpo-1", OperatorID: op, EvseID: evse, Identification: id, PartnerProductID: "AC 1"})
		add("eRoamingAuthorizeStop", oicp.AuthorizeStop{SessionID: session, CPOPartnerSessionID: "cpo-1", OperatorID: op, EvseID: evse, Identification: id})
		add("eRoamingChargeDetailRecord", oicp.ChargeDetailRecord{SessionID: session, CPOPartnerSessionID: "cpo-1", EMPPartnerSessionID: "emp-1",
			PartnerProductID: "AC 1", EvseID: evse, Identification: id, ChargingStart: ts, ChargingEnd: ts2, SessionStart: ts, SessionEnd: ts2,
			MeterValueStart: ptr(100.0), MeterValueEnd: ptr(112.5), MeterValueInBetween: &oicp.MeterValues{MeterValues: []float64{100, 106, 112.5}},
			ConsumedEnergy: 12.5, HubOperatorID: "DE*HUB", HubProviderID: provider,
			SignedMeteringValues: []oicp.SignedMeteringValue{{SignedMeteringValue: "c2lnbmVkLXN0YXJ0", MeteringStatus: oicp.MeteringStart},
				{SignedMeteringValue: "c2lnbmVkLWVuZA", MeteringStatus: oicp.MeteringEnd}},
			CalibrationLawVerificationInfo: &oicp.CalibrationLawVerificationInfo{CalibrationLawCertificateID: "CERT-1", PublicKey: "demo-key",
				MeteringSignatureURL: "https://example.com/verify", MeteringSignatureEncodingFormat: "UTF-8", SignedMeteringValuesVerificationInstruction: "see the URL"}})
	}
	ri := idents()["remote"]
	add("eRoamingChargingNotificationStart", oicp.ChargingNotification{Type: oicp.NotificationStart, SessionID: session, CPOPartnerSessionID: "cpo-1",
		Identification: &ri, EvseID: evse, ChargingStart: ts, SessionStart: ts, MeterValueStart: ptr(100.0), OperatorID: op, PartnerProductID: "AC 1"})
	add("eRoamingChargingNotificationProgress", oicp.ChargingNotification{Type: oicp.NotificationProgress, SessionID: session, Identification: &ri,
		EvseID: evse, ChargingStart: ts, EventOccurred: ts2, ChargingDuration: ptr(int64(4500000)), SessionStart: ts,
		ConsumedEnergyProgress: ptr(6.0), MeterValueStart: ptr(100.0), MeterValueInBetween: &oicp.MeterValues{MeterValues: []float64{103, 106}},
		OperatorID: op, PartnerProductID: "AC 1"})
	add("eRoamingChargingNotificationEnd", oicp.ChargingNotification{Type: oicp.NotificationEnd, SessionID: session, Identification: &ri,
		EvseID: evse, ChargingStart: ts, ChargingEnd: ts2, SessionStart: ts, SessionEnd: ts2, ConsumedEnergy: ptr(12.5),
		MeterValueStart: ptr(100.0), MeterValueEnd: ptr(112.5), MeterValueInBetween: &oicp.MeterValues{MeterValues: []float64{106}},
		OperatorID: op, PartnerProductID: "AC 1", PenaltyTimeStart: ts2})
	add("eRoamingChargingNotificationError", oicp.ChargingNotification{Type: oicp.NotificationError, SessionID: session, Identification: &ri,
		EvseID: evse, OperatorID: op, ErrorType: oicp.ErrorConnector, ErrorAdditionalInfo: "connector jammed"})
	add("eRoamingAuthorizeRemoteStart", oicp.RemoteStart{SessionID: session, EMPPartnerSessionID: "emp-1", ProviderID: provider, EvseID: evse,
		Identification: ri, PartnerProductID: "AC 1"})
	add("eRoamingAuthorizeRemoteStop", oicp.RemoteStop{SessionID: session, EMPPartnerSessionID: "emp-1", ProviderID: provider, EvseID: evse})
	add("eRoamingAuthorizeRemoteReservationStart", oicp.RemoteReservationStart{SessionID: session, ProviderID: provider, EvseID: evse,
		Identification: &ri, PartnerProductID: "Reservation", Duration: 15})
	add("eRoamingAuthorizeRemoteReservationStop", oicp.RemoteReservationStop{SessionID: session, ProviderID: provider, EvseID: evse})
	return m
}

// Every request the package builds is valid against the operation's
// request schema.
func TestRequestsAgainstOpenAPI(t *testing.T) {
	s := spec(t)
	n := 0
	for name, msgs := range requests() {
		sch := s.schema(t, name)
		for i, m := range msgs {
			n++
			if err := validate(sch, m); err != nil {
				t.Errorf("%s #%d: %v", name, i, err)
			}
		}
	}
	t.Logf("%d requests checked against %d request schemas at %s", n, len(requests()), Commit)
	// The check itself refuses what it should: a bad EvseID, a missing
	// required member, an unknown enumeration value.
	sch := s.schema(t, "eRoamingPushEvseStatus")
	for _, bad := range []any{
		oicp.PushEvseStatus{ActionType: oicp.ActionUpdate, OperatorEvseStatus: oicp.OperatorEvseStatus{OperatorID: op, EvseStatusRecord: []oicp.EvseStatusRecord{{EvseID: "not an evse", EvseStatus: oicp.EvseAvailable}}}},
		map[string]any{"ActionType": "update"},
		oicp.PushEvseStatus{ActionType: "replace", OperatorEvseStatus: oicp.OperatorEvseStatus{OperatorID: op, EvseStatusRecord: []oicp.EvseStatusRecord{{EvseID: evse, EvseStatus: oicp.EvseAvailable}}}},
	} {
		if validate(sch, bad) == nil {
			t.Errorf("accepted %+v", bad)
		}
	}
}

// The answers a CPO gives (oicp.Handler's acknowledgements) and the
// answers the package reads (acknowledgements and authorisations) are
// valid against the response schemas.
func TestAnswersAgainstOpenAPI(t *testing.T) {
	s := spec(t)
	ack := s.schema(t, "eRoamingAcknowledgment")
	h := &oicp.Handler{
		RemoteStart: func(_ context.Context, m oicp.RemoteStart) oicp.Acknowledgement {
			return oicp.Acknowledgement{Result: true, StatusCode: oicp.StatusCode{Code: oicp.CodeSuccess}, CPOPartnerSessionID: "cpo-1", EMPPartnerSessionID: m.EMPPartnerSessionID}
		},
		RemoteStop: func(context.Context, oicp.RemoteStop) oicp.Acknowledgement {
			return oicp.NewAck(false, oicp.CodeSessionInvalid, "no such session", "")
		},
		ReservationStart: func(context.Context, oicp.RemoteReservationStart) oicp.Acknowledgement {
			return oicp.NewAck(true, oicp.CodeSuccess, "", "")
		},
	}
	srv := httptest.NewServer(h)
	defer srv.Close()
	reqs := requests()
	for path, body := range map[string]any{
		oicp.PathRemoteStart(provider):      reqs["eRoamingAuthorizeRemoteStart"][0],
		oicp.PathRemoteStop(provider):       reqs["eRoamingAuthorizeRemoteStop"][0],
		oicp.PathReservationStart(provider): reqs["eRoamingAuthorizeRemoteReservationStart"][0],
		oicp.PathReservationStop(provider):  reqs["eRoamingAuthorizeRemoteReservationStop"][0],
	} {
		b, _ := json.Marshal(body)
		resp, err := http.Post(srv.URL+path, "application/json", strings.NewReader(string(b)))
		if err != nil {
			t.Fatal(err)
		}
		var got any
		_ = json.NewDecoder(resp.Body).Decode(&got)
		resp.Body.Close()
		if err := validate(ack, got); err != nil {
			t.Errorf("%s: %v", path, err)
		}
	}
	for _, a := range []oicp.Acknowledgement{oicp.NewAck(true, oicp.CodeSuccess, "Success", session), oicp.NewAck(false, oicp.CodeDataError, "Data error", "")} {
		if err := validate(ack, a); err != nil {
			t.Error(err)
		}
	}
	start, stop := s.schema(t, "eRoamingAuthorizationStart"), s.schema(t, "eRoamingAuthorizationStop")
	for _, a := range []oicp.Authorization{
		{SessionID: session, ProviderID: provider, AuthorizationStatus: oicp.Authorized, StatusCode: oicp.StatusCode{Code: oicp.CodeSuccess}},
		{AuthorizationStatus: oicp.NotAuthorized, StatusCode: oicp.StatusCode{Code: oicp.CodeNoPositiveAuth, Description: "No positive authentication response"}},
	} {
		if err := validate(start, a); err != nil {
			t.Error(err)
		}
		a.AuthorizationStopIdentifications = []oicp.Identification{idents()["mifare"]}
		if err := validate(stop, a); err != nil {
			t.Error(err)
		}
	}
}

// examples pairs each schema with the Go type that reads it.
var examples = map[string]any{
	"eRoamingPushEvseData":                    oicp.PushEvseData{},
	"eRoamingPushEvseStatus":                  oicp.PushEvseStatus{},
	"eRoamingPushPricingProductData":          oicp.PushPricingProductData{},
	"eRoamingPushEVSEPricing":                 oicp.PushEVSEPricing{},
	"eRoamingAuthorizeStart":                  oicp.AuthorizeStart{},
	"eRoamingAuthorizeStop":                   oicp.AuthorizeStop{},
	"eRoamingChargeDetailRecord":              oicp.ChargeDetailRecord{},
	"eRoamingChargingNotificationStart":       oicp.ChargingNotification{},
	"eRoamingChargingNotificationProgress":    oicp.ChargingNotification{},
	"eRoamingChargingNotificationEnd":         oicp.ChargingNotification{},
	"eRoamingChargingNotificationError":       oicp.ChargingNotification{},
	"eRoamingAuthorizeRemoteStart":            oicp.RemoteStart{},
	"eRoamingAuthorizeRemoteStop":             oicp.RemoteStop{},
	"eRoamingAuthorizeRemoteReservationStart": oicp.RemoteReservationStart{},
	"eRoamingAuthorizeRemoteReservationStop":  oicp.RemoteReservationStop{},
	"eRoamingAcknowledgment":                  oicp.Acknowledgement{},
	"eRoamingAuthorizationStart":              oicp.Authorization{},
	"eRoamingAuthorizationStop":               oicp.Authorization{},
	"EvseDataRecord":                          oicp.EvseDataRecord{},
	"Identification":                          oicp.Identification{},
}

// Every member of the description's examples survives a round trip
// through the package's types: no field of the description is dropped
// when the package reads a message.
func TestExamplesRoundTrip(t *testing.T) {
	s := spec(t)
	for name, zero := range examples {
		d, err := s.doc("schemas/" + name + ".yaml")
		if err != nil {
			t.Fatal(err)
		}
		ex := d.(map[string]any)["example"]
		if ex == nil {
			t.Logf("%s has no example", name)
			continue
		}
		in, _ := json.Marshal(ex)
		v := reflect.New(reflect.TypeOf(zero))
		if err := json.Unmarshal(in, v.Interface()); err != nil {
			t.Errorf("%s: the example does not decode: %v", name, err)
			continue
		}
		out, _ := json.Marshal(v.Interface())
		var a, b any
		_ = json.Unmarshal(in, &a)
		_ = json.Unmarshal(out, &b)
		want, got := map[string]bool{}, map[string]bool{}
		keys("", a, want)
		keys("", b, got)
		fold := map[string]string{}
		for k := range got {
			fold[strings.ToLower(k)] = k
		}
		for _, k := range sorted(want) {
			if got[k] {
				continue
			}
			if g, ok := fold[strings.ToLower(k)]; ok {
				// The example spells a member differently from the
				// schema's property; the package reads both (Go's JSON
				// names match without case) and writes the schema's.
				t.Logf("%s: the example's %s is the schema's %s", name, k, g)
				continue
			}
			t.Errorf("%s: %s of the example is dropped", name, k)
		}
	}
}

// Every operation path of the description is one the package builds or
// serves, and the package has no other.
func TestOperationPaths(t *testing.T) {
	s := spec(t)
	d, err := s.doc("openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	paths, _ := d.(map[string]any)["paths"].(map[string]any)
	ours := map[string]bool{
		oicp.PathAuthStart("{operatorID}"): true, oicp.PathAuthStop("{operatorID}"): true,
		oicp.PathRemoteStart("{providerID}"): true, oicp.PathRemoteStop("{providerID}"): true,
		oicp.PathCDR("{operatorID}"):              true,
		oicp.PathReservationStart("{providerID}"): true, oicp.PathReservationStop("{providerID}"): true,
		oicp.PathEvseData("{operatorID}"): true, oicp.PathEvseStatus("{operatorID}"): true,
		oicp.PathPricing("{operatorID}"): true, oicp.PathEVSEPricing("{operatorID}"): true,
		oicp.PathNotifications: true,
	}
	unesc := strings.NewReplacer("%7B", "{", "%7D", "}")
	norm := map[string]bool{}
	for p := range ours {
		norm[unesc.Replace(p)] = true
	}
	for p := range paths {
		if !norm[p] {
			t.Errorf("the description's %s is not covered", p)
		}
		delete(norm, p)
	}
	for p := range norm {
		t.Errorf("%s is not in the description", p)
	}
	t.Logf("%d operation paths at %s", len(paths), Commit)
}
