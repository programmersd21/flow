BINARY    := flow
CMD       := ./cmd/flow
VERSION   := v$(shell cat VERSION 2>/dev/null || echo "0.3.1")
LDFLAGS   := -ldflags "-s -w -X main.version=$(VERSION)"
GOFLAGS   :=

# Cross-platform detection
ifeq ($(OS),Windows_NT)
BINARY    := flow.exe
MKDIR     := mkdir
RMDIR     := -rmdir /s /q
SEP       := \\
else
MKDIR     := mkdir -p
RMDIR     := -rm -rf
SEP       := /
endif

BUILDDIR  := bin

# The linter must be on PATH for the strict quality gate. The version is
# pinned so a new golangci-lint release cannot turn a green tree red
# unexpectedly; override with `make lint GOLANGCI_LINT=...` if needed.
LINT_VERSION := v2.1.6
GOLANGCI_LINT ?= $(shell command -v golangci-lint 2>/dev/null || echo $(shell go env GOPATH)/bin/golangci-lint)

.PHONY: all build install run test vet lint lint-optional tidy clean demo check dev-check bench cross fuzz smoke help

all: build

$(BUILDDIR):
	$(MKDIR) $(BUILDDIR)

## build: compile the binary into ./bin/flow
build: $(BUILDDIR)
	go build $(GOFLAGS) $(LDFLAGS) -o $(BUILDDIR)$(SEP)$(BINARY) $(CMD)

## install: build and install to $GOBIN (or $GOPATH/bin)
install:
	go install $(GOFLAGS) $(LDFLAGS) $(CMD)
	@BIN="$$(go env GOBIN)"; \
	if [ -z "$$BIN" ]; then BIN="$$(go env GOPATH)/bin"; fi; \
	echo "installed: $$BIN/$(BINARY)"; \
	"$$BIN/$(BINARY)" --version; \
	case ":$$PATH:" in \
	  *"$$BIN"*) ;; \
	  *) echo "WARNING: $$BIN is not on your PATH, so '$(BINARY)' will not be found."; \
	     echo "  Add this to ~/.bashrc or ~/.zshrc, then restart your shell:"; \
	     echo "    export PATH=\"\$$PATH:$$BIN\""; ;; \
	esac; \
	FOUND="$$(command -v $(BINARY) 2>/dev/null || true)"; \
	if [ -n "$$FOUND" ] && [ "$$FOUND" != "$$BIN/$(BINARY)" ]; then \
	  echo "WARNING: '$$FOUND' shadows the new binary (shell picks the first match on PATH)."; \
	  echo "  Remove it or put $$BIN earlier on PATH to use the new build."; \
	fi

## run: build and run
run: build
	$(BUILDDIR)$(SEP)$(BINARY)

## tidy: tidy dependencies
tidy:
	go mod tidy

## clean: remove build artifacts
clean:
	$(RMDIR) $(BUILDDIR)

## demo: generate assets/demo.gif with VHS
demo: build
	vhs demo.tape
	@if command -v gifsicle >/dev/null 2>&1; then \
		echo "optimizing with gifsicle..."; \
		gifsicle -O3 --colors 256 --lossy=80 -o assets/demo.gif assets/demo.gif; \
		ls -lh assets/demo.gif; \
	else \
		echo "install gifsicle for further optimization: brew install gifsicle"; \
	fi

## check: the strict quality gate. fails on the first problem, and fails
## loudly rather than skipping when a required tool is missing.
## runs fmt-check, vet, lint, test, race, build.
## CI runs exactly this command.
check: fmt-check vet lint test race build

## dev-check: convenience for a fast local loop without the linter.
## Not the quality gate: use `make check` before pushing.
dev-check: fmt-check vet test build

## fmt: rewrite sources with gofmt
fmt:
	go fmt ./...

## fmt-check: fail if anything is unformatted (no writes)
fmt-check:
	@out=$$(gofmt -l . 2>/dev/null); \
	if [ -n "$$out" ]; then \
		echo "not gofmt'd:"; echo "$$out"; exit 1; \
	fi

## vet: run go vet
vet:
	go vet ./...

## lint: run the pinned linter; missing tool is a failure, not a skip
lint:
	@if ! command -v $(GOLANGCI_LINT) >/dev/null 2>&1; then \
		echo "golangci-lint $(LINT_VERSION) not found on PATH."; \
		echo "install it with: go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(LINT_VERSION)"; \
		echo "(or run the convenience target 'make dev-check', which skips linting)"; \
		exit 1; \
	fi
	$(GOLANGCI_LINT) run ./... --timeout=5m

## lint-optional: run the linter when installed, otherwise say so
lint-optional:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed; skipping"; \
	fi

## vuln: check dependencies for known vulnerabilities, when the tool is present
vuln:
	@if command -v govulncheck >/dev/null 2>&1; then \
		govulncheck ./...; \
	else \
		echo "govulncheck not installed; skipping"; \
	fi

## test: run the test suite
test:
	go test ./...

## race: run the test suite under the race detector
race:
	go test -race ./...

## fuzz: short fuzz run over the fuzz targets
fuzz:
	go test ./internal/ui -run '^$$' -fuzz FuzzHeroLayout -fuzztime 15s
	go test ./internal/ui -run '^$$' -fuzz FuzzSplitHeroNumber -fuzztime 15s
	go test ./internal/ui -run '^$$' -fuzz FuzzTruncate -fuzztime 15s

## bench: run the benchmarks (compare with -count=6 and benchstat)
bench:
	go test -run '^$$' -bench . -benchmem ./...

## cross: cross-compile the release targets CGO-free
cross:
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -o bin/flow-linux-amd64   ./cmd/flow
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build -o bin/flow-linux-arm64   ./cmd/flow
	CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build -o bin/flow-darwin-amd64  ./cmd/flow
	CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -o bin/flow-darwin-arm64  ./cmd/flow
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o bin/flow-windows-amd64.exe ./cmd/flow
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -o bin/flow-windows-arm64.exe ./cmd/flow

## smoke: exercise the real binary the way scripts use it
smoke: build
	$(BUILDDIR)$(SEP)$(BINARY) --version
	$(BUILDDIR)$(SEP)$(BINARY) --once
	$(BUILDDIR)$(SEP)$(BINARY) --once --json
	go test ./cmd/flow -run 'TestCLI' -count=1

## help: print this message
help:
	@sed -n 's/^## //p' $(MAKEFILE_LIST) 2>/dev/null || findstr /B "##" Makefile
