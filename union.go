package shape

import "context"

// UnionSchema composes alternatives that share the same output type.
type UnionSchema[T any] struct {
	alternatives []Schema[T]
	exact        bool
}

// OneOf accepts a value only when exactly one alternative succeeds.
func OneOf[T any](alternatives ...Schema[T]) UnionSchema[T] {
	return newUnion(true, alternatives)
}

// Union accepts the first value that matches at least one alternative.
func Union[T any](alternatives ...Schema[T]) UnionSchema[T] {
	return newUnion(false, alternatives)
}

func newUnion[T any](exact bool, alternatives []Schema[T]) UnionSchema[T] {
	if len(alternatives) == 0 {
		panic("shape: Union and OneOf require at least one schema")
	}
	result := UnionSchema[T]{
		alternatives: append([]Schema[T](nil), alternatives...),
		exact:        exact,
	}
	for _, alternative := range result.alternatives {
		if alternative == nil {
			panic("shape: Union and OneOf alternatives must not be nil")
		}
	}
	return result
}

// Parse implements Schema[T].
func (s UnionSchema[T]) Parse(value any) (T, error) {
	return s.ParseContext(context.Background(), value)
}

// ParseContext implements Schema[T].
func (s UnionSchema[T]) ParseContext(ctx context.Context, value any) (T, error) {
	var zero T
	if err := checkContext(ctx); err != nil {
		return zero, err
	}
	if s.exact {
		var candidate T
		matches := 0
		for _, alternative := range s.alternatives {
			parsed, err := alternative.ParseContext(ctx, value)
			if err == nil {
				candidate = parsed
				matches++
				continue
			}
			if contextErr := contextError(err, ctx); contextErr != nil {
				return zero, contextErr
			}
		}
		if matches == 1 {
			return candidate, nil
		}
		return zero, validationError(ctx, keyedIssue(CodeInvalidUnion, "invalid_union.oneof", 1, matches))
	}
	for _, alternative := range s.alternatives {
		parsed, err := alternative.ParseContext(ctx, value)
		if err == nil {
			return parsed, nil
		}
		if contextErr := contextError(err, ctx); contextErr != nil {
			return zero, contextErr
		}
	}
	return zero, validationError(ctx, keyedIssue(CodeInvalidUnion, "invalid_union", "one or more matches", typeNameOf(value)))
}

func (s UnionSchema[T]) buildJSONSchema(ctx *jsonSchemaBuildContext) (map[string]any, error) {
	alternatives := make([]any, 0, len(s.alternatives))
	for _, alternative := range s.alternatives {
		document, err := buildJSONSchemaWithContext(alternative, ctx)
		if err != nil {
			return nil, err
		}
		alternatives = append(alternatives, document)
	}
	keyword := "anyOf"
	if s.exact {
		keyword = "oneOf"
	}
	return map[string]any{keyword: alternatives}, nil
}
