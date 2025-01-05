package goshape

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"net/mail"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

type stringRule func(string) *Issue

// StringSchema parses, normalizes, and validates string values.
type StringSchema struct {
	trim        bool
	caseMode    int8
	coerce      bool
	rules       []stringRule
	refinements []refinement[string]
	constraints []map[string]any
	metadata    schemaMetadata
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

// ToLower converts the value to lower case before validation. It overrides a
// previous ToUpper call.
func (s StringSchema) ToLower() StringSchema {
	s.caseMode = -1
	return s
}

// ToUpper converts the value to upper case before validation. It overrides a
// previous ToLower call.
func (s StringSchema) ToUpper() StringSchema {
	s.caseMode = 1
	return s
}

// NonEmpty requires at least one Unicode code point after normalization.
func (s StringSchema) NonEmpty() StringSchema { return s.Min(1) }

// Min requires at least n Unicode code points.
func (s StringSchema) Min(n int) StringSchema {
	requireNonNegative("String.Min", n)
	s = s.withRule(func(value string) *Issue {
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
	return s.withConstraint("minLength", n)
}

// Max allows at most n Unicode code points.
func (s StringSchema) Max(n int) StringSchema {
	requireNonNegative("String.Max", n)
	s = s.withRule(func(value string) *Issue {
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
	return s.withConstraint("maxLength", n)
}

// Len requires exactly n Unicode code points.
func (s StringSchema) Len(n int) StringSchema {
	requireNonNegative("String.Len", n)
	s = s.withRule(func(value string) *Issue {
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
	s = s.withConstraint("minLength", n)
	return s.withConstraint("maxLength", n)
}

// Pattern requires a regular-expression match.
func (s StringSchema) Pattern(pattern *regexp.Regexp) StringSchema {
	if pattern == nil {
		panic("goshape: String.Pattern regexp must not be nil")
	}
	s = s.withRule(func(value string) *Issue {
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
	return s.withConstraint("pattern", pattern.String())
}

// StartsWith requires prefix at the beginning of the normalized value.
func (s StringSchema) StartsWith(prefix string) StringSchema {
	s = s.withRule(func(value string) *Issue {
		if strings.HasPrefix(value, prefix) {
			return nil
		}
		return &Issue{Code: CodeInvalidString, Message: "must start with " + strconv.Quote(prefix), Expected: prefix, Received: value}
	})
	return s.withConstraint("pattern", "^"+regexp.QuoteMeta(prefix))
}

// EndsWith requires suffix at the end of the normalized value.
func (s StringSchema) EndsWith(suffix string) StringSchema {
	s = s.withRule(func(value string) *Issue {
		if strings.HasSuffix(value, suffix) {
			return nil
		}
		return &Issue{Code: CodeInvalidString, Message: "must end with " + strconv.Quote(suffix), Expected: suffix, Received: value}
	})
	return s.withConstraint("pattern", regexp.QuoteMeta(suffix)+"$")
}

// Contains requires part within the normalized value.
func (s StringSchema) Contains(part string) StringSchema {
	s = s.withRule(func(value string) *Issue {
		if strings.Contains(value, part) {
			return nil
		}
		return &Issue{Code: CodeInvalidString, Message: "must contain " + strconv.Quote(part), Expected: part, Received: value}
	})
	return s.withConstraint("pattern", regexp.QuoteMeta(part))
}

// Email requires a plain RFC 5322 mailbox address. Display-name forms are not
// accepted; this validates shape, not domain existence or deliverability.
func (s StringSchema) Email() StringSchema {
	s = s.withRule(func(value string) *Issue {
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
	return s.withConstraint("format", "email")
}

// URL requires an absolute URL with a scheme and host.
func (s StringSchema) URL() StringSchema {
	s = s.withRule(func(value string) *Issue {
		parsed, err := url.Parse(value)
		if err == nil && parsed.Scheme != "" && parsed.Host != "" {
			return nil
		}
		return &Issue{Code: CodeInvalidURL, Message: "must be a valid absolute URL", Expected: "url", Received: value}
	})
	return s.withConstraint("format", "uri")
}

// UUID requires a canonical hyphenated UUID string.
func (s StringSchema) UUID() StringSchema {
	s = s.withRule(func(value string) *Issue {
		if isUUID(value) {
			return nil
		}
		return &Issue{Code: CodeInvalidUUID, Message: "must be a valid UUID", Expected: "uuid", Received: value}
	})
	return s.withConstraint("format", "uuid")
}

// IP requires an IPv4 or IPv6 address.
func (s StringSchema) IP() StringSchema {
	s = s.withRule(func(value string) *Issue {
		if net.ParseIP(value) != nil {
			return nil
		}
		return &Issue{Code: CodeInvalidIP, Message: "must be a valid IP address", Expected: "ip", Received: value}
	})
	return s.withConstraint("format", "ip")
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
	if !ok && s.coerce {
		parsed, ok = coerceStringValue(value)
	}
	if !ok {
		return "", validationError(invalidType("string", value))
	}
	if s.trim {
		parsed = strings.TrimSpace(parsed)
	}
	if s.caseMode < 0 {
		parsed = strings.ToLower(parsed)
	} else if s.caseMode > 0 {
		parsed = strings.ToUpper(parsed)
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

// CoerceString returns a string schema that explicitly converts common scalar
// Go values and JSON numbers to their textual representation.
func CoerceString() StringSchema {
	return StringSchema{coerce: true}
}

// URL returns a string schema for absolute URLs.
func URL() StringSchema { return String().URL() }

// UUID returns a string schema for canonical UUIDs.
func UUID() StringSchema { return String().UUID() }

// IP returns a string schema for IPv4 and IPv6 addresses.
func IP() StringSchema { return String().IP() }

func coerceStringValue(value any) (string, bool) {
	switch typed := value.(type) {
	case []byte:
		return string(typed), true
	case bool:
		return strconv.FormatBool(typed), true
	case int:
		return strconv.Itoa(typed), true
	case int8:
		return strconv.FormatInt(int64(typed), 10), true
	case int16:
		return strconv.FormatInt(int64(typed), 10), true
	case int32:
		return strconv.FormatInt(int64(typed), 10), true
	case int64:
		return strconv.FormatInt(typed, 10), true
	case uint:
		return strconv.FormatUint(uint64(typed), 10), true
	case uint8:
		return strconv.FormatUint(uint64(typed), 10), true
	case uint16:
		return strconv.FormatUint(uint64(typed), 10), true
	case uint32:
		return strconv.FormatUint(uint64(typed), 10), true
	case uint64:
		return strconv.FormatUint(typed, 10), true
	case float32:
		return strconv.FormatFloat(float64(typed), 'g', -1, 32), true
	case float64:
		if math.IsNaN(typed) || math.IsInf(typed, 0) {
			return "", false
		}
		return strconv.FormatFloat(typed, 'g', -1, 64), true
	case json.Number:
		return typed.String(), true
	default:
		return "", false
	}
}

func isUUID(value string) bool {
	if len(value) != 36 || value[8] != '-' || value[13] != '-' || value[18] != '-' || value[23] != '-' {
		return false
	}
	for i := 0; i < len(value); i++ {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			continue
		}
		if !strings.ContainsRune("0123456789abcdefABCDEF", rune(value[i])) {
			return false
		}
	}
	return true
}

func (s StringSchema) withRule(rule stringRule) StringSchema {
	s.rules = appendCopy(s.rules, rule)
	return s
}

func (s StringSchema) withConstraint(key string, value any) StringSchema {
	s.constraints = appendCopy(s.constraints, map[string]any{key: value})
	return s
}

// Title sets schema title metadata.
func (s StringSchema) Title(value string) StringSchema {
	s.metadata = s.metadata.title(value)
	return s
}

// Description sets schema description metadata.
func (s StringSchema) Description(value string) StringSchema {
	s.metadata = s.metadata.description(value)
	return s
}

// Example appends an example value.
func (s StringSchema) Example(value string) StringSchema {
	s.metadata = s.metadata.example(value)
	return s
}

// Deprecated marks the schema as deprecated metadata.
func (s StringSchema) Deprecated() StringSchema {
	s.metadata = s.metadata.deprecated()
	return s
}

// DefaultValue sets descriptive default metadata without changing parsing.
func (s StringSchema) DefaultValue(value string) StringSchema {
	s.metadata = s.metadata.defaultValue(value)
	return s
}

// Metadata returns a copy of the string schema metadata.
func (s StringSchema) Metadata() SchemaMetadata { return copyMetadata(s.metadata) }

func (s StringSchema) buildJSONSchema() (map[string]any, error) {
	if err := unsupportedIfRefined(len(s.refinements)); err != nil {
		return nil, err
	}
	document := map[string]any{"type": "string"}
	applyConstraints(document, s.constraints)
	applyMetadata(document, s.metadata)
	if s.trim || s.caseMode != 0 {
		normalization := make([]string, 0, 2)
		if s.trim {
			normalization = append(normalization, "trim")
		}
		if s.caseMode < 0 {
			normalization = append(normalization, "lowercase")
		} else if s.caseMode > 0 {
			normalization = append(normalization, "uppercase")
		}
		document["x-goshape-normalize"] = normalization
	}
	if s.coerce {
		document["x-goshape-coerce"] = true
	}
	return document, nil
}

func requireNonNegative(method string, value int) {
	if value < 0 {
		panic(fmt.Sprintf("goshape: %s requires a non-negative value", method))
	}
}
