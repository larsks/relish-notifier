# Makefile for relish-notifier
GO_SOURCES = $(shell go list -f '{{$$dir := .Dir}}{{range .GoFiles}}{{$$dir}}/{{.}} {{end}}' ./...)
GO_MOD_FILES = go.mod go.sum

# Variables
BINARY_NAME=relish-notifier
INSTALL_PREFIX?=/usr/local
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.version=$(VERSION)"
GOTEST?=go test -v

# Default target
.PHONY: build
build: $(BINARY_NAME)

$(BINARY_NAME): $(GO_SOURCES) $(GO_MOD_FILES)
	go build $(LDFLAGS) -o $@

# Development targets
.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: lint
lint:
	golangci-lint run

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: deps
deps: $(GO_SOURCES) $(GO_MOD_FILES)
	go mod download

# Testing
.PHONY: test
test:
	$(GOTEST) ./...

.PHONY: test-short
test-short:
	go test -v -short ./...

.PHONY: test-cover
test-cover:
	$(GOTEST) -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

.PHONY: bench
bench:
	$(GOTEST) -bench=. -benchmem ./...

# Quality checks
.PHONY: check
check: fmt lint vet test tidy
	@echo "All checks passed"

# Clean build artifacts
.PHONY: clean
clean:
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)-*
	rm -f coverage.out coverage.html

# Install binary to system
.PHONY: install
install: build
	install -d $(INSTALL_PREFIX)/bin
	install -m 755 $(BINARY_NAME) $(INSTALL_PREFIX)/bin/

# Uninstall binary from system
.PHONY: uninstall
uninstall:
	rm -f $(INSTALL_PREFIX)/bin/$(BINARY_NAME)

# Run the application (for testing)
.PHONY: run
run: build
	./$(BINARY_NAME)

.PHONY: run-once
run-once: build
	./$(BINARY_NAME) --once --log-level debug

# Help target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build        Build the binary"
	@echo "  fmt          Format Go code"
	@echo "  vet          Run go vet"
	@echo "  tidy         Tidy go modules"
	@echo "  deps         Download dependencies"
	@echo "  check        Run all quality checks (fmt, vet, tidy)"
	@echo "  clean        Remove build artifacts"
	@echo "  install      Install binary to $(INSTALL_PREFIX)/bin"
	@echo "  uninstall    Remove binary from $(INSTALL_PREFIX)/bin"
	@echo "  run          Build and run the application"
	@echo "  run-once     Build and run with --once --log-level debug"
	@echo "  test         Run all tests"
	@echo "  test-short   Run tests with -short flag"
	@echo "  test-cover   Run tests with coverage report"
	@echo "  bench        Run benchmarks"
	@echo "  help         Show this help message"
