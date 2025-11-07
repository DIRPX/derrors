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

package adapter

import (
	"dirpx.dev/derrors"
	"dirpx.dev/derrors/apis"
)

// ToDescriptor converts a domain-level error together with its resolved
// transport status into a portable ErrorDescriptor.
//
// The descriptor is intended for structured logging, tracing, or message bus
// propagation. It carries both the logical code/reason and the concrete
// transport statuses (HTTP and gRPC).
func ToDescriptor(e *derrors.Error, st apis.Status) apis.ErrorDescriptor {
	if e == nil {
		return apis.ErrorDescriptor{}
	}
	return apis.ErrorDescriptor{
		Code:       string(e.Code),
		Reason:     string(e.Reason),
		HTTPStatus: st.HTTP,
		GRPCCode:   int(st.GRPC),
		Message:    e.Message,
	}
}

// ToView converts a domain-level error into a public ErrorView using the
// resolved status. This function performs no automatic redaction or filtering;
// it exposes exactly what the error instance contains.
//
// If the underlying error implements apis.DetailedError, its details are
// copied into the view as-is. It is up to the caller or API layer to decide
// whether to redact or filter sensitive fields.
func ToView(e *derrors.Error, st apis.Status) apis.ErrorView {
	if e == nil {
		return apis.ErrorView{}
	}
	v := apis.ErrorView{
		Code:    string(e.Code),
		Reason:  string(e.Reason),
		Message: e.Message,
	}
	// If the error provides structured details, propagate them directly.
	if de, ok := any(e).(apis.DetailedError); ok {
		if ds := de.ErrorDetails(); len(ds) > 0 {
			v.Details = ds
		}
	}
	return v
}
