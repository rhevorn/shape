package goshape

import (
	"context"
	"sort"
)

// RecordSchema parses string-keyed input through separate key and value
// schemas into map[K]V.
type RecordSchema[K comparable, V any] struct {
	key         Schema[K]
	value       Schema[V]
	rules       []lengthRule
	constraints []map[string]any
	refinements []refinement[map[K]V]
}

// Record creates a schema with independently typed key and value parsers.
func Record[K comparable, V any](key Schema[K], value Schema[V]) RecordSchema[K, V] {
	if key == nil || value == nil {
		panic("goshape: record key and value schemas must not be nil")
	}
	return RecordSchema[K, V]{key: key, value: value}
}

// Min requires at least n entries.
func (s RecordSchema[K, V]) Min(n int) RecordSchema[K, V] {
	s.rules = appendCopy(s.rules, minLengthRule("Record", n))
	s.constraints = appendCopy(s.constraints, map[string]any{"minProperties": n})
	return s
}

// Max allows at most n entries.
func (s RecordSchema[K, V]) Max(n int) RecordSchema[K, V] {
	s.rules = appendCopy(s.rules, maxLengthRule("Record", n))
	s.constraints = appendCopy(s.constraints, map[string]any{"maxProperties": n})
	return s
}

// NonEmpty requires at least one entry.
func (s RecordSchema[K, V]) NonEmpty() RecordSchema[K, V] { return s.Min(1) }

// Refine adds validation after all keys and values parse successfully.
func (s RecordSchema[K, V]) Refine(fn func(map[K]V) error) RecordSchema[K, V] {
	s.refinements = appendCopy(s.refinements, requireRefinement(fn))
	return s
}

// Parse implements Schema[map[K]V].
func (s RecordSchema[K, V]) Parse(value any) (map[K]V, error) {
	return s.ParseContext(context.Background(), value)
}

// ParseContext implements Schema[map[K]V].
func (s RecordSchema[K, V]) ParseContext(ctx context.Context, value any) (map[K]V, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	input, ok := value.(map[string]any)
	if !ok {
		if typed, typedOK := value.(map[string]V); typedOK {
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
			issues = append(issues, *issue)
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
	result := make(map[K]V, len(input))
	for _, rawKey := range keys {
		parsedKey, keyErr := s.key.ParseContext(ctx, rawKey)
		if keyErr != nil {
			if contextErr := contextError(keyErr, ctx); contextErr != nil {
				return nil, contextErr
			}
			issues = append(issues, prefixIssues(issuesFromError(keyErr), FieldPath(rawKey))...)
			continue
		}
		parsedValue, valueErr := s.value.ParseContext(ctx, input[rawKey])
		if valueErr != nil {
			if contextErr := contextError(valueErr, ctx); contextErr != nil {
				return nil, contextErr
			}
			issues = append(issues, prefixIssues(issuesFromError(valueErr), FieldPath(rawKey))...)
			continue
		}
		if _, duplicate := result[parsedKey]; duplicate {
			issues = append(issues, Issue{Code: CodeInvalidValue, Path: Path{FieldPath(rawKey)}, Message: "key duplicates another parsed key", Expected: "unique parsed key", Received: parsedKey})
			continue
		}
		result[parsedKey] = parsedValue
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

func (s RecordSchema[K, V]) buildJSONSchema(ctx *jsonSchemaBuildContext) (map[string]any, error) {
	if err := unsupportedIfRefined(len(s.refinements)); err != nil {
		return nil, err
	}
	key, err := buildJSONSchemaWithContext(s.key, ctx)
	if err != nil {
		return nil, err
	}
	value, err := buildJSONSchemaWithContext(s.value, ctx)
	if err != nil {
		return nil, err
	}
	document := map[string]any{"type": "object", "propertyNames": key, "additionalProperties": value}
	applyConstraints(document, s.constraints)
	return document, nil
}
