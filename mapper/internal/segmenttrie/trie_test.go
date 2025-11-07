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

package segmenttrie

import "testing"

func TestInsertAndMatch_Simple(t *testing.T) {
	tr := New[int]()
	must(t, tr.Insert("storage.pg", 503))
	must(t, tr.Insert("auth.jwt.verify", 401))
	must(t, tr.Insert("apimachinery.schema.gvk.parse", 400))

	if v, ok, p := tr.MatchWithPattern("storage.pg.connect"); !ok || v != 503 || p != "storage.pg" {
		t.Fatalf("match storage.pg.connect => ok=%v v=%v p=%q; want ok=true v=503 p=storage.pg", ok, v, p)
	}
	if v, ok, p := tr.MatchWithPattern("auth.jwt.verify"); !ok || v != 401 || p != "auth.jwt.verify" {
		t.Fatalf("match auth.jwt.verify => ok=%v v=%v p=%q; want ok=true v=401 p=auth.jwt.verify", ok, v, p)
	}
	if v, ok, p := tr.MatchWithPattern("apimachinery.schema.gvk.parse.kind"); !ok || v != 400 || p != "apimachinery.schema.gvk.parse" {
		t.Fatalf("match gvk.parse.kind => ok=%v v=%v p=%q; want 400, apimachinery.schema.gvk.parse", ok, v, p)
	}
}

func TestWildcard_OneSegment(t *testing.T) {
	tr := New[int]()
	must(t, tr.Insert("auth.*.verify", 498))
	must(t, tr.Insert("auth.jwt.verify", 401)) // exact should beat wildcard at same depth

	// exact match wins
	if v, ok, p := tr.MatchWithPattern("auth.jwt.verify"); !ok || v != 401 || p != "auth.jwt.verify" {
		t.Fatalf("exact must win over wildcard, got ok=%v v=%v p=%q", ok, v, p)
	}
	// wildcard matches a different middle segment
	if v, ok, p := tr.MatchWithPattern("auth.saml.verify.token"); !ok || v != 498 || p != "auth.*.verify" {
		t.Fatalf("wildcard match failed: ok=%v v=%v p=%q", ok, v, p)
	}
	// wildcard must match exactly one segment, not zero
	if _, ok, _ := tr.MatchWithPattern("auth.verify"); ok {
		t.Fatalf("wildcard should not match zero segments")
	}
}

func TestLPM_PrefersDeeperEvenIfExactBranchExists(t *testing.T) {
	tr := New[int]()
	// wildcard path can produce deeper match than an existing (but shallow) exact branch
	must(t, tr.Insert("a.*.c", 7))
	// create an exact branch that doesn't lead to a value at the same depth
	// (common pitfall for greedy algorithms)
	must(t, tr.Insert("a.b", 1))

	if v, ok, p := tr.MatchWithPattern("a.b.c"); !ok || v != 7 || p != "a.*.c" {
		t.Fatalf("LPM must choose wildcard path: ok=%v v=%v p=%q", ok, v, p)
	}
}

func TestInvalidInputs(t *testing.T) {
	tr := New[int]()
	if err := tr.Insert("", 1); err == nil {
		t.Fatalf("empty prefix must be invalid")
	}
	if err := tr.Insert("UPPER.case", 1); err == nil {
		t.Fatalf("uppercase must be invalid")
	}
	if err := tr.Insert("a..b", 1); err == nil {
		t.Fatalf("empty segment must be invalid")
	}
	if err := tr.Insert("*", 1); err == nil {
		t.Fatalf("single wildcard without context should be invalid (segment OK, but prefix must have at least one seg before/after)")
	}
	// NB: The above rule is stylistic; if you want "*" allowed at root, remove this test.

	if _, ok, _ := tr.MatchWithPattern("UPPER.case"); ok {
		t.Fatalf("match should be false for invalid reason")
	}
	if _, ok, _ := tr.MatchWithPattern("a..b"); ok {
		t.Fatalf("match should be false for invalid reason")
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
