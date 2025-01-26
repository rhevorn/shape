package shape

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Numeric is the set of integer and floating-point types supported by the
// generic Number schema, including user-defined types with these underlying
// representations.
type Numeric interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64
}

type number interface{ Numeric }

type numberRule[N number] func(N) *Issue

// IntSchema parses and validates int values and decoder-native JSON numbers.
type IntSchema struct {
	coerce      bool
	rules       []numberRule[int]
	refinements []refinement[int]
	constraints []map[string]any
}

// Int returns an int schema. It does not coerce strings or other Go number
// types.
func Int() IntSchema { return IntSchema{} }

// Min requires a value greater than or equal to bound.
func (s IntSchema) Min(bound int) IntSchema { return s.Gte(bound) }

// Max requires a value less than or equal to bound.
func (s IntSchema) Max(bound int) IntSchema { return s.Lte(bound) }

// Gt requires a value greater than bound.
func (s IntSchema) Gt(bound int) IntSchema {
	s = s.withRule(lowerRule(bound, false))
	return s.withConstraint("exclusiveMinimum", bound)
}

// Gte requires a value greater than or equal to bound.
func (s IntSchema) Gte(bound int) IntSchema {
	s = s.withRule(lowerRule(bound, true))
	return s.withConstraint("minimum", bound)
}

// Lt requires a value less than bound.
func (s IntSchema) Lt(bound int) IntSchema {
	s = s.withRule(upperRule(bound, false))
	return s.withConstraint("exclusiveMaximum", bound)
}

// Lte requires a value less than or equal to bound.
func (s IntSchema) Lte(bound int) IntSchema {
	s = s.withRule(upperRule(bound, true))
	return s.withConstraint("maximum", bound)
}

// Positive requires a value greater than zero.
func (s IntSchema) Positive() IntSchema { return s.Gt(0) }

// Negative requires a value less than zero.
func (s IntSchema) Negative() IntSchema { return s.Lt(0) }

// NonNegative requires a value greater than or equal to zero.
func (s IntSchema) NonNegative() IntSchema { return s.Gte(0) }

// OneOf restricts values to the provided set.
func (s IntSchema) OneOf(values ...int) IntSchema {
	if len(values) == 0 {
		panic("shape: Int.OneOf requires at least one value")
	}
	s = s.withRule(numberOneOfRule(values))
	return s.withConstraint("enum", append([]int(nil), values...))
}

// Refine adds custom validation after built-in rules.
func (s IntSchema) Refine(fn func(int) error) IntSchema {
	s.refinements = appendCopy(s.refinements, requireRefinement(fn))
	return s
}

// Parse implements Schema[int].
func (s IntSchema) Parse(value any) (int, error) {
	return s.ParseContext(context.Background(), value)
}

// ParseContext implements Schema[int].
func (s IntSchema) ParseContext(ctx context.Context, value any) (int, error) {
	if err := checkContext(ctx); err != nil {
		return 0, err
	}
	parsed, ok := value.(int)
	if !ok && s.coerce {
		if converted, convertedOK := coerceInteger(value, strconv.IntSize); convertedOK {
			parsed, ok = int(converted), true
		}
	}
	if !ok {
		if encoded, encodedOK := value.(json.Number); encodedOK {
			parsed64, parseOK := parseJSONInteger(string(encoded), strconv.IntSize)
			if parseOK {
				parsed, ok = int(parsed64), true
			} else {
				return 0, invalidJSONIntegerError("int", encoded)
			}
		}
	}
	if !ok {
		if s.coerce && isNumericCoercionCandidate(value) {
			return 0, validationError(Issue{Code: CodeInvalidNumber, Message: "cannot be converted to int", Expected: "int", Received: value})
		}
		return 0, validationError(invalidType("int", value))
	}
	return parseNumber(ctx, parsed, s.rules, s.refinements)
}
func (s IntSchema) withRule(rule numberRule[int]) IntSchema {
	s.rules = appendCopy(s.rules, rule)
	return s
}
func (s IntSchema) withConstraint(key string, value any) IntSchema {
	s.constraints = appendCopy(s.constraints, map[string]any{key: value})
	return s
}
func (s IntSchema) buildJSONSchema(_ *jsonSchemaBuildContext) (map[string]any, error) {
	return buildNumberJSONSchema("integer", s.coerce, s.constraints, len(s.refinements))
}

// Int64Schema parses and validates int64 values and decoder-native JSON
// numbers.
type Int64Schema struct {
	coerce      bool
	rules       []numberRule[int64]
	refinements []refinement[int64]
	constraints []map[string]any
}

