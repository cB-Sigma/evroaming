package oscp

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// specDir is SigmaM2M's copy of the schemas, when this module is inside
// the SigmaM2M repository.
var specDir = filepath.FromSlash("../../../spec/data/oscp/2.0/schemas")

// The embedded schemas are the unmodified OCA files: byte-identical to
// the digests pinned in schemas/SHA256SUMS (and to SigmaM2M's
// spec/data/oscp/2.0/schemas when that is next to this module), all 26 of
// schemas.zip, and every one loads.
//
// Covers: SIG-IPE-2200
func TestSchemasMatchSpecData(t *testing.T) {
	f, err := os.Open(filepath.Join("schemas", "SHA256SUMS"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sums := map[string]string{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		parts := strings.Fields(sc.Text())
		if len(parts) == 2 {
			sums[filepath.Base(parts[1])] = parts[0]
		}
	}
	files, err := SchemaFiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 26 || len(sums) != 26 {
		t.Fatalf("%d embedded schemas, %d digests; schemas.zip has 26", len(files), len(sums))
	}
	for name, b := range files {
		h := sha256.Sum256(b)
		if hex.EncodeToString(h[:]) != sums[name] {
			t.Errorf("%s differs from schemas/SHA256SUMS", name)
		}
		// Inside the SigmaM2M repository the same files are also kept in
		// spec/data/oscp/2.0/schemas.
		if spec, err := os.ReadFile(filepath.Join(specDir, name)); err == nil && string(spec) != string(b) {
			t.Errorf("%s differs from spec/data/oscp/2.0/schemas", name)
		}
	}
	for _, name := range schemas.Names() {
		if !strings.HasSuffix(name, ".json") {
			t.Errorf("%s", name)
		}
	}
}

// Endpoints and directions of 4.2: who serves and who sends each message.
//
// Covers: SIG-IPE-2207
func TestEndpointDirections(t *testing.T) {
	yes := [][3]string{
		{RoleCP, RoleFP, EPRegister}, {RoleCO, RoleFP, EPRegister}, {RoleFP, RoleCP, EPRegister}, {RoleFP, RoleCO, EPRegister},
		{RoleCP, RoleFP, EPHandshake}, {RoleFP, RoleCO, EPHandshake},
		{RoleFP, RoleCP, EPHandshakeAcknowledge}, {RoleCO, RoleFP, EPHandshakeAcknowledge},
		{RoleCP, RoleFP, EPHeartbeat}, {RoleCO, RoleFP, EPHeartbeat}, {RoleFP, RoleCP, EPHeartbeat}, {RoleFP, RoleCO, EPHeartbeat},
		{RoleCP, RoleFP, EPUpdateGroupCapacityForecast}, {RoleFP, RoleCO, EPUpdateGroupCapacityForecast}, {RoleCO, RoleFP, EPUpdateGroupCapacityForecast},
		{RoleFP, RoleCP, EPAdjustGroupCapacityForecast}, {RoleFP, RoleCP, EPGroupCapacityComplianceError},
		{RoleFP, RoleCP, EPUpdateGroupMeasurements}, {RoleFP, RoleCO, EPUpdateAssetMeasurements},
	}
	for _, y := range yes {
		if !CanSend(y[0], y[1], y[2]) {
			t.Errorf("%s → %s %s refused", y[0], y[1], y[2])
		}
	}
	no := [][3]string{{RoleFP, RoleCP, EPHandshake}, {RoleCP, RoleFP, EPUpdateGroupMeasurements}, {RoleCO, RoleFP, EPUpdateAssetMeasurements}, {RoleCP, RoleCO, EPRegister}, {RoleCP, RoleFP, EPAdjustGroupCapacityForecast}}
	for _, n := range no {
		if CanSend(n[0], n[1], n[2]) {
			t.Errorf("%s → %s %s allowed", n[0], n[1], n[2])
		}
	}
}
