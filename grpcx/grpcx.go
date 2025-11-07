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

package grpcx

import (
	"context"

	"dirpx.dev/derrors/apis"
	"google.golang.org/grpc"
	gcodes "google.golang.org/grpc/codes"
	gstatus "google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/anypb"

	"dirpx.dev/derrors"
	derrorsv1 "dirpx.dev/derrors/api/derrors/v1"
)

// Extras holds optional, rich metadata that can be embedded into
// derrors.v1.ErrorDescriptor. All fields are optional.
type Extras struct {
	// CorrelationID is a client/server correlation token (request ID, idempotency key).
	CorrelationID string

	// TraceID is the distributed trace identifier (W3C traceparent / OpenTelemetry).
	TraceID string

	// SpanID is the span identifier within the trace.
	SpanID string

	// Retry provides client retry/backoff hints.
	Retry *derrorsv1.RetryInfo

	// Quota describes current quota/limit state relevant to this error.
	Quota *derrorsv1.QuotaInfo

	// Violations lists input/validation problems (usually for 4xx).
	Violations []*derrorsv1.Violation

	// Links are human-facing links to docs/support/more info.
	Links []*derrorsv1.Link

	// Causes expose a shallow cause chain (safe subset).
	Causes []*derrorsv1.Cause

	// Env describes the runtime/deployment environment where the error happened.
	Env *derrorsv1.Environment

	// Tags are flat string key/value annotations.
	Tags []*derrorsv1.Tag
}

// MetaFn extracts Extras from context and the domain error.
// It can return an empty Extras if nothing is available.
type MetaFn func(ctx context.Context, e *derrors.Error) Extras

// UnaryServerInterceptor returns a gRPC UnaryServerInterceptor that
// maps derrors.Error into gRPC errors with rich derrors.v1.ErrorDescriptor details.
//
// The provided apis.Mapper is used to map domain error codes/reasons
// into transport status codes.
//
// The optional MetaFn can be used to extract additional metadata from context
// and the domain error to populate the ErrorDescriptor. If nil, no extra metadata
// will be added.
func UnaryServerInterceptor(m apis.Mapper, metaFn MetaFn) grpc.UnaryServerInterceptor {
	if metaFn == nil {
		metaFn = func(context.Context, *derrors.Error) Extras { return Extras{} }
	}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		resp, err := handler(ctx, req)
		if err == nil {
			return resp, nil
		}

		de, ok := err.(*derrors.Error)
		if !ok {
			// Not ours — return as-is.
			return nil, err
		}

		st := m.Status(de.Code, de.Reason)
		ex := metaFn(ctx, de)

		desc := &derrorsv1.ErrorDescriptor{
			// Core identity.
			Code:    string(de.Code),
			Reason:  string(de.Reason),
			Message: de.Message,

			// Transport projections.
			HttpStatus: int32(st.HTTP),
			GrpcCode:   int32(st.GRPC),

			// Correlation / tracing.
			CorrelationId: ex.CorrelationID,
			TraceId:       ex.TraceID,
			SpanId:        ex.SpanID,

			// Client hints.
			Retry:      ex.Retry,
			Quota:      ex.Quota,
			Violations: ex.Violations,

			// Human-facing + diagnostics.
			Links:  ex.Links,
			Causes: ex.Causes,
			Env:    ex.Env,
			Tags:   ex.Tags,
		}

		base := gstatus.New(gcodes.Code(st.GRPC), de.Message)

		// Try to attach descriptor as details. If it fails — return base.
		if anyDesc, err := anypb.New(desc); err == nil {
			if with, err := base.WithDetails(anyDesc); err == nil {
				return nil, with.Err()
			}
		}

		return nil, base.Err()
	}
}

// ExtractDescriptor pulls derrors.v1.ErrorDescriptor out of a gRPC error, if present.
// Useful in tests and client code.
func ExtractDescriptor(err error) (*derrorsv1.ErrorDescriptor, bool) {
	if err == nil {
		return nil, false
	}
	st, ok := gstatus.FromError(err)
	if !ok {
		return nil, false
	}
	for _, d := range st.Details() {
		if ed, ok := d.(*derrorsv1.ErrorDescriptor); ok {
			return ed, true
		}
	}
	return nil, false
}
