# Generate examples using sqlc
generate: plugin-wasm
    cd examples && sqlc -f sqlc.dev.yaml generate

# Compile the Go plugin to WASM.
plugin-wasm:
    GOOS=wasip1 GOARCH=wasm go build -o examples/plugin.wasm ./cmd/sqlc-gen-typescript

# Clean build artifacts
clean:
    rm -f examples/plugin.wasm

# Format Go
fmt:
    go tool goimports -w cmd internal
    go tool modernize -fix ./...

# Lint Go
lint:
    golangci-lint run ./...

# Run unit tests
test:
    go test ./...

# Build everything from scratch
build: clean generate
