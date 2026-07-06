BINARY    := flow
CMD       := ./cmd/flow
VERSION   ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
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

## install: install to $$GOPATH/bin (or ~/go/bin)
install:
	go install $(GOFLAGS) $(LDFLAGS) $(CMD)

## run: build and run
run: build
	$(BUILDDIR)$(SEP)$(BINARY)

## test: run the test suite
test:
	go test ./...

## vet: run go vet
vet:
	go vet ./...

## lint: run golangci-lint (must be installed separately)
lint:
	golangci-lint run ./...

## tidy: tidy and vendor dependencies
tidy:
	go mod tidy

## clean: remove build artifacts
clean:
	$(RMDIR) $(BUILDDIR)

## demo: generate docs/demo.gif with VHS
demo: build
	vhs flow.tape
	@if command -v gifsicle >/dev/null 2>&1; then \
		echo "optimizing with gifsicle..."; \
		gifsicle -O3 --colors 256 --lossy=80 -o docs/demo.gif docs/demo.gif; \
		ls -lh docs/demo.gif; \
	else \
		echo "install gifsicle for further optimization: brew install gifsicle"; \
	fi

## help: print this message
help:
	@sed -n 's/^## //p' $(MAKEFILE_LIST) 2>/dev/null || findstr /B "##" Makefile
