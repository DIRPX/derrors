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
	"fmt"
	"strings"

	"dirpx.dev/derrors/apis"
	"dirpx.dev/derrors/code"
	"dirpx.dev/derrors/mapper/internal/segmenttrie"
	"dirpx.dev/derrors/reason"
	"google.golang.org/grpc/codes"
)

// New constructs an immutable apis.Mapper snapshot.
//
// The resulting apis.Mapper is fully thread-safe and designed for long-lived reuse.
// Each build creates a self-contained mapper instance — no shared references
// to global state or user-provided structures remain.
//
// Build process overview:
//
//  1. Seed the builder with library defaults (HTTP & gRPC).
//  2. Apply user-provided options (defaults, overrides, prefix rules).
//  3. Normalize and validate all reason prefixes (via reason.Normalize/Parse).
//  4. Build per-code segment tries (HTTP & gRPC) supporting longest-prefix-match
//     with '*' as a single-segment wildcard.
//  5. Freeze all maps and tries into immutable copies (fresh allocations).
//
// Errors returned from this function indicate invalid prefixes or configuration
// issues during normalization or trie construction.
func New(opts ...Option) (apis.Mapper, error) {
	// (0) Start with an empty builder.
	// We do not assume any pre-seeded state.
	b := newBuilder()

	// (1) Seed the builder with package-level defaults.
	// Copy into builder-owned maps to prevent external mutation.
	for k, v := range defaultHTTP {
		b.httpDefaults[k] = v
	}
	for k, v := range defaultGRPC {
		// Keep values as int for internal uniformity;
		// convert to codes.Code when freezing the final snapshot.
		b.grpcDefaults[k] = int(v)
	}

	// (2) Apply user-supplied options (defaults, overrides, prefixes, etc.).
	for _, opt := range opts {
		opt(b)
	}

	// (3) Build per-code HTTP prefix tries.
	// Each rule prefix is normalized and validated before insertion.
	httpTrie := make(map[code.Code]*segmenttrie.Trie[int], len(b.httpPrefixes))
	for c, rules := range b.httpPrefixes {
		if len(rules) == 0 {
			continue
		}
		t := segmenttrie.New[int]()
		for _, r := range rules {
			p, err := normalizeAndValidatePrefix(r.prefix)
			if err != nil {
				return nil, fmt.Errorf("mapper: invalid HTTP reason-prefix %q for code %q: %w", r.prefix, c, err)
			}
			if err := t.Insert(p, r.val); err != nil {
				return nil, fmt.Errorf("mapper: cannot insert HTTP prefix %q for code %q: %w", p, c, err)
			}
		}
		httpTrie[c] = t
	}

	// (4) Build per-code gRPC prefix tries.
	// Values are stored as int in the builder and converted to codes.Code here.
	grpcTrie := make(map[code.Code]*segmenttrie.Trie[codes.Code], len(b.grpcPrefixes))
	for c, rules := range b.grpcPrefixes {
		if len(rules) == 0 {
			continue
		}
		t := segmenttrie.New[codes.Code]()
		for _, r := range rules {
			p, err := normalizeAndValidatePrefix(r.prefix)
			if err != nil {
				return nil, fmt.Errorf("mapper: invalid gRPC reason-prefix %q for code %q: %w", r.prefix, c, err)
			}
			if err := t.Insert(p, codes.Code(r.val)); err != nil {
				return nil, fmt.Errorf("mapper: cannot insert gRPC prefix %q for code %q: %w", p, c, err)
			}
		}
		grpcTrie[c] = t
	}

	// (5) Freeze everything into a read-only snapshot.
	// Each map is freshly allocated; tries are shallow-copied (they are immutable).
	m := &mapper{
		httpDefault:  freezeHTTPDefaults(b.httpDefaults),
		grpcDefault:  freezeGRPCDefaults(b.grpcDefaults),
		httpOverride: freezeHTTPOverrides(b.httpOverride),
		grpcOverride: freezeGRPCOverrides(b.grpcOverride),
		httpTrie:     freezeHTTPTrie(httpTrie),
		grpcTrie:     freezeGRPCTrie(grpcTrie),

		fallbackHTTP: b.fallbackHTTP,
		fallbackGRPC: b.fallbackGRPC,
	}

	return m, nil
}

