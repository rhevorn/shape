package goshape

import (
	"context"
	"sort"
)

// ObjectField is a typed field definition accepted by Object. Implementations
// are created with Field.
type ObjectField[T any] interface {
	fieldName() string
	parseAndSet(context.Context, *T, any, bool) ([]Issue, error)
}

// FieldDef connects an input field schema to a strongly typed setter.
type FieldDef[T, V any] struct {
	name         string
	schema       Schema[V]
	setter       func(*T, V)
	optional     bool
	hasDefault   bool
	defaultValue V
}

// Field creates a required object field.
func Field[T, V any](name string, schema Schema[V], setter func(*T, V)) FieldDef[T, V] {
	if name == "" {
		panic("goshape: field name must not be empty")
	}
	if schema == nil {
		panic("goshape: field schema must not be nil")
	}
	if setter == nil {
		panic("goshape: field setter must not be nil")
	}
	return FieldDef[T, V]{name: name, schema: schema, setter: setter}
}

// Optional permits the field to be absent without invoking its setter.
func (f FieldDef[T, V]) Optional() FieldDef[T, V] {
	f.optional = true
	return f
}

// Default assigns value when the input field is absent. Default implies
// optional-on-missing behavior.
func (f FieldDef[T, V]) Default(value V) FieldDef[T, V] {
	f.optional = true
	f.hasDefault = true
	f.defaultValue = value
	return f
}

func (f FieldDef[T, V]) fieldName() string { return f.name }

func (f FieldDef[T, V]) parseAndSet(ctx context.Context, target *T, input any, present bool) ([]Issue, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	if !present {
		if f.hasDefault {
			f.setter(target, f.defaultValue)
			return nil, nil
		}
		if f.optional {
			return nil, nil
		}
		return []Issue{{
			Code:     CodeRequired,
			Path:     Path{FieldPath(f.name)},
			Message:  "field is required",
			Expected: "present",
			Received: "missing",
		}}, nil
	}

	parsed, err := f.schema.ParseContext(ctx, input)
	if err != nil {
		if contextErr := contextError(err, ctx); contextErr != nil {
			return nil, contextErr
		}
		return prefixIssues(issuesFromError(err), FieldPath(f.name)), nil
	}
	f.setter(target, parsed)
	return nil, nil
}

// ObjectSchema parses string-keyed input into T through typed field setters.
type ObjectSchema[T any] struct {
	fields      []ObjectField[T]
	fieldNames  map[string]struct{}
	strict      bool
	refinements []refinement[T]
}

// Object creates an object schema. Unknown fields are stripped by default.
func Object[T any](fields ...ObjectField[T]) ObjectSchema[T] {
	result := ObjectSchema[T]{
		fields:     make([]ObjectField[T], len(fields)),
		fieldNames: make(map[string]struct{}, len(fields)),
	}
	copy(result.fields, fields)
	for _, field := range result.fields {
		if field == nil {
			panic("goshape: object field must not be nil")
		}
		name := field.fieldName()
		if _, exists := result.fieldNames[name]; exists {
			panic("goshape: duplicate object field: " + name)
		}
		result.fieldNames[name] = struct{}{}
	}
	return result
}

// Strict rejects input keys that have no declared field.
func (s ObjectSchema[T]) Strict() ObjectSchema[T] {
	s.strict = true
	return s
}

// Strip ignores input keys that have no declared field. This is the default.
func (s ObjectSchema[T]) Strip() ObjectSchema[T] {
	s.strict = false
	return s
}

// Refine adds cross-field validation after every field parses successfully.
func (s ObjectSchema[T]) Refine(fn func(T) error) ObjectSchema[T] {
	s.refinements = appendCopy(s.refinements, requireRefinement(fn))
	return s
}

// Parse implements Schema[T].
func (s ObjectSchema[T]) Parse(value any) (T, error) {
	return s.ParseContext(context.Background(), value)
}

// ParseContext implements Schema[T].
func (s ObjectSchema[T]) ParseContext(ctx context.Context, value any) (T, error) {
	var zero T
	if err := checkContext(ctx); err != nil {
		return zero, err
	}
	input, ok := value.(map[string]any)
	if !ok {
		return zero, validationError(invalidType("map[string]any", value))
	}

	var candidate T
	var issues []Issue
	for _, field := range s.fields {
		fieldValue, present := input[field.fieldName()]
		fieldIssues, err := field.parseAndSet(ctx, &candidate, fieldValue, present)
		if err != nil {
			return zero, err
		}
		issues = append(issues, fieldIssues...)
	}

	if s.strict {
		unknown := make([]string, 0)
		for key := range input {
			if _, exists := s.fieldNames[key]; !exists {
				unknown = append(unknown, key)
			}
		}
		sort.Strings(unknown)
		for _, key := range unknown {
			issues = append(issues, Issue{
				Code:     CodeUnknownField,
				Path:     Path{FieldPath(key)},
				Message:  "field is not allowed",
				Expected: "declared field",
				Received: key,
			})
		}
	}

	if len(issues) != 0 {
		return zero, &ValidationError{Issues: issues}
	}
	refinementIssues, err := runRefinements(ctx, candidate, s.refinements)
	if err != nil {
		return zero, err
	}
	if len(refinementIssues) != 0 {
		return zero, &ValidationError{Issues: refinementIssues}
	}
	return candidate, nil
}
