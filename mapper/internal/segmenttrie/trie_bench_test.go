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

package segmenttrie

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"
)

// genValidSegment returns a valid segment: [a-z][a-z0-9_]*
func genValidSegment(rng *rand.Rand, min, max int) string {
	n := min + rng.Intn(max-min+1)
	if n < 1 {
		n = 1
	}
	var b strings.Builder
	// first char: [a-z]
	b.WriteByte(byte('a' + rng.Intn(26)))
	// rest: [a-z0-9_]
	for i := 1; i < n; i++ {
		switch rng.Intn(3) {
		case 0:
			b.WriteByte(byte('a' + rng.Intn(26)))
		case 1:
			b.WriteByte(byte('0' + rng.Intn(10)))
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}

// makePrefix builds a dot-separated prefix with optional single-segment wildcards ("*")
// every k segments (if k>0). depth = number of segments.
func makePrefix(rng *rand.Rand, depth int, wildcardEveryK int) string {
	segs := make([]string, depth)
	for i := 0; i < depth; i++ {
		if wildcardEveryK > 0 && (i+1)%wildcardEveryK == 0 {
			segs[i] = "*"
			continue
		}
		segs[i] = genValidSegment(rng, 3, 8)
	}
	return strings.Join(segs, ".")
}

// buildTrie inserts N prefixes of fixed depth into the trie and also returns
// a matching query set (reasons) that are likely to hit via LPM.
func buildTrie(b *testing.B, N, depth, wildcardEveryK int) (*Trie[int], []string) {
	rng := rand.New(rand.NewSource(1)) // deterministic
	tr := New[int]()
	reasons := make([]string, 0, N)

	for i := 0; i < N; i++ {
		p := makePrefix(rng, depth, wildcardEveryK)
		if err := tr.Insert(p, 100+i); err != nil {
			b.Fatalf("insert failed for %q: %v", p, err)
		}

		// build a reason that extends the prefix by +2 segments to test LPM
		ext := p
		if wildcardEveryK > 0 {
			// replace wildcards with some segment to form a valid reason
			parts := strings.Split(ext, ".")
			for j := range parts {
				if parts[j] == "*" {
					parts[j] = genValidSegment(rng, 3, 8)
				}
			}
			ext = strings.Join(parts, ".")
		}
		ext = ext + "." + genValidSegment(rng, 3, 8) + "." + genValidSegment(rng, 3, 8)
		reasons = append(reasons, ext)
	}

	return tr, reasons
}

// ------- INSERT benchmarks -------

func BenchmarkTrieInsert_N16_Depth4_NoWildcard(b *testing.B)   { benchInsert(b, 16, 4, 0) }
func BenchmarkTrieInsert_N128_Depth4_NoWildcard(b *testing.B)  { benchInsert(b, 128, 4, 0) }
func BenchmarkTrieInsert_N1024_Depth4_NoWildcard(b *testing.B) { benchInsert(b, 1024, 4, 0) }
func BenchmarkTrieInsert_N4096_Depth4_NoWildcard(b *testing.B) { benchInsert(b, 4096, 4, 0) }

func BenchmarkTrieInsert_N1024_Depth4_WildcardEvery3(b *testing.B) { benchInsert(b, 1024, 4, 3) }
func BenchmarkTrieInsert_N1024_Depth8_NoWildcard(b *testing.B)     { benchInsert(b, 1024, 8, 0) }

func benchInsert(b *testing.B, N, depth, wildcardEveryK int) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	prefixes := make([]string, N)
	for i := 0; i < N; i++ {
		prefixes[i] = makePrefix(rng, depth, wildcardEveryK)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tr := New[int]()
		for j := 0; j < N; j++ {
			if err := tr.Insert(prefixes[j], j); err != nil {
				b.Fatalf("insert failed: %v", err)
			}
		}
	}
}

// ------- MATCH benchmarks (sequential) -------

func BenchmarkTrieMatch_N16_Depth4_NoWildcard(b *testing.B)   { benchMatch(b, 16, 4, 0) }
func BenchmarkTrieMatch_N128_Depth4_NoWildcard(b *testing.B)  { benchMatch(b, 128, 4, 0) }
func BenchmarkTrieMatch_N1024_Depth4_NoWildcard(b *testing.B) { benchMatch(b, 1024, 4, 0) }
func BenchmarkTrieMatch_N4096_Depth4_NoWildcard(b *testing.B) { benchMatch(b, 4096, 4, 0) }

func BenchmarkTrieMatch_N1024_Depth4_WildcardEvery3(b *testing.B) { benchMatch(b, 1024, 4, 3) }
func BenchmarkTrieMatch_N1024_Depth8_NoWildcard(b *testing.B)     { benchMatch(b, 1024, 8, 0) }

func benchMatch(b *testing.B, N, depth, wildcardEveryK int) {
	tr, reasons := buildTrie(b, N, depth, wildcardEveryK)

	// add a few negative queries (no match)
	rng := rand.New(rand.NewSource(2))
	for i := 0; i < N/8+1; i++ {
		reasons = append(reasons, makePrefix(rng, depth, 0)+"."+genValidSegment(rng, 3, 8))
	}

	b.ReportAllocs()
	b.ResetTimer()
	idx := 0
	var sum int // prevent DCE
	for i := 0; i < b.N; i++ {
		r := reasons[idx]
		if v, ok := tr.Match(r); ok {
			sum += v
		}
		idx++
		if idx == len(reasons) {
			idx = 0
		}
	}
	// use sum
	if sum == 42 {
		b.Log("keep")
	}
}

// ------- MATCH benchmarks (parallel) -------

func BenchmarkTrieMatchParallel_N1024_Depth4_NoWildcard(b *testing.B) {
	tr, reasons := buildTrie(b, 1024, 4, 0)
	benchMatchParallel(b, tr, reasons)
}

func BenchmarkTrieMatchParallel_N4096_Depth4_WildcardEvery3(b *testing.B) {
	tr, reasons := buildTrie(b, 4096, 4, 3)
	benchMatchParallel(b, tr, reasons)
}

func benchMatchParallel(b *testing.B, tr *Trie[int], reasons []string) {
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		rng := rand.New(rand.NewSource(int64(rand.Int())))
		for pb.Next() {
			r := reasons[rng.Intn(len(reasons))]
			_, _ = tr.Match(r)
		}
	})
}

