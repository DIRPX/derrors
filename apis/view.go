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

// ViewProvider is implemented by errors that can produce a transport-friendly,
// self-contained representation of themselves.
//
// This is useful for HTTP/gRPC adapters that want to send "the canonical form"
// of the error to the client without having to know about the concrete error
// type.
//
// The returned view MUST be safe to marshal (to JSON/proto) and SHOULD contain
// all information that is safe to disclose to the client.
type ViewProvider interface {
	error

	// ErrorView returns a transport-friendly snapshot of the error.
	ErrorView() ErrorView
}

// ErrorView is a minimal, serializable representation of an error.
//
// This is *not* the concrete error type used internally — it is the shape that
// we are comfortable exposing over the wire or logging. Keeping it here (in
// apis) allows both HTTP and gRPC adapters to share the same struct.
type ErrorView struct {
	// Code is the canonical error code, e.g. "invalid", "not_found",
	// "already_exists".
	//
	// Implementations SHOULD store only normalized, validated codes here.
	Code string `json:"code"`
	// Reason is the more specific sub-classification, e.g. "schema.group",
	// "resource.missing".
	//
	// It MAY be empty when the descriptor applies to the whole code.
	// Implementations SHOULD store only normalized, validated reasons here.
	Reason string `json:"reason,omitempty"`
	// Message is an optional human-friendly message.
	//
	// This is typically either the error's own message or a default message
	// taken from the descriptor.
	Message string `json:"message,omitempty"`
	// Details is an optional list of additional details about the error.
	//
	// The exact shape of each detail is implementation-specific.
	Details []Detail `json:"details,omitempty"`
}
