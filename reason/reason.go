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
	"bytes"
	"encoding"
	"errors"
	"regexp"
	"strings"
)

// Reason is the canonical, validated representation of an error reason.
//
// Reasons are dot-separated hierarchical identifiers with a small, fixed depth.
// Each segment names a module, component, or operation inside the system.
//
// Example valid reasons:
//
//   - "apimachinery.schema.gvk.parse"
//   - "apimachinery.schema.types.validate"
//   - "storage.pg.connect"
//   - "auth.jwt.verify"
//   - "network.dns.resolve"
//
// The intent is to make it easy to programmatically build such identifiers from
// known package/component/operation names, and later to let mappers/loggers
// quickly match on these prefixes.
type Reason string

// MinLength and MaxLength define the allowed length range for a canonical
// reason string.
//
// We allow reasons to be a bit longer than codes, because they often contain
// multiple segments (module.component.operation).
const (
	// MinLength is the minimum length for a non-empty reason.
	// We keep it at 3 so that trivial values like "x" are not considered
	// meaningful reasons. Remember: the empty string is still allowed and
	// means "no reason provided".
	MinLength = 3

	// MaxLength is the maximum length for a valid reason.
	// 128 characters is enough even for 4 segments with descriptive names.
	MaxLength = 128
)

const (
	// reasonFmt is the canonical regular expression used to validate reasons.
	//
	// We accept 1 to 4 segments, dot-separated, each segment:
	//
	//   - starts with a lowercase ASCII letter [a-z]
	//   - continues with lowercase letters, digits, or underscore [a-z0-9_]*
	//
	// Examples that match:
	//
	//	"apimachinery.schema.gvk.parse"
	//	"storage.pg.connect"
	//	"auth.jwt.verify"
	//	"network.dns"
	//
	// Examples that DO NOT match:
	//
	//	"ApiMachinery..."  (uppercase)
	//	"apimachinery/schema" (slash)
	//	"apimachinery..schema" (empty segment)
	//	"1schema.parse" (digit first)
	//
	// NOTE: empty string ("") is treated separately as "optional reason" and does
	// not go through this regexp.
	reasonFmt = `^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*){0,3}$`
)

var (
	// reasonRe is the compiled regexp for the above pattern.
	reasonRe = regexp.MustCompile(reasonFmt)
)

var (
	// ErrReasonInvalidFormat is returned when a reason does not conform to
	// the expected format.
	ErrReasonInvalidFormat = errors.New("derrors: invalid reason format")
	// ErrReasonInvalidLength is returned when a reason is too short or too long.
	ErrReasonInvalidLength = errors.New("derrors: invalid reason length")
)

// Ensure Reason implements encoding.TextMarshaler / encoding.TextUnmarshaler.
var (
	_ encoding.TextMarshaler   = (*Reason)(nil)
	_ encoding.TextUnmarshaler = (*Reason)(nil)
)

// Empty is the zero-value reason. It is considered "not provided" and is valid
// to store in error structs. Callers that require a non-empty, canonical reason
// should explicitly call Validate.
var Empty Reason = ""

// Normalize takes an arbitrary string and tries to bring it closer to the
// canonical reason form.
//
// We do *very* conservative transformations:
//
//   - trim spaces
//   - lower-case
//   - convert "/" to "." (because callers may build paths with slashes)
//   - replace "-" with "_" (to align with code-style identifiers)
//
// It does NOT guarantee validity — callers should still call Parse/Validate.
func Normalize(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "/", ".")
	s = strings.ReplaceAll(s, "-", "_")
	return s
}

// Parse takes a user-provided string, normalizes it and validates it.
// On success it returns a canonical Reason value.
//
// Parse also accepts the empty string and returns reason.Empty without error.
// This is what makes Reason an "optional" part of the error model.
func Parse(s string) (Reason, error) {
	s = Normalize(s)
	if s == "" {
		return Empty, nil
	}
	if err := validate(s); err != nil {
		return Empty, err
	}
	return Reason(s), nil
}

// MustParse is the panic-on-error variant of Parse. It is useful for
// declaring package-level reason constants in var/const blocks.
//
// NOTE: unlike Parse, MustParse does NOT allow the empty string — passing
// an empty string here is almost always a programmer error.
func MustParse(s string) Reason {
	r, err := Parse(s)
	if err != nil {
		panic(err)
	}
	if r == Empty {
		panic("derrors: empty reason in MustParse")
	}
	return r
}

// Validate checks whether the provided Reason is in canonical form.
//
// The empty reason ("") is considered valid here, because the whole point of
// this type is to be optional. If you need to enforce "must be non-empty",
// add that check at call site.
func Validate(r Reason) error {
	if r == Empty {
		return nil
	}
	return validate(string(r))
}

// String returns the canonical string representation of the reason.
func (r Reason) String() string {
	return string(r)
}

// MarshalText implements encoding.TextMarshaler.
//
// We allow marshaling of the empty reason as an empty slice to not break
// JSON/YAML encoders that rely on TextMarshaler.
func (r Reason) MarshalText() ([]byte, error) {
	if err := Validate(r); err != nil {
		return nil, err
	}
	if r == Empty {
		return []byte{}, nil
	}
	return []byte(r), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
//
// It normalizes and validates the provided text before assigning.
// An empty or whitespace-only input will produce reason.Empty.
func (r *Reason) UnmarshalText(text []byte) error {
	s := string(bytes.TrimSpace(text))
	parsed, err := Parse(s)
	if err != nil {
		return err
	}
	*r = parsed
	return nil
}

// validate is the internal helper that checks length and format.
func validate(s string) error {
	if len(s) < MinLength || len(s) > MaxLength {
		return ErrReasonInvalidLength
	}
	if !reasonRe.MatchString(s) {
		return ErrReasonInvalidFormat
	}
	return nil
}
