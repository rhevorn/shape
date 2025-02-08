package shape

import (
	"context"
	"sort"
)

// MapSchema parses JSON objects (string keys on the wire) into map[K]V.
// The key schema turns each raw string key into K; the value schema parses V.
type MapSchema[K comparable, V any] struct {
	key         Schema[K]
	value       Schema[V]
	rules       []lengthRule
	constraints []map[string]any
	refinements []refinement[map[K]V]
}

// Min requires at least n entries.
func (s MapSchema[K, V]) Min(n int) MapSchema[K, V] {
	s.rules = appendCopy(s.rules, minLengthRule("Map", n))
	s.constraints = appendCopy(s.constraints, map[string]any{"minProperties": n})
	return s
}

// Max allows at most n entries.
func (s MapSchema[K, V]) Max(n int) MapSchema[K, V] {
	s.rules = appendCopy(s.rules, maxLengthRule("Map", n))
	s.constraints = appendCopy(s.constraints, map[string]any{"maxProperties": n})
	return s
}

// NonEmpty requires at least one entry.
func (s MapSchema[K, V]) NonEmpty() MapSchema[K, V] { return s.Min(1) }

// Map returns a schema for map[K]V. The first argument parses keys, the second
// parses values (same order as typical key/value APIs).
//
//	Map(String(), Int())                         // map[string]int
//	Map(String(), Any())                         // map[string]any
//	Map(String().ToLower(), String())            // normalize keys
//	Map(Transform(String(), strconv.Atoi), Bool()) // map[int]bool
func Map[K comparable, V any](key Schema[K], value Schema[V]) MapSchema[K, V] {
	if key == nil {
		panic("shape: map key schema must not be nil")
	}
	if value == nil {
		panic("shape: map value schema must not be nil")
	}
	return MapSchema[K, V]{key: key, value: value}
}

// Refine adds custom validation after every entry parses successfully.
func (s MapSchema[K, V]) Refine(fn func(map[K]V) error) MapSchema[K, V] {
	s.refinements = appendCopy(s.refinements, requireRefinement(fn))
	return s
}

// Parse implements Schema[map[K]V].
func (s MapSchema[K, V]) Parse(value any) (map[K]V, error) {
	return s.ParseContext(context.Background(), value)
}

// ParseContext implements Schema[map[K]V].
func (s MapSchema[K, V]) ParseContext(ctx context.Context, value any) (map[K]V, error) {
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
		return nil, validationError(ctx, invalidType("map[string]any", value))
	}

	var issues []Issue
	for _, rule := range s.rules {
		if issue := rule(len(input)); issue != nil {
			issues, _ = appendIssuesBounded(issues, *issue)
		}
	}
	if len(issues) != 0 {
		return nil, validationIssues(ctx, issues)
	}

	keys := make([]string, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	result := make(map[K]V, len(input))
	for _, rawKey := range keys {
		if err := checkContext(ctx); err != nil {
			return nil, err
		}
		parsedKey, keyErr := s.key.ParseContext(ctx, rawKey)
		if keyErr != nil {
			if contextErr := contextError(keyErr, ctx); contextErr != nil {
				return nil, contextErr
			}
			var capped bool
			issues, capped = appendIssuesBounded(issues, prefixIssues(issuesFromError(keyErr), FieldPath(rawKey))...)
			if capped {
				break
			}
			continue
		}
		parsedValue, valueErr := s.value.ParseContext(ctx, input[rawKey])
		if valueErr != nil {
			if contextErr := contextError(valueErr, ctx); contextErr != nil {
				return nil, contextErr
			}
			var capped bool
			issues, capped = appendIssuesBounded(issues, prefixIssues(issuesFromError(valueErr), FieldPath(rawKey))...)
			if capped {
				break
			}
			continue
		}
		if _, duplicate := result[parsedKey]; duplicate {
			var capped bool
			dup := keyedIssue(CodeInvalidValue, "invalid_value.duplicate_key", "unique parsed key", parsedKey)
			dup.Path = Path{FieldPath(rawKey)}
			issues, capped = appendIssuesBounded(issues, dup)
			if capped {
				break
			}
			continue
		}
		result[parsedKey] = parsedValue
	}
	if len(issues) != 0 {
		return nil, validationIssues(ctx, issues)
	}

	refinementIssues, err := runRefinements(ctx, result, s.refinements)
	if err != nil {
		return nil, err
	}
	if len(refinementIssues) != 0 {
		return nil, validationIssues(ctx, refinementIssues)
	}
	return result, nil
}

func (s MapSchema[K, V]) buildJSONSchema(ctx *jsonSchemaBuildContext) (map[string]any, error) {
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
	document := map[string]any{
		"type":                 "object",
		"propertyNames":        key,
		"additionalProperties": value,
	}
	applyConstraints(document, s.constraints)
	return document, nil
}
