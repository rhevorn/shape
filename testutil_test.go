package shape

import (
	"errors"
	"reflect"
	"testing"
)

func requireIssues(t *testing.T, err error) []Issue {
	t.Helper()
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("expected *ValidationError, got %T: %v", err, err)
	}
	return validation.Issues
}

func requireIssueCodes(t *testing.T, err error, want ...string) []Issue {
	t.Helper()
	issues := requireIssues(t, err)
	got := make([]string, len(issues))
	for i, issue := range issues {
		got[i] = issue.Code
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("issue codes = %v, want %v", got, want)
	}
	return issues
}

func requirePanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	fn()
}
