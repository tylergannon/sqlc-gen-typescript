package generator_test

import (
	"context"
	"os"
	"testing"

	"github.com/sqlc-dev/plugin-sdk-go/plugin"

	"github.com/tylergannon/sqlc-gen-typescript/internal/generator"
)

func TestGenerateAuthorsFixtureMatchesOracle(t *testing.T) {
	resp, err := generator.Generate(context.Background(), authorsRequest())
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.GetFiles()) != 1 {
		t.Fatalf("got %d files, want 1", len(resp.GetFiles()))
	}
	file := resp.GetFiles()[0]
	if file.GetName() != "query_sql.ts" {
		t.Fatalf("got file %q, want query_sql.ts", file.GetName())
	}

	want, err := os.ReadFile("../../examples/node-postgres/src/db/query_sql.ts")
	if err != nil {
		t.Fatal(err)
	}
	if got := string(file.GetContents()); got != string(want) {
		t.Fatalf("generated fixture mismatch\n--- got ---\n%s\n--- want ---\n%s", got, string(want))
	}
}

func authorsRequest() *plugin.GenerateRequest {
	return &plugin.GenerateRequest{
		Catalog: &plugin.Catalog{
			DefaultSchema: "public",
			Schemas: []*plugin.Schema{{
				Name: "public",
				Enums: []*plugin.Enum{{
					Name: "author_status",
					Vals: []string{"active", "inactive", "pending"},
				}},
			}},
		},
		Queries: []*plugin.Query{
			{
				Name:     "GetAuthor",
				Cmd:      ":one",
				Filename: "query.sql",
				Text: `SELECT id, name, bio, status, profile, notes FROM authors
WHERE id = $1 LIMIT 1`,
				Params:  []*plugin.Parameter{{Column: column("id", "bigint", true)}},
				Columns: authorColumns(),
			},
			{
				Name:     "ListAuthors",
				Cmd:      ":many",
				Filename: "query.sql",
				Text: `SELECT id, name, bio, status, profile, notes FROM authors
ORDER BY name`,
				Columns: authorColumns(),
			},
			{
				Name:     "CreateAuthor",
				Cmd:      ":one",
				Filename: "query.sql",
				Text: `INSERT INTO authors (
  name, bio, status, profile, notes
) VALUES (
  $1, $2, $3, $4, $5
)
RETURNING id, name, bio, status, profile, notes`,
				Params: []*plugin.Parameter{
					{Column: column("name", "text", true)},
					{Column: column("bio", "text", false)},
					{Column: column("status", "author_status", true)},
					{Column: column("profile", "jsonb", true)},
					{Column: column("notes", "jsonb", false)},
				},
				Columns: authorColumns(),
			},
			{
				Name:     "ListAuthorsByStatus",
				Cmd:      ":many",
				Filename: "query.sql",
				Text: `SELECT id, name, bio, status, profile, notes FROM authors
WHERE status = $1
ORDER BY name`,
				Params:  []*plugin.Parameter{{Column: column("status", "author_status", true)}},
				Columns: authorColumns(),
			},
			{
				Name:     "DeleteAuthor",
				Cmd:      ":exec",
				Filename: "query.sql",
				Text: `DELETE FROM authors
WHERE id = $1`,
				Params: []*plugin.Parameter{{Column: column("id", "bigint", true)}},
			},
		},
	}
}

func authorColumns() []*plugin.Column {
	return []*plugin.Column{
		column("id", "bigint", true),
		column("name", "text", true),
		column("bio", "text", false),
		column("status", "author_status", true),
		column("profile", "jsonb", true),
		column("notes", "jsonb", false),
	}
}

func column(name string, typeName string, notNull bool) *plugin.Column {
	return &plugin.Column{
		Name:    name,
		NotNull: notNull,
		Type:    &plugin.Identifier{Name: typeName},
	}
}
