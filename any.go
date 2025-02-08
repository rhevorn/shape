package shape

import "context"

// AnySchema accepts any Go value and returns it as any.
type AnySchema struct{}

// Any returns a schema for an arbitrary JSON/Go value (output type any).
// Useful with Map: Map(String(), Any()) yields map[string]any.
func Any() AnySchema { return AnySchema{} }

// Parse implements Schema[any].
func (AnySchema) Parse(value any) (any, error) {
	return value, nil
}

// ParseContext implements Schema[any].
func (AnySchema) ParseContext(ctx context.Context, value any) (any, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	return value, nil
}

func (AnySchema) buildJSONSchema(_ *jsonSchemaBuildContext) (map[string]any, error) {
	// Empty schema object: accept any JSON value.
	return map[string]any{}, nil
}
