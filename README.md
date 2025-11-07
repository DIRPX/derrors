# derrors — predictable, fast, boring errors for Go

A small, opinionated toolkit for **application errors** in services. It gives you:

- A tiny, transport‑agnostic `*derrors.Error` type you can pass anywhere.
- A **status mapper** that deterministically converts `(Code, Reason)` → `{HTTP, gRPC}` with
  longest‑prefix matching over dotted reasons (supports `*` per segment).
- Thin adapters for **HTTP** (`httpx`) and **gRPC** (`grpcx`) so your wire formats are consistent.
- Versioned **contracts** in `api/derrors/v1`: JSON Schema for the HTTP **View** and Protobuf for the rich **Descriptor**.

The philosophy is simple: keep domain errors small, put transport decisions at the edge, and make mapping explicit and testable.

---

## Contents

- [Why this exists](#why-this-exists)
- [Install](#install)
- [At a glance](#at-a-glance)
- [Core types](#core-types)
- [Codes & Reasons](#codes--reasons)
- [Status mapper](#status-mapper)
- [HTTP adapter (`httpx`)](#http-adapter-httpx)
- [gRPC adapter (`grpcx`)](#grpc-adapter-grpcx)
- [Contracts (wire formats)](#contracts-wire-formats)
- [End‑to‑end examples](#endtoend-examples)
- [Performance](#performance)
- [Testing & fuzzing](#testing--fuzzing)
- [Buf & codegen](#buf--codegen)
- [Project layout](#project-layout)
- [Versioning & compatibility](#versioning--compatibility)
- [FAQ](#faq)

---

## Why this exists

Service code shouldn’t reinvent error handling in every handler. We want:

- **Predictability** — the same domain error maps to the same HTTP/gRPC status everywhere.
- **Separation of concerns** — domain errors avoid transport baggage (status codes, correlation, trace).
- **Explained behavior** — you can inspect exactly *why* a certain status was chosen (override vs prefix vs default).
- **Performance** — hot paths must be allocation‑free and very fast.

`derrors` is a practical distillation of those goals.

---

## Install

```bash
go get dirpx.dev/derrors@latest
```

> The repo is split into small packages (`code`, `reason`, `mapper`, `httpx`, `grpcx`) you can import individually.

---

## At a glance

```go
e := &derrors.Error{
Code:    code.Unavailable,
Reason:  reason.MustParse("storage.pg.connect_timeout"),
Message: "temporarily unavailable",
}

// Map to HTTP/gRPC statuses:
st := m.Status(e.Code, e.Reason) // override > prefix(LPM) > default > fallback

// HTTP:
httpx.Writer{Mapper: m}.Write(w, e, httpx.Meta{
Correlation: correlationIDFrom(ctx),
TraceID:     traceIDFrom(ctx),
SpanID:      spanIDFrom(ctx),
// RetryAfterSeconds, Links, Fields if needed
})

// gRPC (interceptor):
srv := grpc.NewServer(grpc.UnaryInterceptor(
grpcx.UnaryServerInterceptor(m, func(ctx context.Context, e *derrors.Error) grpcx.Extras {
return grpcx.Extras{
CorrelationID: correlationIDFrom(ctx),
TraceID:       traceIDFrom(ctx),
SpanID:        spanIDFrom(ctx),
// Retry/Quota/Violations/Links/Causes/Env/Tags if you have them
}
}),
))
```

---

## Core types

### `*derrors.Error`
A small value‑like error that owns **only** what the *error itself* should own.

```go
type Error struct {
Code    code.Code      // business code (e.g., "unavailable", "invalid")
Reason  reason.Reason  // dotted reason ("storage.pg.connect_timeout"), optional
Message string         // short, safe, human‑readable message
Details map[string]any // structured ad‑hoc details (safe keys only)
Cause   error          // wrapped technical cause
}

func (e *Error) Error() string
func (e *Error) Unwrap() error
func (e *Error) WithReason(r reason.Reason) *Error // copy‑on‑write
func (e *Error) WithMessage(msg string) *Error     // copy‑on‑write
func (e *Error) WithDetail(k string, v any) *Error // copy‑on‑write
```

> **No transport fields** (status, correlation, trace/span) live here. Adapters inject those at the boundary.

---

## Codes & Reasons

- **Codes** (`code.Code`) are short business labels like `"unavailable"`, `"invalid"`, `"conflict"`, etc. You own the set.
- **Reasons** (`reason.Reason`) are normalized **dotted paths** used for routing and mapping:
    - Segments: `a-z`, digits, `_` in the middle; start with `a-z`.
    - Examples: `storage.pg.connect_timeout`, `auth.jwt.verify`.
    - **Mapper prefixes** may include `*` to match **exactly one** segment: `auth.*.verify`.

Helpers:

```go
r := reason.MustParse("storage.pg.connect_timeout")
s := r.String()     // canonical lowercase dotted form
```

---

## Status mapper

Deterministic mapping from `(Code, Reason)` to `{HTTP, gRPC}` with this precedence:

1. **Override** — per‑code hard override.
2. **Prefix** — longest prefix match over `Reason` using a **segment trie**
   (segments separated by `.`, `*` matches one segment).
3. **Default** — per‑code default mapping.
4. **Fallback** — global fallback (e.g., 500/Internal).

Configure once:

```go
m, _ := mapper.New(
  mapper.WithHTTPDefaults(map[code.Code]int{
    code.Unavailable: 503,
    code.Invalid:     400,
  }),
  mapper.WithGRPCDefaults(map[code.Code]codes.Code{
    code.Unavailable: codes.Unavailable,
    code.Invalid:     codes.InvalidArgument,
  }),

  // Highest priority:
  mapper.WithHTTPOverride(code.Canceled, 408),
  mapper.WithGRPCOverride(code.Canceled, codes.Canceled),

  // Prefix rules (segment‑aware LPM; "*" matches one segment)
  mapper.WithHTTPPrefix(code.Unavailable, "storage.pg", 503),
  mapper.WithGRPCPrefix(code.Unavailable, "storage.pg", codes.Unavailable),
)
```

Explain decisions (great for unit tests and debugging):

```
code="unavailable" reason="storage.pg.connect_timeout"
http: source=prefix pattern="storage.pg" -> 503
grpc: source=prefix pattern="storage.pg" -> UNAVAILABLE(14)
```

**Under the hood:** a compact **segment trie** explores exact and wildcard branches. The hot path is allocation‑free.

---

## HTTP adapter (`httpx`)

`httpx.Writer` renders the **HTTP View** (public JSON) and sets the HTTP status from the mapper.

Input:
- `*derrors.Error`
- `httpx.Meta` with optional `Correlation`, `TraceID`, `SpanID`, `RetryAfterSeconds`, `Links`, `Fields`

Output:
- Body matches `api/derrors/v1/error.view.schema.json` exactly.
- Status is `Mapper.Status(err.Code, err.Reason).HTTP`.
- `Retry-After` header is set if `RetryAfterSeconds > 0`.

**Encoding options**:

- Go struct + `encoding/json` (simple), **or**
- `derrors.v1.ErrorView` + `protojson` (exact parity with schema; snake_case via `json_name`).

The provided implementation uses **`protojson`** for 1:1 parity.

---

## gRPC adapter (`grpcx`)

`grpcx.UnaryServerInterceptor` converts any returned `*derrors.Error` into:

- `status.Code` from the mapper (`Mapper.Status(...).GRPC`)
- `status.WithDetails(derrors.v1.ErrorDescriptor)` — the **rich descriptor** (protobuf) carrying
  code, reason, message, mapped HTTP/gRPC, correlation, trace/span, and optional retry/quota/violations/links/causes/env/tags.

Helper for tests:

```go
desc, ok := grpcx.ExtractDescriptor(err) // read ErrorDescriptor back from details
```

---

## Contracts (wire formats)

The library ships **contracts** under `api/derrors/v1`:

- `error.view.schema.json` — JSON Schema for the **HTTP View** (public payload).
- `error.proto` — Protobuf for the rich **ErrorDescriptor** (gRPC details / logs / buses).
- *(Optional)* `error.view.proto` — Protobuf for **ErrorView** if you prefer to emit HTTP JSON via `protojson`.

> Keep contracts versioned. Add optional fields freely; bump `v1 → v2` for breaking semantics.

---

## End‑to‑end examples

### HTTP handler

```go
func createOrder(w http.ResponseWriter, r *http.Request) {
  if err := svc.Create(r.Context()); err != nil {
    de := &derrors.Error{
      Code:    code.Unavailable,
      Reason:  reason.MustParse("storage.pg.connect_timeout"),
      Message: "temporarily unavailable",
      Cause:   err,
    }
    httpx.Writer{Mapper: m}.Write(w, de, httpx.Meta{
      Correlation: idFromHeaders(r),
      TraceID:     traceIDFrom(r.Context()),
      SpanID:      spanIDFrom(r.Context()),
      // RetryAfterSeconds: 5,
      // Links, Fields...
    })
    return
  }
  w.WriteHeader(http.StatusCreated)
}
```

### gRPC server

```go
srv := grpc.NewServer(grpc.UnaryInterceptor(
  grpcx.UnaryServerInterceptor(m, func(ctx context.Context, e *derrors.Error) grpcx.Extras {
    return grpcx.Extras{
      CorrelationID: correlationIDFrom(ctx),
      TraceID:       traceIDFrom(ctx),
      SpanID:        spanIDFrom(ctx),
      // Retry/Quota/Violations/Links/Causes/Env/Tags if needed
    }
  }),
))
```

### Mapper tests

```go
func TestExplain(t *testing.T) {
  out := m.Explain(code.Unavailable, reason.MustParse("storage.pg.connect_timeout"))
  require.Contains(t, out, `source=prefix`)
}
```

---

## Performance

Hot‑path targets (observed on a modern laptop; your results may vary):

- `BenchmarkMapperStatus_Default` ~ **15–25 ns/op**, **0 alloc/op**
- `BenchmarkMapperStatus_Override` ~ **10–20 ns/op**, **0 alloc/op**
- `BenchmarkMapperStatus_PrefixHit` ~ **80–120 ns/op**, **0 alloc/op**
- Trie match (depth 4–8, with/without `*`) ~ **80–300 ns/op**, **0 alloc/op**

Why it’s fast:

- Segment trie explores exact + wildcard branches with a tiny DFS and no heap churn on steady‑state.
- Mapper keeps prebuilt tries per code and uses straight‑line checks (override → trie → default).

---

## Testing & fuzzing

We include:

- **Unit tests** for mapper precedence and invalid inputs.
- **Golden test** fixing the `Explain()` output format.
- **Fuzz test** that differential‑checks trie matching against a naive longest‑prefix matcher (catches wildcard edge cases).

Useful commands:

```bash
# run all tests
go test ./...

# race detector
go test -race ./...

# trie fuzzing (package-by-package)
go test ./mapper/internal/segmenttrie -run=^$ -fuzz=Fuzz -fuzztime=30s

# benchmarks
go test ./mapper/internal/segmenttrie -bench=. -benchmem
go test ./mapper -bench=BenchmarkMapperStatus -benchmem
```

---

## Buf & codegen

We use **Buf v2** with module rooted at `api/`. Typical workflow:

```bash
# format, lint, generate
make proto-format
make proto-lint
make proto-gen

# check breaking changes vs main
make proto-breaking BASE=origin/main
```

Generation target can be colocated (`out: api`) or a separate artifacts dir (`out: gen/go`). Choose one and keep imports consistent.

---

## Project layout

```
api/
  derrors/
    v1/
      error.proto               # rich ErrorDescriptor (protobuf)
      error.view.schema.json    # HTTP ErrorView (JSON Schema)
      error.schema.json         # rich ErrorDescriptor (JSON Schema for logs/bus)
      # optional: error.view.proto (proto View for protojson HTTP)

httpx/
  httpx.go                      # HTTP writer → View JSON (protojson)

grpcx/
  grpcx.go                      # gRPC interceptor → Status + Details(Descriptor)

mapper/
  builder.go
  defaults.go
  mapper.go
  helpers.go
  doc.go
  explain_golden_test.go
  mapper_test.go
  internal/segmenttrie/
    trie.go
    trie_test.go
    trie_bench_test.go

code/, reason/
  code.go, codes.go, reason.go, tests, docs

apis/
  mapper.go, status.go, interfaces... (transport‑agnostic)
```

---

## Versioning & compatibility

- Contracts live in `api/derrors/v1`. Additive changes are OK (optional fields). For breaking changes, fork to `v2`.
- The Go API follows semver; breaking changes will bump the major version tag.

---

## FAQ

**Why separate View (HTTP) and Descriptor (gRPC)?**  
They serve different audiences. View is a slim public payload; Descriptor is rich and portable for details/logs/internal tooling. Keeping them separate avoids over‑exposing internals to HTTP clients.

**Do I have to use proto for HTTP?**  
No. We use `protojson` for strict parity with the schema, but you can emit a plain Go struct if you prefer. The contract stays the same.

**Where do correlation/trace/span belong?**  
In adapters (httpx/grpcx) and logs — not in the domain error.

**Can I override status per service?**  
Yes. Use per‑code overrides and reason‑prefix rules. Precedence is explicit and testable.
