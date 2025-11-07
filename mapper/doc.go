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

// Package mapper provides deterministic, immutable mappings from logical
// derrors codes (dirpx.dev/derrors/code) and optional reasons
// (dirpx.dev/derrors/reason) to transport-level statuses for HTTP and gRPC.
//
// # Overview
//
// In dirpx errors are expressed in two parts:
//
//  1. a high-level Code (e.g. code.Unavailable, code.Invalid),
//  2. an optional, more specific Reason (e.g. "storage.pg.connect_timeout").
//
// Transport layers (HTTP handlers, REST gateways, gRPC servers) need to turn
// this pair into concrete status codes. Package mapping does that in a way that
// is:
//
//   - immutable — a Mapper is a snapshot, safe for concurrent reuse;
//   - overridable — callers can change library defaults per Code;
//   - prefix-aware — callers can add fine-grained rules for specific reasons;
//   - dual — HTTP and gRPC are resolved with the same logic.
//
// # Resolution model
//
// A Mapper resolves statuses in the following order:
//
//  1. exact override for the Code;
//  2. per-Code longest-prefix-match (LPM) on the Reason;
//  3. per-Code default (library or user-adjusted);
//  4. global fallback (500 / codes.Internal).
//
// Prefix rules are segment-aware: reasons are treated as "."-separated segments,
// and "*" matches exactly one segment. For example:
//
//	WithHTTPPrefix(code.Unavailable, "storage.pg", http.StatusServiceUnavailable)
//	WithHTTPPrefix(code.Unavailable, "storage.*.connect", http.StatusServiceUnavailable)
//
// The more specific prefix wins.
//
// # Library defaults
//
// The package ships with sensible defaults for common dirpx codes, mapping them
// to standard net/http constants and grpc/codes values (e.g. code.Invalid -> 400 /
// InvalidArgument, code.Unauthenticated -> 401 / Unauthenticated, code.Unavailable
// -> 503 / Unavailable). These can be adjusted at build time.
//
// # Building a mapper
//
// A Mapper is created once and reused:
//
//	m, err := mapping.New(
//	    mapping.WithHTTPOverride(code.Canceled, 499),           // nginx-style
//	    mapping.WithHTTPPrefix(code.Unavailable, "storage.pg", 503),
//	)
//	if err != nil {
//	    // invalid prefix, etc.
//	}
//
//	st := m.Status(code.Unavailable, reason.Of("storage.pg.connect_timeout"))
//	// st.HTTP == 503, st.GRPC == codes.Unavailable
//
// # Diagnostics
//
// For debugging and tests, Mapper.Explain returns a human-readable trace of how
// a particular (code, reason) was resolved, including which tier matched and, for
// prefixes, which pattern was used.
//
// This is intended for inspection and logging, not for stable machine parsing.
//
// # Immutability
//
// All user-provided inputs are copied during New. After construction, the Mapper
// does not observe further changes to the caller's maps or slices. This makes it
// safe to share a single instance across handlers, goroutines, and requests.
package mapper
