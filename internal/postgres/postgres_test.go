package postgres_test

import (
	"strings"
	"testing"

	"github.com/sqlc-dev/plugin-sdk-go/plugin"

	"github.com/tylergannon/sqlc-gen-typescript/internal/postgres"
)

func TestColumnTypeMapsJSONBToJsonValue(t *testing.T) {
	typ, err := postgres.ColumnType(nil, column("profile", "jsonb", true))
	if err != nil {
		t.Fatal(err)
	}
	if got := typ.Render(); got != "JsonValue" {
		t.Fatalf("got %q, want JsonValue", got)
	}
}

func TestColumnTypePreservesJSONNullability(t *testing.T) {
	typ, err := postgres.ColumnType(nil, column("notes", "jsonb", false))
	if err != nil {
		t.Fatal(err)
	}
	if got := typ.Render(); got != "JsonValue | null" {
		t.Fatalf("got %q, want JsonValue | null", got)
	}
}

func TestColumnTypeErrorsOnMissingMetadata(t *testing.T) {
	_, err := postgres.ColumnType(nil, &plugin.Column{Name: "payload"})
	if err == nil {
		t.Fatal("expected missing metadata error")
	}
	if !strings.Contains(err.Error(), "missing PostgreSQL type metadata") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func column(name string, typeName string, notNull bool) *plugin.Column {
	return &plugin.Column{
		Name:    name,
		NotNull: notNull,
		Type:    &plugin.Identifier{Name: typeName},
	}
}
