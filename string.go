package goshape

import (
	"context"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"unicode/utf8"
)

type stringRule func(string) *Issue

// StringSchema parses, normalizes, and validates string values.
type StringSchema struct {
	trim        bool
	rules       []stringRule
	refinements []refinement[string]
}

// String returns a strict string schema.
func String() StringSchema {
	return StringSchema{}
}

// Trim removes leading and trailing Unicode whitespace before validation.
func (s StringSchema) Trim() StringSchema {
	s.trim = true
	return s
}

// Min requires at least n Unicode code points.
func (s StringSchema) Min(n int) StringSchema {
	requireNonNegative("String.Min", n)
	return s.withRule(func(value string) *Issue {
		length := utf8.RuneCountInString(value)
		if length >= n {
			return nil
		}
		return &Issue{
			Code:     CodeTooSmall,
			Message:  fmt.Sprintf("must contain at least %d characters", n),
			Expected: n,
			Received: length,
		}
	})
}

// Max allows at most n Unicode code points.
func (s StringSchema) Max(n int) StringSchema {
	requireNonNegative("String.Max", n)
	return s.withRule(func(value string) *Issue {
		length := utf8.RuneCountInString(value)
		if length <= n {
			return nil
		}
		return &Issue{
			Code:     CodeTooBig,
			Message:  fmt.Sprintf("must contain at most %d characters", n),
			Expected: n,
			Received: length,
		}
	})
}

// Len requires exactly n Unicode code points.
func (s StringSchema) Len(n int) StringSchema {
	requireNonNegative("String.Len", n)
	return s.withRule(func(value string) *Issue {
		length := utf8.RuneCountInString(value)
		if length == n {
			return nil
		}
		code := CodeTooSmall
		if length > n {
			code = CodeTooBig
		}
		return &Issue{
			Code:     code,
			Message:  fmt.Sprintf("must contain exactly %d characters", n),
			Expected: n,
			Received: length,
		}
	})
}

// Pattern requires a regular-expression match.
func (s StringSchema) Pattern(pattern *regexp.Regexp) StringSchema {
	if pattern == nil {
		panic("goshape: String.Pattern regexp must not be nil")
	}
	return s.withRule(func(value string) *Issue {
		if pattern.MatchString(value) {
			return nil
		}
		return &Issue{
			Code:     CodeInvalidFormat,
			Message:  "must match the required pattern",
			Expected: pattern.String(),
			Received: value,
		}
	})
}

// Email requires a plain RFC 5322 mailbox address. Display-name forms are not
// accepted; this validates shape, not domain existence or deliverability.
func (s StringSchema) Email() StringSchema {
	return s.withRule(func(value string) *Issue {
		address, err := mail.ParseAddress(value)
		if err == nil && address.Address == value && strings.Contains(value, "@") {
			return nil
		}
		return &Issue{
			Code:     CodeInvalidEmail,
			Message:  "must be a valid email address",
			Expected: "email",
			Received: value,
		}
	})
}

// Refine adds custom validation after normalization and built-in rules.
func (s StringSchema) Refine(fn func(string) error) StringSchema {
	s.refinements = appendCopy(s.refinements, requireRefinement(fn))
	return s
}

// Parse implements Schema[string].
func (s StringSchema) Parse(value any) (string, error) {
	return s.ParseContext(context.Background(), value)
}

// ParseContext implements Schema[string].
func (s StringSchema) ParseContext(ctx context.Context, value any) (string, error) {
	if err := checkContext(ctx); err != nil {
		return "", err
	}
	parsed, ok := value.(string)
	if !ok {
		return "", validationError(invalidType("string", value))
	}
	if s.trim {
		parsed = strings.TrimSpace(parsed)
	}

	issues := make([]Issue, 0)
	for _, rule := range s.rules {
		if issue := rule(parsed); issue != nil {
			issues = append(issues, *issue)
		}
	}
	refinementIssues, err := runRefinements(ctx, parsed, s.refinements)
	if err != nil {
		return "", err
	}
	issues = append(issues, refinementIssues...)
	if len(issues) != 0 {
		return "", &ValidationError{Issues: issues}
	}
	return parsed, nil
}

func (s StringSchema) withRule(rule stringRule) StringSchema {
	s.rules = appendCopy(s.rules, rule)
	return s
}

func requireNonNegative(method string, value int) {
	if value < 0 {
		panic(fmt.Sprintf("goshape: %s requires a non-negative value", method))
	}
}
