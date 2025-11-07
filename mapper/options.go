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
	"dirpx.dev/derrors/code"
)

// Option configures the Mapper at build time.
// All options are applied to an internal builder and then frozen into
// an immutable Mapper.
type Option func(*builder)

// WithHTTPDefault sets or replaces the library-level default HTTP status
// for the given error code. This affects the fallback value used when
// no per-reason override is found.
func WithHTTPDefault(c code.Code, http int) Option {
	return func(b *builder) { b.httpDefaults[c] = http }
}

// WithGRPCDefault sets or replaces the library-level default gRPC status
// for the given error code. This affects the fallback value used when
// no per-reason override is found.
func WithGRPCDefault(c code.Code, grpc int) Option {
	return func(b *builder) { b.grpcDefaults[c] = grpc }
}

// WithHTTPOverride registers an exact HTTP override for the given code.
// Overrides take precedence over defaults but still sit below per-reason
// prefix matches (LPM) for that code.
func WithHTTPOverride(c code.Code, http int) Option {
	return func(b *builder) { b.httpOverride[c] = http }
}

// WithGRPCOverride registers an exact gRPC override for the given code.
// Overrides take precedence over defaults but still sit below per-reason
// prefix matches (LPM) for that code.
func WithGRPCOverride(c code.Code, grpc int) Option {
	return func(b *builder) { b.grpcOverride[c] = grpc }
}

// WithHTTPPrefix adds an HTTP longest-prefix-match rule for the given code.
// The rule is evaluated against the reason (dot-separated). A more specific
// prefix wins. Use "*" to match a single segment.
func WithHTTPPrefix(c code.Code, prefix string, http int) Option {
	return func(b *builder) { b.httpPrefixes[c] = append(b.httpPrefixes[c], prefixRule{prefix, http}) }
}

// WithGRPCPrefix adds a gRPC longest-prefix-match rule for the given code.
// The rule is evaluated against the reason (dot-separated). A more specific
// prefix wins. Use "*" to match a single segment.
func WithGRPCPrefix(c code.Code, prefix string, grpc int) Option {
	return func(b *builder) { b.grpcPrefixes[c] = append(b.grpcPrefixes[c], prefixRule{prefix, grpc}) }
}
