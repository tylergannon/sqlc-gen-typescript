package options

import (
	"encoding/json"
	"fmt"

	"github.com/sqlc-dev/plugin-sdk-go/plugin"
)

const (
	RuntimePostgres = "postgres"
	BigintNumber    = "number"
)

type Options struct {
	Driver    string     `json:"driver,omitempty"`
	Runtime   string     `json:"runtime,omitempty"`
	Bigint    string     `json:"bigint,omitempty"`
	Overrides []Override `json:"overrides,omitempty"`
}

type Override struct {
	DBType   string `json:"db_type,omitempty"`
	Column   string `json:"column,omitempty"`
	Nullable *bool  `json:"nullable,omitempty"`
	TSType   string `json:"ts_type,omitempty"`
}

func Parse(req *plugin.GenerateRequest) (*Options, error) {
	opts := Options{
		Driver:  RuntimePostgres,
		Runtime: RuntimePostgres,
		Bigint:  BigintNumber,
	}
	if len(req.GetPluginOptions()) > 0 {
		if err := json.Unmarshal(req.GetPluginOptions(), &opts); err != nil {
			return nil, fmt.Errorf("unmarshalling plugin options: %w", err)
		}
	}
	if opts.Driver == "" {
		opts.Driver = RuntimePostgres
	}
	if opts.Runtime == "" {
		opts.Runtime = RuntimePostgres
	}
	if opts.Bigint == "" {
		opts.Bigint = BigintNumber
	}
	if opts.Driver != RuntimePostgres {
		return nil, fmt.Errorf("unsupported driver %q: only postgres is currently supported", opts.Driver)
	}
	if opts.Runtime != RuntimePostgres && opts.Runtime != "postgres.js" {
		return nil, fmt.Errorf("unsupported runtime %q: only postgres is currently supported", opts.Runtime)
	}
	if opts.Bigint != BigintNumber {
		return nil, fmt.Errorf("unsupported bigint policy %q: only number is currently supported", opts.Bigint)
	}
	for i, override := range opts.Overrides {
		if override.TSType == "" {
			return nil, fmt.Errorf("invalid override %d: ts_type is required", i)
		}
		if override.DBType == "" && override.Column == "" {
			return nil, fmt.Errorf("invalid override %d: db_type or column is required", i)
		}
	}
	return &opts, nil
}
