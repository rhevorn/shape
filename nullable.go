package goshape

import "context"

// NullableSchema accepts nil or a value parsed by its inner schema. Non-nil
// values are returned through a pointer so nil remains distinguishable.
type NullableSchema[T any] struct{ schema Schema[T] }

// Nullable creates a nil-aware schema.
func Nullable[T any](schema Schema[T]) NullableSchema[T] {
	if schema == nil {
		panic("goshape: nullable schema must not be nil")
	}
	return NullableSchema[T]{schema: schema}
}

// Parse implements Schema[*T].
func (s NullableSchema[T]) Parse(value any) (*T, error) {
	return s.ParseContext(context.Background(), value)
}

// ParseContext implements Schema[*T].
func (s NullableSchema[T]) ParseContext(ctx context.Context, value any) (*T, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	if value == nil {
		return nil, nil
	}
	parsed, err := s.schema.ParseContext(ctx, value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func (s NullableSchema[T]) buildJSONSchema() (map[string]any, error) {
	inner, err := buildJSONSchema(s.schema)
	if err != nil {
		return nil, err
	}
	return map[string]any{"anyOf": []any{inner, map[string]any{"type": "null"}}}, nil
}
