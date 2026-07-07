package options_test

import (
	"strings"
	"testing"

	"github.com/sqlc-dev/plugin-sdk-go/plugin"

	"github.com/tylergannon/sqlc-gen-typescript/internal/options"
)

func TestParseOverridesAcceptsStructuredTypeAndConvert(t *testing.T) {
	req := &plugin.GenerateRequest{
		PluginOptions: []byte(`{
			"overrides": [
				{
					"db_type": "uuid",
					"nullable": true,
					"ts_type": {
						"import": "$lib/model/types",
						"type": "UUID"
					}
				},
				{
					"column": "users.birthday",
					"type": "DateTime",
					"raw_type": "string",
					"convert": {
						"import": "$lib/model/types",
						"type": "strToDateTime"
					}
				}
			]
		}`),
	}

	opts, err := options.Parse(req)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(opts.Overrides); got != 2 {
		t.Fatalf("got %d overrides, want 2", got)
	}
	if got := opts.Overrides[0].TSType.Name; got != "UUID" {
		t.Fatalf("got first ts_type %q, want UUID", got)
	}
	if got := opts.Overrides[0].TSType.ImportPath; got != "$lib/model/types" {
		t.Fatalf("got first import %q, want $lib/model/types", got)
	}
	if got := opts.Overrides[1].TSType.Name; got != "DateTime" {
		t.Fatalf("got second type alias %q, want DateTime", got)
	}
	if got := opts.Overrides[1].RawType.Name; got != "string" {
		t.Fatalf("got raw type %q, want string", got)
	}
	if got := opts.Overrides[1].Convert.Name; got != "strToDateTime" {
		t.Fatalf("got converter %q, want strToDateTime", got)
	}
}

func TestParseOverridesRejectsRawTypeWithoutConvert(t *testing.T) {
	req := &plugin.GenerateRequest{
		PluginOptions: []byte(`{
			"overrides": [
				{
					"db_type": "int8",
					"ts_type": "number",
					"raw_type": "string"
				}
			]
		}`),
	}

	_, err := options.Parse(req)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "raw_type requires convert") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseOverridesRejectsDBTypeAndColumnTogether(t *testing.T) {
	req := &plugin.GenerateRequest{
		PluginOptions: []byte(`{
			"overrides": [
				{
					"db_type": "uuid",
					"column": "users.id",
					"ts_type": "UUID"
				}
			]
		}`),
	}

	_, err := options.Parse(req)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "db_type and column are mutually exclusive") {
		t.Fatalf("unexpected error: %v", err)
	}
}
