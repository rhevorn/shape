package validate

import (
	"context"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/rhevorn/shape/internal/stringrule"
)

// StringValidator validates strings with length, content, and format rules.
type StringValidator struct{ base[string] }

func (v StringValidator) add(fn check[string]) StringValidator { v.base = v.base.add(fn); return v }

// Validate collects issues using a background context.
func (v StringValidator) Validate(value string) error { return v.base.Validate(value) }

// ValidateContext collects issues and observes cancellation.
func (v StringValidator) ValidateContext(ctx context.Context, value string) error {
	return v.base.ValidateContext(ctx, value)
}

// ValidateFirst stops after the first issue.
func (v StringValidator) ValidateFirst(value string) error { return v.base.ValidateFirst(value) }

// ValidateFirstContext stops after the first issue and observes cancellation.
func (v StringValidator) ValidateFirstContext(ctx context.Context, value string) error {
	return v.base.ValidateFirstContext(ctx, value)
}

// NotEmpty rejects the empty string.
func (v StringValidator) NotEmpty() StringValidator {
	return v.add(func(_ context.Context, s string) error {
		if s == "" {
			return issueError(CodeInvalidValue, "not_empty.string", nil, s)
		}
		return nil
	})
}

// MinLength requires at least n Unicode code points.
func (v StringValidator) MinLength(n int) StringValidator {
	if n < 0 {
		panic("validate: negative length")
	}
	return v.add(func(_ context.Context, s string) error {
		if utf8.RuneCountInString(s) < n {
			return issueError(CodeTooSmall, "too_small.string", n, s)
		}
		return nil
	})
}

// MaxLength allows at most n Unicode code points.
func (v StringValidator) MaxLength(n int) StringValidator {
	if n < 0 {
		panic("validate: negative length")
	}
	return v.add(func(_ context.Context, s string) error {
		if utf8.RuneCountInString(s) > n {
			return issueError(CodeTooBig, "too_big.string", n, s)
		}
		return nil
	})
}

// Len requires exactly n Unicode code points.
func (v StringValidator) Len(n int) StringValidator {
	if n < 0 {
		panic("validate: negative length")
	}
	return v.add(func(_ context.Context, s string) error {
		if utf8.RuneCountInString(s) != n {
			return issueError(CodeInvalidValue, "string.len", n, s)
		}
		return nil
	})
}

// OneOf requires equality with one listed string.
func (v StringValidator) OneOf(values ...string) StringValidator {
	if len(values) == 0 {
		panic("validate: OneOf requires values")
	}
	values = append([]string(nil), values...)
	return v.add(func(_ context.Context, s string) error {
		for _, x := range values {
			if s == x {
				return nil
			}
		}
		return issueError(CodeInvalidEnum, "invalid_enum", values, s)
	})
}

// Pattern requires a match against the supplied Go regular expression.
func (v StringValidator) Pattern(expr string) StringValidator {
	re, err := regexp.Compile(expr)
	if err != nil || expr == "" {
		panic("validate: invalid pattern")
	}
	return v.add(func(_ context.Context, s string) error {
		if !re.MatchString(s) {
			return issueError(CodeInvalidFormat, "pattern", expr, s)
		}
		return nil
	})
}

// StartsWith requires prefix x.
func (v StringValidator) StartsWith(x string) StringValidator {
	return v.add(func(_ context.Context, s string) error {
		if !strings.HasPrefix(s, x) {
			return issueError(CodeInvalidFormat, "startswith", x, s)
		}
		return nil
	})
}

// EndsWith requires suffix x.
func (v StringValidator) EndsWith(x string) StringValidator {
	return v.add(func(_ context.Context, s string) error {
		if !strings.HasSuffix(s, x) {
			return issueError(CodeInvalidFormat, "endswith", x, s)
		}
		return nil
	})
}

// Contains requires substring x.
func (v StringValidator) Contains(x string) StringValidator {
	return v.add(func(_ context.Context, s string) error {
		if !strings.Contains(s, x) {
			return issueError(CodeInvalidFormat, "contains", x, s)
		}
		return nil
	})
}

// Email requires one plain email mailbox address.
func (v StringValidator) Email() StringValidator {
	return v.add(func(_ context.Context, s string) error {
		if !stringrule.Email(s) {
			return issueError(CodeInvalidEmail, "email", nil, s)
		}
		return nil
	})
}

// URL requires an absolute URL with a host.
func (v StringValidator) URL() StringValidator {
	return v.add(func(_ context.Context, s string) error {
		if !stringrule.URL(s) {
			return issueError(CodeInvalidURL, "url", nil, s)
		}
		return nil
	})
}

// UUID requires canonical hyphenated UUID syntax.
func (v StringValidator) UUID() StringValidator {
	return v.add(func(_ context.Context, s string) error {
		if !stringrule.UUID(s) {
			return issueError(CodeInvalidUUID, "uuid", nil, s)
		}
		return nil
	})
}

// IP requires an IPv4 or IPv6 address.
func (v StringValidator) IP() StringValidator {
	return v.add(func(_ context.Context, s string) error {
		if !stringrule.IP(s) {
			return issueError(CodeInvalidIP, "ip", nil, s)
		}
		return nil
	})
}

// Refine appends custom validation rules.
func (v StringValidator) Refine(fns ...func(string) error) StringValidator {
	for _, fn := range fns {
		if fn == nil {
			panic("validate: nil refine")
		}
		f := fn
		v = v.add(func(_ context.Context, s string) error { return f(s) })
	}
	return v
}

// RefineContext appends context-aware custom validation rules.
func (v StringValidator) RefineContext(fns ...func(context.Context, string) error) StringValidator {
	for _, fn := range fns {
		if fn == nil {
			panic("validate: nil refine")
		}
		v = v.add(fn)
	}
	return v
}

// And appends validators that run after this validator's rules.
func (v StringValidator) And(vs ...Validator[string]) StringValidator {
	v.base = v.base.andAll(vs...)
	return v
}

// Label sets the human-readable label on otherwise unlabeled issues.
func (v StringValidator) Label(s string) StringValidator {
	v.base = v.base.clone()
	v.base.label = s
	return v
}
