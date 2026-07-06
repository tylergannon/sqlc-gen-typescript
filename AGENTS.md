# Repository Guidelines

## Project Structure & Module Organization

This repository builds a narrow `sqlc` TypeScript generator for PostgreSQL and `postgres.js`.
Core generator code lives in `src/`: `src/app.ts` is the Javy entrypoint, `src/drivers/` contains driver-specific emission logic, and `src/validate.ts` holds shared validation helpers. Generated protobuf bindings are in `src/gen/plugin/codegen_pb.ts`; refresh them with the Just build flow rather than hand-editing. Tests are colocated with implementation as `*.test.ts`. Example `sqlc` projects live under `examples/`, including generated output fixtures such as `examples/*/src/db/query_sql.ts`.

## Build, Test, and Development Commands

- `bun install`: install JavaScript dependencies.
- `bun test` or `just test`: run Bun unit tests.
- `just lint`: run `oxlint` with type-aware TypeScript checks.
- `just fmt`: format TypeScript with `oxfmt`.
- `just out-js`: regenerate protobuf bindings and bundle `src/app.ts` into `out.js`.
- `just plugin-wasm`: build `examples/plugin.wasm` with Javy.
- `just generate`: build the WASM plugin and regenerate examples with `sqlc-dev`.
- `just build`: clean generated artifacts, rebuild the plugin, and regenerate examples.

`buf`, `javy`, `sqlc-dev`, Bun, and Just are expected for the full build path. Set `JAVY_PATH` if `javy` is not on `PATH`.

## Coding Style & Naming Conventions

Use strict TypeScript, ES modules, and two-space indentation as enforced by `oxfmt`. Prefer small pure helpers for type mapping and validation, and use the TypeScript compiler factory APIs when emitting syntax trees. Keep generated TypeScript identifiers consistent with existing helpers: `pascalCase` for exported types, camelCase for query functions and argument properties, and `JsonValue` for JSON/JSONB shapes.

## Testing Guidelines

Tests use `bun:test` with `describe`, `it`, and `expect`. Add tests next to the code they cover, named `*.test.ts`. For code generation behavior, render TypeScript AST nodes and assert on exact output or important substrings. Cover both successful emission and error messages for invalid sqlc metadata.

## Commit & Pull Request Guidelines

Recent commits are short, imperative summaries such as `Narrow generated JSON types` or scoped fixes such as `fix examples`. Keep commits focused on one behavior change. Pull requests should explain the generator behavior affected, list the commands run, and mention any regenerated files under `src/gen/` or `examples/`. Link issues when applicable and include generated output diffs when changing emitted TypeScript.

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

4. The `Release` workflow runs on `v*` tags. It installs Bun, Just, Buf, and Javy, runs `just test`, builds `examples/plugin.wasm`, renames it to `sqlc-gen-typescript.wasm`, calculates SHA256, and publishes a GitHub Release.

Before tagging, verify CI on `main` passed. The `ci` workflow runs `just lint`, `just test`, `just plugin-wasm`, and `sqlc -f sqlc.dev.yaml diff` under `examples/`. The `examples` workflow runs both example clients against Postgres 16.