// Int64 returns an int64 schema. It does not coerce strings or other Go number
// types.
func Int64() Int64Schema { return Int64Schema{} }

// Min requires a value greater than or equal to bound.
func (s Int64Schema) Min(bound int64) Int64Schema { return s.Gte(bound) }

// Max requires a value less than or equal to bound.
func (s Int64Schema) Max(bound int64) Int64Schema { return s.Lte(bound) }

// Gt requires a value greater than bound.
func (s Int64Schema) Gt(bound int64) Int64Schema {
	s = s.withRule(lowerRule(bound, false))
	return s.withConstraint("exclusiveMinimum", bound)
}

// Gte requires a value greater than or equal to bound.
func (s Int64Schema) Gte(bound int64) Int64Schema {
	s = s.withRule(lowerRule(bound, true))
	return s.withConstraint("minimum", bound)
}

// Lt requires a value less than bound.
func (s Int64Schema) Lt(bound int64) Int64Schema {
	s = s.withRule(upperRule(bound, false))
	return s.withConstraint("exclusiveMaximum", bound)
}

// Lte requires a value less than or equal to bound.
func (s Int64Schema) Lte(bound int64) Int64Schema {
	s = s.withRule(upperRule(bound, true))
	return s.withConstraint("maximum", bound)
}

// Positive requires a value greater than zero.
func (s Int64Schema) Positive() Int64Schema { return s.Gt(0) }

// Negative requires a value less than zero.
func (s Int64Schema) Negative() Int64Schema { return s.Lt(0) }

// NonNegative requires a value greater than or equal to zero.
func (s Int64Schema) NonNegative() Int64Schema { return s.Gte(0) }

// OneOf restricts values to the provided set.
func (s Int64Schema) OneOf(values ...int64) Int64Schema {
	if len(values) == 0 {
		panic("shape: Int64.OneOf requires at least one value")
	}
	s = s.withRule(numberOneOfRule(values))
	return s.withConstraint("enum", append([]int64(nil), values...))
}

// Refine adds custom validation after built-in rules.
func (s Int64Schema) Refine(fn func(int64) error) Int64Schema {
	s.refinements = appendCopy(s.refinements, requireRefinement(fn))
	return s
}

// Parse implements Schema[int64].
func (s Int64Schema) Parse(value any) (int64, error) {
	return s.ParseContext(context.Background(), value)
}

// ParseContext implements Schema[int64].
func (s Int64Schema) ParseContext(ctx context.Context, value any) (int64, error) {
	if err := checkContext(ctx); err != nil {
		return 0, err
	}
	parsed, ok := value.(int64)
	if !ok && s.coerce {
		parsed, ok = coerceInteger(value, 64)
	}
	if !ok {
		if encoded, encodedOK := value.(json.Number); encodedOK {
			parsed, ok = parseJSONInteger(string(encoded), 64)
			if !ok {
				return 0, invalidJSONIntegerError("int64", encoded)
			}
		}
	}
	if !ok {
		if s.coerce && isNumericCoercionCandidate(value) {
			return 0, validationError(Issue{Code: CodeInvalidNumber, Message: "cannot be converted to int64", Expected: "int64", Received: value})
		}
		return 0, validationError(invalidType("int64", value))
	}
	return parseNumber(ctx, parsed, s.rules, s.refinements)
}
func (s Int64Schema) withRule(rule numberRule[int64]) Int64Schema {
	s.rules = appendCopy(s.rules, rule)
	return s
}
func (s Int64Schema) withConstraint(key string, value any) Int64Schema {
	s.constraints = appendCopy(s.constraints, map[string]any{key: value})
	return s
}
func (s Int64Schema) buildJSONSchema(_ *jsonSchemaBuildContext) (map[string]any, error) {
	return buildNumberJSONSchema("integer", s.coerce, s.constraints, len(s.refinements))
}

// Float64Schema parses and validates finite float64 values and decoder-native
// JSON numbers.
type Float64Schema struct {
	coerce      bool
	rules       []numberRule[float64]
	refinements []refinement[float64]
	constraints []map[string]any
}

// Float64 returns a float64 schema. It does not coerce strings or other Go
// number types.
func Float64() Float64Schema { return Float64Schema{} }

// Min requires a value greater than or equal to bound.
func (s Float64Schema) Min(bound float64) Float64Schema { return s.Gte(bound) }

// Max requires a value less than or equal to bound.
func (s Float64Schema) Max(bound float64) Float64Schema { return s.Lte(bound) }

