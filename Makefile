BINARY    := flow
CMD       := ./cmd/flow
VERSION   ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS   := -ldflags "-s -w -X main.version=$(VERSION)"
GOFLAGS   :=

.PHONY: all build install clean test lint vet tidy demo

all: build

## build: compile the binary into ./bin/flow
build:
	@mkdir -p bin
	go build $(GOFLAGS) $(LDFLAGS) -o bin/$(BINARY) $(CMD)

## install: install to $GOPATH/bin (or ~/go/bin)
install:
	go install $(GOFLAGS) $(LDFLAGS) $(CMD)

## run: build and run with the default hero view
run: build
	./bin/$(BINARY)

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
	rm -rf bin/

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
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## //'
