# Type Overrides Reference

Use this file before adding or changing `options.overrides`.

## Override Forms

Use `db_type` for all columns of a PostgreSQL type:

```yaml
options:
  driver: postgres
  overrides:
    - db_type: "uuid"
      nullable: false
      ts_type:
        import: "$lib/model/types"
        type: "UUID"
```

Use `column` for one specific column:

```yaml
options:
  overrides:
    - column: "users.birthday"
      ts_type:
        import: "$lib/model/types"
        type: "DateTime"
      raw_type: "string"
      convert:
        import: "$lib/model/types"
        type: "strToDateTime"
```

Column overrides take precedence over `db_type` overrides.

## Matching Rules

`db_type` overrides match nullable and non-nullable columns separately. Add both
forms when one type should apply to both:

```yaml
overrides:
  - db_type: "uuid"
    nullable: false
    ts_type: "UUID"
  - db_type: "uuid"
    nullable: true
    ts_type: "UUID"
```

Column overrides ignore `nullable`.

Column names may match:

- `table.column`
- `schema.table.column`
- `catalog.schema.table.column`

Use the most specific form needed to avoid collisions.

## Type Symbols

`ts_type`, `raw_type`, and `convert` accept either a string or an object:

```yaml
ts_type: "EmailAddress"
```

```yaml
ts_type:
  import: "$lib/model/types"
  type: "EmailAddress"
```

`name` is accepted as an alias for `type` inside symbol objects, but prefer
`type`.

For convenience, override-level `type` is accepted as an alias for `ts_type`:

```yaml
- db_type: "int8"
  nullable: false
  type: "number"
```

## Raw Types And Converters

Use `raw_type` when postgres.js returns a different TypeScript value than the
public generated type. `raw_type` requires `convert`.

```yaml
- db_type: "int8"
  nullable: false
  type: "number"
  raw_type: "string"
  convert: "Number"
```

When `convert` is present on an output column override, generated query
functions fetch a raw row type and map that column through the converter before
returning the public row type. Parameter values are still passed to postgres.js
directly.

Converters may be globals:

```yaml
convert: "Number"
```

Or imports:

```yaml
convert:
  import: "$lib/model/types"
  type: "strToDateTime"
```

## Validation Rules

Each override must have:

- `ts_type` or override-level `type`.
- Exactly one of `db_type` or `column`.

Invalid combinations:

- Both `db_type` and `column`.
- Neither `db_type` nor `column`.
- `raw_type` without `convert`.
- Empty type names.
- Import paths with leading or trailing whitespace.

## Complete Example

```yaml
options:
  driver: postgres
  overrides:
    - db_type: "uuid"
      nullable: false
      ts_type:
        import: "$lib/model/types"
        type: "UUID"
    - db_type: "uuid"
      nullable: true
      ts_type:
        import: "$lib/model/types"
        type: "UUID"
    - db_type: "int8"
      nullable: false
      type: "number"
      raw_type: "string"
      convert: "Number"
    - column: "users.email"
      ts_type:
        import: "$lib/model/types"
        type: "EmailAddress"
    - column: "users.birthday"
      type: "DateTime"
      raw_type: "string"
      convert:
        import: "$lib/model/types"
        type: "strToDateTime"
```