// mapper is an immutable mapper implementation that combines
// per-code defaults, per-code exact overrides, and per-code segment-aware
// prefix tries for reasons. Lookups are O(depth) and safe for concurrent use
// once constructed.
type mapper struct {
	// httpDefault holds the base HTTP status for a given logical error code.
	// Used when no per-reason rule and no override are present.
	httpDefault map[code.Code]int

	// grpcDefault holds the base gRPC status for a given logical error code.
	grpcDefault map[code.Code]codes.Code

	// httpOverride holds explicit HTTP statuses for specific codes.
	// These take precedence over defaults but are below per-reason LPM rules.
	httpOverride map[code.Code]int

	// grpcOverride holds explicit gRPC statuses for specific codes.
	grpcOverride map[code.Code]codes.Code

	// httpTrie stores per-code tries that resolve HTTP statuses based on
	// reason prefixes (dot-separated, with "*" for one-segment wildcards).
	httpTrie map[code.Code]*segmenttrie.Trie[int]

	// grpcTrie stores per-code tries that resolve gRPC statuses based on
	// reason prefixes.
	grpcTrie map[code.Code]*segmenttrie.Trie[codes.Code]

	// fallbackHTTP is used when there is no mapper at all for a code.
	// Typically http.StatusInternalServerError.
	fallbackHTTP int

	// fallbackGRPC is used when there is no mapper at all for a code.
	// Typically codes.Internal.
	fallbackGRPC codes.Code
}

// HTTPStatus resolves an HTTP status for the given code and reason.
//
// Resolution order (highest to lowest):
//  1. exact per-code override (explicitly registered);
//  2. per-code longest-prefix-match rule on the reason;
//  3. per-code default (library or user overridden);
//  4. hardcoded ultimate fallback (500).
//
// The reason is treated as a dot-separated string; LPM rules are stored per code.
func (m *mapper) HTTPStatus(c code.Code, r reason.Reason) int {
	// 1. Fast path: exact override for this code.
	if v, ok := m.httpOverride[c]; ok {
		return v
	}

	// 2. Per-code prefix LPM over the reason.
	if idx, ok := m.httpTrie[c]; ok && idx != nil {
		if v, ok := idx.Match(string(r)); ok {
			return v
		}
	}

	// 3. Per-code default.
	if v, ok := m.httpDefault[c]; ok {
		return v
	}

	// 4. Ultimate fallback: HTTP must never be zero.
	return 500
}

// GRPCStatus resolves a gRPC status for the given code and reason.
// Uses the same precedence as HTTPStatus, but returns gRPC codes.
//
// Resolution order:
//  1. exact per-code override;
//  2. per-code LPM by reason;
//  3. per-code default;
//  4. hardcoded fallback (codes.Internal).
func (m *mapper) GRPCStatus(c code.Code, r reason.Reason) codes.Code {
	// 1. Exact override.
	if v, ok := m.grpcOverride[c]; ok {
		return v
	}

	// 2. Trie-based LPM for this code.
	if idx, ok := m.grpcTrie[c]; ok && idx != nil {
		if v, ok := idx.Match(string(r)); ok {
			return v
		}
	}

	// 3. Default for this code.
	if v, ok := m.grpcDefault[c]; ok {
		return v
	}

	// 4. Ultimate fallback.
	return codes.Internal
}

// Status resolves both HTTP and gRPC using the same inputs.
// This keeps HTTP/GRPC decisions consistent for a single logical error.
func (m *mapper) Status(c code.Code, r reason.Reason) apis.Status {
	return apis.Status{
		HTTP: m.HTTPStatus(c, r),
		GRPC: m.GRPCStatus(c, r),
	}
}

// Explain produces a textual trace of how the mapper resolved HTTP and gRPC
// statuses for a particular (code, reason) pair.
//
// This is primarily a diagnostic tool: it shows which tier matched
// (override, prefix, default, or fallback) and, for prefix matches,
// which pattern was used.
//
// Example output:
//
//	code="unavailable" reason="storage.pg.connect_timeout"
//	http:  source=prefix pattern="storage.pg" -> 503
//	grpc:  source=default -> UNAVAILABLE(14)
//
// Notes:
//   - source ∈ {override | prefix | default | fallback}
//   - pattern is the rule as it was stored in the trie (may contain "*")
func (m *mapper) Explain(c code.Code, r reason.Reason) string {
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "code=%q reason=%q\n", c, r)

	// ---- HTTP ----
	switch src, httpLine := m.explainHTTP(c, r); src {
	case "override", "prefix", "default", "fallback":
		_, _ = fmt.Fprintln(&b, httpLine)
	default:
		// Defensive: unexpected source.
		_, _ = fmt.Fprintln(&b, "http:  source=unknown")
	}

	// ---- gRPC ----
	switch src, grpcLine := m.explainGRPC(c, r); src {
	case "override", "prefix", "default", "fallback":
		_, _ = fmt.Fprintln(&b, grpcLine)
	default:
		_, _ = fmt.Fprintln(&b, "grpc:  source=unknown")
	}

	return strings.TrimSuffix(b.String(), "\n")
}

