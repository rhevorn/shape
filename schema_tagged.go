package shape

import (
	"context"
	"io"

	"github.com/rhevorn/shape/internal/pipeline"
	"github.com/rhevorn/shape/internal/tagged"
	"github.com/rhevorn/shape/validate"
)

// TaggedSpec is the immutable tagged struct Schema returned by Struct. Apply
// and Refine add whole-struct behavior after the compiled field behavior.
type TaggedSpec[T any] struct {
	base       tagged.Runtime[T]
	validator  validate.Validator[T]
	transforms []pipeline.Step[T]
	custom     bool
}

var _ Schema[struct{}] = TaggedSpec[struct{}]{}
var _ JSONSchema[struct{}] = TaggedSpec[struct{}]{}

func newTaggedSpec[T any](plan *tagged.Plan) TaggedSpec[T] {
	base := tagged.Runtime[T]{Plan: plan}
	return TaggedSpec[T]{base: base, validator: base}
}

func (s TaggedSpec[T]) Apply(steps ...func(T) (T, error)) TaggedSpec[T] {
	if len(steps) == 0 {
		return s
	}
	s.transforms = pipeline.Append(s.transforms, steps...)
	s.custom = true
	return s
}

func (s TaggedSpec[T]) ApplyContext(steps ...func(context.Context, T) (T, error)) TaggedSpec[T] {
	if len(steps) == 0 {
		return s
	}
	s.transforms = pipeline.AppendContext(s.transforms, steps...)
	s.custom = true
	return s
}

func (s TaggedSpec[T]) Refine(rules ...func(T) error) TaggedSpec[T] {
	if len(rules) == 0 {
		return s
	}
	after := validate.Value[T]().Refine(rules...)
	s.validator = validate.Value[T]().And(s.validator, after)
	s.custom = true
	return s
}

func (s TaggedSpec[T]) RefineContext(rules ...func(context.Context, T) error) TaggedSpec[T] {
	if len(rules) == 0 {
		return s
	}
	after := validate.Value[T]().RefineContext(rules...)
	s.validator = validate.Value[T]().And(s.validator, after)
	s.custom = true
	return s
}

// Transform applies tag and whole-struct transforms without validation.
func (s TaggedSpec[T]) Transform(value T) (T, error) {
	return s.TransformContext(context.Background(), value)
}

// TransformContext applies transforms and observes context cancellation.
func (s TaggedSpec[T]) TransformContext(ctx context.Context, value T) (T, error) {
	return s.transformContext(ctx, value, false)
}

// transformDecodedContext is the owned entry point: the value came straight
// from JSON decoding, so it is already a private copy.
func (s TaggedSpec[T]) transformDecodedContext(ctx context.Context, value T) (T, error) {
	return s.transformContext(ctx, value, true)
}

func (s TaggedSpec[T]) transformContext(ctx context.Context, value T, owned bool) (T, error) {
	var zero T
	out, err := s.base.TransformContext(ctx, value)
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
func (s TaggedSpec[T]) Validate(value T) error { return s.validator.Validate(value) }

// ValidateContext collects issues and observes context cancellation.
func (s TaggedSpec[T]) ValidateContext(ctx context.Context, value T) error {
	return s.validator.ValidateContext(ctx, value)
}

// ValidateFirst stops after the first issue.
func (s TaggedSpec[T]) ValidateFirst(value T) error { return s.validator.ValidateFirst(value) }

// ValidateFirstContext stops after the first issue and observes cancellation.
func (s TaggedSpec[T]) ValidateFirstContext(ctx context.Context, value T) error {
	return s.validator.ValidateFirstContext(ctx, value)
}

// ParseJSON decodes, transforms, and validates one JSON value.
func (s TaggedSpec[T]) ParseJSON(source []byte, options ...JSONOptions) (T, error) {
	return ParseJSON(s, source, options...)
}

// ParseJSONContext decodes, transforms, and validates with ctx.
func (s TaggedSpec[T]) ParseJSONContext(ctx context.Context, source []byte, options ...JSONOptions) (T, error) {
	return ParseJSONContext(ctx, s, source, options...)
}

// ParseJSONReader reads, transforms, and validates one JSON value.
func (s TaggedSpec[T]) ParseJSONReader(reader io.Reader, options ...JSONOptions) (T, error) {
	return ParseJSONReader(s, reader, options...)
}

// ParseJSONReaderContext is the context-aware reader form of ParseJSON.
func (s TaggedSpec[T]) ParseJSONReaderContext(ctx context.Context, reader io.Reader, options ...JSONOptions) (T, error) {
	return ParseJSONReaderContext(ctx, s, reader, options...)
}

func (s TaggedSpec[T]) schemaPlan() (*tagged.Plan, error) {
	if s.custom {
		return nil, &UnsupportedSchemaError{Feature: "custom Apply or Refine"}
	}
	return s.base.Plan, nil
}
