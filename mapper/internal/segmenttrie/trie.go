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

import (
	"errors"
	"strings"
)

// Trie is a segment-aware prefix index for dot-separated keys (reasons).
// Each node represents one segment; the wildcard "*" matches exactly one segment.
// The trie supports longest-prefix-match (LPM) with segment boundaries, so
// a more specific rule wins over a shorter one.
type Trie[T any] struct {
	// children contains next segments, including "*" for a single-segment wildcard.
	children map[string]*Trie[T]
	// hasVal marks that this node carries a value for the prefix ending here.
	hasVal bool
	val    T
	// pattern is the canonical dotted prefix (with '*' if wildcard was used)
	// for this node, set only when hasVal=true. It is used by MatchWithPattern
	// for Explain(), so we don't build strings during lookup.
	pattern string
}

var (
	// ErrInvalidPrefix is returned when inserting a prefix that is empty,
	// has empty segments, contains invalid characters, or consists only of wildcards.
	ErrInvalidPrefix = errors.New("segmenttrie: invalid prefix")
)

// New creates an empty trie ready for inserts.
func New[T any]() *Trie[T] {
	return &Trie[T]{children: make(map[string]*Trie[T])}
}

// Insert adds a dot-separated prefix to the trie and associates it with val.
//
// Examples:
//
//	"storage.pg"
//	"auth.jwt.verify"
//	"auth.*.verify"
//
// The wildcard "*" matches exactly one segment.
// A prefix made only of "*" segments is rejected, because it is too generic.
// Returns ErrInvalidPrefix on malformed input.
func (t *Trie[T]) Insert(prefix string, val T) error {
	if t == nil {
		return ErrInvalidPrefix
	}
	segs, ok := splitAndValidate(prefix, true /* allowWildcard */)
	if !ok || len(segs) == 0 {
		return ErrInvalidPrefix
	}

	// Require at least one non-wildcard segment to avoid catching everything.
	allWild := true
	for _, s := range segs {
		if s != "*" {
			allWild = false
			break
		}
	}
	if allWild {
		return ErrInvalidPrefix
	}

	cur := t
	for _, s := range segs {
		child, exists := cur.children[s]
		if !exists {
			child = New[T]()
			cur.children[s] = child
		}
		cur = child
	}
	cur.hasVal = true
	cur.val = val
	if cur.pattern == "" {
		// build pattern once; cost is at build time, not on hot path
		cur.pattern = prefix
	}
	return nil
}

// Match finds the best (deepest) prefix match for a full reason string.
// The reason is treated as a dot-separated sequence of segments.
// Both exact segment matches and "*" wildcard branches are explored.
// It returns (value, true) on success.
// If the reason is invalid or nothing matches, it returns the zero value and false.
func (t *Trie[T]) Match(reason string) (T, bool) {
	var zero T
	if t == nil {
		return zero, false
	}
	// empty reason => match only if root has value
	bestDepth := -1
	var bestVal T

	// dfs scans the next segment starting at byte offset 'off', with 'depth'
	// segments already consumed. It returns the best depth reachable from here.
	var dfs func(n *Trie[T], off, depth int) int
	dfs = func(n *Trie[T], off, depth int) int {
		if n.hasVal && depth > bestDepth {
			bestDepth = depth
			bestVal = n.val
		}
		if off >= len(reason) {
			return depth
		}

		// parse next segment [off:next), validating [a-z][a-z0-9_]*
		i := off
		// first char
		c := reason[i]
		if c < 'a' || c > 'z' {
			return depth // invalid segment => stop this path
		}
		i++
		for i < len(reason) {
			c = reason[i]
			if c == '.' {
				break
			}
			if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_') {
				return depth // invalid char => stop
			}
			i++
		}
		seg := reason[off:i] // substring; no heap alloc

		// exact branch
		if next, ok := n.children[seg]; ok {
			// skip '.' if present
			nextOff := i
			if nextOff < len(reason) && reason[nextOff] == '.' {
				nextOff++
			}
			_ = dfs(next, nextOff, depth+1)
		}
		// wildcard branch
		if next, ok := n.children["*"]; ok {
			nextOff := i
			if nextOff < len(reason) && reason[nextOff] == '.' {
				nextOff++
			}
			_ = dfs(next, nextOff, depth+1)
		}
		return depth
	}

	_ = dfs(t, 0, 0)
	if bestDepth < 0 {
		return zero, false
	}
	return bestVal, true
}

// MatchWithPattern returns value + the stored rule pattern (if any) for Explain().
// It reuses the same zero-alloc traversal as MatchValue but keeps a pointer
// to the deepest node that had a value; the pattern string is taken from node.
func (t *Trie[T]) MatchWithPattern(reason string) (T, bool, string) {
	var zero T
	if t == nil {
		return zero, false, ""
	}
	bestDepth := -1
	var bestVal T
	var bestPat string

	var dfs func(n *Trie[T], off, depth int)
	dfs = func(n *Trie[T], off, depth int) {
		if n.hasVal && depth > bestDepth {
			bestDepth = depth
			bestVal = n.val
			bestPat = n.pattern
		}
		if off >= len(reason) {
			return
		}
		// parse next segment (same as in MatchValue)
		i := off
		c := reason[i]
		if c < 'a' || c > 'z' {
			return
		}
		i++
		for i < len(reason) {
			c = reason[i]
			if c == '.' {
				break
			}
			if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_') {
				return
			}
			i++
		}
		seg := reason[off:i]
		nextOff := i
		if nextOff < len(reason) && reason[nextOff] == '.' {
			nextOff++
		}

		if next, ok := n.children[seg]; ok {
			dfs(next, nextOff, depth+1)
		}
		if next, ok := n.children["*"]; ok {
			dfs(next, nextOff, depth+1)
		}
	}

	dfs(t, 0, 0)
	if bestDepth < 0 {
		return zero, false, ""
	}
	return bestVal, true, bestPat
}

// splitAndValidate splits a dot-separated string into segments and validates
// each segment according to validSegment(). When allowWildcard=true,
// a segment that is exactly "*" is accepted.
// Returns (segments, true) on success, or (nil, false) on invalid input.
//
// Note: an empty string is treated as an empty (but valid) segment list
// to make matching against "" possible in callers.
func splitAndValidate(s string, allowWildcard bool) ([]string, bool) {
	if s == "" {
		return []string{}, true
	}
	segs := strings.Split(s, ".")
	for _, seg := range segs {
		if !validSegment(seg, allowWildcard) {
			return nil, false
		}
	}
	return segs, true
}

// validSegment reports whether seg is a valid trie segment.
// Rules:
//   - empty segments are invalid;
//   - when allowWildcard=true, the segment "*" is allowed;
//   - otherwise the segment must match: [a-z][a-z0-9_]*
//
// These rules keep reason prefixes simple, predictable and easy to normalize.
func validSegment(seg string, allowWildcard bool) bool {
	if seg == "" {
		return false
	}
	if allowWildcard && seg == "*" {
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
