package goshape

import "context"

// Transform parses with schema and converts the successful value to a new
// output type.
func Transform[A, B any](schema Schema[A], fn func(A) (B, error)) Schema[B] {
	if schema == nil {
		panic("goshape: transformed schema must not be nil")
	}
	if fn == nil {
		panic("goshape: transform function must not be nil")
	}
	return transformSchema[A, B]{schema: schema, transform: fn}
}

type transformSchema[A, B any] struct {
	schema    Schema[A]
	transform func(A) (B, error)
}

func (s transformSchema[A, B]) Parse(value any) (B, error) {
	return s.ParseContext(context.Background(), value)
}

func (s transformSchema[A, B]) ParseContext(ctx context.Context, value any) (B, error) {
	parsed, err := s.schema.ParseContext(ctx, value)
	if err != nil {
		var zero B
		return zero, err
	}
	if err := checkContext(ctx); err != nil {
		var zero B
		return zero, err
	}
	result, err := s.transform(parsed)
	if err != nil {
		var zero B
		return zero, validationError(Issue{
			Code:    CodeTransformFailed,
			Message: err.Error(),
		})
	}
	if err := checkContext(ctx); err != nil {
		var zero B
		return zero, err
	}
	return result, nil
}