// explainHTTP returns the origin ("override", "prefix", "default", "fallback")
// and a formatted line describing how the HTTP status was chosen.
func (m *mapper) explainHTTP(c code.Code, r reason.Reason) (source, line string) {
	// 1) exact per-code override
	if v, ok := m.httpOverride[c]; ok {
		return "override", fmt.Sprintf("http: source=override -> %d", v)
	}

	// 2) per-code LPM against the reason
	if idx, ok := m.httpTrie[c]; ok && idx != nil {
		if v, ok2, pat := idx.MatchWithPattern(string(r)); ok2 {
			return "prefix", fmt.Sprintf("http: source=prefix pattern=%q -> %d", pat, v)
		}
	}

	// 3) per-code default
	if v, ok := m.httpDefault[c]; ok {
		return "default", fmt.Sprintf("http: source=default -> %d", v)
	}

	// 4) global fallback
	return "fallback", fmt.Sprintf("http source=fallback -> %d", m.fallbackHTTP)
}

// explainGRPC returns the origin ("override", "prefix", "default", "fallback")
// and a formatted line describing how the gRPC status was chosen.
func (m *mapper) explainGRPC(c code.Code, r reason.Reason) (source, line string) {
	// 1) exact per-code override
	if v, ok := m.grpcOverride[c]; ok {
		return "override", fmt.Sprintf("grpc: source=override -> %s(%d)", strings.ToUpper(v.String()), int(v))
	}

	// 2) per-code LPM against the reason
	if idx, ok := m.grpcTrie[c]; ok && idx != nil {
		if v, ok2, pat := idx.MatchWithPattern(string(r)); ok2 {
			return "prefix", fmt.Sprintf("grpc: source=prefix pattern=%q -> %s(%d)", pat, strings.ToUpper(v.String()), int(v))
		}
	}

	// 3) per-code default
	if v, ok := m.grpcDefault[c]; ok {
		return "default", fmt.Sprintf("grpc: source=default -> %s(%d)", strings.ToUpper(v.String()), int(v))
	}

	// 4) global fallback
	return "fallback", fmt.Sprintf("grpc: source=fallback -> %s(%d)", strings.ToUpper(m.fallbackGRPC.String()), int(m.fallbackGRPC))
}

// normalizeAndValidatePrefix ensures a reason prefix is canonical and valid.
// It forbids empty strings as prefixes and delegates structural checks to reason.Parse.
func normalizeAndValidatePrefix(raw string) (string, error) {
	p := reason.Normalize(raw)
	if p == "" {
		return "", fmt.Errorf("empty prefix")
	}
	segs := strings.Split(p, ".")
	allWild := true
	for _, seg := range segs {
		if !validPrefixSegment(seg) { // allows "*" or [a-z][a-z0-9_]*
			return "", fmt.Errorf("invalid segment %q", seg)
		}
		if seg != "*" {
			allWild = false
		}
	}
	if allWild {
		return "", fmt.Errorf("prefix cannot consist of '*' only")
	}
	return p, nil
}

// validPrefixSegment reports whether seg is a valid trie segment for prefixes.
// Rules:
//   - empty segments are invalid;
//   - the segment "*" is allowed;
//   - otherwise the segment must match: [a-z][a-z0-9_]*
func validPrefixSegment(seg string) bool {
	if seg == "" {
		return false
	}
	if seg == "*" {
		return true
	}
	// [a-z][a-z0-9_]*
	if seg[0] < 'a' || seg[0] > 'z' {
		return false
	}
	for i := 1; i < len(seg); i++ {
		c := seg[i]
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' {
			continue
		}
		return false
	}
	return true
}
