# PostgreSQL Dialect Reference

Use this file before changing SQL schema or query files for this generator.

## Query Shape

Write normal sqlc PostgreSQL queries. Generated functions wrap the original SQL
in postgres.js tagged templates and interpolate parameters from a generated args
object.

Prefer explicit named parameters when sqlc might infer unstable names:

```sql
-- name: GetAuthor :one
SELECT id, name
FROM authors
WHERE id = sqlc.arg(id)::bigint;
```

For nullable parameters, use `sqlc.narg` and cast when inference is ambiguous:

```sql
WHERE nickname = sqlc.narg(nickname)::text
```

If the generator reports ambiguous names, disambiguate with `sqlc.arg`,
`sqlc.narg`, or explicit output column aliases.

## Nullability

The generator follows sqlc metadata:

- `NOT NULL` columns become non-null TypeScript fields.
- Nullable columns become `T | null`.
- Nullable parameters become `T | null`.

When sqlc cannot infer a parameter or result type, add an explicit PostgreSQL
cast in the query.

## JSON And JSONB

PostgreSQL `json` and `jsonb` columns emit generated `JsonValue` aliases rather
than `any`:

```ts
export type JsonPrimitive = string | number | boolean | null;
export type JsonValue =
  | JsonPrimitive
  | readonly JsonValue[]
  | { readonly [key: string]: JsonValue | undefined };
```

Scalar JSON parameters are serialized with `sql.json(args.field)` in generated
queries. Nullable JSON parameters preserve SQL `NULL`:

```ts
args.field === null ? null : sql.json(args.field)
```

Use a non-nullable JSON parameter if the application needs to write the JSON
literal `null` rather than SQL `NULL`.

## Enums

PostgreSQL enums generate TypeScript string union types. Keep enum labels stable
or treat generated output changes as API changes.

## Bigint

Current generated TypeScript policy maps PostgreSQL `int8`, `bigint`,
`bigserial`, and `serial8` to `number`.

postgres.js returns OID `20` values as strings by default. Configure the
postgres.js bigint parser to `Number` only when values are expected to stay
below `Number.MAX_SAFE_INTEGER`.

For precision-preserving public APIs, add type overrides that keep these values
as strings or domain-specific types and include converters only when safe.

## Unsupported Or Ambiguous Types

If generation fails with an unrecognized PostgreSQL type:

1. Check whether sqlc failed to infer the type. Add an explicit cast such as
   `sqlc.arg(name)::text`.
2. If the type is known but needs an application type, add an override.
3. If the type should be supported natively by the generator, update the
   generator's PostgreSQL type policy and tests rather than hiding the issue in
   generated code.
