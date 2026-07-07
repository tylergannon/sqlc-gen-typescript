package options

import (
	"encoding/json"
	"fmt"
	"strings"

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
	DBType   string     `json:"db_type,omitempty"`
	Column   string     `json:"column,omitempty"`
	Nullable *bool      `json:"nullable,omitempty"`
	Unsigned bool       `json:"unsigned,omitempty"`
	TSType   SymbolSpec `json:"ts_type"`
	RawType  SymbolSpec `json:"raw_type"`
	Convert  SymbolSpec `json:"convert"`
}

type SymbolSpec struct {
	ImportPath string
	Name       string
}

func (s SymbolSpec) IsZero() bool {
	return s.Name == "" && s.ImportPath == ""
}

func (s *SymbolSpec) UnmarshalJSON(data []byte) error {
	var name string
	if err := json.Unmarshal(data, &name); err == nil {
		s.Name = name
		s.ImportPath = ""
		return nil
	}

	var obj struct {
		ImportPath string `json:"import"`
		Type       string `json:"type"`
		Name       string `json:"name"`
	}
	if err := json.Unmarshal(data, &obj); err != nil {
		return fmt.Errorf("expected string or object: %w", err)
	}
	name = obj.Type
	if name == "" {
		name = obj.Name
	}
	s.Name = name
	s.ImportPath = obj.ImportPath
	return nil
}

func (o *Override) UnmarshalJSON(data []byte) error {
	var raw struct {
		DBType   string          `json:"db_type"`
		Column   string          `json:"column"`
		Nullable *bool           `json:"nullable"`
		Unsigned bool            `json:"unsigned"`
		TSType   json.RawMessage `json:"ts_type"`
		Type     json.RawMessage `json:"type"`
		RawType  json.RawMessage `json:"raw_type"`
		Convert  json.RawMessage `json:"convert"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	o.DBType = raw.DBType
	o.Column = raw.Column
	o.Nullable = raw.Nullable
	o.Unsigned = raw.Unsigned
	if len(raw.TSType) > 0 {
		if err := json.Unmarshal(raw.TSType, &o.TSType); err != nil {
			return fmt.Errorf("ts_type: %w", err)
		}
	} else if len(raw.Type) > 0 {
		if err := json.Unmarshal(raw.Type, &o.TSType); err != nil {
			return fmt.Errorf("type: %w", err)
		}
	}
	if len(raw.RawType) > 0 {
		if err := json.Unmarshal(raw.RawType, &o.RawType); err != nil {
			return fmt.Errorf("raw_type: %w", err)
		}
	}
	if len(raw.Convert) > 0 {
		if err := json.Unmarshal(raw.Convert, &o.Convert); err != nil {
			return fmt.Errorf("convert: %w", err)
		}
	}
	return nil
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
		if override.TSType.Name == "" {
			return nil, fmt.Errorf("invalid override %d: ts_type is required", i)
		}
		if strings.TrimSpace(override.TSType.Name) == "" {
			return nil, fmt.Errorf("invalid override %d: ts_type is required", i)
		}
		if strings.TrimSpace(override.TSType.ImportPath) != override.TSType.ImportPath {
			return nil, fmt.Errorf("invalid override %d: ts_type import path must not have leading or trailing whitespace", i)
		}
		if override.DBType == "" && override.Column == "" {
			return nil, fmt.Errorf("invalid override %d: db_type or column is required", i)
		}
		if override.DBType != "" && override.Column != "" {
			return nil, fmt.Errorf("invalid override %d: db_type and column are mutually exclusive", i)
		}
		if !override.RawType.IsZero() && override.Convert.IsZero() {
			return nil, fmt.Errorf("invalid override %d: raw_type requires convert", i)
		}
		if !override.RawType.IsZero() {
			if strings.TrimSpace(override.RawType.Name) == "" {
				return nil, fmt.Errorf("invalid override %d: raw_type is required", i)
			}
			if strings.TrimSpace(override.RawType.ImportPath) != override.RawType.ImportPath {
				return nil, fmt.Errorf("invalid override %d: raw_type import path must not have leading or trailing whitespace", i)
			}
		}
		if !override.Convert.IsZero() {
			if strings.TrimSpace(override.Convert.Name) == "" {
				return nil, fmt.Errorf("invalid override %d: convert type is required", i)
			}
			if strings.TrimSpace(override.Convert.ImportPath) != override.Convert.ImportPath {
				return nil, fmt.Errorf("invalid override %d: convert import path must not have leading or trailing whitespace", i)
			}
		}
	}
	return &opts, nil
}
