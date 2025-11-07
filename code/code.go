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
	"bytes"
	"encoding"
	"errors"
	"regexp"
	"strings"
)

// Code is the canonical, validated representation of an error code.
//
// It is defined as a separate type (not just string) so that other packages
// can explicitly declare which values they expect and to avoid accidental
// mixing of raw user input with normalized values.
//
// IMPORTANT: Empty codes ("") are NOT allowed. Every error MUST have a
// non-empty code.
type Code string

// MinLength and MaxLength define the allowed length range for a canonical
// derrors code.
//
// We keep these values as separate constants so they can be referenced in
// validation errors, tests, or in other packages that want to mirror the same
// constraints.
const (
	// MinLength is the minimum length for a valid code.
	// We require at least 3 characters so that ultra-short and ambiguous
	// identifiers like "a" or "x1" are not accepted.
	MinLength = 3

	// MaxLength is the maximum length for a valid code.
	// 64 characters is enough for descriptive codes like "too_many_attempts"
	// while still preventing unbounded or accidental long strings.
	MaxLength = 64
)

const (
	// codeFmt is the canonical regular expression used to validate error codes.
	//
	// The pattern is intentionally kept in a separate constant so that:
	//   - it can be referenced from tests;
	//   - it is obvious that the regexp below is derived from this exact pattern;
	//   - it is easy to keep the regexp and the length constraints in sync.
	//
	// Pattern breakdown:
	//
	//	^ - start of string;
	//	[a-z] - first character must be a lowercase ASCII letter;
	//	[a-z0-9_]{2,63} - the remaining characters may be lowercase letters,
	//	                  digits or underscore; the quantifier {2,63} makes the
	//	                  total length 3..64 characters (1 + 2..63);
	//	$ - end of string;
	//
	// IMPORTANT: the numeric range {2,63} is tied to MinLength / MaxLength above.
	// If you change MinLength / MaxLength, make sure to adjust this pattern as well.
	codeFmt = `^[a-z][a-z0-9_]{2,63}$`
)

var (
	// codeRe is the compiled regular expression used at runtime to validate
	// that a string is a canonical derrors code.
	//
	// We precompile it so that repeated validations (e.g. in registries or in
	// hot paths) do not pay the compilation cost over and over again.
	//
	// Examples of valid codes:
	//   - "invalid"
	//   - "not_found"
	//   - "already_exists"
	//   - "internal"
	//
	// Examples of invalid codes:
	//   - "Invalid"     (uppercase)
	//   - "not-found"   (dash instead of underscore)
	//   - "x"           (too short)
	//   - "1notvalid"   (does not start with a letter)
	codeRe = regexp.MustCompile(codeFmt)
)

var (
	// ErrCodeInvalid is returned when a value cannot be parsed or validated
	// as a derrors code.
	//
	// Having a dedicated sentinel error makes it easier for callers and tests
	// to detect "this is about code format" vs "this is some other error".
	ErrCodeInvalid = errors.New("derrors: invalid code")
)

// Ensure Code implements encoding.TextMarshaler / encoding.TextMarshaler
// so it can be embedded into larger config or API structs.
var (
	_ encoding.TextMarshaler   = (*Code)(nil)
	_ encoding.TextUnmarshaler = (*Code)(nil)
)

// Empty is the zero-value code. It is considered "not provided" and is valid
// to store in error structs. Callers that require a non-empty, canonical code
// should explicitly call Validate.
var Empty Code = ""

// Parse takes a user-provided string, normalizes it and validates it.
// On success it returns a canonical Code value.
func Parse(s string) (Code, error) {
	s = Normalize(s)
	if err := validate(s); err != nil {
		return Empty, err
	}
	return Code(s), nil
}

// MustParse is the panic-on-error variant of Parse. It is useful for
// declaring package-level constants in init() or var blocks.
func MustParse(s string) Code {
	c, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return c
}

// Normalize takes an arbitrary string and tries to bring it closer to the
// canonical code form.
//
// This function is intentionally conservative: it only performs obvious,
// non-lossy transformations:
//
//   - trims surrounding spaces;
//   - lowercases the value;
//   - replaces '-' with '_';
//
// It does NOT guarantee that the result is valid — callers should still call
// Validate/Parse after normalization.
func Normalize(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "-", "_")
	return s
}

// Validate checks whether the provided Code is valid.
// The empty code ("") is considered invalid.
func Validate(c Code) error {
	return validate(string(c))
}

// String returns the canonical string representation of the code.
func (c Code) String() string {
	return string(c)
}

// MarshalText implements encoding.TextMarshaler.
//
// It always returns the canonical string representation.
func (c Code) MarshalText() ([]byte, error) {
	if err := Validate(c); err != nil {
		return nil, err
	}
	return []byte(c), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
//
// It normalizes and validates the provided text before assigning.
func (c *Code) UnmarshalText(text []byte) error {
	// We copy into a buffer to avoid changing the input slice.
	s := string(bytes.TrimSpace(text))
	parsed, err := Parse(s)
	if err != nil {
		return err
	}
	*c = parsed
	return nil
}

// validate is a helper that checks whether the provided string is a valid code.
func validate(s string) error {
	if !codeRe.MatchString(s) {
		return ErrCodeInvalid
	}
	return nil
}
