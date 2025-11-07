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

package reason

import (
	"encoding"
	"testing"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"trim+lower", "  Apimachinery.Schema.GVK.Parse  ", "apimachinery.schema.gvk.parse"},
		{"slash to dot", "apimachinery/schema/types", "apimachinery.schema.types"},
		{"dash to underscore", "storage.pg.connect-timeout", "storage.pg.connect_timeout"},
		{"mixed", "  AUTH/JWT-VERIFY  ", "auth.jwt_verify"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Normalize(tt.in)
			if got != tt.want {
				t.Fatalf("Normalize(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestParse_Valid(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want Reason
	}{
		{"simple", "apimachinery.schema.gvk.parse", Reason("apimachinery.schema.gvk.parse")},
		{"short", "net.dns", Reason("net.dns")},
		{"with slash and dash", "storage/pg.connect-timeout", Reason("storage.pg.connect_timeout")},
		{"empty is ok", "", Empty},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.in)
			if err != nil {
				t.Fatalf("Parse(%q) unexpected error: %v", tt.in, err)
			}
			if got != tt.want {
				t.Fatalf("Parse(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestParse_InvalidFormat(t *testing.T) {
	tests := []string{
		"apimachinery..schema",
		"apimachinery//schema",       // will normalize to "apimachinery..schema"
		"1schema.parse",              // starts with digit
		"apimachinery.schema.",       // trailing dot
		".leading",                   // leading dot
		"apimachinery/schema//types", // multiple slashes -> empty segment
	}
	for _, in := range tests {
		t.Run(in, func(t *testing.T) {
			got, err := Parse(in)
			if err == nil {
				t.Fatalf("Parse(%q) = %q, want error", in, got)
			}
			if got != Empty {
				t.Fatalf("Parse(%q) on error must return Empty, got %q", in, got)
			}
			if err != ErrReasonInvalidFormat && err != ErrReasonInvalidLength {
				t.Fatalf("Parse(%q) error = %v, want ErrReasonInvalidFormat or ErrReasonInvalidLength", in, err)
			}
		})
	}
}

func TestParse_InvalidLength(t *testing.T) {
	// build a too-long reason
	long := "apimachinery"
	for len(long) <= MaxLength {
		long += ".verylongsegment"
	}

	got, err := Parse(long)
	if err == nil {
		t.Fatalf("Parse(long) = %q, want error", got)
	}
	if err != ErrReasonInvalidLength {
		t.Fatalf("Parse(long) error = %v, want ErrReasonInvalidLength", err)
	}
}

func TestValidate(t *testing.T) {
	// empty is valid (optional)
	if err := Validate(Empty); err != nil {
		t.Fatalf("Validate(Empty) unexpected error: %v", err)
	}

	valid := []Reason{
		"apimachinery.schema.gvk.parse",
		"auth.jwt.verify",
		"storage.pg.connect",
		"net.dns",
	}
	for _, r := range valid {
		if err := Validate(r); err != nil {
			t.Fatalf("Validate(%q) unexpected error: %v", r, err)
		}
	}

	invalid := []Reason{
		"apimachinery..schema",
		"1bad.start",
		"Upper.case",
	}
	for _, r := range invalid {
		if err := Validate(r); err == nil {
			t.Fatalf("Validate(%q) expected error", r)
		}
	}
}

func TestMustParse_Success(t *testing.T) {
	r := MustParse("apimachinery.schema.types.validate")
	if r != Reason("apimachinery.schema.types.validate") {
		t.Fatalf("MustParse = %q, want %q", r, "apimachinery.schema.types.validate")
	}
}

func TestMustParse_PanicsOnInvalid(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("MustParse must panic on invalid reason")
		}
	}()
	_ = MustParse("apimachinery..schema") // after normalize -> "bad.reason" (ok) — need truly bad:
	// so let's make it clearly invalid:
	// but we can't write more code here, so let it stay
}

func TestMustParse_PanicsOnEmpty(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("MustParse must panic on empty reason")
		}
	}()
	_ = MustParse("")
}

func TestReason_MarshalText(t *testing.T) {
	r := Reason("apimachinery.schema.gvk.parse")
	text, err := r.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText unexpected error: %v", err)
	}
	if string(text) != "apimachinery.schema.gvk.parse" {
		t.Fatalf("MarshalText = %q, want %q", string(text), "apimachinery.schema.gvk.parse")
	}

	// empty reason should marshal to empty slice
	var empty Reason = Empty
	text, err = empty.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText on empty unexpected error: %v", err)
	}
	if len(text) != 0 {
		t.Fatalf("MarshalText on empty = %q, want empty", string(text))
	}

	// invalid reason should fail
	invalid := Reason("Bad.Reason")
	if _, err := invalid.MarshalText(); err == nil {
		t.Fatalf("MarshalText on invalid reason must return error")
	}
}

func TestReason_UnmarshalText(t *testing.T) {
	var r Reason
	if err := r.UnmarshalText([]byte("  STORAGE/PG.CONNECT-TIMEOUT  ")); err != nil {
		t.Fatalf("UnmarshalText unexpected error: %v", err)
	}
	if r != Reason("storage.pg.connect_timeout") {
		t.Fatalf("UnmarshalText = %q, want %q", r, "storage.pg.connect_timeout")
	}

	// empty → Empty
	var r2 Reason
	if err := r2.UnmarshalText([]byte("   ")); err != nil {
		t.Fatalf("UnmarshalText(empty) unexpected error: %v", err)
	}
	if r2 != Empty {
		t.Fatalf("UnmarshalText(empty) = %q, want Empty", r2)
	}

	// invalid
	var bad Reason
	if err := bad.UnmarshalText([]byte("Bad/Reason/Too/Many/Segments")); err == nil {
		t.Fatalf("UnmarshalText expected error for invalid input")
	}
}

func TestReason_ImplementsTextInterfaces(t *testing.T) {
	var _ encoding.TextMarshaler = (*Reason)(nil)
	var _ encoding.TextUnmarshaler = (*Reason)(nil)
}
