// Copyright 2026 Abhinav Gautam
// SPDX-License-Identifier: Apache-2.0

// Package oicpspec checks the oicp package against Hubject's OpenAPI
// description of the OICP 2.3 CPO side (github.com/hubject/oicp-cpo-2.3-api-doc),
// read at a pinned commit. Hubject's files are CC BY-SA 4.0 and are not
// copied into this repository: the tests fetch them (SIGMA_EVROAMING_OICP_SPEC=1)
// or read a checkout (SIGMA_EVROAMING_OICP_DIR=<dir>), and are skipped
// otherwise.
package oicpspec

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"
)

// Commit is the commit of hubject/oicp-cpo-2.3-api-doc the tests read
// (2026-05-08, "Update eRoamingChargeDetailRecord.yaml (#21)").
const Commit = "f5a8e2f2eac29c76ad840f855181e54b11394e68"

// Base is the raw URL of the description's src directory at Commit.
const Base = "https://raw.githubusercontent.com/hubject/oicp-cpo-2.3-api-doc/" + Commit + "/src/"

type source struct {
	dir string // a checkout of the repository, or "" to fetch
	mu  sync.Mutex
	raw map[string][]byte
}

var (
	srcOnce sync.Once
	src     *source
)

// spec returns the source of the description, or skips the test.
func spec(t *testing.T) *source {
	t.Helper()
	dir := os.Getenv("SIGMA_EVROAMING_OICP_DIR")
	if dir == "" && os.Getenv("SIGMA_EVROAMING_OICP_SPEC") != "1" {
		t.Skip("set SIGMA_EVROAMING_OICP_SPEC=1 (fetch at the pinned commit) or SIGMA_EVROAMING_OICP_DIR=<checkout>")
	}
	srcOnce.Do(func() { src = &source{dir: dir, raw: map[string][]byte{}} })
	return src
}

// file returns a file of src/ (rel is relative to it).
func (s *source) file(rel string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if b, ok := s.raw[rel]; ok {
		return b, nil
	}
	var b []byte
	var err error
	if s.dir != "" {
		b, err = os.ReadFile(filepath.Join(s.dir, "src", filepath.FromSlash(rel)))
	} else {
		b, err = fetch(Base + rel)
	}
	if err != nil {
		return nil, err
	}
	s.raw[rel] = b
	return b, nil
}

func fetch(u string) ([]byte, error) {
	c := &http.Client{Timeout: 30 * time.Second}
	var last error
	for i := 0; i < 3; i++ {
		resp, err := c.Get(u)
		if err != nil {
			last = err
			continue
		}
		b, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			last = fmt.Errorf("GET %s: %s", u, resp.Status)
			continue
		}
		return b, err
	}
	return nil, last
}

// doc reads a YAML file of src/ as JSON-shaped values.
func (s *source) doc(rel string) (any, error) {
	b, err := s.file(rel)
	if err != nil {
		return nil, err
	}
	var y any
	if err := yaml.Unmarshal(b, &y); err != nil {
		return nil, fmt.Errorf("%s: %w", rel, err)
	}
	j, err := json.Marshal(y)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", rel, err)
	}
	return jsonschema.UnmarshalJSON(bytes.NewReader(j))
}

// Load is the jsonschema loader: URLs below Base are files of src/.
func (s *source) Load(u string) (any, error) {
	rel, ok := strings.CutPrefix(u, Base)
	if !ok {
		return nil, fmt.Errorf("%s is outside the description", u)
	}
	return s.doc(rel)
}

// compiler compiles the description's schemas as JSON Schema draft 4
// (OpenAPI 3.0's schema objects are close to it; its own keywords such
// as example and nullable are ignored), asserting formats.
func (s *source) compiler() *jsonschema.Compiler {
	c := jsonschema.NewCompiler()
	c.DefaultDraft(jsonschema.Draft4)
	c.AssertFormat()
	c.UseLoader(s)
	return c
}

// schema compiles schemas/<name>.yaml.
func (s *source) schema(t *testing.T, name string) *jsonschema.Schema {
	t.Helper()
	sch, err := s.compiler().Compile(Base + "schemas/" + name + ".yaml")
	if err != nil {
		t.Fatalf("compiling %s: %v", name, err)
	}
	return sch
}

// validate marshals v and validates it against sch.
func validate(sch *jsonschema.Schema, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(b))
	if err != nil {
		return err
	}
	if err := sch.Validate(inst); err != nil {
		return fmt.Errorf("%s\n%v", b, err)
	}
	return nil
}

// keys lists the member paths of a JSON value (array elements as []).
func keys(prefix string, v any, out map[string]bool) {
	switch x := v.(type) {
	case map[string]any:
		for k, e := range x {
			p := prefix + "/" + k
			out[p] = true
			keys(p, e, out)
		}
	case []any:
		for _, e := range x {
			keys(prefix+"[]", e, out)
		}
	}
}

func sorted(m map[string]bool) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
