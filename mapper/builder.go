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
	"net/http"

	"dirpx.dev/derrors/code"
	"google.golang.org/grpc/codes"
)

type prefixRule struct {
	// prefix is the raw, dot-separated reason prefix (may contain "*").
	// It is validated/normalized when we build the per-code trie.
	prefix string
	// val is the numeric transport status to apply when this prefix matches.
	// For HTTP this is the final value; for gRPC we store ints in the builder
	// and convert to codes.Code later.
	val int
}

type builder struct {
	// user-provided adjustments (applied on top of library defaults)

	// httpDefaults holds per-code HTTP defaults that override library defaults.
	httpDefaults map[code.Code]int
	// grpcDefaults holds per-code gRPC defaults as ints; converted to codes.Code in New().
	grpcDefaults map[code.Code]int

	// httpOverride holds exact per-code HTTP overrides (higher than defaults).
	httpOverride map[code.Code]int
	// grpcOverride holds exact per-code gRPC overrides as ints; converted in New().
	grpcOverride map[code.Code]int

	// httpPrefixes holds per-code LPM rules for HTTP, defined as raw prefixRule
	// and later compiled into a segment trie.
	httpPrefixes map[code.Code][]prefixRule
	// grpcPrefixes holds per-code LPM rules for gRPC.
	grpcPrefixes map[code.Code][]prefixRule

	// global fallbacks used when a code has no default at all.
	fallbackHTTP int
	fallbackGRPC codes.Code
}

// newBuilder creates an empty builder with maps pre-sized
// to hold typical numbers of entries.
func newBuilder() *builder {
	return &builder{
		// we size the maps roughly to the number of built-in defaults
		httpDefaults: make(map[code.Code]int, len(defaultHTTP)),
		grpcDefaults: make(map[code.Code]int, len(defaultGRPC)),

		// overrides and prefixes are usually few
		httpOverride: make(map[code.Code]int),
		grpcOverride: make(map[code.Code]int),
		httpPrefixes: make(map[code.Code][]prefixRule),
		grpcPrefixes: make(map[code.Code][]prefixRule),

		// hard fallbacks if the code was never seen
		fallbackHTTP: http.StatusInternalServerError,
		fallbackGRPC: codes.Internal,
	}
}
