.PHONY: all build test vet fmt check examples build-demo clean ci-lint ci-test ci-test-submodules ci-build-demo

# Default: run full CI-like validation
all: check

# ---------------------------------------------------------------------------
# Build
# ---------------------------------------------------------------------------
build:
	go build ./...

examples: build-demo

build-demo:
	go build ./demo/...

# ---------------------------------------------------------------------------
# Test
# ---------------------------------------------------------------------------
test:
	go test ./...

test-race:
	go test -race ./...

test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
	@echo "HTML report: go tool cover -html=coverage.out"

bench:
	go test -bench=. -benchmem ./...

# ---------------------------------------------------------------------------
# Static analysis
# ---------------------------------------------------------------------------
vet:
	go vet ./...

fmt:
	gofmt -l -w .

fmt-check:
	@test -z "$$(gofmt -l .)" || { echo "gofmt needed on:"; gofmt -l .; exit 1; }

# ---------------------------------------------------------------------------
# Combined checks (CI gate)
# ---------------------------------------------------------------------------
check: fmt-check build vet test

ci-lint: vet

ci-test:
	go test -race -count=1 ./...

ci-test-submodules:
	@if [ -d otel ]; then \
		echo "==> testing otel"; \
		cd otel && go test ./...; \
	fi

ci-build-demo: build-demo

# ---------------------------------------------------------------------------
# Cleanup
# ---------------------------------------------------------------------------
clean:
	rm -f coverage.out
	go clean -cache -testcache
