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

// defaultHTTP defines the library's built-in HTTP mappings for well-known error codes.
// These are only defaults: callers are expected to wrap or override them at the boundary
// where HTTP is actually produced (REST gateway, HTTP handler, etc.).
//
// The intent is to stay close to common REST conventions, but still reflect
// higher-level dirpx error semantics (unavailable, draining, throttled, etc.).
var defaultHTTP = map[code.Code]int{
	// 5xx — server / dependency / transient issues.
	code.Internal:         http.StatusInternalServerError, // Generic internal failure; do not expose internal details.
	code.Unavailable:      http.StatusServiceUnavailable,  // Service or a required dependency is temporarily unreachable.
	code.NotReady:         http.StatusServiceUnavailable,  // Service exists but is not yet ready to serve traffic.
	code.Draining:         http.StatusServiceUnavailable,  // Service is deliberately refusing new work (shutting down, draining).
	code.Overloaded:       http.StatusServiceUnavailable,  // Service cannot accept more requests right now.
	code.DependencyFailed: http.StatusBadGateway,          // Upstream/downstream dependency failed in a way visible to the client.
	code.Timeout:          http.StatusGatewayTimeout,      // Operation exceeded the time budget.
	// Note: 499 is a non-standard but widely used code (nginx) for "client closed request".
	// We use it here for canceled by default; integrators may switch to 408.
	code.Canceled: http.StatusRequestTimeout,

	// 4xx — client/protocol/resource issues.
	code.Invalid:             http.StatusBadRequest, // Malformed input, validation errors, contract violation.
	code.Missing:             http.StatusBadRequest, // Required field/parameter/resource reference is missing.
	code.Unsupported:         http.StatusBadRequest, // Known but unsupported operation/content/option.
	code.Expired:             http.StatusBadRequest, // Resource or token has expired; client may refresh and retry.
	code.DeprecationRejected: http.StatusBadRequest, // Client used a deprecated/removed feature or version; request must be changed.
	code.TooEarly:            http.StatusTooEarly,   // Client is too early (e.g. before a readiness or scheduled time).

	code.NotFound: http.StatusNotFound, // Target resource does not exist (or is not visible to the caller).
	code.Gone:     http.StatusGone,     // Resource used to exist but is permanently gone.

	// Conflicts and concurrency.
	code.AlreadyExists:      http.StatusConflict,           // Resource creation clash — it already exists.
	code.Conflict:           http.StatusConflict,           // General conflicting update/action.
	code.StaleVersion:       http.StatusConflict,           // Optimistic lock / resource version mismatch.
	code.PreconditionFailed: http.StatusPreconditionFailed, // If-Match / preconditions failed on the resource.

	// AuthN / AuthZ.
	code.Unauthenticated:    http.StatusUnauthorized, // No/invalid credentials — caller must authenticate.
	code.InvalidCredentials: http.StatusUnauthorized, // Credentials were present but invalid (revoked, bad token, etc.).
	code.TokenInvalid:       http.StatusUnauthorized, // Token was present but malformed, unknown, or failed validation.
	code.TokenExpired:       http.StatusUnauthorized, // Token is structurally OK but its lifetime is over; client must re-auth.
	code.TokenRevoked:       http.StatusUnauthorized, // Token was explicitly revoked by the issuer/policy; cannot be reused.
	code.SessionExpired:     http.StatusUnauthorized, // Session-based auth expired; client must establish a new session/login.
	code.PermissionDenied:   http.StatusForbidden,    // Caller is authenticated but not allowed to perform the action.

	// Rate/quotas.
	code.Throttled:     http.StatusTooManyRequests, // Server is asking the client to slow down (burst protection).
	code.RateLimited:   http.StatusTooManyRequests, // Client hit a rate limit.
	code.QuotaExceeded: http.StatusTooManyRequests, // Client exceeded an allocated quota.
}

// defaultGRPC defines the library's built-in gRPC mappings for well-known error codes.
// These values are chosen to align with canonical gRPC status codes while still preserving
// higher-level meanings from derrors. As with HTTP, callers may override these at the
// transport edge if a different mapper policy is required.
var defaultGRPC = map[code.Code]codes.Code{
	// Internal / server-side / unexpected.
	code.Internal: codes.Internal,

	// Input / preconditions / protocol.
	code.Invalid:            codes.InvalidArgument,    // Bad input shape or validation errors.
	code.Missing:            codes.InvalidArgument,    // Required field or parameter missing.
	code.Unsupported:        codes.InvalidArgument,    // Unsupported option/content.
	code.PreconditionFailed: codes.FailedPrecondition, // Resource preconditions not met.
	code.DependencyFailed:   codes.FailedPrecondition, // Dependency cannot satisfy the request right now.
	code.Expired:            codes.FailedPrecondition, // Token/resource expired.
	code.TooEarly:           codes.FailedPrecondition, // Request made before allowed time.

	// Resource state / versioning.
	code.NotFound:            codes.NotFound,           // Resource does not exist (or not visible).
	code.Gone:                codes.NotFound,           // gRPC has no 410; NotFound is the closest practical choice.
	code.DeprecationRejected: codes.FailedPrecondition, // Server no longer accepts this version/feature; client must adjust.

	// Conflicts / concurrency.
	code.AlreadyExists: codes.AlreadyExists, // Create on existing resource.
	code.Conflict:      codes.Aborted,       // General conflict (concurrent updates, etc.).
	code.StaleVersion:  codes.Aborted,       // Optimistic concurrency error.

	// AuthN / AuthZ.
	code.Unauthenticated:    codes.Unauthenticated,
	code.InvalidCredentials: codes.Unauthenticated, // Credentials present but invalid — still unauthenticated.
	code.TokenInvalid:       codes.Unauthenticated, // Caller is not authenticated because the token is invalid.
	code.TokenExpired:       codes.Unauthenticated, // Caller is not authenticated because the token is expired.
	code.TokenRevoked:       codes.Unauthenticated, // Caller is not authenticated because the token was revoked.
	code.SessionExpired:     codes.Unauthenticated, // Caller is not authenticated because the session has ended.
	code.PermissionDenied:   codes.PermissionDenied,

	// Availability / load / lifecycle.
	code.Unavailable: codes.Unavailable, // Service or dependency temporarily unavailable.
	code.NotReady:    codes.Unavailable, // Service exists but cannot serve yet.
	code.Draining:    codes.Unavailable, // Service refuses work intentionally.
	code.Overloaded:  codes.Unavailable, // Backpressure / resource exhaustion on server.

	// Time / cancellation.
	code.Timeout:  codes.DeadlineExceeded, // Time budget exceeded.
	code.Canceled: codes.Canceled,         // Caller canceled or context expired upstream.

	// Rate/quotas.
	code.Throttled:     codes.ResourceExhausted, // Server asks the client to back off.
	code.RateLimited:   codes.ResourceExhausted, // Rate limit hit.
	code.QuotaExceeded: codes.ResourceExhausted, // Quota exhausted.
}
