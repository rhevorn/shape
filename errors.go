package shape

import (
	"errors"
	"fmt"
)

// Stable machine-readable validation issue codes.
const (
	CodeInvalidType     = "invalid_type"
	CodeRequired        = "required"
	CodeTooSmall        = "too_small"
	CodeTooBig          = "too_big"
	CodeTooDeep         = "too_deep"
	CodeInvalidFormat   = "invalid_format"
	CodeInvalidValue    = "invalid_value"
	CodeInvalidEnum     = "invalid_enum"
	CodeInvalidUnion    = "invalid_union"
	CodeInvalidString   = "invalid_string"
	CodeInvalidNumber   = "invalid_number"
	CodeInvalidEmail    = "invalid_email"
	CodeInvalidURL      = "invalid_url"
	CodeInvalidUUID     = "invalid_uuid"
	CodeInvalidIP       = "invalid_ip"
	CodeUnknownField    = "unknown_field"
	CodeTooManyIssues   = "too_many_issues"
	CodeCustom          = "custom"
	CodeTransformFailed = "transform_failed"
)

// DefaultMaxIssues is the maximum number of issues retained by one composite
// validation result. The final issue reports truncation when more failures are
// found.
const DefaultMaxIssues = 100

// Issue describes one validation failure.
type Issue struct {
	Code     string `json:"code"`
	Path     Path   `json:"path"`
	Message  string `json:"message"`
	Expected any    `json:"expected,omitempty"`
	Received any    `json:"received,omitempty"`
}

// NewIssue creates an issue suitable for returning from a refinement.
func NewIssue(code, message string) *Issue {
	return &Issue{Code: code, Message: message}
}

// Error implements error.
func (i Issue) Error() string {
	if len(i.Path) == 0 {
		return i.Message
	}
	return fmt.Sprintf("%s: %s", i.Path.String(), i.Message)
}

// ValidationError contains all validation issues found during a parse.
type ValidationError struct {
	Issues []Issue `json:"issues"`
}

// Error implements error.
func (e *ValidationError) Error() string {
	if e == nil || len(e.Issues) == 0 {
		return "validation failed"
	}
	if len(e.Issues) == 1 {
		return "validation failed: " + e.Issues[0].Error()
	}
	return fmt.Sprintf(
		"validation failed with %d issues; first issue: %s",
		len(e.Issues),
		e.Issues[0].Error(),
	)
}

func validationError(issue Issue) *ValidationError {
	return &ValidationError{Issues: []Issue{issue}}
}

func issuesFromError(err error) []Issue {
	if err == nil {
		return nil
	}

	var validation *ValidationError
	if errors.As(err, &validation) {
		if validation == nil {
			return []Issue{{Code: CodeCustom, Message: "validation failed"}}
		}
		result, _ := appendIssuesBounded(nil, validation.Issues...)
		return result
	}

	var issue *Issue
	if errors.As(err, &issue) {
		if issue == nil {
			return []Issue{{Code: CodeCustom, Message: "validation failed"}}
		}
		return []Issue{*issue}
	}

	return []Issue{{Code: CodeCustom, Message: err.Error()}}
}

func tooManyIssues() Issue {
	return Issue{
		Code:     CodeTooManyIssues,
		Message:  "additional validation issues were omitted",
		Expected: fmt.Sprintf("at most %d reported issues", DefaultMaxIssues),
	}
}

// appendIssuesBounded preserves issue order while limiting retained work. The
// boolean result tells callers that no further siblings need to be visited.
func appendIssuesBounded(current []Issue, additions ...Issue) ([]Issue, bool) {
	if len(additions) == 0 {
		return current, len(current) > 0 && current[len(current)-1].Code == CodeTooManyIssues
	}
	if len(current) >= DefaultMaxIssues {
		current = current[:DefaultMaxIssues]
		current[DefaultMaxIssues-1] = tooManyIssues()
		return current, true
	}
	remaining := DefaultMaxIssues - len(current)
	if len(additions) <= remaining {
		return append(current, additions...), false
	}
	if remaining > 1 {
		current = append(current, additions[:remaining-1]...)
	}
	current = append(current, tooManyIssues())
	return current, true
}

func prefixIssues(issues []Issue, segment PathSegment) []Issue {
	result := make([]Issue, len(issues))
	for i, issue := range issues {
		result[i] = issue
		result[i].Path = issue.Path.prefixed(segment)
	}
	return result
}
