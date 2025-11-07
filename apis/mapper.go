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

import (
	"dirpx.dev/derrors/code"
	"dirpx.dev/derrors/reason"
	"google.golang.org/grpc/codes"
)

// Mapper is an immutable, concurrency-safe view of the mapper rules.
// It resolves a logical derrors code (and optionally a reason) into transport
// statuses for HTTP and gRPC.
type Mapper interface {
	// HTTPStatus returns the HTTP status code for the given error code and reason.
	// If no reason-specific rule exists, the mapper must fall back to the code-level rule.
	HTTPStatus(c code.Code, r reason.Reason) int

	// GRPCStatus returns the gRPC status code for the given error code and reason.
	// If no reason-specific rule exists, the mapper must fall back to the code-level rule.
	GRPCStatus(c code.Code, r reason.Reason) codes.Code

	// Status resolves both HTTP and gRPC in a single call, using the same matching logic.
	Status(c code.Code, r reason.Reason) Status

	// Explain returns a human-readable description of which rule matched.
	// Implementations may return an empty string in production builds.
	Explain(c code.Code, r reason.Reason) string
}

// Status represents a resolved pair of transport statuses for a single error.
// It is the final output of the mapper and can be written directly to HTTP/gRPC.
type Status struct {
	HTTP int        // Resolved HTTP status code (net/http compatible).
	GRPC codes.Code // Resolved gRPC status code.
}
