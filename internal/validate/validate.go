package validate

import (
	"fmt"
	"sort"
	"strings"
)

const namedParamsDocsURL = "https://docs.sqlc.dev/en/latest/howto/named_parameters.html"

type UniqueNamesOptions struct {
	Kind      string
	QueryName string
	FileName  string
	Names     []string
}

func AssertUniqueNames(opts UniqueNamesOptions) error {
	counts := map[string]int{}
	for _, name := range opts.Names {
		counts[name]++
	}

	var duplicates []string
	for name, count := range counts {
		if count > 1 {
			duplicates = append(duplicates, name)
		}
	}
	if len(duplicates) == 0 {
		return nil
	}
	sort.Strings(duplicates)

	lines := make([]string, 0, len(duplicates))
	for _, duplicate := range duplicates {
		lines = append(lines, "- "+duplicate)
	}

	return fmt.Errorf(strings.Join([]string{
		"sqlc-gen-typescript: ambiguous %s names for query '%s' (%s)",
		"",
		"The TypeScript generator produced duplicate identifier(s):",
		"%s",
		"",
		"Disambiguate using named parameters (sqlc.arg/sqlc.narg) or explicit column aliases.",
		"Docs: %s",
	}, "\n"), opts.Kind, opts.QueryName, opts.FileName, strings.Join(lines, "\n"), namedParamsDocsURL)
}
