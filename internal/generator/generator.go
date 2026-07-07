package generator

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/sqlc-dev/plugin-sdk-go/plugin"

	"github.com/tylergannon/sqlc-gen-typescript/internal/model"
	"github.com/tylergannon/sqlc-gen-typescript/internal/options"
	"github.com/tylergannon/sqlc-gen-typescript/internal/postgres"
	"github.com/tylergannon/sqlc-gen-typescript/internal/ts"
	"github.com/tylergannon/sqlc-gen-typescript/internal/validate"
)

func Generate(_ context.Context, req *plugin.GenerateRequest) (*plugin.GenerateResponse, error) {
	opts, err := options.Parse(req)
	if err != nil {
		return nil, err
	}

	enumMap := postgres.BuildEnumMap(req)
	queryMap := map[string][]*plugin.Query{}
	var filenames []string
	for _, query := range req.GetQueries() {
		filename := query.GetFilename()
		if _, ok := queryMap[filename]; !ok {
			filenames = append(filenames, filename)
		}
		queryMap[filename] = append(queryMap[filename], query)
	}

	resp := &plugin.GenerateResponse{}
	for _, filename := range filenames {
		renderFile, err := buildFile(filename, queryMap[filename], enumMap, opts)
		if err != nil {
			return nil, err
		}
		resp.Files = append(resp.Files, &plugin.File{
			Name:     strings.Replace(filename, ".", "_", 1) + ".ts",
			Contents: []byte(ts.Render(renderFile)),
		})
	}
	return resp, nil
}

func buildFile(filename string, queries []*plugin.Query, enumMap postgres.EnumMap, opts *options.Options) (ts.File, error) {
	renderFile := ts.File{}
	fileEnums := map[string]bool{}
	for _, query := range queries {
		renderQuery := &ts.Query{
			Name:      query.GetName(),
			Cmd:       query.GetCmd(),
			Text:      query.GetText(),
			ArgsName:  query.GetName() + "Args",
			RowName:   query.GetName() + "Row",
			RawParams: query.GetParams(),
		}

		if len(query.GetParams()) > 0 {
			names := make([]string, 0, len(query.GetParams()))
			for i, param := range query.GetParams() {
				name := model.ArgName(i, param.GetColumn().GetName())
				names = append(names, name)
				field, err := fieldForColumn(name, param.GetColumn(), enumMap, opts, false)
				if err != nil {
					return ts.File{}, fmt.Errorf("error in query %q (%s): %w", query.GetName(), filename, err)
				}
				renderQuery.Params = append(renderQuery.Params, field)
				trackColumnSharedTypes(&renderFile, fileEnums, enumMap, param.GetColumn())
			}
			if err := validate.AssertUniqueNames(validate.UniqueNamesOptions{
				Kind:      "argument",
				QueryName: query.GetName(),
				FileName:  filename,
				Names:     names,
			}); err != nil {
				return ts.File{}, err
			}
		}

		if len(query.GetColumns()) > 0 {
			names := make([]string, 0, len(query.GetColumns()))
			for i, column := range query.GetColumns() {
				name := model.ColName(i, column.GetName())
				names = append(names, name)
				field, err := fieldForColumn(name, column, enumMap, opts, true)
				if err != nil {
					return ts.File{}, fmt.Errorf("error in query %q (%s): %w", query.GetName(), filename, err)
				}
				renderQuery.Columns = append(renderQuery.Columns, field)
				trackColumnSharedTypes(&renderFile, fileEnums, enumMap, column)
			}
			if err := validate.AssertUniqueNames(validate.UniqueNamesOptions{
				Kind:      "column",
				QueryName: query.GetName(),
				FileName:  filename,
				Names:     names,
			}); err != nil {
				return ts.File{}, err
			}
		}

		if renderQuery.RowName == query.GetName()+"Row" && len(renderQuery.Columns) == 0 {
			renderQuery.RowName = ""
		}
		renderFile.Queries = append(renderFile.Queries, renderQuery)
	}

	for _, query := range queries {
		for _, param := range query.GetParams() {
			appendEnumDecl(&renderFile, fileEnums, enumMap, param.GetColumn())
		}
		for _, column := range query.GetColumns() {
			appendEnumDecl(&renderFile, fileEnums, enumMap, column)
		}
	}
	return renderFile, nil
}

