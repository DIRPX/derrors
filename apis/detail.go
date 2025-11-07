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

package apis

// Detail represents a single structured piece of information attached to an
// error. This is a *view type* — small, transport-friendly, and suitable for
// JSON or proto mapper.
//
// We keep it in apis so that different parts of the system (validators,
// HTTP/gRPC adapters, loggers) can speak about "details" without importing
// the concrete error implementation.
//
// Typical usages:
//   - report which field failed validation;
//   - report expected vs actual values;
//   - report conflicting resource versions.
type Detail struct {
	// Type is a short classifier of the detail, e.g. "field", "conflict",
	// "extra", "missing", etc. Callers MAY leave it empty, but providing it
	// makes client-side handling simpler.
	Type string `json:"type,omitempty"`

	// Field carries the logical path to the failing field, e.g.
	// "metadata.name" or "spec.replicas". For non-field errors this may be
	// empty.
	Field string `json:"field,omitempty"`

	// Reason is a short, human-friendly explanation, e.g. "required",
	// "not_unique", "invalid_format". This is NOT the same as the top-level
	// error reason, but often corresponds to it.
	Reason string `json:"reason,omitempty"`

	// Info carries optional extra structured data (for example, allowed
	// values, maximum length, conflicting resource name, etc.). Keys and
	// values should be chosen so that they survive JSON/proto round-trips.
	Info map[string]string `json:"info,omitempty"`
}
