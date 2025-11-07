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
	"dirpx.dev/derrors/code"
	"dirpx.dev/derrors/mapper/internal/segmenttrie"
	"google.golang.org/grpc/codes"
)

// freezeHTTPDefaults makes an immutable copy of the HTTP defaults map.
// Used when finalizing the mapper so later mutations to the builder
// (or caller-owned maps) cannot affect the mapper.
func freezeHTTPDefaults(src map[code.Code]int) map[code.Code]int {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[code.Code]int, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// freezeGRPCDefaults makes an immutable copy of the gRPC defaults map,
// converting builder-style int values into typed gRPC codes.
func freezeGRPCDefaults(src map[code.Code]int) map[code.Code]codes.Code {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[code.Code]codes.Code, len(src))
	for k, v := range src {
		dst[k] = codes.Code(v)
	}
	return dst
}

// freezeHTTPOverrides makes an immutable copy of the HTTP overrides map.
func freezeHTTPOverrides(src map[code.Code]int) map[code.Code]int {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[code.Code]int, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// freezeGRPCOverrides makes an immutable copy of the gRPC overrides map,
// converting ints into typed gRPC codes.
func freezeGRPCOverrides(src map[code.Code]int) map[code.Code]codes.Code {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[code.Code]codes.Code, len(src))
	for k, v := range src {
		dst[k] = codes.Code(v)
	}
	return dst
}

// freezeHTTPTrie makes a shallow copy of the per-code HTTP tries.
// Each trie is considered immutable after build, so we only need to
// protect the top-level map.
func freezeHTTPTrie(src map[code.Code]*segmenttrie.Trie[int]) map[code.Code]*segmenttrie.Trie[int] {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[code.Code]*segmenttrie.Trie[int], len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// freezeGRPCTrie makes a shallow copy of the per-code gRPC tries.
func freezeGRPCTrie(src map[code.Code]*segmenttrie.Trie[codes.Code]) map[code.Code]*segmenttrie.Trie[codes.Code] {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[code.Code]*segmenttrie.Trie[codes.Code], len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// defaultHTTPOr makes an immutable copy of the given HTTP map.
// It is used at build time to detach the mapper from caller-provided maps.
func defaultHTTPOr(m map[code.Code]int) map[code.Code]int {
	if len(m) == 0 {
		return nil
	}
	out := make(map[code.Code]int, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// defaultGRPCOr makes an immutable copy of the given gRPC map.
//
// NOTE: the builder may temporarily keep gRPC codes as ints (for symmetry
// with HTTP). This helper normalizes them into map[code.Code]codes.Code.
func defaultGRPCOr(m map[code.Code]int) map[code.Code]codes.Code {
	if len(m) == 0 {
		return nil
	}
	out := make(map[code.Code]codes.Code, len(m))
	for k, v := range m {
		out[k] = codes.Code(v)
	}
	return out
}

// toGRPCMap converts a map of int gRPC codes into a typed gRPC map and
// copies it. This is useful when builder options stored values as int.
func toGRPCMap(m map[code.Code]int) map[code.Code]codes.Code {
	if len(m) == 0 {
		return nil
	}
	out := make(map[code.Code]codes.Code, len(m))
	for k, v := range m {
		out[k] = codes.Code(v)
	}
	return out
}

// nilIfEmptyInt normalizes an empty HTTP map to nil to reduce allocations
// and simplify nil checks in the mapper.
func nilIfEmptyInt(m map[code.Code]int) map[code.Code]int {
	if len(m) == 0 {
		return nil
	}
	return m
}

// nilIfEmptyTrieInt normalizes an empty HTTP trie map to nil.
func nilIfEmptyTrieInt(m map[code.Code]*segmenttrie.Trie[int]) map[code.Code]*segmenttrie.Trie[int] {
	if len(m) == 0 {
		return nil
	}
	return m
}

// nilIfEmptyTrieGRPC normalizes an empty gRPC trie map to nil.
func nilIfEmptyTrieGRPC(m map[code.Code]*segmenttrie.Trie[codes.Code]) map[code.Code]*segmenttrie.Trie[codes.Code] {
	if len(m) == 0 {
		return nil
	}
	return m
}
