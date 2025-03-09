package validate

import (
	"context"
	"errors"
	"fmt"
)

const (
	CodeTooSmall      = "too_small"
	CodeTooBig        = "too_big"
	CodeInvalidFormat = "invalid_format"
	CodeInvalidValue  = "invalid_value"
	CodeInvalidEnum   = "invalid_enum"
	CodeInvalidNumber = "invalid_number"
	CodeInvalidEmail  = "invalid_email"
	CodeInvalidURL    = "invalid_url"
	CodeInvalidUUID   = "invalid_uuid"
	CodeInvalidIP     = "invalid_ip"
	CodeTooDeep       = "too_deep"
	CodeTooManyIssues = "too_many_issues"
	CodeCustom        = "custom"
)

const DefaultMaxIssues = 100

type Issue struct {
	Code      string `json:"code"`
	Path      Path   `json:"path"`
	Message   string `json:"message"`
	Label     string `json:"label,omitempty"`
	Expected  any    `json:"expected,omitempty"`
	Received  any    `json:"received,omitempty"`
	messageID string
}

func (i Issue) Error() string {
	if len(i.Path) == 0 {
		return i.Message
	}
	return fmt.Sprintf("%s: %s", i.Path.String(), i.Message)
}

type Error struct {
	Issues []Issue `json:"issues"`
}

func (e *Error) Error() string {
	if e == nil || len(e.Issues) == 0 {
		return "validation failed"
	}
	if len(e.Issues) == 1 {
		return "validation failed: " + e.Issues[0].Error()
	}
	return fmt.Sprintf("validation failed with %d issues; first issue: %s", len(e.Issues), e.Issues[0].Error())
}
func issue(code, key string, expected, received any) error {
	return &Error{Issues: []Issue{{Code: code, messageID: key, Expected: expected, Received: received}}}
}

func customIssue(err error) []Issue {
	if err == nil {
		return nil
	}
	var ve *Error
	if errors.As(err, &ve) && ve != nil {
		return append([]Issue(nil), ve.Issues...)
	}
	return []Issue{{Code: CodeCustom, Message: err.Error()}}
}

func finish(ctx context.Context, issues []Issue) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(issues) == 0 {
		return nil
	}
	lang := languageFromContext(ctx)
	for i := range issues {
		issues[i] = renderIssue(issues[i], lang)
	}
	return &Error{Issues: issues}
}

func appendIssues(dst []Issue, additions []Issue, first bool) ([]Issue, bool) {
	if first && len(additions) > 0 {
		return append(dst, additions[0]), true
	}
	remaining := DefaultMaxIssues - len(dst)
	if remaining <= 0 {
		dst[DefaultMaxIssues-1] = Issue{
			Code: CodeTooManyIssues, messageID: "too_many_issues", Expected: DefaultMaxIssues,
		}
		return dst, true
	}
	if len(additions) > remaining {
		if remaining > 1 {
			dst = append(dst, additions[:remaining-1]...)
		}
		dst = append(dst, Issue{Code: CodeTooManyIssues, messageID: "too_many_issues", Expected: DefaultMaxIssues})
		return dst, true
	}
	dst = append(dst, additions...)
	return dst, false
}

func prefix(issues []Issue, segment PathSegment) []Issue {
	out := make([]Issue, len(issues))
	for i, v := range issues {
		out[i] = v
		out[i].Path = append(Path{segment}, v.Path...)
	}
	return out
}

func withLabel(issues []Issue, label string) []Issue {
	if label == "" {
		return issues
	}
	out := append([]Issue(nil), issues...)
	for i := range out {
		if out[i].Label == "" {
			out[i].Label = label
		}
	}
	return out
}
