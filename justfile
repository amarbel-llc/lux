# Lux: LSP Multiplexer

default:
    @just --list

# Build the binary
build:
    nix build

# Build using go directly (faster for dev)
build-go:
    go build -o lux ./cmd/lux

# Run tests
test:
    go test ./...

# Run tests with verbose output
test-v:
    go test -v ./...

# Format code
fmt:
    go fmt ./...
    shfmt -w -i 2 -ci ./*.sh 2>/dev/null || true

# Lint code
lint:
    go vet ./...

# Run the LSP server
serve:
    go run ./cmd/lux serve

# Show configured LSPs
list:
    go run ./cmd/lux list

# Check LSP status
status:
    go run ./cmd/lux status

# Add a new LSP from a flake
add flake:
    go run ./cmd/lux add "{{flake}}"

# Run in nix develop shell
dev:
    nix develop

# Clean build artifacts
clean:
    rm -f lux
    rm -rf result

# Update go dependencies
deps:
    go mod tidy

# Generate vendored dependencies for nix
vendor:
    go mod vendor
