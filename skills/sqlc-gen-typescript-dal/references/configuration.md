# Configuration Reference

Use this file when creating or changing `sqlc.yaml` or generation commands for
`tylergannon/sqlc-gen-typescript`.

## Supported Surface

- sqlc config version: `version: "2"`.
- Database engine: `postgresql`.
- Generator target: TypeScript query wrappers for the `postgres` npm package.
- Runtime environments: Node.js and Vercel-compatible JavaScript runtimes.
- Published plugin artifact:
  `https://github.com/tylergannon/sqlc-gen-typescript/releases/download/v0.6.0/sqlc-gen-typescript.wasm`

Do not configure Bun SQL, MySQL, SQLite, or non-postgres.js clients for this
generator.

## Minimal sqlc.yaml

```yaml
version: "2"
plugins:
  - name: ts
    wasm:
      url: https://github.com/tylergannon/sqlc-gen-typescript/releases/download/v0.6.0/sqlc-gen-typescript.wasm
sql:
  - schema: "db/schema.sql"
    queries: "db/query.sql"
    engine: "postgresql"
    codegen:
      - plugin: ts
        out: "src/db"
        options:
          driver: postgres
```

`options.driver: postgres` is optional because it is the default, but include it
in new configs for clarity.

## Options

```yaml
options:
  driver: postgres
  runtime: postgres
  bigint: number
  overrides: []
```

Current accepted values:

- `driver`: `postgres` only.
- `runtime`: `postgres` or `postgres.js`.
- `bigint`: `number` only.
- `overrides`: see `type-overrides.md`.

The generator rejects unsupported option values. Future option names are already
reserved, so do not invent additional values unless the installed generator
supports them.

## Runtime Setup

Install postgres.js:

```sh
npm install postgres
```

Create one configured SQL client and pass it to generated functions:

```ts
import postgres from "postgres";

export const sql = postgres(process.env.DATABASE_URL, {
  transform: postgres.camel,
  types: {
    bigint: {
      to: 20,
      from: [20],
      serialize: String,
      parse: Number,
    },
  },
});
```

`transform: postgres.camel` is required when SQL columns are snake_case and
generated TypeScript property names are camelCase.

The bigint parser is required for the current generated type policy:
PostgreSQL `int8`, `bigint`, `bigserial`, and `serial8` are emitted as
TypeScript `number`, while postgres.js returns OID `20` values as strings by
default.

## Local Plugin Builds

For development against this repository, build a local WASM plugin:

```sh
GOOS=wasip1 GOARCH=wasm go build -o examples/plugin.wasm ./cmd/sqlc-gen-typescript
```

Then point sqlc at a file URL:

```yaml
plugins:
  - name: ts
    wasm:
      url: file:///absolute/path/to/plugin.wasm
```

For this repository, prefer:

```sh
just generate
```

For downstream application repos, prefer the repo's own generation wrapper when
one exists; otherwise run:

```sh
sqlc generate
```

## Proof Checklist

Run the narrowest commands that prove the generated DAL works in the target
repo:

- `sqlc generate` or the repo's generation wrapper.
- TypeScript compile or check command, such as `npm run typecheck`,
  `pnpm check`, or `tsc --noEmit`.
- Unit/integration tests that import generated query functions or exercise the
  postgres.js DAL.

If generation fails, fix schema, query, config, or override inputs. Do not edit
generated TypeScript by hand.
