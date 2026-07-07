package postgres

import (
	"fmt"
	"strings"

	"github.com/sqlc-dev/plugin-sdk-go/plugin"

	"github.com/tylergannon/sqlc-gen-typescript/internal/model"
)

type EnumMap map[string]*plugin.Enum

func BuildEnumMap(req *plugin.GenerateRequest) EnumMap {
	enumMap := EnumMap{}
	defaultSchema := req.GetCatalog().GetDefaultSchema()
	if defaultSchema == "" {
		defaultSchema = "public"
	}
	for _, schema := range req.GetCatalog().GetSchemas() {
		if schema.GetName() == "pg_catalog" || schema.GetName() == "information_schema" {
			continue
		}
		for _, enumDef := range schema.GetEnums() {
			enumMap[strings.ToLower(enumDef.GetName())] = enumDef
			if schema.GetName() != defaultSchema {
				enumMap[strings.ToLower(schema.GetName()+"."+enumDef.GetName())] = enumDef
			}
		}
	}
	return enumMap
}

func EnumName(enumMap EnumMap, col *plugin.Column) string {
	typeName := NormalizedTypeName(col)
	if typeName == "" {
		return ""
	}
	if _, ok := enumMap[typeName]; ok {
		return typeName
	}
	return ""
}

func IsJSONColumn(col *plugin.Column) bool {
	typeName := NormalizedTypeName(col)
	return typeName == "json" || typeName == "jsonb"
}

func IsScalarJSONColumn(col *plugin.Column) bool {
	return IsJSONColumn(col) && !col.GetIsArray() && col.GetArrayDims() == 0
}

func NormalizedTypeName(col *plugin.Column) string {
	if col == nil || col.GetType() == nil {
		return ""
	}
	typeName := col.GetType().GetName()
	if col.GetType().GetSchema() != "" {
		typeName = col.GetType().GetSchema() + "." + typeName
	}
	typeName = strings.TrimPrefix(typeName, "pg_catalog.")
	return strings.ToLower(typeName)
}

func ColumnType(enumMap EnumMap, col *plugin.Column) (model.TypeRef, error) {
	if col == nil || col.GetType() == nil {
		name := "unknown"
		if col != nil && col.GetName() != "" {
			name = col.GetName()
		}
		return model.TypeRef{}, fmt.Errorf("missing PostgreSQL type metadata for column %q; try adding an explicit cast or named parameter in your query", name)
	}

	originalTypeName := col.GetType().GetName()
	typeName := NormalizedTypeName(col)
	if typeName == "" {
		return model.TypeRef{}, fmt.Errorf("missing PostgreSQL type metadata for column %q; try adding an explicit cast or named parameter in your query", col.GetName())
	}

	var typ string
	if _, ok := enumMap[typeName]; ok {
		typ = model.PascalCase(typeName)
	} else {
		switch typeName {
		case "bool", "boolean":
			typ = "boolean"
		case "bytea":
			typ = "Buffer"
		case "date", "timestamp", "timestamp without time zone", "timestamptz", "timestamp with time zone":
			typ = "Date"
		case "float4", "real", "float8", "float", "double precision", "int2", "smallint", "int4", "int", "integer", "int8", "bigint", "bigserial", "serial", "serial2", "serial4", "serial8", "smallserial", "oid":
			typ = "number"
		case "json", "jsonb":
			typ = "JsonValue"
		case "void":
			typ = "void"
		case "text", "varchar", "character varying", "char", "character", "bpchar", "name", "uuid", "citext", "inet", "cidr", "macaddr", "macaddr8", "money", "numeric", "decimal", "xml", "bit", "varbit", "bit varying", "interval", "time", "time without time zone", "timetz", "time with time zone", "tsvector", "tsquery":
			typ = "string"
		case "point", "line", "lseg", "box", "path", "polygon", "circle":
			return model.TypeRef{}, fmt.Errorf("unrecognized PostgreSQL type: %q; please add support for this type in sqlc-gen-typescript/internal/postgres", originalTypeName)
		default:
			return model.TypeRef{}, fmt.Errorf("unrecognized PostgreSQL type: %q for column %q; this usually means sqlc couldn't infer the type; try adding an explicit cast like \"sqlc.arg(%s)::text\" or \"sqlc.narg('%s')\" in your query; if this is a valid PostgreSQL type that needs support, please add it to sqlc-gen-typescript/internal/postgres", originalTypeName, unknownColumnName(col), unknownColumnName(col), unknownColumnName(col))
		}
	}

	arrayDim := int(col.GetArrayDims())
	if col.GetIsArray() && arrayDim == 0 {
		arrayDim = 1
	}
	return model.TypeRef{
		Name:     typ,
		Nullable: !col.GetNotNull(),
		ArrayDim: arrayDim,
	}, nil
}

func unknownColumnName(col *plugin.Column) string {
	if col.GetName() == "" {
		return "unknown"
	}
	return col.GetName()
}
