package shape

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/rhevorn/shape/internal/spec"
	"github.com/rhevorn/shape/internal/validationlocale"
	"github.com/rhevorn/shape/internal/validationmsg"
	"github.com/rhevorn/shape/validate"
)

type ruleError struct {
	code               string
	key                string
	expected, received any
}

func (e *ruleError) Error() string { return "shape: validation rule failed: " + e.key }

func ruleFailure(name string, expected, received any) error {
	code, key := validate.CodeInvalidValue, "invalid_value"
	switch name {
	case "min", "gte":
		code = validate.CodeTooSmall
		if collectionValue(received) {
			key = "too_small.collection"
		} else {
			key = "number.gte"
		}
	case "gt", "positive":
		code = validate.CodeTooSmall
		key = "number.gt"
	case "max", "lte":
		code = validate.CodeTooBig
		if collectionValue(received) {
			key = "too_big.collection"
		} else {
			key = "number.lte"
		}
	case "lt", "negative":
		code = validate.CodeTooBig
		key = "number.lt"
	case "nonnegative":
		code = validate.CodeTooSmall
		key = "number.gte"
	case "minlength":
		code = validate.CodeTooSmall
		key = "too_small.string"
	case "maxlength":
		code = validate.CodeTooBig
		key = "too_big.string"
	case "len":
		if collectionValue(received) {
			key = "collection.len"
		} else {
			key = "string.len"
		}
	case "between":
		key = "number.between"
	case "notnull":
		key = "not_null"
	case "notempty":
		switch reflect.TypeOf(received).Kind() {
		case reflect.String:
			key = "not_empty.string"
		case reflect.Slice, reflect.Map:
			key = "not_empty.collection"
		case reflect.Pointer:
			key = "not_empty.pointer"
		}
	case "pattern":
		code = validate.CodeInvalidFormat
		key = "pattern"
	case "startswith":
		code = validate.CodeInvalidFormat
		key = "startswith"
	case "endswith":
		code = validate.CodeInvalidFormat
		key = "endswith"
	case "contains":
		code = validate.CodeInvalidFormat
		key = "contains"
	case "email":
		code = validate.CodeInvalidEmail
		key = "email"
	case "url":
		code = validate.CodeInvalidURL
		key = "url"
	case "uuid":
		code = validate.CodeInvalidUUID
		key = "uuid"
	case "ip":
		code = validate.CodeInvalidIP
		key = "ip"
	case "oneof":
		code = validate.CodeInvalidEnum
		key = "invalid_enum"
	case "unique":
		key = "unique"
	case "unique_limit":
		code = validate.CodeTooBig
		key = "unique_limit"
	}
	return &ruleError{code: code, key: key, expected: expected, received: received}
}

func ruleIssue(ctx context.Context, err error, path validate.Path, label string) (validate.Issue, bool) {
	var failure *ruleError
	if !errors.As(err, &failure) {
		return validate.Issue{}, false
	}
	return validate.Issue{
		Code: failure.code, Path: cloneValidatePath(path), Label: label,
		Message: validationmsg.Render(
			validationlocale.Get(ctx) == uint8(validate.SimplifiedChinese),
			failure.key,
			label,
			failure.expected,
		),
		Expected: failure.expected, Received: failure.received,
	}, true
}

