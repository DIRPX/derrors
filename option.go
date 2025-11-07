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

import "dirpx.dev/derrors/reason"

// Option is a functional option for constructing or transforming an Error.
// It always takes an *Error and returns a (possibly new) *Error.
type Option func(*Error) *Error

// WithReasonOption sets the Reason on the error being constructed.
// Intended to be used with E(...).
func WithReasonOption(r reason.Reason) Option {
	return func(e *Error) *Error {
		return e.WithReason(r)
	}
}

// WithDetailOption adds a single detail key/value on construction.
// Intended to be used with E(...).
func WithDetailOption(k string, v any) Option {
	return func(e *Error) *Error {
		return e.WithDetail(k, v)
	}
}

// WithDetailsOption merges multiple detail key/values on construction.
// Intended to be used with E(...).
func WithDetailsOption(kv map[string]any) Option {
	return func(e *Error) *Error {
		return e.WithDetails(kv)
	}
}

// WithCauseOption attaches a cause on construction.
// Intended to be used with E(...).
func WithCauseOption(err error) Option {
	return func(e *Error) *Error {
		return e.WithCause(err)
	}
}
