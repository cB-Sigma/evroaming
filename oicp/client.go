package oicp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// PathEvseData is eRoamingPushEvseData. The Path functions give the CPO
// operations' paths at Hubject, below its base URL (the CPO OpenAPI
// description's paths); op is the CPO's OperatorID.
func PathEvseData(op string) string {
	return "/evsepush/v23/operators/" + url.PathEscape(op) + "/data-records"
}

// PathEvseStatus is eRoamingPushEvseStatus.
func PathEvseStatus(op string) string {
	return "/evsepush/v21/operators/" + url.PathEscape(op) + "/status-records"
}

// PathAuthStart is eRoamingAuthorizeStart.
func PathAuthStart(op string) string {
	return "/charging/v21/operators/" + url.PathEscape(op) + "/authorize/start"
}

// PathAuthStop is eRoamingAuthorizeStop.
func PathAuthStop(op string) string {
	return "/charging/v21/operators/" + url.PathEscape(op) + "/authorize/stop"
}

// PathCDR is eRoamingChargeDetailRecord.
func PathCDR(op string) string {
	return "/cdrmgmt/v22/operators/" + url.PathEscape(op) + "/charge-detail-record"
}

// PathPricing is eRoamingPushPricingProductData.
func PathPricing(op string) string {
	return "/dynamicpricing/v10/operators/" + url.PathEscape(op) + "/pricing-products"
}

// PathEVSEPricing is eRoamingPushEVSEPricing.
func PathEVSEPricing(op string) string {
	return "/dynamicpricing/v10/operators/" + url.PathEscape(op) + "/evse-pricing"
}

// PathNotifications is eRoamingChargingNotifications.
const PathNotifications = "/notificationmgmt/v11/charging-notifications"

// HubjectError is a negative answer from Hubject: an HTTP status other
// than 200, or Result false with a status code.
type HubjectError struct {
	HTTP int
	Code string
	Text string
}

func (e *HubjectError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("Hubject answered %s: %s", e.Code, e.Text)
	}
	return fmt.Sprintf("Hubject answered HTTP %d: %s", e.HTTP, e.Text)
}

// Retryable says whether sending again may succeed (Hubject's system
// codes, or no answer at all).
func Retryable(err error) bool {
	he, ok := err.(*HubjectError)
	if !ok {
		return true
	}
	if he.HTTP >= 500 || he.HTTP == http.StatusTooManyRequests {
		return true
	}
	switch he.Code {
	case CodeHubjectSystem, CodeHubjectDatabase, CodeDataTransaction, CodeSystemError, CodePartnerNotFound:
		return true
	}
	return false
}

// Client posts OICP messages to Hubject.
type Client struct {
	// BaseURL is Hubject's base URL (the paths are appended).
	BaseURL string
	// HTTP is the client, with the CPO's client certificate for mutual
	// TLS (nil: http.DefaultClient).
	HTTP *http.Client
	// Link, when set, is told after each call whether Hubject answered
	// usably (ok) or what went wrong.
	Link func(ok bool, msg string)
}

func (c *Client) link(ok bool, msg string) {
	if c.Link != nil {
		c.Link(ok, msg)
	}
}

// Post sends body to Hubject at path and decodes its answer into out.
func (c *Client) Post(ctx context.Context, path string, body, out any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+path, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json")
	hc := c.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}
	resp, err := hc.Do(req)
	if err != nil {
		c.link(false, err.Error())
		return fmt.Errorf("Hubject is not reachable: %w", err)
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode != http.StatusOK {
		he := &HubjectError{HTTP: resp.StatusCode, Text: strings.TrimSpace(truncate(string(rb), 300))}
		// An answer of another status may still carry a StatusCode.
		var ack Acknowledgement
		if json.Unmarshal(rb, &ack) == nil && ack.StatusCode.Code != "" {
			he.Code, he.Text = ack.StatusCode.Code, StatusText(ack.StatusCode)
		}
		c.link(false, he.Error())
		return he
	}
	if err := json.Unmarshal(rb, out); err != nil {
		c.link(false, "Hubject's answer is not JSON")
		return fmt.Errorf("Hubject's answer is not the expected JSON: %w", err)
	}
	c.link(true, "")
	return nil
}

// PostAck sends body and wants an acknowledgement with Result true; a
// false Result is a *HubjectError.
func (c *Client) PostAck(ctx context.Context, path string, body any) (Acknowledgement, error) {
	var ack Acknowledgement
	if err := c.Post(ctx, path, body, &ack); err != nil {
		return ack, err
	}
	if !ack.Result {
		he := &HubjectError{HTTP: http.StatusOK, Code: ack.StatusCode.Code, Text: StatusText(ack.StatusCode)}
		c.link(false, he.Error())
		return ack, he
	}
	return ack, nil
}

// StatusText is a status code's description and additional information,
// at most 300 bytes.
func StatusText(s StatusCode) string {
	t := s.Description
	if s.AdditionalInfo != "" {
		if t != "" {
			t += ": "
		}
		t += s.AdditionalInfo
	}
	return truncate(t, 300)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