func collectionValue(v any) bool {
	t := reflect.TypeOf(v)
	return t != nil && (t.Kind() == reflect.Slice || t.Kind() == reflect.Map)
}
func numericKind(k reflect.Kind) bool {
	return k >= reflect.Int && k <= reflect.Float64 && k != reflect.Uintptr
}
func compareNumber(a, b reflect.Value) int {
	switch a.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if a.Int() < b.Int() {
			return -1
		}
		if a.Int() > b.Int() {
			return 1
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if a.Uint() < b.Uint() {
			return -1
		}
		if a.Uint() > b.Uint() {
			return 1
		}
	default:
		if a.Float() < b.Float() {
			return -1
		}
		if a.Float() > b.Float() {
			return 1
		}
	}
	return 0
}
func compileRule(t reflect.Type, r spec.Rule) compiledRule {
	name := spec.RuleName(r)
	bad := func() { panic(fmt.Sprintf("shape: rule %s does not support %v or its arguments", name, t)) }
	args := spec.RuleArguments(r)
	if name == "custom" {
		check := spec.RuleCheck(r)
		if spec.RuleTargetType(r) != t || check == nil {
			bad()
		}
		return check
	}
	var check func(context.Context, reflect.Value) bool
	var expected any
	if len(args) > 0 {
		expected = args[0]
	}
	switch name {
	case "notnull":
		if !nilable(t) || len(args) != 0 {
			bad()
		}
		check = func(_ context.Context, v reflect.Value) bool { return !v.IsNil() }
	case "notempty":
		if len(args) != 0 || (t.Kind() != reflect.String && t.Kind() != reflect.Slice && t.Kind() != reflect.Map && t.Kind() != reflect.Pointer) {
			bad()
		}
		check = func(_ context.Context, v reflect.Value) bool {
			switch v.Kind() {
			case reflect.String:
				return v.Len() > 0
			case reflect.Slice, reflect.Map:
				return !v.IsNil() && v.Len() > 0
			case reflect.Pointer:
				return !v.IsNil()
			}
			return true
		}
	case "minlength", "maxlength", "len":
		if len(args) != 1 || (t.Kind() != reflect.String && (name != "len" || (t.Kind() != reflect.Slice && t.Kind() != reflect.Map))) {
			bad()
		}
		n, ok := args[0].(int)
		if !ok || n < 0 {
			bad()
		}
		check = func(_ context.Context, v reflect.Value) bool {
			l := v.Len()
			if v.Kind() == reflect.String {
				l = utf8.RuneCountInString(v.String())
			}
			switch name {
			case "minlength":
				return l >= n
			case "maxlength":
				return l <= n
			}
			return l == n
		}
	case "oneof":
		if len(args) == 0 || (t.Kind() != reflect.String && !numericKind(t.Kind())) {
			bad()
		}
		values := make([]reflect.Value, len(args))
		expectedValues := make([]any, len(args))
		for i, arg := range args {
			value := reflect.ValueOf(arg)
			if !value.IsValid() || value.Type() != t {
				var ok bool
				value, ok = safeNumericConversion(value, t)
				if !ok {
					bad()
				}
			}
			values[i] = value
			expectedValues[i] = value.Interface()
		}
		expected = expectedValues
		check = func(_ context.Context, v reflect.Value) bool {
			for _, candidate := range values {
				if t.Kind() == reflect.String {
					if v.String() == candidate.String() {
						return true
					}
				} else if compareNumber(v, candidate) == 0 {
					return true
				}
			}
			return false
		}
	case "min", "max", "gt", "gte", "lt", "lte", "between", "positive", "negative", "nonnegative":
		collection := t.Kind() == reflect.Slice || t.Kind() == reflect.Map
		if collection {
			if (name != "min" && name != "max") || len(args) != 1 {
				bad()
			}
			bound, ok := safeNumericConversion(reflect.ValueOf(args[0]), reflect.TypeFor[int]())
			if !ok || bound.Int() < 0 {
				bad()
			}
			n := int(bound.Int())
			check = func(_ context.Context, v reflect.Value) bool {
				if name == "min" {
					return v.Len() >= n
				}
				return v.Len() <= n
			}
			break
		}
		if !numericKind(t.Kind()) {
			bad()
		}
		switch name {
		case "positive", "negative", "nonnegative":
			if len(args) != 0 {
				bad()
			}
			args = []any{reflect.Zero(t).Interface()}
			expected = args[0]
		case "between":
			if len(args) != 2 {
				bad()
			}
		default:
			if len(args) != 1 {
				bad()
			}
		}
		bounds := make([]reflect.Value, len(args))
		for i, a := range args {
			v := reflect.ValueOf(a)
			if !v.IsValid() || v.Type() != t {
				var ok bool
				v, ok = safeNumericConversion(v, t)
				if !ok {
					bad()
				}
			}
			bounds[i] = v
		}
		expected = bounds[0].Interface()
		if name == "between" {
			values := make([]any, len(bounds))
			for i := range bounds {
				values[i] = bounds[i].Interface()
			}
			expected = values
		}
		if name == "between" && compareNumber(bounds[0], bounds[1]) > 0 {
			bad()
		}
		check = func(_ context.Context, v reflect.Value) bool {
			c := compareNumber(v, bounds[0])
			switch name {
			case "min", "gte", "nonnegative":
				return c >= 0
			case "gt", "positive":
				return c > 0
			case "max", "lte":
				return c <= 0
			case "lt", "negative":
				return c < 0
			case "between":
				return c >= 0 && compareNumber(v, bounds[1]) <= 0
			}
			return false
		}
	case "startswith", "endswith", "contains", "pattern":
		if t.Kind() != reflect.String || len(args) != 1 {
			bad()
		}
		s, ok := args[0].(string)
		if !ok {
			bad()
		}
		var re *regexp.Regexp
		if name == "pattern" {
			var err error
			re, err = regexp.Compile(s)
			if err != nil || s == "" {
				bad()
			}
		}
		check = func(_ context.Context, v reflect.Value) bool {
			switch name {
			case "startswith":
				return strings.HasPrefix(v.String(), s)
			case "endswith":
				return strings.HasSuffix(v.String(), s)
			case "contains":
				return strings.Contains(v.String(), s)
			}
			return re.MatchString(v.String())
		}
	case "email", "url", "uuid", "ip":
		if t.Kind() != reflect.String || len(args) != 0 {
			bad()
		}
		check = func(_ context.Context, v reflect.Value) bool {
			s := v.String()
			switch name {
			case "email":
				a, e := mail.ParseAddress(s)
				return e == nil && a.Address == s && !strings.ContainsAny(s, "\r\n")
			case "url":
				u, e := url.ParseRequestURI(s)
				return e == nil && u.Scheme != "" && u.Host != ""
			case "ip":
				return net.ParseIP(s) != nil
			case "uuid":
				return uuidPattern.MatchString(s)
			}
			return false
		}
	case "unique":
		if t.Kind() != reflect.Slice || len(args) != 0 {
			bad()
		}
		return func(ctx context.Context, v reflect.Value) error {
			// Bound quadratic deep comparisons; comparable values use a hash set.
			fastComparable := deepEqualMatchesComparable(t.Elem())
			if !fastComparable && v.Len() > defaultMaxDeepUniqueItems {
				return ruleFailure("unique_limit", defaultMaxDeepUniqueItems, v.Len())
			}
			seen := make(map[any]bool)
			for i := 0; i < v.Len(); i++ {
				if e := ctx.Err(); e != nil {
					return e
				}
				a := v.Index(i)
				if fastComparable {
					k := a.Interface()
					if seen[k] {
						return ruleFailure("unique", "unique", k)
					}
					seen[k] = true
					continue
				}
				for j := 0; j < i; j++ {
					if e := ctx.Err(); e != nil {
						return e
					}
					if reflect.DeepEqual(a.Interface(), v.Index(j).Interface()) {
						return ruleFailure("unique", "unique", a.Interface())
					}
				}
			}
			return nil
		}
	default:
		bad()
	}
	return func(ctx context.Context, v reflect.Value) error {
		if check(ctx, v) {
			return nil
		}
		return ruleFailure(name, expected, v.Interface())
	}
}

func deepEqualMatchesComparable(t reflect.Type) bool {
	if !t.Comparable() {
		return false
	}
	switch t.Kind() {
	case reflect.Pointer, reflect.Interface:
		return false
	case reflect.Array:
		return deepEqualMatchesComparable(t.Elem())
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			if !deepEqualMatchesComparable(t.Field(i).Type) {
				return false
			}
		}
	}
	return true
}

var uuidPattern = regexp.MustCompile("^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$")

const defaultMaxDeepUniqueItems = 1024
