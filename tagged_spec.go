package shape

import (
	"context"
	"io"

	"github.com/rhevorn/shape/validate"
)

// TaggedSpec is the immutable tagged struct Schema returned by Struct. Apply
// and Refine add whole-struct behavior after the compiled field behavior.
type TaggedSpec[T any] struct {
	base       structSchema[T]
	validator  validate.Validator[T]
	transforms []wholeTransformStep[T]
	custom     bool
}

var _ Schema[struct{}] = TaggedSpec[struct{}]{}

func newTaggedSpec[T any](plan *valuePlan) TaggedSpec[T] {
	base := structSchema[T]{p: plan}
	return TaggedSpec[T]{base: base, validator: base}
}

func (s TaggedSpec[T]) Apply(steps ...func(T) (T, error)) TaggedSpec[T] {
	if len(steps) == 0 {
		return s
	}
	s.transforms = appendWholeApply(s.transforms, steps...)
	s.custom = true
	return s
}

func (s TaggedSpec[T]) ApplyContext(steps ...func(context.Context, T) (T, error)) TaggedSpec[T] {
	if len(steps) == 0 {
		return s
	}
	s.transforms = appendWholeApplyContext(s.transforms, steps...)
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

func (s TaggedSpec[T]) Transform(value T) (T, error) {
	return s.TransformContext(context.Background(), value)
}

func (s TaggedSpec[T]) TransformContext(ctx context.Context, value T) (T, error) {
	var zero T
	out, err := s.base.TransformContext(ctx, value)
	if err != nil {
		return zero, normalizeTransformError(ctx, err)
	}
	out, err = runWholeTransforms(ctx, s.transforms, out)
	if err != nil {
		return zero, normalizeTransformError(ctx, err)
	}
	return out, nil
}

func (s TaggedSpec[T]) transformDecodedContext(ctx context.Context, value T) (T, error) {
	var zero T
	out, err := s.base.transformDecodedContext(ctx, value)
	if err != nil {
		return zero, normalizeTransformError(ctx, err)
	}
	out, err = runWholeTransforms(ctx, s.transforms, out)
	if err != nil {
		return zero, normalizeTransformError(ctx, err)
	}
	return out, nil
}

func (s TaggedSpec[T]) Validate(value T) error { return s.validator.Validate(value) }
func (s TaggedSpec[T]) ValidateContext(ctx context.Context, value T) error {
	return s.validator.ValidateContext(ctx, value)
}
func (s TaggedSpec[T]) ValidateFirst(value T) error { return s.validator.ValidateFirst(value) }
func (s TaggedSpec[T]) ValidateFirstContext(ctx context.Context, value T) error {
	return s.validator.ValidateFirstContext(ctx, value)
}

func (s TaggedSpec[T]) ParseJSON(source []byte, options ...JSONOptions) (T, error) {
	return parseJSON(s, source, options...)
}
func (s TaggedSpec[T]) ParseJSONContext(ctx context.Context, source []byte, options ...JSONOptions) (T, error) {
	return parseJSONContext(ctx, s, source, options...)
}
func (s TaggedSpec[T]) ParseJSONReader(reader io.Reader, options ...JSONOptions) (T, error) {
	return parseJSONReader(s, reader, options...)
}
func (s TaggedSpec[T]) ParseJSONReaderContext(ctx context.Context, reader io.Reader, options ...JSONOptions) (T, error) {
	return parseJSONReaderContext(ctx, s, reader, options...)
}
func (s TaggedSpec[T]) BindJSON(target *T, source []byte, options ...JSONOptions) error {
	return bindJSON(s, target, source, options...)
}
func (s TaggedSpec[T]) BindJSONContext(ctx context.Context, target *T, source []byte, options ...JSONOptions) error {
	return bindJSONContext(ctx, s, target, source, options...)
}
func (s TaggedSpec[T]) BindJSONReader(target *T, reader io.Reader, options ...JSONOptions) error {
	return bindJSONReader(s, target, reader, options...)
}
func (s TaggedSpec[T]) BindJSONReaderContext(ctx context.Context, target *T, reader io.Reader, options ...JSONOptions) error {
	return bindJSONReaderContext(ctx, s, target, reader, options...)
}

func (s TaggedSpec[T]) exportPlan() (*valuePlan, error) {
	if s.custom {
		return nil, &UnsupportedSchemaError{Operation: "custom Apply or Refine"}
	}
	return s.base.p, nil
}
