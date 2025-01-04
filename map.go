package goshape

import (
	"context"
	"sort"
)

// MapSchema parses map[string]any values with one schema for every value.
type MapSchema[T any] struct {
	value       Schema[T]
	refinements []refinement[map[string]T]
}

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
		return nil, validationError(invalidType("map[string]any", value))
	}

	keys := make([]string, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	result := make(map[string]T, len(input))
	var issues []Issue
	for _, key := range keys {
		if err := checkContext(ctx); err != nil {
			return nil, err
		}
		parsed, err := s.value.ParseContext(ctx, input[key])
		if err != nil {
			if contextErr := contextError(err, ctx); contextErr != nil {
				return nil, contextErr
			}
			issues = append(issues, prefixIssues(issuesFromError(err), FieldPath(key))...)
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
