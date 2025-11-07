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

// ErrorDescriptor is a flat, transport-friendly description of a known
// (code, reason) pair.
//
// This type intentionally uses strings (not the internal Code / Reason value
// types) so that it can live in the public "apis" layer and be used by
// adapters (HTTP, gRPC) and by user-defined registries.
//
// Implementations may choose to store a richer descriptor internally, but
// this shape is what the rest of the system can rely on.
type ErrorDescriptor struct {
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

	// HTTPStatus is an optional HTTP status that should be used when this
	// (code, reason) is exposed over HTTP. A value of 0 means "not specified".
	HTTPStatus int `json:"http_status,omitempty"`

	// GRPCCode is an optional gRPC status code (as integer) that should be
	// used when this (code, reason) is exposed over gRPC. A value of 0 means
	// "not specified".
	GRPCCode int `json:"grpc_code,omitempty"`

	// Message is an optional human-friendly default message or template that
	// can be used when the error instance itself did not provide one.
	Message string `json:"message,omitempty"`
}
