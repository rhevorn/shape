package goshape

import (
	"context"
	"encoding/json"
	"math"
	"reflect"
	"strconv"
)

// EnumSchema restricts a comparable value to a fixed set.
type EnumSchema[T comparable] struct {
	values      []T
	refinements []refinement[T]
}

// Enum creates a schema that accepts one of values.
func Enum[T comparable](values ...T) EnumSchema[T] {
	if len(values) == 0 {
		panic("goshape: Enum requires at least one value")
	}
	return EnumSchema[T]{values: append([]T(nil), values...)}
}

// Refine adds custom validation after enum membership validation.
func (s EnumSchema[T]) Refine(fn func(T) error) EnumSchema[T] {
	s.refinements = appendCopy(s.refinements, requireRefinement(fn))
	return s
}

// Parse implements Schema[T].
func (s EnumSchema[T]) Parse(value any) (T, error) {
	return s.ParseContext(context.Background(), value)
}

// ParseContext implements Schema[T].
func (s EnumSchema[T]) ParseContext(ctx context.Context, value any) (T, error) {
	var zero T
	if err := checkContext(ctx); err != nil {
		return zero, err
	}
	parsed, ok := parseComparable[T](value)
	if !ok {
		return zero, validationError(invalidType(genericTypeName[T](), value))
	}
	for _, allowed := range s.values {
		if parsed == allowed {
			issues, err := runRefinements(ctx, parsed, s.refinements)
			if err != nil {
				return zero, err
			}
			if len(issues) != 0 {
				return zero, &ValidationError{Issues: issues}
			}
			return parsed, nil
		}
	}
	return zero, validationError(Issue{Code: CodeInvalidEnum, Message: "must be one of the allowed values", Expected: s.values, Received: parsed})
}

func (s EnumSchema[T]) buildJSONSchema(_ *jsonSchemaBuildContext) (map[string]any, error) {
	if err := unsupportedIfRefined(len(s.refinements)); err != nil {
		return nil, err
	}
	return map[string]any{"enum": append([]T(nil), s.values...)}, nil
}

// LiteralSchema requires equality with one fixed value.
type LiteralSchema[T comparable] struct{ value T }

// Literal creates a schema for one fixed comparable value.
func Literal[T comparable](value T) LiteralSchema[T] { return LiteralSchema[T]{value: value} }

// Parse implements Schema[T].
func (s LiteralSchema[T]) Parse(value any) (T, error) {
	return s.ParseContext(context.Background(), value)
}

// ParseContext implements Schema[T].
func (s LiteralSchema[T]) ParseContext(ctx context.Context, value any) (T, error) {
	var zero T
	if err := checkContext(ctx); err != nil {
		return zero, err
	}
	parsed, ok := parseComparable[T](value)
	if !ok {
		return zero, validationError(invalidType(genericTypeName[T](), value))
	}
	if parsed != s.value {
		return zero, validationError(Issue{Code: CodeInvalidValue, Message: "must equal the literal value", Expected: s.value, Received: parsed})
	}
	return parsed, nil
}

func (s LiteralSchema[T]) buildJSONSchema(_ *jsonSchemaBuildContext) (map[string]any, error) {
	return map[string]any{"const": s.value}, nil
}

func parseComparable[T comparable](value any) (T, bool) {
	if parsed, ok := value.(T); ok {
		return parsed, true
	}
	var zero T
	targetType := reflect.TypeOf(zero)
	if number, ok := value.(json.Number); ok && targetType != nil {
		switch targetType.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			parsed, valid := parseJSONInteger(number.String(), targetType.Bits())
			if valid {
				converted := reflect.New(targetType).Elem()
				converted.SetInt(parsed)
				return converted.Interface().(T), true
			}
		case reflect.Float32, reflect.Float64:
			parsed, valid := parseJSONFloat(number.String())
			if valid && (targetType.Kind() != reflect.Float32 || !math.IsInf(float64(float32(parsed)), 0)) {
				converted := reflect.New(targetType).Elem()
				converted.SetFloat(parsed)
				return converted.Interface().(T), true
			}
		}
	}
	switch any(zero).(type) {
	case int:
		if number, ok := value.(json.Number); ok {
			parsed, valid := parseJSONInteger(number.String(), strconv.IntSize)
			if valid {
				return any(int(parsed)).(T), true
			}
		}
	case int64:
		if number, ok := value.(json.Number); ok {
			parsed, valid := parseJSONInteger(number.String(), 64)
			if valid {
				return any(parsed).(T), true
			}
		}
	case float64:
		if number, ok := value.(json.Number); ok {
			parsed, valid := parseJSONFloat(number.String())
			if valid {
				return any(parsed).(T), true
			}
		}
	}
	if targetType != nil && value != nil {
		source := reflect.ValueOf(value)
		if source.Type().ConvertibleTo(targetType) && source.Kind() == targetType.Kind() {
			return source.Convert(targetType).Interface().(T), true
		}
	}
	return zero, false
}
