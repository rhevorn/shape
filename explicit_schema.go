package shape

import (
	"context"
	"io"

	"github.com/rhevorn/shape/transform"
	"github.com/rhevorn/shape/validate"
)

// StructSpec is the immutable explicit struct Schema returned by New.
type StructSpec[T any] struct {
	transformer transform.Transformer[T]
	validator   validate.Validator[T]
}

var _ Schema[struct{}] = StructSpec[struct{}]{}

// Apply appends one or more whole-struct transforms after all field transforms.
func (s StructSpec[T]) Apply(steps ...func(T) (T, error)) StructSpec[T] {
	after := transform.Value[T]().Apply(steps...)
	s.transformer = transform.Value[T]().Then(s.transformer, after)
	return s
}

// ApplyContext is the context-aware form of Apply.
func (s StructSpec[T]) ApplyContext(steps ...func(context.Context, T) (T, error)) StructSpec[T] {
	after := transform.Value[T]().ApplyContext(steps...)
	s.transformer = transform.Value[T]().Then(s.transformer, after)
	return s
}

// Refine appends one or more whole-struct validation rules after all field rules.
func (s StructSpec[T]) Refine(rules ...func(T) error) StructSpec[T] {
	after := validate.Value[T]().Refine(rules...)
	s.validator = validate.Value[T]().And(s.validator, after)
	return s
}

// RefineContext is the context-aware form of Refine.
func (s StructSpec[T]) RefineContext(rules ...func(context.Context, T) error) StructSpec[T] {
	after := validate.Value[T]().RefineContext(rules...)
	s.validator = validate.Value[T]().And(s.validator, after)
	return s
}

func (s StructSpec[T]) Transform(value T) (T, error) {
	return s.TransformContext(context.Background(), value)
}

func (s StructSpec[T]) TransformContext(ctx context.Context, value T) (T, error) {
	out, err := s.transformer.TransformContext(ctx, value)
	if err != nil {
		var zero T
		return zero, normalizeTransformError(ctx, err)
	}
	return out, nil
}

func (s StructSpec[T]) Validate(value T) error {
	return s.validator.Validate(value)
}

func (s StructSpec[T]) ValidateContext(ctx context.Context, value T) error {
	return s.validator.ValidateContext(ctx, value)
}

func (s StructSpec[T]) ValidateFirst(value T) error {
	return s.validator.ValidateFirst(value)
}

func (s StructSpec[T]) ValidateFirstContext(ctx context.Context, value T) error {
	return s.validator.ValidateFirstContext(ctx, value)
}

func (s StructSpec[T]) ParseJSON(source []byte, options ...JSONOptions) (T, error) {
	return parseJSON(s, source, options...)
}

func (s StructSpec[T]) ParseJSONContext(ctx context.Context, source []byte, options ...JSONOptions) (T, error) {
	return parseJSONContext(ctx, s, source, options...)
}

func (s StructSpec[T]) ParseJSONReader(reader io.Reader, options ...JSONOptions) (T, error) {
	return parseJSONReader(s, reader, options...)
}

func (s StructSpec[T]) ParseJSONReaderContext(ctx context.Context, reader io.Reader, options ...JSONOptions) (T, error) {
	return parseJSONReaderContext(ctx, s, reader, options...)
}

func (s StructSpec[T]) BindJSON(target *T, source []byte, options ...JSONOptions) error {
	return bindJSON(s, target, source, options...)
}

func (s StructSpec[T]) BindJSONContext(ctx context.Context, target *T, source []byte, options ...JSONOptions) error {
	return bindJSONContext(ctx, s, target, source, options...)
}

func (s StructSpec[T]) BindJSONReader(target *T, reader io.Reader, options ...JSONOptions) error {
	return bindJSONReader(s, target, reader, options...)
}

func (s StructSpec[T]) BindJSONReaderContext(ctx context.Context, target *T, reader io.Reader, options ...JSONOptions) error {
	return bindJSONReaderContext(ctx, s, target, reader, options...)
}
