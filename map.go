package goshape

import (
	"context"
	"sort"
)

// MapSchema parses map[string]any values with one schema for every value.
type MapSchema[T any] struct {
	value       Schema[T]
	refinements []refinement[map[string]T]
	rules       []lengthRule
	constraints []map[string]any
}

// Min requires at least n entries.
func (s MapSchema[T]) Min(n int) MapSchema[T] {
	s.rules = appendCopy(s.rules, minLengthRule("Map", n))
	s.constraints = appendCopy(s.constraints, map[string]any{"minProperties": n})
	return s
}

// Max allows at most n entries.
func (s MapSchema[T]) Max(n int) MapSchema[T] {
	s.rules = appendCopy(s.rules, maxLengthRule("Map", n))
	s.constraints = appendCopy(s.constraints, map[string]any{"maxProperties": n})
	return s
}

// NonEmpty requires at least one entry.
func (s MapSchema[T]) NonEmpty() MapSchema[T] { return s.Min(1) }

// Map returns a schema for a string-keyed map.
func Map[T any](value Schema[T]) MapSchema[T] {
	if value == nil {
		panic("goshape: map value schema must not be nil")
	}
	return MapSchema[T]{value: value}
}

// Refine adds custom validation after every map value parses successfully.
func (s MapSchema[T]) Refine(fn func(map[string]T) error) MapSchema[T] {
	s.refinements = appendCopy(s.refinements, requireRefinement(fn))
	return s
}

// Parse implements Schema[map[string]T].
func (s MapSchema[T]) Parse(value any) (map[string]T, error) {
	return s.ParseContext(context.Background(), value)
}

// ParseContext implements Schema[map[string]T].
func (s MapSchema[T]) ParseContext(ctx context.Context, value any) (map[string]T, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	input, ok := value.(map[string]any)
	if !ok {
		if typed, typedOK := value.(map[string]T); typedOK {
			input = make(map[string]any, len(typed))
			for key, item := range typed {
				input[key] = item
			}
			ok = true
		}
	}
	if !ok {
		return nil, validationError(invalidType("map[string]any", value))
	}

	var issues []Issue
	for _, rule := range s.rules {
		if issue := rule(len(input)); issue != nil {
			issues, _ = appendIssuesBounded(issues, *issue)
		}
	}
	if len(issues) != 0 {
		return nil, &ValidationError{Issues: issues}
	}
	keys := make([]string, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	result := make(map[string]T, len(input))
	for _, key := range keys {
		if err := checkContext(ctx); err != nil {
			return nil, err
		}
		parsed, err := s.value.ParseContext(ctx, input[key])
		if err != nil {
			if contextErr := contextError(err, ctx); contextErr != nil {
				return nil, contextErr
			}
			var capped bool
			issues, capped = appendIssuesBounded(issues, prefixIssues(issuesFromError(err), FieldPath(key))...)
			if capped {
				break
			}
			continue
		}
		result[key] = parsed
	}
	if len(issues) != 0 {
		return nil, &ValidationError{Issues: issues}
	}

	refinementIssues, err := runRefinements(ctx, result, s.refinements)
	if err != nil {
		return nil, err
	}
	if len(refinementIssues) != 0 {
		return nil, &ValidationError{Issues: refinementIssues}
	}
	return result, nil
}

func (s MapSchema[T]) buildJSONSchema(ctx *jsonSchemaBuildContext) (map[string]any, error) {
	if err := unsupportedIfRefined(len(s.refinements)); err != nil {
		return nil, err
	}
	value, err := buildJSONSchemaWithContext(s.value, ctx)
	if err != nil {
		return nil, err
	}
	document := map[string]any{"type": "object", "additionalProperties": value}
	applyConstraints(document, s.constraints)
	return document, nil
}