// ------- Depth stress (LPM across deeper trees) -------

func BenchmarkTrieMatch_LPM_Depth4_ExactVsDeeper(b *testing.B)  { benchLPMDepth(b, 4) }
func BenchmarkTrieMatch_LPM_Depth8_ExactVsDeeper(b *testing.B)  { benchLPMDepth(b, 8) }
func BenchmarkTrieMatch_LPM_Depth12_ExactVsDeeper(b *testing.B) { benchLPMDepth(b, 12) }

func benchLPMDepth(b *testing.B, depth int) {
	// Build chain: a.b.c... (depth), plus deeper specializations at the tail
	rng := rand.New(rand.NewSource(3))
	tr := New[int]()
	base := make([]string, depth)
	for i := 0; i < depth; i++ {
		base[i] = genValidSegment(rng, 3, 6)
		// Insert progressively longer prefixes; value increases with depth
		p := strings.Join(base[:i+1], ".")
		if err := tr.Insert(p, 100+i); err != nil {
			b.Fatalf("insert failed: %v", err)
		}
	}
	// Query extends deepest prefix by +2 segments so that LPM hits the deepest
	q := strings.Join(base, ".") + "." + genValidSegment(rng, 3, 6) + "." + genValidSegment(rng, 3, 6)

	b.ReportAllocs()
	b.ResetTimer()
	var sink int
	for i := 0; i < b.N; i++ {
		if v, ok := tr.Match(q); ok {
			sink += v
		}
	}
	if sink == 0 {
		b.Fatalf("unexpected zero")
	}
}

// ------- helper to print scenario info (optional) -------

func BenchmarkScenarioPrint(b *testing.B) {
	// For quick, manual sanity check; not performance-relevant.
	Ns := []int{16, 128, 1024, 4096}
	for _, N := range Ns {
		for _, depth := range []int{4, 8} {
			for _, k := range []int{0, 3} {
				b.Run(fmt.Sprintf("N=%d/depth=%d/wcEvery=%d", N, depth, k), func(b *testing.B) {
					tr, qs := buildTrie(b, N, depth, k)
					if tr == nil || len(qs) == 0 {
						b.Fatal("broken builder")
					}
				})
			}
		}
	}
}
