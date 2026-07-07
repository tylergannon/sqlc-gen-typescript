package validate_test

import (
	"strings"
	"testing"

	"github.com/tylergannon/sqlc-gen-typescript/internal/validate"
)

func TestAssertUniqueNamesThrowsForDuplicateArguments(t *testing.T) {
	err := validate.AssertUniqueNames(validate.UniqueNamesOptions{
		Kind:      "argument",
		QueryName: "GetJobRunStats",
		FileName:  "scheduled_job_runs.sql",
		Names:     []string{"startedAt", "startedAt"},
	})
	if err == nil {
		t.Fatal("expected duplicate argument error")
	}
	if msg := err.Error(); !containsAll(msg, "sqlc.arg", "named parameters") {
		t.Fatalf("expected helpful duplicate argument message, got:\n%s", msg)
	}
}

func TestAssertUniqueNamesAllowsUniqueNames(t *testing.T) {
	err := validate.AssertUniqueNames(validate.UniqueNamesOptions{
		Kind:      "column",
		QueryName: "ListUsers",
		FileName:  "users.sql",
		Names:     []string{"id", "createdAt"},
	})
	if err != nil {
		t.Fatalf("expected unique names to pass: %v", err)
	}
}

func TestAssertUniqueNamesThrowsForDuplicateColumns(t *testing.T) {
	err := validate.AssertUniqueNames(validate.UniqueNamesOptions{
		Kind:      "column",
		QueryName: "ListJobRuns",
		FileName:  "job_runs.sql",
		Names:     []string{"startedAt", "startedAt"},
	})
	if err == nil {
		t.Fatal("expected duplicate column error")
	}
	if msg := err.Error(); !containsAll(msg, "sqlc.narg", "column aliases", "named parameters") {
		t.Fatalf("expected helpful duplicate column message, got:\n%s", msg)
	}
}

func containsAll(s string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(s, part) {
			return false
		}
	}
	return true
}
