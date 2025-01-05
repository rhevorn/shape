package goshape

import "context"

// UnionSchema accepts the first successful schema among alternatives sharing
// the same output type.
type UnionSchema[T any] struct{ alternatives []Schema[T] }

// OneOf creates a union of schemas with the same output type.
func OneOf[T any](alternatives ...Schema[T]) UnionSchema[T] {
	if len(alternatives) == 0 {
		panic("goshape: OneOf requires at least one schema")
	}
	result := UnionSchema[T]{alternatives: append([]Schema[T](nil), alternatives...)}
	for _, alternative := range result.alternatives {
		if alternative == nil {
			panic("goshape: OneOf alternative must not be nil")
		}
	}
	return result
}

// Union is an alias for OneOf.
func Union[T any](alternatives ...Schema[T]) UnionSchema[T] { return OneOf(alternatives...) }

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
	for _, alternative := range s.alternatives {
		parsed, err := alternative.ParseContext(ctx, value)
		if err == nil {
			return parsed, nil
		}
		if contextErr := contextError(err, ctx); contextErr != nil {
			return zero, contextErr
		}
	}
	return zero, validationError(Issue{
		Code:     CodeInvalidUnion,
		Message:  "must match at least one schema",
		Expected: len(s.alternatives),
		Received: typeNameOf(value),
	})
}

func (s UnionSchema[T]) buildJSONSchema() (map[string]any, error) {
	alternatives := make([]any, 0, len(s.alternatives))
	for _, alternative := range s.alternatives {
		document, err := buildJSONSchema(alternative)
		if err != nil {
			return nil, err
		}
		alternatives = append(alternatives, document)
	}
	return map[string]any{"anyOf": alternatives}, nil
}
