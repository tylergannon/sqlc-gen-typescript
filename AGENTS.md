# Repository Guidelines

## Project Structure & Module Organization

This repository builds a narrow `sqlc` TypeScript generator for PostgreSQL and `postgres.js`.
Core generator code is Go. `cmd/sqlc-gen-typescript/main.go` is the plugin entrypoint, `internal/generator/` orchestrates sqlc requests, `internal/postgres/` holds PostgreSQL type policy, `internal/ts/` renders TypeScript, and `internal/validate/` holds shared validation helpers. Architecture notes live in `docs/architecture/`. Tests are colocated with implementation as `*_test.go`. Example `sqlc` projects live under `examples/`, including generated output fixtures such as `examples/*/src/db/query_sql.ts`.

## Build, Test, and Development Commands

- `go test ./...` or `just test`: run Go unit tests.
- `just lint`: run `golangci-lint run ./...`.
- `just fmt`: format Go with `goimports` and `modernize`.
- `just plugin-wasm`: build `examples/plugin.wasm` with `GOOS=wasip1 GOARCH=wasm`.
- `just generate`: build the WASM plugin and regenerate examples with `sqlc`.
- `just build`: clean generated artifacts, rebuild the plugin, and regenerate examples.

Go, `sqlc`, Just, `golangci-lint`, and the Go tool dependencies declared in `go.mod` are expected for the full build path.

## Coding Style & Naming Conventions

Use idiomatic Go formatted by `gofmt`/`goimports`. Prefer small pure helpers for type mapping and validation. Keep sqlc metadata analysis separate from TypeScript rendering so future output targets and type overrides can share the same model. Keep generated TypeScript identifiers consistent with existing helpers: `PascalCase` for exported types, camelCase for query functions and argument properties, and `JsonValue` for JSON/JSONB shapes.

## Testing Guidelines

Tests use Go's standard `testing` package. Add tests next to the code they cover, named `*_test.go`. For code generation behavior, assert exact output or important substrings. Keep the authors example golden test passing before changing generated TypeScript intentionally. Cover both successful emission and error messages for invalid sqlc metadata.

## Commit & Pull Request Guidelines

Recent commits are short, imperative summaries such as `Narrow generated JSON types` or scoped fixes such as `fix examples`. Keep commits focused on one behavior change. Pull requests should explain the generator behavior affected, list the commands run, and mention any regenerated files under `examples/`. Link issues when applicable and include generated output diffs when changing emitted TypeScript.

## Deploy Pipeline

This repository is standalone at `tylergannon/sqlc-gen-typescript`; do not target or recreate a `pagerguild` upstream remote. Keep `origin` pointed at `https://github.com/tylergannon/sqlc-gen-typescript.git`.

Releases are not self-versioning. Publishing is tag-driven:

1. Merge the intended commits to `main` and push `origin main`.
2. Choose the next semver tag manually, for example `v0.5.4`.
3. Create and push the tag:

   ```sh
   git tag v0.5.4
   git push origin v0.5.4
   ```

4. The `Release` workflow runs on `v*` tags. It installs Go and Just, runs `just test`, builds `examples/plugin.wasm`, renames it to `sqlc-gen-typescript.wasm`, calculates SHA256, and publishes a GitHub Release.

Before tagging, verify CI on `main` passed. The `ci` workflow runs `just lint`, `just test`, `just plugin-wasm`, and `sqlc -f sqlc.dev.yaml diff` under `examples/`. The `examples` workflow runs both example clients against Postgres 16.
