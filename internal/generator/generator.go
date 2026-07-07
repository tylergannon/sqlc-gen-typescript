package generator

import (
	"context"
	"fmt"
	"strings"

	"github.com/sqlc-dev/plugin-sdk-go/plugin"

	"github.com/tylergannon/sqlc-gen-typescript/internal/model"
	"github.com/tylergannon/sqlc-gen-typescript/internal/options"
	"github.com/tylergannon/sqlc-gen-typescript/internal/postgres"
	"github.com/tylergannon/sqlc-gen-typescript/internal/ts"
	"github.com/tylergannon/sqlc-gen-typescript/internal/validate"
)

func Generate(_ context.Context, req *plugin.GenerateRequest) (*plugin.GenerateResponse, error) {
	if _, err := options.Parse(req); err != nil {
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
		renderFile, err := buildFile(filename, queryMap[filename], enumMap)
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

func buildFile(filename string, queries []*plugin.Query, enumMap postgres.EnumMap) (ts.File, error) {
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
				field, err := fieldForColumn(name, param.GetColumn(), enumMap)
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
				field, err := fieldForColumn(name, column, enumMap)
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

func fieldForColumn(name string, column *plugin.Column, enumMap postgres.EnumMap) (*ts.Field, error) {
	typ, err := postgres.ColumnType(enumMap, column)
	if err != nil {
		return nil, err
	}
	return &ts.Field{Name: name, Type: typ}, nil
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
