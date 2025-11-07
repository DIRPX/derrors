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

package derrors

import (
	"errors"
	"strings"
	"testing"

	"dirpx.dev/derrors/code"
	"dirpx.dev/derrors/reason"
)

func mustReason(t *testing.T, s string) reason.Reason {
	t.Helper()
	r, err := reason.Parse(s)
	if err != nil {
		t.Fatalf("parse reason: %v", err)
	}
	return r
}

func TestError_Basics(t *testing.T) {
	e := E(code.Unavailable, "db is down",
		WithReasonOption(mustReason(t, "storage.pg.connect_timeout")),
		WithDetailOption("node", "pg-2"),
	)

	if e.Code != code.Unavailable {
		t.Fatal("code mismatch")
	}
	if e.Reason == "" {
		t.Fatal("reason must be set")
	}
	if e.Details["node"] != "pg-2" {
		t.Fatal("detail missing")
	}

	s := e.Error()
	wantSubs := []string{"unavailable", "storage.pg.connect_timeout", "db is down"}
	for _, sub := range wantSubs {
		if !strings.Contains(s, sub) {
			t.Fatalf("Error() missing %q in %q", sub, s)
		}
	}
}

func TestError_Immutability_CopyOnWrite(t *testing.T) {
	e1 := E(code.Invalid, "bad").WithDetail("k1", 1)
	e2 := e1.WithDetail("k2", 2)

	if len(e1.Details) != 1 || len(e2.Details) != 2 {
		t.Fatal("details size mismatch")
	}
	if _, ok := e1.Details["k2"]; ok {
		t.Fatal("original mutated")
	}
}

func TestError_WithCause_Unwrap(t *testing.T) {
	root := errors.New("root")
	e := E(code.Internal, "x").WithCause(root)
	if !errors.Is(e, root) {
		t.Fatal("errors.Is failed")
	}
	if errors.Unwrap(e) != root {
		t.Fatal("Unwrap failed")
	}
}

func TestError_WithDetails_Merge(t *testing.T) {
	e := E(code.Invalid, "x").WithDetails(map[string]any{"a": 1})
	e2 := e.WithDetails(map[string]any{"b": 2, "a": 3})
	if e.Details["a"] != 1 {
		t.Fatal("original mutated")
	}
	if e2.Details["a"] != 3 || e2.Details["b"] != 2 {
		t.Fatal("merge failed")
	}
}
