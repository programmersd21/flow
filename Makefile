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

.PHONY: all build install run test vet lint tidy clean demo help

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

## check: format, vet, lint, and test
## check: the one command that says whether a change is safe.
## Runs formatting, vet, lint, tests, and the race detector.
check: fmt-check vet lint test race build

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

## lint: run golangci-lint if it is installed, otherwise say so and continue
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed; skipping (go install honnef.co/go/tools/cmd/staticcheck@latest)"; \
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

## help: print this message
help:
	@sed -n 's/^## //p' $(MAKEFILE_LIST) 2>/dev/null || findstr /B "##" Makefile