// Gt requires a value greater than bound.
func (s Float64Schema) Gt(bound float64) Float64Schema {
	s = s.withRule(lowerRule(bound, false))
	return s.withConstraint("exclusiveMinimum", bound)
}

// Gte requires a value greater than or equal to bound.
func (s Float64Schema) Gte(bound float64) Float64Schema {
	s = s.withRule(lowerRule(bound, true))
	return s.withConstraint("minimum", bound)
}

// Lt requires a value less than bound.
func (s Float64Schema) Lt(bound float64) Float64Schema {
	s = s.withRule(upperRule(bound, false))
	return s.withConstraint("exclusiveMaximum", bound)
}

// Lte requires a value less than or equal to bound.
func (s Float64Schema) Lte(bound float64) Float64Schema {
	s = s.withRule(upperRule(bound, true))
	return s.withConstraint("maximum", bound)
}

// Positive requires a value greater than zero.
func (s Float64Schema) Positive() Float64Schema { return s.Gt(0) }

// Negative requires a value less than zero.
func (s Float64Schema) Negative() Float64Schema { return s.Lt(0) }

// NonNegative requires a value greater than or equal to zero.
func (s Float64Schema) NonNegative() Float64Schema { return s.Gte(0) }

// OneOf restricts values to the provided set.
func (s Float64Schema) OneOf(values ...float64) Float64Schema {
	if len(values) == 0 {
		panic("shape: Float64.OneOf requires at least one value")
	}
	s = s.withRule(numberOneOfRule(values))
	return s.withConstraint("enum", append([]float64(nil), values...))
}

// Refine adds custom validation after built-in rules.
func (s Float64Schema) Refine(fn func(float64) error) Float64Schema {
	s.refinements = appendCopy(s.refinements, requireRefinement(fn))
	return s
}

// Parse implements Schema[float64].
func (s Float64Schema) Parse(value any) (float64, error) {
	return s.ParseContext(context.Background(), value)
}

// ParseContext implements Schema[float64].
func (s Float64Schema) ParseContext(ctx context.Context, value any) (float64, error) {
	if err := checkContext(ctx); err != nil {
		return 0, err
	}
	parsed, ok := value.(float64)
	if !ok && s.coerce {
		parsed, ok = coerceFloat64Value(value)
	}
	if !ok {
		if encoded, encodedOK := value.(json.Number); encodedOK {
			parsed, ok = parseJSONFloat(string(encoded))
			if !ok {
				return 0, validationError(Issue{
					Code:     CodeInvalidNumber,
					Message:  "must be a finite float64 number",
					Expected: "float64",
					Received: encoded.String(),
				})
			}
		}
	}
	if !ok {
		if s.coerce && isNumericCoercionCandidate(value) {
			return 0, validationError(Issue{Code: CodeInvalidNumber, Message: "cannot be converted to float64", Expected: "float64", Received: value})
		}
		return 0, validationError(invalidType("float64", value))
	}
	if math.IsNaN(parsed) || math.IsInf(parsed, 0) {
		return 0, validationError(Issue{
			Code:     CodeInvalidNumber,
			Message:  "must be a finite number",
			Expected: "finite float64",
			Received: parsed,
		})
	}
	return parseNumber(ctx, parsed, s.rules, s.refinements)
}
func (s Float64Schema) withRule(rule numberRule[float64]) Float64Schema {
	s.rules = appendCopy(s.rules, rule)
	return s
}
func (s Float64Schema) withConstraint(key string, value any) Float64Schema {
	s.constraints = appendCopy(s.constraints, map[string]any{key: value})
	return s
}
func (s Float64Schema) buildJSONSchema(_ *jsonSchemaBuildContext) (map[string]any, error) {
	return buildNumberJSONSchema("number", s.coerce, s.constraints, len(s.refinements))
}

func lowerRule[N number](bound N, inclusive bool) numberRule[N] {
	return func(value N) *Issue {
		valid := value > bound
		comparison := "greater than"
		if inclusive {
			valid = value >= bound
			comparison = "greater than or equal to"
		}
		if valid {
			return nil
		}
		return &Issue{
			Code:     CodeTooSmall,
			Message:  fmt.Sprintf("must be %s %v", comparison, bound),
			Expected: bound,
			Received: value,
		}
	}
}

func upperRule[N number](bound N, inclusive bool) numberRule[N] {
	return func(value N) *Issue {
		valid := value < bound
		comparison := "less than"
		if inclusive {
			valid = value <= bound
			comparison = "less than or equal to"
		}
		if valid {
			return nil
		}
		return &Issue{
			Code:     CodeTooBig,
			Message:  fmt.Sprintf("must be %s %v", comparison, bound),
			Expected: bound,
			Received: value,
		}
	}
}

