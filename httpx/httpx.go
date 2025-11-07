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

package httpx

import (
	"net/http"
	"strconv"

	"dirpx.dev/derrors"
	derrorsv1 "dirpx.dev/derrors/api/derrors/v1"
	"dirpx.dev/derrors/apis"
	"google.golang.org/protobuf/encoding/protojson"
)

// Meta carries extra context that the HTTP layer can add on top of derrors.Error.
// All fields are optional and typically come from request context, headers,
// rate-limiter output, or router-level logic.
type Meta struct {
	Correlation       string
	TraceID           string
	SpanID            string
	RetryAfterSeconds int32
	Links             []*derrorsv1.Link
	Fields            []*derrorsv1.Violation
}

// Writer is a thin adapter that knows how to turn a derrors.Error into an HTTP
// response using the provided status mapper.
type Writer struct {
	Mapper apis.Mapper
}

// Write serializes a View that conforms to error.view.schema.json and writes it
// to the response writer. The HTTP status is resolved via the Mapper.
//
// No automatic redaction or filtering is performed here: whatever is present
// in the error and Meta is exposed as-is. Higher-level handlers should apply
// policies if needed.
func (w Writer) Write(rw http.ResponseWriter, err *derrors.Error, meta Meta) {
	if err == nil {
		return
	}

	st := w.Mapper.Status(err.Code, err.Reason)

	view := &derrorsv1.ErrorView{
		Code:              string(err.Code),
		Message:           err.Message,
		Reason:            string(err.Reason),
		Correlation:       meta.Correlation,
		TraceId:           meta.TraceID,
		SpanId:            meta.SpanID,
		RetryAfterSeconds: meta.RetryAfterSeconds,
		Links:             meta.Links,
		Fields:            meta.Fields,
	}

	rw.Header().Set("Content-Type", "application/json")
	if meta.RetryAfterSeconds > 0 {
		rw.Header().Set("Retry-After", strconv.Itoa(int(meta.RetryAfterSeconds)))
	}
	rw.WriteHeader(st.HTTP)

	// IMPORTANT: protobuf JSON through protojson must be used to ensure
	// proper serialization of nested structures, field names (json_name),
	// and well-known types.
	b, _ := (protojson.MarshalOptions{
		EmitUnpopulated: false,
		UseProtoNames:   false, // use json_name
	}).Marshal(view)
	_, _ = rw.Write(b)
}
