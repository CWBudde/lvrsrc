# lvrsrc justfile
# Development automation for LabVIEW RSRC/VI file toolkit

set shell := ["bash", "-uc"]

# Default recipe - show available commands
default:
    @just --list

# Note: Install dependencies manually or use the GitHub Actions workflow
# treefmt: Download from https://github.com/numtide/treefmt/releases
# Go tools: go install mvdan.cc/gofumpt@latest && go install github.com/daixiang0/gci@latest && go install mvdan.cc/sh/v3/cmd/shfmt@latest
# golangci-lint: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
# prettier: npm install -g prettier

# Format all code using treefmt
fmt:
    treefmt --allow-missing-formatter

# Check if code is formatted correctly
check-formatted:
    treefmt --allow-missing-formatter --fail-on-change

# Run linters
lint:
    golangci-lint run --timeout=2m

# Run linters with auto-fix
lint-fix:
    golangci-lint run --fix --timeout=2m

# Ensure go.mod is tidy
check-tidy:
    go mod tidy
    git diff --exit-code go.mod go.sum

# Run all tests
test:
    go test -v -timeout 120s ./...

# Run tests with coverage
test-coverage:
    go test -v -timeout 120s -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html

# Run fuzz tests (short budget for local runs)
fuzz SECONDS="30":
    go test -fuzz=. -fuzztime={{ SECONDS }}s ./internal/rsrcwire/...

# Run all checks (formatting, linting, tests, tidiness)
check: check-formatted lint test check-tidy

# Build lvrsrc CLI tool
build:
    go build -o bin/lvrsrc ./cmd/lvrsrc

# Install lvrsrc to $GOPATH/bin
install:
    go install ./cmd/lvrsrc

# Clean build artifacts
clean:
    rm -rf bin/
    rm -f coverage.out coverage.html

# Inspect a VI/RSRC file
inspect FILE="testdata/vi/example.vi":
    go run ./cmd/lvrsrc inspect "{{ FILE }}"

# List resources in a VI/RSRC file
list-resources FILE="testdata/vi/example.vi":
    go run ./cmd/lvrsrc list-resources "{{ FILE }}"

# Dump raw resource data from a VI/RSRC file
dump FILE="testdata/vi/example.vi":
    go run ./cmd/lvrsrc dump "{{ FILE }}"

# Apply format fixes and linting
fix:
    just lint-fix
    just fmt
