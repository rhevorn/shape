package validate

import (
	"context"
	"net"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"
)

type StringValidator struct{ base[string] }

func (v StringValidator) add(fn check[string]) StringValidator { v.base = v.base.add(fn); return v }
func (v StringValidator) NotEmpty() StringValidator {
	return v.add(func(_ context.Context, s string) error {
		if s == "" {
			return issue(CodeInvalidValue, "not_empty.string", nil, s)
		}
		return nil
	})
}
func (v StringValidator) MinLength(n int) StringValidator {
	if n < 0 {
		panic("validate: negative length")
	}
	return v.add(func(_ context.Context, s string) error {
		if utf8.RuneCountInString(s) < n {
			return issue(CodeTooSmall, "too_small.string", n, s)
		}
		return nil
	})
}
func (v StringValidator) MaxLength(n int) StringValidator {
	if n < 0 {
		panic("validate: negative length")
	}
	return v.add(func(_ context.Context, s string) error {
		if utf8.RuneCountInString(s) > n {
			return issue(CodeTooBig, "too_big.string", n, s)
		}
		return nil
	})
}
func (v StringValidator) Len(n int) StringValidator {
	if n < 0 {
		panic("validate: negative length")
	}
	return v.add(func(_ context.Context, s string) error {
		if utf8.RuneCountInString(s) != n {
			return issue(CodeInvalidValue, "string.len", n, s)
		}
		return nil
	})
}
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
		return issue(CodeInvalidEnum, "invalid_enum", values, s)
	})
}
func (v StringValidator) Pattern(expr string) StringValidator {
	re, err := regexp.Compile(expr)
	if err != nil || expr == "" {
		panic("validate: invalid pattern")
	}
	return v.add(func(_ context.Context, s string) error {
		if !re.MatchString(s) {
			return issue(CodeInvalidFormat, "pattern", expr, s)
		}
		return nil
	})
}
func (v StringValidator) StartsWith(x string) StringValidator {
	return v.add(func(_ context.Context, s string) error {
		if !strings.HasPrefix(s, x) {
			return issue(CodeInvalidFormat, "startswith", x, s)
		}
		return nil
	})
}
func (v StringValidator) EndsWith(x string) StringValidator {
	return v.add(func(_ context.Context, s string) error {
		if !strings.HasSuffix(s, x) {
			return issue(CodeInvalidFormat, "endswith", x, s)
		}
		return nil
	})
}
func (v StringValidator) Contains(x string) StringValidator {
	return v.add(func(_ context.Context, s string) error {
		if !strings.Contains(s, x) {
			return issue(CodeInvalidFormat, "contains", x, s)
		}
		return nil
	})
}
func (v StringValidator) Email() StringValidator {
	return v.add(func(_ context.Context, s string) error {
		a, e := mail.ParseAddress(s)
		if e != nil || a.Address != s || strings.ContainsAny(s, "\r\n") {
			return issue(CodeInvalidEmail, "email", nil, s)
		}
		return nil
	})
}
func (v StringValidator) URL() StringValidator {
	return v.add(func(_ context.Context, s string) error {
		u, e := url.ParseRequestURI(s)
		if e != nil || u.Scheme == "" || u.Host == "" {
			return issue(CodeInvalidURL, "url", nil, s)
		}
		return nil
	})
}

var uuidRE = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func (v StringValidator) UUID() StringValidator {
	return v.add(func(_ context.Context, s string) error {
		if !uuidRE.MatchString(s) {
			return issue(CodeInvalidUUID, "uuid", nil, s)
		}
		return nil
	})
}
func (v StringValidator) IP() StringValidator {
	return v.add(func(_ context.Context, s string) error {
		if net.ParseIP(s) == nil {
			return issue(CodeInvalidIP, "ip", nil, s)
		}
		return nil
	})
}
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
func (v StringValidator) RefineContext(fns ...func(context.Context, string) error) StringValidator {
	for _, fn := range fns {
		if fn == nil {
			panic("validate: nil refine")
		}
		v = v.add(fn)
	}
	return v
}
func (v StringValidator) And(vs ...Validator[string]) Validator[string] { return v.base.And(vs...) }
func (v StringValidator) Label(s string) StringValidator {
	v.base = v.base.clone()
	v.base.label = s
	return v
}