func fieldForColumn(name string, column *plugin.Column, enumMap postgres.EnumMap, opts *options.Options, allowConverter bool) (*ts.Field, error) {
	typ, err := postgres.ColumnType(enumMap, column)
	if err != nil {
		return nil, err
	}
	field := &ts.Field{Name: name, Type: typ}
	override := matchOverride(opts, column)
	if override == nil {
		return field, nil
	}

	field.Type = typeRefWithOverride(typ, override.TSType)
	if allowConverter && !override.Convert.IsZero() {
		field.WireType = typ
		if !override.RawType.IsZero() {
			field.WireType = typeRefWithOverride(typ, override.RawType)
		}
		field.Converter = &ts.ImportRef{
			Name:       override.Convert.Name,
			ImportPath: override.Convert.ImportPath,
		}
	}
	return field, nil
}

func typeRefWithOverride(base model.TypeRef, spec options.SymbolSpec) model.TypeRef {
	base.Name = spec.Name
	base.ImportPath = spec.ImportPath
	return base
}

func matchOverride(opts *options.Options, column *plugin.Column) *options.Override {
	if opts == nil {
		return nil
	}
	for i := range opts.Overrides {
		override := &opts.Overrides[i]
		if override.Column != "" && columnOverrideMatches(override.Column, column) {
			return override
		}
	}
	for i := range opts.Overrides {
		override := &opts.Overrides[i]
		if override.DBType == "" {
			continue
		}
		if normalizeOverrideType(override.DBType) != postgres.NormalizedTypeName(column) {
			continue
		}
		nullable := false
		if override.Nullable != nil {
			nullable = *override.Nullable
		}
		if nullable != !column.GetNotNull() {
			continue
		}
		if override.Unsigned != column.GetUnsigned() {
			continue
		}
		return override
	}
	return nil
}

func columnOverrideMatches(pattern string, column *plugin.Column) bool {
	pattern = strings.ToLower(pattern)
	return slices.Contains(columnOverrideCandidates(column), pattern)
}

func columnOverrideCandidates(column *plugin.Column) []string {
	if column == nil {
		return nil
	}
	columnNames := []string{column.GetName()}
	if column.GetOriginalName() != "" && column.GetOriginalName() != column.GetName() {
		columnNames = append(columnNames, column.GetOriginalName())
	}

	var candidates []string
	table := column.GetTable()
	for _, columnName := range columnNames {
		columnName = strings.ToLower(columnName)
		if columnName == "" {
			continue
		}
		if table != nil && table.GetName() != "" {
			tableName := strings.ToLower(table.GetName())
			schemaName := strings.ToLower(table.GetSchema())
			catalogName := strings.ToLower(table.GetCatalog())
			candidates = append(candidates, tableName+"."+columnName)
			if schemaName != "" {
				candidates = append(candidates, schemaName+"."+tableName+"."+columnName)
			}
			if catalogName != "" && schemaName != "" {
				candidates = append(candidates, catalogName+"."+schemaName+"."+tableName+"."+columnName)
			}
		}
		if column.GetScope() != "" {
			candidates = append(candidates, strings.ToLower(column.GetScope())+"."+columnName)
		}
		candidates = append(candidates, columnName)
	}
	return candidates
}

func normalizeOverrideType(typeName string) string {
	return strings.TrimPrefix(strings.ToLower(typeName), "pg_catalog.")
}

func trackColumnSharedTypes(file *ts.File, _ map[string]bool, enumMap postgres.EnumMap, column *plugin.Column) {
	if postgres.IsJSONColumn(column) {
		file.UsesJSON = true
	}
	_ = postgres.EnumName(enumMap, column)
}

func appendEnumDecl(file *ts.File, seen map[string]bool, enumMap postgres.EnumMap, column *plugin.Column) {
	enumName := postgres.EnumName(enumMap, column)
	if enumName == "" || seen[enumName] {
		return
	}
	seen[enumName] = true
	enumDef := enumMap[enumName]
	file.Enums = append(file.Enums, ts.EnumDecl{
		Name: enumName,
		Vals: enumDef.GetVals(),
	})
}
