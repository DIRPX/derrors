/*
   Copyright 2025 The DIRPX Authors

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package mapper

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"dirpx.dev/derrors/code"
	"dirpx.dev/derrors/reason"
	"google.golang.org/grpc/codes"
)

var update = flag.Bool("update", false, "update golden files")

// TestExplain_Golden verifies Explain() output is stable and human-friendly.
// Update golden with: go test ./derrors/mapper -run Explain_Golden -update
func TestExplain_Golden(t *testing.T) {
	m, err := New(
		// prefix rules
		WithHTTPPrefix(code.Unavailable, "storage.pg", 503),
		WithGRPCPrefix(code.Unavailable, "storage.pg", int(codes.Unavailable)),
		// exact override rules
		WithHTTPOverride(code.Canceled, 408),
		WithGRPCOverride(code.Canceled, int(codes.Canceled)),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	var b strings.Builder

	// Case 1: prefix hit
	exp1 := m.Explain(code.Unavailable, mustReason("storage.pg.connect_timeout"))
	b.WriteString(exp1)
	b.WriteString("\n---\n")

	// Case 2: override
	exp2 := m.Explain(code.Canceled, reason.Empty)
	b.WriteString(exp2)
	b.WriteString("\n")

	got := b.String()

	goldenPath := filepath.Join("testdata", "explain.golden")
	if *update {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
			t.Fatalf("mkdir testdata: %v", err)
		}
		if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Logf("updated %s", goldenPath)
		return
	}

	wantBytes, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v (run with -update to create)", err)
	}
	want := string(wantBytes)

	// normalize trailing newlines to avoid EOF newline mismatches
	normalize := func(s string) string { return strings.TrimRight(s, "\r\n") }

	if normalize(want) != normalize(got) {
		t.Fatalf("Explain() output mismatch.\n--- want ---\n%s\n--- got ---\n%s", want, got)
	}
}
