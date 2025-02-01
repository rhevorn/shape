package shape

import "context"

// Refine adds value-level validation to any schema without changing its output
// type. Concrete schema builders also expose a Refine method for fluent use.
func Refine[T any](schema Schema[T], fn func(T) error) Schema[T] {
	if schema == nil {
		panic("shape: refined schema must not be nil")
	}
	return refineSchema[T]{schema: schema, refine: requireRefinement(fn)}
}

type refineSchema[T any] struct {
	schema Schema[T]
	refine refinement[T]
}

func (s refineSchema[T]) Parse(value any) (T, error) {
	return s.ParseContext(context.Background(), value)
}

func (s refineSchema[T]) ParseContext(ctx context.Context, value any) (T, error) {
	parsed, err := s.schema.ParseContext(ctx, value)
	if err != nil {
		return parsed, err
	}
	if err := checkContext(ctx); err != nil {
		var zero T
		return zero, err
	}
	refinementErr := s.refine(parsed)
	if contextErr := contextError(refinementErr, ctx); contextErr != nil {
		var zero T
		return zero, contextErr
	}
	if refinementErr != nil {
		var zero T
		return zero, validationIssues(ctx, issuesFromError(refinementErr))
	}
	if err := checkContext(ctx); err != nil {
		var zero T
		return zero, err
	}
	return parsed, nil
}

func (s refineSchema[T]) buildJSONSchema(_ *jsonSchemaBuildContext) (map[string]any, error) {
	return nil, &UnsupportedSchemaError{Operation: "custom refinement"}
}
