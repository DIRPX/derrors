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

package code

// Core / generic domain error codes
//
// These codes describe high-level, transport-agnostic error classes that your
// business logic and validation code will use most often.
const (
	// Internal indicates an internal, non-classified service/domain error.
	// Use this as the fallback when no more specific domain code applies.
	// The root cause is typically attached as the error cause; a Reason is
	// usually not provided.
	//
	// Transport mapper is framework-specific.
	// Can be mapped to an HTTP 500.
	Internal Code = "internal"

	// Invalid indicates that the input value, entity, or payload violates
	// a structural or semantic invariant.
	// Use this when the format, range, charset, pattern, or cross-field
	// consistency is wrong.
	// Often paired with reasons like: ReasonPattern, ReasonCharset,
	// ReasonMismatch, ReasonTooShort, ReasonTooLong.
	//
	// Transport mapper is framework-specific.
	// Can be mapped to an HTTP 400.
	Invalid Code = "invalid"

	// Missing indicates that a required value or structure is absent.
	// Use this when a field, parameter, header, or related object is empty,
	// nil, or not supplied at all.
	// Often paired with ReasonEmpty or ReasonZeroValue.
	//
	// Transport mapper is framework-specific.
	// Can be mapped to an HTTP 400.
	Missing Code = "missing"

	// Unsupported indicates that the requested operation, value, feature,
	// or configuration is not supported in the current runtime or policy.
	// Use this for disabled providers, disallowed algorithms, or unavailable
	// capability flags.
	// Often paired with ReasonUnsupported or ReasonNotAllowed.
	//
	// Transport mapper is framework-specific.
	// Can be mapped to an HTTP 400 or 501 depending on policy.
	Unsupported Code = "unsupported"
)

// Runtime / operation control error codes
//
// These codes describe transient, operational conditions that affect
// the ability to complete the requested operation.
const (
	// Unavailable indicates that a required downstream dependency or
	// service is temporarily unreachable.
	// Use this for DB outages, network partitions, or unavailable IDPs.
	// The underlying technical cause should be carried in the error cause.
	// Often paired with ReasonNetwork or ReasonDependency.
	//
	// Transport mapper is framework-specific.
	// Can be mapped to an HTTP 503.
	Unavailable Code = "unavailable"

	// Timeout indicates that the operation could not complete within the
	// allotted time budget.
	// Use this for exceeded RPC/DB timeouts; the cause may be
	// context.DeadlineExceeded or similar.
	// Often paired with ReasonTimeout.
	//
	// Transport mapper is framework-specific.
	// Can be mapped to an HTTP 504.
	Timeout Code = "timeout"

	// Canceled indicates that the operation was explicitly canceled by
	// the caller or by context propagation.
	// Use this when work was in progress but needs to stop.
	// Often paired with ReasonCanceled.
	//
	// Transport mapper is framework-specific.
	// Can be mapped to an HTTP 499 (client closed request) or 408 depending on policy.
	Canceled Code = "canceled"

	// DependencyFailed indicates that the operation could not complete
	// because a required sub-operation or dependency call failed, even
	// though the dependency itself was reachable.
	// Use this when a downstream returned an application-level failure
	// that makes continuing impossible.
	// Often paired with ReasonDependency or a more specific reason.
	//
	// Transport mapper is framework-specific.
	// Can be mapped to an HTTP 502 or 424 depending on policy.
	DependencyFailed Code = "dependency_failed"

	// NotReady indicates that the target component is alive but not yet
	// ready to serve traffic.
	// Use this during startup, warm-up, cache priming, or rolling updates
	// when readiness probes would fail.
	// Often paired with ReasonInitializing or ReasonNotReady.
	//
	// Transport mapper is framework-specific.
	// Can be mapped to an HTTP 503.
	NotReady Code = "not_ready"

	// Draining indicates that the component is intentionally leaving service
	// (draining connections, shutting down, or being removed from rotation)
	// and therefore refuses new work.
	// Often paired with ReasonShutdown or ReasonMaintenance.
	//
	// Transport mapper is framework-specific.
	// Can be mapped to an HTTP 503.
	Draining Code = "draining"

	// Overloaded indicates that the service or node is currently under
	// excessive load and cannot accept or process the request efficiently.
	// Use this when queues are full, worker pools are saturated, or global
	// concurrency limits are reached.
	// Often paired with ReasonOverload or ReasonResourceExhausted.
	//
	// Transport mapper is framework-specific.
	// Can be mapped to an HTTP 503.
	Overloaded Code = "overloaded"

	// Throttled indicates that the caller is being temporarily restricted
	// from performing the requested operation due to domain-specific
	// throttling policies.
	// Use this for business-level throttles that are not strictly rate
	// limits (e.g., temporary holds, soft bans, or cooldown periods).
	// Often paired with ReasonThrottled.
	//
	// Transport mapper is framework-specific.
	// Can be mapped to an HTTP 429.
	Throttled Code = "throttled"
)

