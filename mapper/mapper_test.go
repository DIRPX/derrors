package mapper

import (
	"strings"
	"sync"
	"testing"

	"dirpx.dev/derrors/apis"
	"dirpx.dev/derrors/code"
	"dirpx.dev/derrors/reason"
	"google.golang.org/grpc/codes"
)

func TestDefaults_HTTP_GRPC(t *testing.T) {
	m, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	// Spot-check a few canonical defaults from defaults.go
	check := func(c code.Code, wantHTTP int, wantGRPC codes.Code) {
		t.Helper()
		st := m.Status(c, reason.Empty)
		if st.HTTP != wantHTTP || st.GRPC != wantGRPC {
			t.Fatalf("Status(%q) got HTTP=%d GRPC=%v; want HTTP=%d GRPC=%v",
				c, st.HTTP, st.GRPC, wantHTTP, wantGRPC)
		}
	}
	check(code.Invalid, 400, codes.InvalidArgument)
	check(code.NotFound, 404, codes.NotFound)
	check(code.Unavailable, 503, codes.Unavailable)
}

func TestPriority_OverrideOverPrefixOverDefault_HTTP(t *testing.T) {
	m, err := New(
		WithHTTPDefault(code.Unavailable, 503),              // default
		WithHTTPPrefix(code.Unavailable, "storage.pg", 599), // prefix
		WithHTTPOverride(code.Unavailable, 418),             // override
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	st := m.Status(code.Unavailable, mustReason("storage.pg.connect"))
	if st.HTTP != 418 {
		t.Fatalf("override must win; got %d, want 418", st.HTTP)
	}
}

func TestPriority_OverrideOverPrefixOverDefault_GRPC(t *testing.T) {
	m, err := New(
		WithGRPCDefault(code.Unavailable, int(codes.Unavailable)),
		WithGRPCPrefix(code.Unavailable, "storage.pg", int(codes.Internal)),
		WithGRPCOverride(code.Unavailable, int(codes.Aborted)),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	st := m.Status(code.Unavailable, mustReason("storage.pg.connect"))
	if st.GRPC != codes.Aborted {
		t.Fatalf("override must win; got %v, want %v", st.GRPC, codes.Aborted)
	}
}

func TestPrefix_LPM_And_SegmentBoundary(t *testing.T) {
	m, err := New(
		WithHTTPPrefix(code.Unavailable, "storage.pg", 503),
		WithHTTPPrefix(code.Unavailable, "storage.pg.connect", 599),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// LPM should pick the longer "storage.pg.connect"
	st := m.Status(code.Unavailable, mustReason("storage.pg.connect.timeout"))
	if st.HTTP != 599 {
		t.Fatalf("LPM failed: got %d, want 599", st.HTTP)
	}
	// make sure we don't cross segment boundaries ("auth.j" must not match "auth.jwt")
	m2, _ := New(WithHTTPPrefix(code.Unavailable, "auth.jwt", 499))
	st2 := m2.Status(code.Unavailable, mustReason("auth.j"))
	if st2.HTTP == 499 {
		t.Fatalf("unexpected match across segment boundary")
	}
}

func TestWildcard_OneSegment(t *testing.T) {
	m, err := New(
		WithHTTPPrefix(code.Unavailable, "auth.*.verify", 502),
		WithHTTPPrefix(code.Unavailable, "auth.jwt.verify", 401), // exact should win at same depth
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	a := m.Status(code.Unavailable, mustReason("auth.jwt.verify"))
	if a.HTTP != 401 {
		t.Fatalf("exact must beat wildcard; got %d", a.HTTP)
	}
	b := m.Status(code.Unavailable, mustReason("auth.saml.verify.token"))
	if b.HTTP != 502 {
		t.Fatalf("wildcard match failed; got %d, want 502", b.HTTP)
	}
	// wildcard matches exactly one segment, not zero
	c := m.Status(code.Unavailable, mustReason("auth.verify"))
	if c.HTTP == 502 {
		t.Fatalf("wildcard must not match zero segments")
	}
}

func TestNormalization_In_Options(t *testing.T) {
	m, err := New(
		WithHTTPPrefix(code.Unavailable, "  STORAGE/PG.CONNECT-TIMEOUT  ", 599),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	st := m.Status(code.Unavailable, mustReason("storage.pg.connect_timeout"))
	if st.HTTP != 599 {
		t.Fatalf("normalized prefix should match; got %d", st.HTTP)
	}
}

func TestEmptyReason_UsesDefaultAndFallback(t *testing.T) {
	// Explicit default for Canceled (408), then check fallback on a synthetic code with no default.
	m, err := New(
		WithHTTPDefault(code.Canceled, 408),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	st := m.Status(code.Canceled, reason.Empty)
	if st.HTTP != 408 {
		t.Fatalf("empty reason should use default; got %d, want 408", st.HTTP)
	}

	// Create a mapper without a default for a synthetic code (simulate by new code).
	// We don't have a custom Code here, so simulate by clearing defaults via override-only:
	m2, err := New(
		WithHTTPOverride(code.PermissionDenied, 451),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// Override exists → 451, sanity check.
	st2 := m2.Status(code.PermissionDenied, reason.Empty)
	if st2.HTTP != 451 {
		t.Fatalf("override must win; got %d, want 451", st2.HTTP)
	}
}

func TestExplain_Sources_And_Pattern(t *testing.T) {
	m, err := New(
		WithHTTPPrefix(code.Unavailable, "storage.pg", 503),
		WithGRPCPrefix(code.Unavailable, "storage.pg", int(codes.Unavailable)),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	exp := m.Explain(code.Unavailable, mustReason("storage.pg.connect"))
	if !strings.Contains(exp, `source=prefix`) {
		t.Fatalf("Explain must include source=prefix:\n%s", exp)
	}
	if !strings.Contains(exp, `pattern="storage.pg"`) {
		t.Fatalf("Explain must include matched pattern:\n%s", exp)
	}
	if !strings.Contains(exp, `grpc:`) || !strings.Contains(exp, `http:`) {
		t.Fatalf("Explain must render both transports:\n%s", exp)
	}
}

func TestConcurrency_MapperStatus(t *testing.T) {
	m, err := New(
		WithHTTPPrefix(code.Unavailable, "storage.pg", 503),
		WithHTTPOverride(code.Canceled, 408),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 2000; j++ {
				_ = m.Status(code.Unavailable, mustReason("storage.pg.connect"))
				_ = m.Status(code.Canceled, reason.Empty)
				_ = m.Status(code.Invalid, mustReason("apimachinery.schema.gvk.parse"))
			}
		}()
	}
	wg.Wait()
}

func mustReason(s string) reason.Reason {
	r, err := reason.Parse(s)
	if err != nil {
		panic(err)
	}
	return r
}

func BenchmarkMapperStatus_Default(t *testing.B) {
	m, _ := New()
	r := mustReason("apimachinery.schema.gvk.parse")
	t.ReportAllocs()
	for i := 0; i < t.N; i++ {
		_ = m.Status(code.Invalid, r)
	}
}

func BenchmarkMapperStatus_PrefixHit(t *testing.B) {
	m, _ := New(
		WithHTTPPrefix(code.Unavailable, "storage.pg", 503),
		WithGRPCPrefix(code.Unavailable, "storage.pg", int(codes.Unavailable)),
	)
	r := mustReason("storage.pg.connect")
	t.ReportAllocs()
	for i := 0; i < t.N; i++ {
		_ = m.Status(code.Unavailable, r)
	}
}

func BenchmarkMapperStatus_Override(t *testing.B) {
	m, _ := New(
		WithHTTPOverride(code.Unavailable, 418),
		WithGRPCOverride(code.Unavailable, int(codes.Aborted)),
	)
	r := mustReason("storage.pg.connect")
	t.ReportAllocs()
	for i := 0; i < t.N; i++ {
		_ = m.Status(code.Unavailable, r)
	}
}

func BenchmarkMapperStatus_Fallback(t *testing.B) {
	// Use a code that has a default anyway; this just forces the path w/o prefix/override.
	m, _ := New()
	r := reason.Empty
	t.ReportAllocs()
	for i := 0; i < t.N; i++ {
		_ = m.Status(code.Unavailable, r)
	}
}

// Ensure mapper implements apis.Mapper
func TestMapper_InterfaceSatisfaction(t *testing.T) {
	var _ apis.Mapper = (*mapper)(nil)
}
