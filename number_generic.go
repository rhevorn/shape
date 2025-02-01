package shape

import (
	"context"
	"encoding/json"
	"math"
	"reflect"
	"strconv"
	"strings"
)

// NumberSchema parses and validates any built-in or named numeric type.
// Strict parsing accepts exactly N; JSON and explicit coercion use a small
// reflection bridge to construct named values without weakening type safety.
type NumberSchema[N Numeric] struct {
	coerce      bool
	rules       []numberRule[N]
	refinements []refinement[N]
	constraints []map[string]any
}

// Number returns a strict generic numeric schema.
func Number[N Numeric]() NumberSchema[N] { return NumberSchema[N]{} }

// CoerceNumber returns a generic numeric schema that accepts scalar numeric and
// textual representations when conversion is in range and non-lossy for
// integer targets.
func CoerceNumber[N Numeric]() NumberSchema[N] { return NumberSchema[N]{coerce: true} }

// Min requires a value greater than or equal to bound.
func (s NumberSchema[N]) Min(bound N) NumberSchema[N] { return s.Gte(bound) }

// Max requires a value less than or equal to bound.
func (s NumberSchema[N]) Max(bound N) NumberSchema[N] { return s.Lte(bound) }

// Gt requires a value greater than bound.
func (s NumberSchema[N]) Gt(bound N) NumberSchema[N] {
	s.rules = appendCopy(s.rules, lowerRule(bound, false))
	s.constraints = appendCopy(s.constraints, map[string]any{"exclusiveMinimum": bound})
	return s
}

// Gte requires a value greater than or equal to bound.
func (s NumberSchema[N]) Gte(bound N) NumberSchema[N] {
	s.rules = appendCopy(s.rules, lowerRule(bound, true))
	s.constraints = appendCopy(s.constraints, map[string]any{"minimum": bound})
	return s
}

// Lt requires a value less than bound.
func (s NumberSchema[N]) Lt(bound N) NumberSchema[N] {
	s.rules = appendCopy(s.rules, upperRule(bound, false))
	s.constraints = appendCopy(s.constraints, map[string]any{"exclusiveMaximum": bound})
	return s
}

// Lte requires a value less than or equal to bound.
func (s NumberSchema[N]) Lte(bound N) NumberSchema[N] {
	s.rules = appendCopy(s.rules, upperRule(bound, true))
	s.constraints = appendCopy(s.constraints, map[string]any{"maximum": bound})
	return s
}

// Positive requires a value greater than zero.
func (s NumberSchema[N]) Positive() NumberSchema[N] { return s.Gt(0) }

// Negative requires a value less than zero. Unsigned schemas can never satisfy
// this rule.
func (s NumberSchema[N]) Negative() NumberSchema[N] { return s.Lt(0) }

// NonNegative requires a value greater than or equal to zero.
func (s NumberSchema[N]) NonNegative() NumberSchema[N] { return s.Gte(0) }

// OneOf restricts values to the provided set.
func (s NumberSchema[N]) OneOf(values ...N) NumberSchema[N] {
	if len(values) == 0 {
		panic("shape: Number.OneOf requires at least one value")
	}
	s.rules = appendCopy(s.rules, numberOneOfRule(values))
	s.constraints = appendCopy(s.constraints, map[string]any{"enum": append([]N(nil), values...)})
	return s
}

// Refine adds custom validation after built-in rules.
func (s NumberSchema[N]) Refine(fn func(N) error) NumberSchema[N] {
	s.refinements = appendCopy(s.refinements, requireRefinement(fn))
	return s
}

// Parse implements Schema[N].
func (s NumberSchema[N]) Parse(value any) (N, error) {
	return s.ParseContext(context.Background(), value)
}

// ParseContext implements Schema[N].
func (s NumberSchema[N]) ParseContext(ctx context.Context, value any) (N, error) {
	var zero N
	if err := checkContext(ctx); err != nil {
		return zero, err
	}
	parsed, ok := value.(N)
	_, jsonNumberInput := value.(json.Number)
	if !ok {
		if encoded, encodedOK := value.(json.Number); encodedOK {
			parsed, ok = parseGenericJSONNumber[N](encoded.String())
		} else if s.coerce {
			parsed, ok = coerceGenericNumber[N](value)
		}
	}
	if !ok {
		if jsonNumberInput {
			return zero, validationError(ctx, keyedIssue(CodeInvalidNumber, "number.json_range", genericTypeName[N](), value))
		}
		if s.coerce && isNumericCoercionCandidate(value) {
			return zero, validationError(ctx, keyedIssue(CodeInvalidNumber, "number.convert", genericTypeName[N](), value))
		}
		return zero, validationError(ctx, invalidType(genericTypeName[N](), value))
	}
	converted := float64(parsed)
	if math.IsNaN(converted) || math.IsInf(converted, 0) {
		return zero, validationError(ctx, keyedIssue(CodeInvalidNumber, "number.finite", genericTypeName[N](), parsed))
	}
	return parseNumber(ctx, parsed, s.rules, s.refinements)
}

func (s NumberSchema[N]) buildJSONSchema(_ *jsonSchemaBuildContext) (map[string]any, error) {
	kind := "integer"
	targetKind := reflect.TypeFor[N]().Kind()
	if targetKind == reflect.Float32 || targetKind == reflect.Float64 {
		kind = "number"
	}
	return buildNumberJSONSchema(kind, s.coerce, s.constraints, len(s.refinements))
}

func parseGenericJSONNumber[N Numeric](value string) (N, bool) {
	var zero N
	target := reflect.New(reflect.TypeFor[N]()).Elem()
	switch target.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		parsed, ok := parseJSONInteger(value, target.Type().Bits())
		if !ok {
			return zero, false
		}
		target.SetInt(parsed)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		parsed, ok := parseJSONUnsigned(value, target.Type().Bits())
		if !ok {
			return zero, false
		}
		target.SetUint(parsed)
	case reflect.Float32, reflect.Float64:
		parsed, err := strconv.ParseFloat(value, target.Type().Bits())
		if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) {
			return zero, false
		}
		target.SetFloat(parsed)
	default:
		return zero, false
	}
	return target.Interface().(N), true
}

func parseJSONUnsigned(value string, bits int) (uint64, bool) {
	limit := uint64(math.MaxUint64)
	if bits > 0 && bits < 64 {
		limit = uint64(1)<<bits - 1
	}
	parsed, negative, ok := parseDecimalInteger(value, limit, limit)
	if !ok || negative {
		return 0, false
	}
	return parsed, true
}

func coerceGenericNumber[N Numeric](value any) (N, bool) {
	if text, ok := value.(string); ok {
		return parseGenericJSONNumber[N](strings.TrimSpace(text))
	}
	text, ok := coerceStringValue(value)
	if !ok {
		var zero N
		return zero, false
	}
	return parseGenericJSONNumber[N](text)
}
