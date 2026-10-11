// Copyright 2026 Abhinav Gautam
// SPDX-License-Identifier: Apache-2.0

package oscp_test

import (
	"encoding/json"
	"fmt"

	"github.com/cB-Sigma/evroaming/oscp"
)

// A capacity provider checks a forecast against the OCA schema and the
// rules the schema lacks before sending it, and that its role may send it.
func Example() {
	f := oscp.GroupCapacityForecast{GroupID: "site-1", Type: oscp.TypeConsumption, ForecastedBlocks: []oscp.ForecastedBlock{
		{Capacity: 32, Phase: "ALL", Unit: "A", StartTime: "2026-10-06T10:00:00Z", EndTime: "2026-10-06T10:15:00Z"}}}
	b, _ := json.Marshal(f)
	fmt.Println(oscp.Validate(oscp.SchemaUpdateGroupCapacityForecast, b))
	fmt.Println(oscp.CanSend(oscp.RoleCP, oscp.RoleFP, oscp.EPUpdateGroupCapacityForecast))
	// Output:
	// <nil>
	// true
}
