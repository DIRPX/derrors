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

package derrors

import (
	"fmt"

	"dirpx.dev/derrors/code"
	"dirpx.dev/derrors/reason"
)

// Error is the canonical rich error type for dirpx.
//
// It carries:
//   - Code: high-level, normalized error code (required);
//   - Reason: optional, more specific machine-friendly cause;
//   - Message: human-oriented description (what went wrong);
//   - Details: arbitrary key/value payload (for logging / HTTP body);
//   - Cause: wrapped underlying error for debugging / unwrapping.
//
// All mutation helpers (WithX) return a shallow copy, so Error instances
// can be safely shared and modified in a functional style.
type Error struct {
	// Code is the primary classification of the error, e.g. "unavailable",
	// "invalid", "not_found". Must be a normalized code from derrors/code.
	Code code.Code

	// Reason refines the Code with a machine-usable marker, e.g.
	// "storage.pg.connect_timeout" or "auth.jwt.expired".
	// May be empty when the Code is descriptive enough.
	Reason reason.Reason

	// Message is a human-readable explanation. This is what should end up
	// in logs or in the "message" field of an HTTP error response.
	Message string

	// Details is an optional, shallow map of extra fields. Use this to expose
	// structured error data to API clients (ids, limits, resource names, etc.).
	// The map is treated as immutable: WithDetail/WithDetails always copy it.
	Details map[string]any

	// Cause holds the wrapped underlying error (if any). This is used for
	// errors.Is / errors.As and for debugging in lower layers.
	Cause error
}

// E is a convenience constructor for Error.
//
// Usage:
//
//	return derrors.E(code.Unavailable, "storage is down",
//	    derrors.WithReasonOption("storage.pg.connect_timeout"),
//	    derrors.WithDetailOption("host", "db:5432"),
//	)
//
// It always returns a *new* Error and applies all provided options in order.
func E(c code.Code, msg string, opts ...Option) *Error {
	e := &Error{Code: c, Message: msg}
	for _, opt := range opts {
		e = opt(e)
	}
	return e
}

// Error implements the built-in error interface.
//
// The format is:
//
//	<code>: <message>
//
// or, when Reason is present:
//
//	<code>:<reason>: <message>
//
// This makes the error both human- and machine-scannable in logs.
func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Reason != "" {
		return fmt.Sprintf("%s:%s: %s", e.Code, e.Reason, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause, enabling errors.Is / errors.As chains.
func (e *Error) Unwrap() error { return e.Cause }

// WithReason returns a shallow copy of e with the given Reason set.
// The original error is not modified.
func (e *Error) WithReason(r reason.Reason) *Error {
	cp := *e
	cp.Reason = r
	return &cp
}

// WithMessage returns a shallow copy of e with a replaced human message.
// Useful when you want to keep the Code/Reason but present the message
// in a different language or context.
func (e *Error) WithMessage(msg string) *Error {
	cp := *e
	cp.Message = msg
	return &cp
}

// WithDetail returns a shallow copy of e with one extra key/value in Details.
//
// The method always copies the map to preserve immutability. This prevents
// surprising modifications across goroutines or shared error values.
func (e *Error) WithDetail(k string, v any) *Error {
	cp := *e
	// No details yet — create a new single-entry map.
	if len(cp.Details) == 0 {
		cp.Details = map[string]any{k: v}
		return &cp
	}
	// Copy existing details and add one more.
	m := make(map[string]any, len(cp.Details)+1)
	for k0, v0 := range cp.Details {
		m[k0] = v0
	}
	m[k] = v
	cp.Details = m
	return &cp
}

// WithDetails returns a shallow copy of e with all provided kv merged into Details.
//
// If the Error already has Details, both maps are copied and merged,
// with kv taking precedence on key conflicts.
func (e *Error) WithDetails(kv map[string]any) *Error {
	if len(kv) == 0 {
		return e
	}
	cp := *e
	// No existing details — just copy kv.
	if len(cp.Details) == 0 {
		m := make(map[string]any, len(kv))
		for k, v := range kv {
			m[k] = v
		}
		cp.Details = m
		return &cp
	}
	// Merge existing + new.
	m := make(map[string]any, len(cp.Details)+len(kv))
	for k0, v0 := range cp.Details {
		m[k0] = v0
	}
	for k, v := range kv {
		m[k] = v
	}
	cp.Details = m
	return &cp
}

// WithCause returns a shallow copy of e with the given underlying cause attached.
// If err is nil, the original error is returned unchanged.
func (e *Error) WithCause(err error) *Error {
	if err == nil {
		return e
	}
	cp := *e
	cp.Cause = err
	return &cp
}
