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

// Package reason defines an optional, structured refinement for derrors codes.
//
// Where Code answers “what kind of error is this?” (invalid, not_found,
// unavailable, ...), Reason can answer “where / in what operation / in what
// component this happened?”, e.g.:
//
//   - "apimachinery.schema.gvk.parse"
//   - "storage.pg.connect"
//   - "auth.jwt.verify"
//
// Reason is intentionally optional: the zero value ("") is allowed and
// indicates that no further refinement is provided. This lets callers attach
// a reason only when they actually have a meaningful, stable one to report.
package reason
