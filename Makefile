.PHONY: help build test bench clean run fmt lint vet coverage wasm

# Default target
help:
	@echo "Poker Solver - Makefile targets:"
	@echo ""
	@echo "  make build     - Build the poker-solver binary"
	@echo "  make test      - Run all tests"
	@echo "  make bench     - Run all benchmarks"
	@echo "  make coverage  - Generate test coverage report"
	@echo "  make run       - Build and run the solver (use ARGS=\"...\" to pass arguments)"
	@echo "  make fmt       - Format all Go code"
	@echo "  make vet       - Run go vet on all packages"
	@echo "  make lint      - Run golint (requires golint installed)"
	@echo "  make clean     - Remove build artifacts"
	@echo "  make wasm      - Build WebAssembly binary"
	@echo ""

# Build the native binary
build:
	@echo "Building poker-solver..."
	go build -o bin/poker-solver ./cmd/poker-solver

# Run all tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run all tests with coverage
coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

# Run benchmarks and save to file
bench-save:
	@echo "Running benchmarks and saving to bench.txt..."
	go test -bench=. -benchmem ./... | tee bench.txt

# Format all Go code
fmt:
	@echo "Formatting Go code..."
	go fmt ./...

# Run go vet
vet:
	@echo "Running go vet..."
	go vet ./...

# Run golint (requires: go install golang.org/x/lint/golint@latest)
lint:
	@echo "Running golint..."
	golint ./...

# Build and run the solver
run: build
	@echo "Running poker-solver..."
	./bin/poker-solver $(ARGS)

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -f coverage.out coverage.html bench.txt
	rm -f poker-solver.wasm

# Build WebAssembly binary
wasm:
	@echo "Building WASM binary..."
	GOOS=js GOARCH=wasm go build -ldflags="-s -w" -o poker-solver.wasm ./wasm

# Run tests in verbose mode with race detection
test-race:
	@echo "Running tests with race detection..."
	go test -v -race ./...

# Check for common issues
check: fmt vet test
	@echo "All checks passed!"