func numberOneOfRule[N number](allowed []N) numberRule[N] {
	values := append([]N(nil), allowed...)
	return func(value N) *Issue {
		for _, candidate := range values {
			if value == candidate {
				return nil
			}
		}
		return &Issue{Code: CodeInvalidEnum, Message: "must be one of the allowed values", Expected: values, Received: value}
	}
}

func buildNumberJSONSchema(kind string, coerce bool, constraints []map[string]any, refinementCount int) (map[string]any, error) {
	if err := unsupportedIfRefined(refinementCount); err != nil {
		return nil, err
	}
	document := map[string]any{"type": kind}
	applyConstraints(document, constraints)
	if coerce {
		document["x-shape-coerce"] = true
	}
	return document, nil
}

func parseNumber[N number](ctx context.Context, value N, rules []numberRule[N], refinements []refinement[N]) (N, error) {
	issues := make([]Issue, 0)
	for _, rule := range rules {
		if issue := rule(value); issue != nil {
			issues, _ = appendIssuesBounded(issues, *issue)
		}
	}
	refinementIssues, err := runRefinements(ctx, value, refinements)
	if err != nil {
		var zero N
		return zero, err
	}
	issues, _ = appendIssuesBounded(issues, refinementIssues...)
	if len(issues) != 0 {
		var zero N
		return zero, &ValidationError{Issues: issues}
	}
	return value, nil
}

func parseJSONInteger(value string, bits int) (int64, bool) {
	positiveLimit := uint64(math.MaxInt64)
	negativeLimit := positiveLimit + 1
	if bits > 0 && bits < 64 {
		positiveLimit = uint64(1)<<(bits-1) - 1
		negativeLimit = positiveLimit + 1
	}
	magnitude, negative, ok := parseDecimalInteger(value, positiveLimit, negativeLimit)
	if !ok {
		return 0, false
	}
	if negative {
		if magnitude == uint64(math.MaxInt64)+1 {
			return math.MinInt64, true
		}
		return -int64(magnitude), true
	}
	return int64(magnitude), true
}

// parseDecimalInteger validates base-10 integer syntax, decimal fractions, and
// exponents without materializing an arbitrary-precision value. Its work and
// allocation are bounded by the encoded input length and the target width.
func parseDecimalInteger(value string, positiveLimit, negativeLimit uint64) (uint64, bool, bool) {
	if value == "" {
		return 0, false, false
	}
	index := 0
	negative := false
	if value[index] == '+' || value[index] == '-' {
		negative = value[index] == '-'
		index++
		if index == len(value) {
			return 0, false, false
		}
	}

	digits := make([]byte, 0, len(value))
	integerDigits := 0
	for index < len(value) && value[index] >= '0' && value[index] <= '9' {
		digits = append(digits, value[index])
		integerDigits++
		index++
	}
	if integerDigits == 0 {
		return 0, false, false
	}
	fractionDigits := 0
	if index < len(value) && value[index] == '.' {
		index++
		for index < len(value) && value[index] >= '0' && value[index] <= '9' {
			digits = append(digits, value[index])
			fractionDigits++
			index++
		}
		if fractionDigits == 0 {
			return 0, false, false
		}
	}

	exponent := 0
	if index < len(value) && (value[index] == 'e' || value[index] == 'E') {
		index++
		exponentNegative := false
		if index < len(value) && (value[index] == '+' || value[index] == '-') {
			exponentNegative = value[index] == '-'
			index++
		}
		if index == len(value) || value[index] < '0' || value[index] > '9' {
			return 0, false, false
		}
		// Any exponent beyond the input length plus the target's 20 decimal
		// digits has the same fit outcome. Saturating prevents int overflow.
		exponentLimit := len(value)
		if exponentLimit <= int(^uint(0)>>1)-20 {
			exponentLimit += 20
		}
		for index < len(value) && value[index] >= '0' && value[index] <= '9' {
			digit := int(value[index] - '0')
			if exponent <= exponentLimit {
				if exponent > (exponentLimit-digit)/10 {
					exponent = exponentLimit
				} else {
					exponent = exponent*10 + digit
				}
			}
			index++
		}
		if exponentNegative {
			exponent = -exponent
		}
	}
	if index != len(value) {
		return 0, false, false
	}

	firstNonZero := 0
	for firstNonZero < len(digits) && digits[firstNonZero] == '0' {
		firstNonZero++
	}
	if firstNonZero == len(digits) {
		return 0, false, true
	}
	digits = digits[firstNonZero:]
	shift := exponent - fractionDigits
	if shift < 0 {
		trim := -shift
		if trim > len(digits) {
			return 0, false, false
		}
		for _, digit := range digits[len(digits)-trim:] {
			if digit != '0' {
				return 0, false, false
			}
		}
		digits = digits[:len(digits)-trim]
		shift = 0
		for len(digits) > 0 && digits[0] == '0' {
			digits = digits[1:]
		}
		if len(digits) == 0 {
			return 0, false, true
		}
	}
	if shift > 20 || len(digits) > 20-shift {
		return 0, false, false
	}

	limit := positiveLimit
	if negative {
		limit = negativeLimit
	}
	var magnitude uint64
	for _, digit := range digits {
		value := uint64(digit - '0')
		if magnitude > (limit-value)/10 {
			return 0, false, false
		}
		magnitude = magnitude*10 + value
	}
	for range shift {
		if magnitude > limit/10 {
			return 0, false, false
		}
		magnitude *= 10
	}
	return magnitude, negative && magnitude != 0, true
}

