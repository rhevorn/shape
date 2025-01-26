package shape

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestIssueAndValidationError(t *testing.T) {
	t.Parallel()

	issue := Issue{
		Code:     CodeTooSmall,
		Path:     Path{FieldPath("age")},
		Message:  "must be at least 18",
		Expected: 18,
		Received: 16,
	}
	err := validationError(issue)

	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatal("errors.As did not find *ValidationError")
	}
	if got, want := err.Error(), "validation failed: age: must be at least 18"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}

	data, marshalErr := json.Marshal(issue)
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	if got, want := string(data), `{"code":"too_small","path":["age"],"message":"must be at least 18","expected":18,"received":16}`; got != want {
		t.Fatalf("json.Marshal(Issue) = %s, want %s", got, want)
	}
}

func TestIssuesFromErrorCopiesValidationIssues(t *testing.T) {
	t.Parallel()

	source := &ValidationError{Issues: []Issue{{Code: CodeRequired, Message: "is required"}}}
	issues := issuesFromError(source)
	issues[0].Code = CodeCustom

	if source.Issues[0].Code != CodeRequired {
		t.Fatal("issuesFromError returned aliased issue storage")
	}
}
