package shape

import "context"

// RefineContext adds context-aware validation to any schema.
func RefineContext[T any](schema Schema[T], fn func(context.Context, T) error) Schema[T] {
	if schema == nil {
		panic("shape: refined schema must not be nil")
	}
	if fn == nil {
		panic("shape: context refinement function must not be nil")
	}
	return contextRefineSchema[T]{schema: schema, refine: fn}
}

type contextRefineSchema[T any] struct {
	schema Schema[T]
	refine func(context.Context, T) error
}

func (s contextRefineSchema[T]) Parse(value any) (T, error) {
	return s.ParseContext(context.Background(), value)
}

func (s contextRefineSchema[T]) ParseContext(ctx context.Context, value any) (T, error) {
	var zero T
	parsed, err := s.schema.ParseContext(ctx, value)
	if err != nil {
		return zero, err
	}
	if err := s.refine(ctx, parsed); err != nil {
		if contextErr := contextError(err, ctx); contextErr != nil {
			return zero, contextErr
		}
		return zero, validationIssues(ctx, issuesFromError(err))
	}
	if err := checkContext(ctx); err != nil {
		return zero, err
	}
	return parsed, nil
}

func (s contextRefineSchema[T]) buildJSONSchema(_ *jsonSchemaBuildContext) (map[string]any, error) {
	return nil, &UnsupportedSchemaError{Operation: "context refinement"}
}