func parseJSONFloat(value string) (float64, bool) {
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsInf(parsed, 0) || math.IsNaN(parsed) {
		return 0, false
	}
	return parsed, true
}

func invalidJSONIntegerError(expected string, value json.Number) error {
	return validationError(Issue{
		Code:     CodeInvalidNumber,
		Message:  "must be an integer within the target type's range",
		Expected: expected,
		Received: value.String(),
	})
}

// CoerceInt returns an int schema that explicitly accepts lossless numeric and
// base-10 string representations.
func CoerceInt() IntSchema { return IntSchema{coerce: true} }

// CoerceInt64 returns an int64 schema that explicitly accepts lossless numeric
// and base-10 string representations.
func CoerceInt64() Int64Schema { return Int64Schema{coerce: true} }

// CoerceFloat64 returns a float64 schema that explicitly accepts Go numeric
// values and base-10 string representations.
func CoerceFloat64() Float64Schema { return Float64Schema{coerce: true} }

// CoerceFloat is an alias for CoerceFloat64.
func CoerceFloat() Float64Schema { return CoerceFloat64() }

func coerceInteger(value any, bits int) (int64, bool) {
	switch typed := value.(type) {
	case int:
		return boundedInteger(int64(typed), bits)
	case int8:
		return boundedInteger(int64(typed), bits)
	case int16:
		return boundedInteger(int64(typed), bits)
	case int32:
		return boundedInteger(int64(typed), bits)
	case int64:
		return boundedInteger(typed, bits)
	case uint:
		if uint64(typed) > math.MaxInt64 {
			return 0, false
		}
		return boundedInteger(int64(typed), bits)
	case uint8:
		return boundedInteger(int64(typed), bits)
	case uint16:
		return boundedInteger(int64(typed), bits)
	case uint32:
		return boundedInteger(int64(typed), bits)
	case uint64:
		if typed > math.MaxInt64 {
			return 0, false
		}
		return boundedInteger(int64(typed), bits)
	case float32:
		return coerceFloatInteger(float64(typed), bits)
	case float64:
		return coerceFloatInteger(typed, bits)
	case string:
		return parseJSONInteger(strings.TrimSpace(typed), bits)
	case json.Number:
		return parseJSONInteger(typed.String(), bits)
	default:
		return 0, false
	}
}

func boundedInteger(value int64, bits int) (int64, bool) {
	if bits > 0 && bits < 64 {
		maximum := int64(1)<<(bits-1) - 1
		minimum := -maximum - 1
		if value < minimum || value > maximum {
			return 0, false
		}
	}
	return value, true
}

func coerceFloatInteger(value float64, bits int) (int64, bool) {
	if math.IsNaN(value) || math.IsInf(value, 0) || math.Trunc(value) != value {
		return 0, false
	}
	if value < math.MinInt64 || value >= -float64(math.MinInt64) {
		return 0, false
	}
	return boundedInteger(int64(value), bits)
}

func coerceFloat64Value(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int8:
		return float64(typed), true
	case int16:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case uint:
		return float64(typed), true
	case uint8:
		return float64(typed), true
	case uint16:
		return float64(typed), true
	case uint32:
		return float64(typed), true
	case uint64:
		return float64(typed), true
	case string:
		return parseJSONFloat(strings.TrimSpace(typed))
	case json.Number:
		return parseJSONFloat(typed.String())
	default:
		return 0, false
	}
}

func isNumericCoercionCandidate(value any) bool {
	switch value.(type) {
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64, string, json.Number:
		return true
	default:
		return false
	}
}
