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

// Package code provides parsing, normalization and validation for derrors error codes.
//
// A "code" is the top-level, machine-readable classification of an error,
// such as "invalid", "not_found", "conflict" or "internal". Codes are meant
// to be:
//
//   - short and stable;
//   - lowercased;
//   - underscore-separated (not dash-separated);
//   - suitable for use in JSON/proto payloads and for lookup in registries.
//
// IMPORTANT: Empty codes ("") are NOT allowed. Every error MUST have a
// non-empty code.
//
// This package defines the canonical representation and the functions that
// convert arbitrary user input to that canonical form.
package code
