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

// CodedError represents an error that is classified into a well-defined,
// machine-readable error *code*.
//
// A code usually denotes a broad category, such as:
//   - "invalid"        — validation failed,
//   - "not_found"      — a referenced object does not exist,
//   - "conflict"       — concurrent modification or version mismatch,
//   - "internal"       — unexpected server-side failure.
//
// Codes are intended to be stable and enumerable. They are the primary value
// that higher-level adapters (HTTP, gRPC) will use to decide which status
// code to return to the client.
//
// Implementations are expected to return a *canonicalized* code string — i.e.,
// normalized to the format enforced by the derrors/code package (lowercase,
// underscores, length limits, etc.). Adapters should treat unknown or empty
// codes as internal/server errors.
type CodedError interface {
	error

	// ErrorCode returns the machine-readable error code.
	//
	// The returned value MUST be non-empty and MUST already be normalized
	// according to the rules of the derrors subsystem. Callers should not try
	// to "fix" or "guess" the value here — if it's invalid, it should be
	// handled as an internal error at the boundary.
	ErrorCode() string
}

// ReasonedError represents an error that provides a more specific, contextual
// *reason* in addition to the high-level code.
//
// While the code answers the question "what kind of error is this?", the
// reason answers "which exact subcase of that error happened?".
//
// Examples:
//
//	code:   "invalid"
//	reason: "schema.group" -> the group name in schema is invalid
//
//	code:   "not_found"
//	reason: "resource.missing" -> the referenced resource does not exist
//
// Reasons are typically hierarchical, dot-separated strings, and are expected
// to be validated/normalized by the derrors/reason package.
//
// Having a separate interface for reasons allows code to gracefully degrade:
// if an error does not provide a reason, the caller can still act on the code.
type ReasonedError interface {
	error

	// ErrorReason returns the specific error reason.
	//
	// The returned value MAY be empty if the error does not provide a more
	// specific sub-classification. Callers should be prepared to handle the
	// empty case.
	ErrorReason() string
}

// DetailedError represents an error that exposes zero or more structured
// details. This is especially useful for validation scenarios where multiple
// fields may fail at once and the caller needs to show *all* of them.
//
// Implementations SHOULD return a slice that is safe to iterate over and that
// will not be modified by the callee. Returning nil is allowed and simply
// means "no extra details".
type DetailedError interface {
	error

	// ErrorDetails returns structured details of the error. May return nil.
	ErrorDetails() []Detail
}

// CausedError represents an error that exposes its underlying cause.
//
// While Go 1.13 introduced errors.Unwrap, having this interface in apis lets
// us work with wrapped errors even in places where we don't want to depend on
// errors.As / errors.Is directly, or where we want to keep the contract
// explicit.
//
// Implementations SHOULD return the direct, immediate cause of the error. If
// there is no underlying cause, they SHOULD return nil.
type CausedError interface {
	error

	// Cause returns the underlying error that triggered this error, if any.
	// May return nil.
	Cause() error
}
