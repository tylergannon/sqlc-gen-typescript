# sqlc-gen-typescript

TypeScript code generation for [`sqlc`](https://sqlc.dev/) targeting PostgreSQL and
[`postgres.js`](https://github.com/porsager/postgres).

This is standalone software maintained at
[`tylergannon/sqlc-gen-typescript`](https://github.com/tylergannon/sqlc-gen-typescript).
The supported install path is the WASM binary published from this repository.
The plugin is implemented in Go and compiled to WASI WASM for releases.

## Agent Skill

This repository publishes a skill for agents that need to use this generator to
build a PostgreSQL TypeScript DAL:

```sh
npx skills add tylergannon/sqlc-gen-typescript --skill sqlc-gen-typescript-dal
```

Use `$sqlc-gen-typescript-dal` when configuring `sqlc`, generating postgres.js
query wrappers, choosing generator options, handling PostgreSQL dialect details,
or adding type overrides and converters.

## Status

The current generator is intentionally narrow:

- PostgreSQL only.
- `postgres` npm package only.
- Node.js/Vercel-compatible runtime output.
- Pure generated TypeScript; no native database driver dependency.

Older README content mentioned Bun SQL, MySQL, SQLite, and upstream sync
workflows. Those are not the current product surface.

## Install

Add the release binary to `sqlc.yaml`:

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

Then run:

```sh
sqlc generate
```

## Runtime Setup

Install postgres.js in your application:

```sh
npm install postgres
```

Create one configured SQL client and pass it to generated functions:

```ts
import postgres from "postgres";
import { getAuthor } from "./db/query_sql";

const sql = postgres(process.env.DATABASE_URL, {
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

const author = await getAuthor(sql, { id: 1 });
```

`transform: postgres.camel` is required when SQL columns are snake_case and the
generated TypeScript property names are camelCase.

The `bigint` parser is required for the current generated type policy:
PostgreSQL `int8`, `bigint`, `bigserial`, and `serial8` are emitted as
TypeScript `number`. postgres.js returns OID `20` values as strings by default
to avoid precision loss. This project currently chooses ergonomic `number`
types, so applications must parse OID `20` with `Number` if they select or
return bigint columns.

Only use this `number` policy when the values are expected to stay below
`Number.MAX_SAFE_INTEGER`.

## Generated Code Shape

The generator emits postgres.js tagged-template wrappers:

```ts
import type { Sql } from "postgres";

export interface GetAuthorArgs {
  id: number;
}

export interface GetAuthorRow {
  id: number;
  name: string;
}

export async function getAuthor(
  sql: Sql,
  args: GetAuthorArgs,
): Promise<GetAuthorRow | null> {
  const rows = await sql<GetAuthorRow[]>`SELECT id, name FROM authors
WHERE id = ${args.id} LIMIT 1`;
  return rows[0] ?? null;
}
```

Generated TypeScript types are compile-time types. By default, generated code
does not post-process rows or validate runtime values. Runtime parsing belongs
to postgres.js configuration unless you configure an explicit override
converter.

PostgreSQL `json` and `jsonb` columns are emitted as `JsonValue` instead of
`any`:

```ts
export type JsonPrimitive = string | number | boolean | null;
export type JsonValue =
  | JsonPrimitive
  | readonly JsonValue[]
  | { readonly [key: string]: JsonValue | undefined };
```

JSON parameters are serialized with `sql.json(...)` in generated queries. For
nullable JSON parameters, JavaScript `null` is sent as SQL `NULL`; use a
non-nullable JSON parameter if you need to write the JSON literal `null`.

## Vercel Notes

The generated code depends on the `postgres` npm package. postgres.js is pure
JavaScript and is suitable for Vercel's Node.js runtime.

Do not use Bun SQL for Vercel deployments. Bun SQL was part of older fork
history and is not supported by the current generator.

## Local Development

Run checks:

```sh
just test
just lint
```

Build the WASM plugin:

```sh
just plugin-wasm
```

The build expects Go, Just, and `sqlc` to be installed. The release workflow
publishes `sqlc-gen-typescript.wasm` to GitHub Releases.

To regenerate the examples with a local plugin build:

```sh
just generate
```

To test a local plugin build:

```yaml
plugins:
  - name: ts
    wasm:
      url: file:///absolute/path/to/plugin.wasm
```

## Type Overrides

Use `overrides` to replace the generated TypeScript type for a PostgreSQL type
or for a specific column:

```yaml
options:
  driver: postgres
  overrides:
    - db_type: "uuid"
      nullable: true
      ts_type:
        import: "$lib/model/types"
        type: "UUID"
    - column: "users.birthday"
      ts_type:
        import: "$lib/model/types"
        type: "DateTime"
      raw_type: "string"
      convert:
        import: "$lib/model/types"
        type: "strToDateTime"
    - db_type: "int8"
      nullable: false
      type: "number"
      raw_type: "string"
      convert: "Number"
```

Column overrides take precedence over `db_type` overrides. A `db_type` override
matches nullable and non-nullable columns separately, following sqlc's Go
generator behavior. Add both `nullable: true` and non-nullable overrides if you
want one database type to be replaced in both cases. Column overrides ignore
`nullable`.

`ts_type` can be a string for locally available or global types, or an object
with `import` and `type` when the generated file should import a named type.
For convenience, `type` is accepted as an alias for `ts_type` on an override.

When `convert` is present on an output column override, generated query
functions fetch a raw row type and map that column through the named converter
before returning the public row type. Parameter values are still passed to
postgres.js directly. Use `raw_type` when the database driver returns a
different TypeScript shape than the public type. For example, postgres.js
returns PostgreSQL `int8` values as strings by default, so a BIGINT-to-`number`
override can use `raw_type: "string"` and `convert: "Number"`.

The current bigint behavior remains hard-coded as `number`. A future version can
add a `bigint: number | string | bigint` option for projects that prefer
precision-preserving strings or native JavaScript `bigint`.

See `examples/type-overrides` for a working fixture that exercises imported
types, global type shorthand, nullable and non-nullable `db_type` overrides,
column overrides, raw driver types, output converters, BIGINT-to-`number`
conversion, and JSON parameters.
