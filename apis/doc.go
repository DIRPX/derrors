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

// Package apis defines the public Go-level contracts for dirpx error handling.
//
// The goal of this package is to provide *small, composable* interfaces that
// other dirpx packages can depend on without importing the concrete error
// implementation (which will live in other subpackages, e.g. derrors/errors,
// derrors/code, derrors/reason, etc.).
//
// In other words: this package is the "surface" that HTTP adapters, gRPC
// adapters, validation code and business logic can target. Concrete error
// types should implement these interfaces, but callers should not rely on the
// concrete types.
//
// This package must remain lightweight and should not introduce heavy
// dependencies, so it only contains interfaces and very small view types.
package apis
