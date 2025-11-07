# Root directory for protobuf API definitions.
PROTO_DIR := api

# Buf binary (override with `make BUF=/path/to/buf ...` if needed).
BUF := buf

# Default base ref for breaking-change checks.
# Can be overridden: `make proto-breaking BASE=origin/main`
BASE ?= origin/main

.PHONY: \
	proto-gen proto-lint proto-breaking proto-format proto-clean \
	all fmt lint test race bench fuzz \
	explain-golden explain-golden-update

# ============================================================
# Proto / Buf
# ============================================================

# Generate Go (and other) artifacts from proto files via buf.gen.yaml.
proto-gen:
	$(BUF) generate

# Run Buf linter on all proto files.
proto-lint:
	$(BUF) lint

# Check for breaking changes against a given git ref/tag/branch.
# By default compares against $(BASE).
proto-breaking:
	$(BUF) breaking --against ".git#ref=$(BASE)"

# Format .proto files in-place according to Buf formatting rules.
proto-format:
	$(BUF) format -w

# Remove generated protobuf Go files.
proto-clean:
	@find $(PROTO_DIR) -name '*.pb.go' -delete

# ============================================================
# Go
# ============================================================

# Run common Go steps: format, lint, test.
all: fmt lint test

# Go formatting. If gofumpt is installed, use it too (optional).
fmt:
	go fmt ./...
	@command -v gofumpt >/dev/null 2>&1 && gofumpt -w . || true

# Go linting. golangci-lint is optional — if not installed, skip silently.
lint:
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run ./... || true

# Run unit tests.
test:
	go test ./...

# Run tests with the race detector.
race:
	go test -race ./...

# Benchmarks for the mapping/segment trie packages.
bench:
	go test ./mapper/internal/segmenttrie -bench=. -benchmem
	go test ./mapper -bench=BenchmarkMapperStatus -benchmem

# Run fuzzing on the trie (30s).
fuzz:
	go test ./mapper -run=^$ -fuzz=Fuzz -fuzztime=30s

# ============================================================
# Golden tests for Explain()
# ============================================================

# Run explain-golden test without updating fixtures.
explain-golden:
	go test ./derrors/mapper -run Explain_Golden

# Re-generate golden files for explain tests.
explain-golden-update:
	go test ./derrors/mapper -run Explain_Golden -update
