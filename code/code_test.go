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

package code

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
		{"trim spaces", "  internal  ", "internal"},
		{"to lower", "InVaLiD", "invalid"},
		{"dash to underscore", "not-found", "not_found"},
		{"mixed", "  ALREADY-EXISTS  ", "already_exists"},
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
		want Code
	}{
		{"simple", "internal", Code("internal")},
		{"with spaces", "  not_found  ", Code("not_found")},
		{"upper", "CONFLICT", Code("conflict")},
		{"dash", "already-exists", Code("already_exists")},
		{"min length", "abc", Code("abc")},
		{"max length", "a" + string(make([]byte, 0)) + "bc", Code("abc")},
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

func TestParse_Invalid(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{"empty", ""},
		{"too short", "a"},
		{"starts with digit", "1invalid"},
		{"contains dash after normalize", "x-"},
		{"uppercase with dash giving too short", "-"},
		{"too long", "a_very_long_code_that_is_definitely_more_than_sixty_four_characters_long"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.in)
			if err == nil {
				t.Fatalf("Parse(%q) = %q, want error", tt.in, got)
			}
			if got != Empty {
				t.Fatalf("Parse(%q) on error must return Empty, got %q", tt.in, got)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	valid := []Code{
		"internal",
		"not_found",
		"already_exists",
		"abc",
	}
	for _, c := range valid {
		if err := Validate(c); err != nil {
			t.Fatalf("Validate(%q) unexpected error: %v", c, err)
		}
	}

	invalid := []Code{
		"",          // empty
		"ab",        // too short
		"Invalid",   // uppercase
		"not-found", // dash
	}
	for _, c := range invalid {
		if err := Validate(c); err == nil {
			t.Fatalf("Validate(%q) expected error", c)
		}
	}
}

func TestMustParse_PanicsOnInvalid(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("MustParse should panic on invalid input")
		}
	}()
	_ = MustParse("INVALID CODE ??")
}

func TestMustParse_SucceedsOnValid(t *testing.T) {
	c := MustParse("not_found")
	if c != Code("not_found") {
		t.Fatalf("MustParse(valid) = %q, want %q", c, "not_found")
	}
}

func TestCode_String(t *testing.T) {
	c := Code("internal")
	if c.String() != "internal" {
		t.Fatalf("String() = %q, want %q", c.String(), "internal")
	}
}

func TestCode_MarshalText(t *testing.T) {
	c := Code("internal")
	text, err := c.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText() unexpected error: %v", err)
	}
	if string(text) != "internal" {
		t.Fatalf("MarshalText() = %q, want %q", string(text), "internal")
	}

	// invalid code should fail MarshalText
	invalid := Code("Invalid-Dash")
	if _, err := invalid.MarshalText(); err == nil {
		t.Fatalf("MarshalText() on invalid code must return error")
	}
}

func TestCode_UnmarshalText(t *testing.T) {
	var c Code
	if err := c.UnmarshalText([]byte("  NOT-FOUND  ")); err != nil {
		t.Fatalf("UnmarshalText() unexpected error: %v", err)
	}
	if c != Code("not_found") {
		t.Fatalf("UnmarshalText() = %q, want %q", c, "not_found")
	}

	// invalid
	var bad Code
	if err := bad.UnmarshalText([]byte("!@#")); err == nil {
		t.Fatalf("UnmarshalText() expected error for invalid input")
	}
}

func TestCode_ImplementsTextInterfaces(t *testing.T) {
	var _ encoding.TextMarshaler = (*Code)(nil)
	var _ encoding.TextUnmarshaler = (*Code)(nil)
}

func TestRegexAndLengthAreConsistent(t *testing.T) {
	// sanity: codeFmt should enforce 3..64
	if MinLength != 3 {
		t.Fatalf("MinLength changed, update tests")
	}
	if MaxLength != 64 {
		t.Fatalf("MaxLength changed, update tests")
	}

	// check a 64-char valid code: first char letter, rest underscores
	long := "a"
	for len(long) < MaxLength {
		long += "a"
	}

	if len(long) != MaxLength {
		t.Fatalf("constructed long code has len=%d, want %d", len(long), MaxLength)
	}

	if _, err := Parse(long); err != nil {
		t.Fatalf("expected %q to be valid (len=%d): %v", long, len(long), err)
	}

	// now 65 chars
	longer := long + "a"
	if _, err := Parse(longer); err == nil {
		t.Fatalf("expected %q (len=%d) to be invalid", longer, len(longer))
	}
}