// Resource / state / concurrency error codes
//
// These codes describe common CRUD / resource-level conditions.
const (
	// NotFound indicates that the requested entity does not exist in the
	// current domain scope or storage.
	// Use this for lookups by ID, name, key, or reference.
	// A reason may be omitted, or augmented with ReasonInvalidValue when the
	// lookup key itself is malformed.
	//
	// Transport mapper is framework-specific.
	// Can be mapped to an HTTP 404.
	NotFound Code = "not_found"

	// AlreadyExists indicates that the target entity cannot be created
	// because an entity with the same primary identity already exists.
	// This is semantically different from Conflict: here the server
	// clearly says “this identity is already taken”.
	//
	// Transport mapper is framework-specific.
	// Often mapped to HTTP 409.
	AlreadyExists Code = "already_exists"

	// Conflict indicates a domain-state conflict or uniqueness violation.
	// Use this for version mismatches, concurrent updates, and collisions
	// that are not strictly “already exists” cases.
	// Often paired with ReasonDuplicate or ReasonMismatch.
	//
	// Transport mapper is framework-specific.
	// Can be mapped to an HTTP 409.
	Conflict Code = "conflict"

	// PreconditionFailed indicates that the operation could not proceed
	// because a required precondition was not met (for example, version/ETag
	// mismatch, resource not in the expected state).
	// Use this when the client must re-fetch or re-evaluate current state
	// before retrying.
	//
	// Transport mapper is framework-specific.
	// Can be mapped to an HTTP 412.
	PreconditionFailed Code = "precondition_failed"

	// Gone indicates that the resource existed before but is no longer
	// available (deleted, retired, deprovisioned).
	// Use this when the resource is known to have existed but cannot
	// be accessed anymore.
	//
	// Transport mapper is framework-specific.
	// Can be mapped to an HTTP 410.
	Gone Code = "gone"

	// StaleVersion indicates that the client sent a version/resource that is
	// no longer current (different from precondition_failed when the client
	// expressed a specific precondition).
	// Useful when you want to hint the client to re-fetch.
	//
	// Transport mapper is framework-specific.
	// Can be mapped to an HTTP 409 or 412 depending on policy.
	StaleVersion Code = "stale_version"

	// DeprecationRejected indicates that the caller tried to use a deprecated
	// or removed feature/version that is no longer accepted by the server.
	//
	// Transport mapper is framework-specific.
	// This can be surfaced as 400 or 410 depending on API policy.
	DeprecationRejected Code = "deprecation_rejected"
)

