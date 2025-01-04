package goshape

import "context"

// Refine adds value-level validation to any schema without changing its output
// type. Concrete schema builders also expose a Refine method for fluent use.
func Refine[T any](schema Schema[T], fn func(T) error) Schema[T] {
	if schema == nil {
		panic("goshape: refined schema must not be nil")
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
	if err := s.refine(parsed); err != nil {
		var zero T
		return zero, &ValidationError{Issues: issuesFromError(err)}
	}
	if err := checkContext(ctx); err != nil {
		var zero T
		return zero, err
	}
	return parsed, nil
}
