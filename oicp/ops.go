// Copyright 2026 Abhinav Gautam
// SPDX-License-Identifier: Apache-2.0

package oicp

import "context"

// One method per operation a CPO sends to Hubject; op is the CPO's
// OperatorID (it goes into the path). Each wants Result true (or, for
// the authorisations, any decoded answer) and returns a *HubjectError
// otherwise, as Post and PostAck do.

// PushEvseData sends eRoamingPushEvseData (V2.3).
func (c *Client) PushEvseData(ctx context.Context, m PushEvseData) (Acknowledgement, error) {
	return c.PostAck(ctx, PathEvseData(m.OperatorEvseData.OperatorID), m)
}

// PushEvseStatus sends eRoamingPushEvseStatus (V2.1).
func (c *Client) PushEvseStatus(ctx context.Context, m PushEvseStatus) (Acknowledgement, error) {
	return c.PostAck(ctx, PathEvseStatus(m.OperatorEvseStatus.OperatorID), m)
}

// PushPricingProductData sends eRoamingPushPricingProductData (V1.0).
func (c *Client) PushPricingProductData(ctx context.Context, m PushPricingProductData) (Acknowledgement, error) {
	return c.PostAck(ctx, PathPricing(m.PricingProductData.OperatorID), m)
}

// PushEVSEPricing sends eRoamingPushEVSEPricing (V1.0) for operator op.
func (c *Client) PushEVSEPricing(ctx context.Context, op string, m PushEVSEPricing) (Acknowledgement, error) {
	return c.PostAck(ctx, PathEVSEPricing(op), m)
}

// AuthorizeStart sends eRoamingAuthorizeStart (V2.1) and returns
// Hubject's eRoamingAuthorizationStart, whatever its status.
func (c *Client) AuthorizeStart(ctx context.Context, m AuthorizeStart) (Authorization, error) {
	var a Authorization
	err := c.Post(ctx, PathAuthStart(m.OperatorID), m, &a)
	return a, err
}

// AuthorizeStop sends eRoamingAuthorizeStop (V2.1) and returns Hubject's
// eRoamingAuthorizationStop.
func (c *Client) AuthorizeStop(ctx context.Context, m AuthorizeStop) (Authorization, error) {
	var a Authorization
	err := c.Post(ctx, PathAuthStop(m.OperatorID), m, &a)
	return a, err
}

// SendChargeDetailRecord sends eRoamingChargeDetailRecord (V2.2) for
// operator op.
func (c *Client) SendChargeDetailRecord(ctx context.Context, op string, m ChargeDetailRecord) (Acknowledgement, error) {
	return c.PostAck(ctx, PathCDR(op), m)
}

// SendChargingNotification sends eRoamingChargingNotification (V1.1) of
// any type.
func (c *Client) SendChargingNotification(ctx context.Context, m ChargingNotification) (Acknowledgement, error) {
	return c.PostAck(ctx, PathNotifications, m)
}