// Authentication / authorization / tokens
//
// These codes let you distinguish "no auth", "bad auth", and "not allowed".
// This is important because HTTP has 401 vs 403, and gRPC has Unauthenticated
// vs PermissionDenied.
const (
	// Unauthenticated indicates that the caller is not authenticated or
	// the authentication context could not be established.
	// Use this when there is no valid session/token, or when the identity
	// cannot be verified at all.
	// Different from InvalidCredentials, which is about *failed* credential
	// verification when an auth attempt was made.
	//
	// Transport mapper is framework-specific.
	// Can be mapped to an HTTP 401 or gRPC 16 Unauthenticated.
	Unauthenticated Code = "unauthenticated"

	// InvalidCredentials indicates that the provided primary credentials
	// are not valid for the claimed principal.
	// Use this for bad username/password, client secret, OTP, or confirmation
	// code.
	// Different from Unauthenticated: here we *do* have an auth attempt,
	// but it failed.
	//
	// Transport mapper is framework-specific.
	// Can be mapped to an HTTP 401 or gRPC 16 Unauthenticated.
	InvalidCredentials Code = "invalid_credentials"

	// PermissionDenied indicates that the caller is authenticated but does
	// not have sufficient privileges to perform the target operation.
	// Use this for RBAC/ABAC/tenant isolation failures.
	// Often paired with ReasonNotAllowed or ReasonScope.
	//
	// Transport mapper is framework-specific.
	// Can be mapped to an HTTP 403 or gRPC 7 PermissionDenied.
	PermissionDenied Code = "permission_denied"

	// TokenInvalid indicates that a token is present but does not pass
	// format/signature/claims validation.
	// Use this for JWT/OAuth/internal tokens with bad signatures, issuers,
	// audiences, KID, or algorithms.
	// Often paired with ReasonSignature, ReasonClaims, ReasonIssuer,
	// ReasonAudience, ReasonKid, ReasonAlgorithm.
	//
	// Transport mapper is framework-specific.
	// Can be mapped to an HTTP 401 or 440 depending on policy.
	TokenInvalid Code = "token_invalid"

	// TokenExpired indicates that the token was otherwise valid but is
	// past its expiration/validity time.
	// Use this for exp/nbf checks and token freshness policies.
	// Often paired with ReasonExpired or ReasonNotBefore / ReasonClockSkew.
	//
	// Transport mapper is framework-specific.
	// Can be mapped to an HTTP 401 or 440 depending on policy.
	TokenExpired Code = "token_expired"

	// TokenRevoked indicates that the token has been actively revoked by
	// domain policy.
	// Use this for forced session/logout/invalidation scenarios.
	// Often paired with ReasonRevoked.
	//
	// Transport mapper is framework-specific.
	// Can be mapped to an HTTP 401 or 440 depending on policy.
	TokenRevoked Code = "token_revoked"

	// SessionExpired indicates that the server-side or stateful session
	// used to authenticate the caller has expired.
	// Use this for cookie/session storage/state objects that have their own
	// TTL, distinct from token TTLs.
	// Different from TokenExpired, which is about cryptographic tokens.
	//
	// Transport mapper is framework-specific.
	// Can be mapped to an HTTP 440 (Microsoft extension) or 401 depending on policy.
	SessionExpired Code = "session_expired"
)

// Rate, quota, lifecycle / time-based
//
// These codes describe throttling, quotas and time windows.
const (
	// Expired indicates that the target domain object is no longer valid
	// due to time-based expiration.
	// Use this for challenges, confirmation links, one-time tokens, or any
	// business entity with a TTL.
	// Often paired with ReasonExpired or ReasonNotBefore / ReasonClockSkew.
	//
	// Transport mapper is framework-specific.
	// Can be mapped to an HTTP 410 or 400 depending on policy.
	Expired Code = "expired"

	// TooEarly indicates that the server is unwilling to risk processing a
	// request that might be replayed (idempotency/0-RTT related).
	// Use this for early replays, premature retries, or operations that
	// are attempted before their allowed time window.
	// Often paired with ReasonTooEarly.
	//
	// Transport mapper is framework-specific.
	// Can be mapped to an HTTP 425.
	TooEarly Code = "too_early"

	// RateLimited indicates that the caller has exceeded the allowed
	// request or action rate in the current time window.
	// Use this for MFA attempts, authentication throttling, or integration
	// anti-abuse logic. Transport layers may add retry metadata.
	// Often paired with ReasonRateLimited.
	//
	// Transport mapper is framework-specific.
	// Can be mapped to an HTTP 429.
	RateLimited Code = "rate_limited"

	// QuotaExceeded indicates that the caller or tenant exceeded a configured
	// resource quota (objects, bytes, connections, etc).
	// Use this for storage limits, user counts, API usage quotas, or
	// any other domain-specific quota.
	//
	// Transport mapper is framework-specific.
	// Often mapped to 429 or 403 depending on policy.
	QuotaExceeded Code = "quota_exceeded"
)
