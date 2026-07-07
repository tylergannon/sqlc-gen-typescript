# Go Port Architecture

## Goal

Port `sqlc-gen-typescript` from the current TypeScript/Javy implementation to a
Go sqlc plugin while preserving the emitted TypeScript API. The initial Go
version should be behavior-compatible with the existing generator, then become
the foundation for type overrides and additional output targets.

## Upstream Plugin Shape

The Go plugin should use `github.com/sqlc-dev/plugin-sdk-go` directly:

- `cmd/sqlc-gen-typescript/main.go` calls `codegen.Run(generator.Generate)`.
- `generator.Generate` accepts `*plugin.GenerateRequest` and returns
  `*plugin.GenerateResponse`.
- The same binary entrypoint can build as a local process plugin or as WASM with
  `GOOS=wasip1 GOARCH=wasm`.

There is no scaffold generator worth adopting here. `sqlc-gen-go` is useful as a
reference for project shape and build targets, but this project emits
TypeScript, so the core generation logic should stay purpose-built.

## Package Boundaries

Keep the initial packages small and explicit:

- `cmd/sqlc-gen-typescript`: plugin executable entrypoint.
- `internal/generator`: request orchestration, query grouping, file assembly.
- `internal/options`: parsed plugin options with defaults.
- `internal/model`: output-neutral generation model for queries, fields, enums,
  and parameter render decisions.
- `internal/postgres`: PostgreSQL metadata interpretation and TypeScript type
  policy for the default `postgres.js` target.
- `internal/ts`: TypeScript source rendering.
- `internal/validate`: shared validation such as duplicate generated names.

The important boundary is between sqlc metadata and TypeScript rendering.
Postgres type resolution should produce a model-level `TypeRef`, not raw text
that is scattered through templates. Rendering can still be simple string
building, but policy should not live in string concatenation.

## Output Targets

The initial target is `postgres.js`, matching the current repository behavior.
However, the model should allow future outputs such as Bun SQL without rewiring
query analysis:

```text
sqlc request -> model -> target renderer -> files
```

A target owns:

- imports and shared declarations,
- database client parameter type,
- query function body shape,
- parameter expression rendering,
- target-specific JSON serialization behavior.

For `postgres.js`, scalar JSON parameters render as `sql.json(args.field)` and
nullable scalar JSON parameters preserve SQL `NULL` with
`args.field === null ? null : sql.json(args.field)`.

## Options And Overrides

The generator keeps a stable option namespace:

```yaml
options:
  driver: postgres
  runtime: postgres
  bigint: number
  overrides:
    - db_type: uuid
      nullable: true
      ts_type:
        import: "$lib/model/types"
        type: "UUID"
    - column: users.birthday
      ts_type:
        import: "$lib/model/types"
        type: "DateTime"
      raw_type: string
      convert:
        import: "$lib/model/types"
        type: "strToDateTime"
```

Current rules:

- `driver` remains accepted as `postgres` for compatibility.
- `runtime` can later distinguish `postgres`, `bun`, or another emitted client.
- `bigint` can later support `number`, `string`, and `bigint`.
- Column overrides take precedence over `db_type` overrides.
- `db_type` overrides match nullable and non-nullable columns separately.
- Column overrides ignore `nullable`.
- `ts_type` may be a string or an `{ import, type }` object.
- `raw_type` may be a string or an `{ import, type }` object for converter
  source values when the driver returns a different type than the public API.
- `convert` may be used on output column overrides to map raw postgres.js row
  values into the public TypeScript row type.

## Parity Contract

The Go port should prove parity before deleting TypeScript sources:

- Unit tests cover type mapping, JSON parameter rendering, duplicate names, and
  request-to-file generation.
- A golden test compares the generated authors example to the checked-in
  TypeScript fixture.
- `sqlc -f examples/sqlc.dev.yaml diff` must pass against the Go-built WASM.

Intentional generated output changes should be called out explicitly. Otherwise,
the initial port should preserve the current public TypeScript shape.
