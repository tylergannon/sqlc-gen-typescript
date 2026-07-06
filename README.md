# sqlc-gen-typescript

TypeScript code generation for [`sqlc`](https://sqlc.dev/) targeting PostgreSQL and
[`postgres.js`](https://github.com/porsager/postgres).

This is PagerGuild-owned software. It is not maintained as an upstreamable fork,
and the supported install path is the WASM binary published from this repository.

## Status

The current generator is intentionally narrow:

- PostgreSQL only.
- `postgres` npm package only.
- Node.js/Vercel-compatible runtime output.
- Pure generated TypeScript; no native database driver dependency.

Older README content mentioned Bun SQL, MySQL, SQLite, and upstream sync
workflows. Those are not the current product surface.

## Install

Add the PagerGuild release binary to `sqlc.yaml`:

```yaml
version: "2"
plugins:
  - name: ts
    wasm:
      url: https://github.com/pagerguild/sqlc-gen-typescript/releases/download/v0.5.2/sqlc-gen-typescript.wasm
      sha256: 6c6d36cf48b74e952fccb078b961ee91b75b4ceec59ad522505c0b992d29d149
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

Generated TypeScript types are compile-time types. The generated code does not
post-process rows or validate runtime values. Runtime parsing belongs to
postgres.js configuration.

## Vercel Notes

The generated code depends on the `postgres` npm package. postgres.js is pure
JavaScript and is suitable for Vercel's Node.js runtime.

Do not use Bun SQL for Vercel deployments. Bun SQL was part of older fork
history and is not supported by the current generator.

## Local Development

Install dependencies:

```sh
bun install
```

Run checks:

```sh
just test
just lint
```

Build the WASM plugin:

```sh
just plugin-wasm
```

The build expects `buf` and `javy` to be installed. The release workflow installs
Javy 8.0.0 and publishes `sqlc-gen-typescript.wasm` to GitHub Releases.

To test a local plugin build:

```yaml
plugins:
  - name: ts
    wasm:
      url: file:///absolute/path/to/plugin.wasm
```

## Planned Type Overrides

The current bigint behavior is hard-coded. A future version should support an
option such as:

```yaml
options:
  driver: postgres
  bigint: number # number | string | bigint
```

That would let projects choose between ergonomic numbers, precision-preserving
strings, or native JavaScript `bigint` with matching postgres.js parser setup.
