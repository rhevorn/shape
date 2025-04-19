package shape

import (
	"context"
	"io"

	"github.com/rhevorn/shape/internal/pipeline"
	"github.com/rhevorn/shape/transform"
	"github.com/rhevorn/shape/validate"
)

// StructSpec is the immutable explicit struct Schema returned by New.
type StructSpec[T any] struct {
	transformer transform.Transformer[T]
	validator   validate.Validator[T]
	transforms  []pipeline.Step[T]
}

var _ Schema[struct{}] = StructSpec[struct{}]{}
var _ JSONSchema[struct{}] = StructSpec[struct{}]{}

// Apply appends one or more whole-struct transforms after all field transforms.
func (s StructSpec[T]) Apply(steps ...func(T) (T, error)) StructSpec[T] {
	s.transforms = pipeline.Append(s.transforms, steps...)
	return s
}

// ApplyContext is the context-aware form of Apply.
func (s StructSpec[T]) ApplyContext(steps ...func(context.Context, T) (T, error)) StructSpec[T] {
	s.transforms = pipeline.AppendContext(s.transforms, steps...)
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

// Transform applies field and whole-struct transforms without validation.
func (s StructSpec[T]) Transform(value T) (T, error) {
	return s.TransformContext(context.Background(), value)
}

// TransformContext applies transforms and observes context cancellation.
func (s StructSpec[T]) TransformContext(ctx context.Context, value T) (T, error) {
	return s.transformContext(ctx, value, false)
}

func (s StructSpec[T]) transformDecodedContext(ctx context.Context, value T) (T, error) {
	return s.transformContext(ctx, value, true)
}

func (s StructSpec[T]) transformContext(ctx context.Context, value T, owned bool) (T, error) {
	var zero T
	out, err := s.transformer.TransformContext(ctx, value)
	if err != nil {
		return zero, normalizeTransformError(ctx, err)
	}
	out, err = pipeline.Apply(ctx, s.transforms, out, owned)
	if err != nil {
		return zero, normalizeTransformError(ctx, err)
	}
	return out, nil
}

// Validate collects validation issues without transforming value.
func (s StructSpec[T]) Validate(value T) error {
	return s.validator.Validate(value)
}

// ValidateContext collects issues and observes context cancellation.
func (s StructSpec[T]) ValidateContext(ctx context.Context, value T) error {
	return s.validator.ValidateContext(ctx, value)
}

// ValidateFirst stops after the first issue.
func (s StructSpec[T]) ValidateFirst(value T) error {
	return s.validator.ValidateFirst(value)
}

// ValidateFirstContext stops after the first issue and observes cancellation.
func (s StructSpec[T]) ValidateFirstContext(ctx context.Context, value T) error {
	return s.validator.ValidateFirstContext(ctx, value)
}

// ParseJSON decodes, transforms, and validates one JSON value.
func (s StructSpec[T]) ParseJSON(source []byte, options ...JSONOptions) (T, error) {
	return ParseJSON(s, source, options...)
}

// ParseJSONContext decodes, transforms, and validates with ctx.
func (s StructSpec[T]) ParseJSONContext(ctx context.Context, source []byte, options ...JSONOptions) (T, error) {
	return ParseJSONContext(ctx, s, source, options...)
}

// ParseJSONReader reads, transforms, and validates one JSON value.
func (s StructSpec[T]) ParseJSONReader(reader io.Reader, options ...JSONOptions) (T, error) {
	return ParseJSONReader(s, reader, options...)
}

// ParseJSONReaderContext is the context-aware reader form of ParseJSON.
func (s StructSpec[T]) ParseJSONReaderContext(ctx context.Context, reader io.Reader, options ...JSONOptions) (T, error) {
	return ParseJSONReaderContext(ctx, s, reader, options...)
}
