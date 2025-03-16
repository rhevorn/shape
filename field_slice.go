package shape

import (
	"context"
	"io"

	"github.com/rhevorn/shape/transform"
	"github.com/rhevorn/shape/validate"
)

// SliceSpec defines behavior for a slice and each of its elements.
type SliceSpec[T any] struct {
	name        string
	transformer transform.SliceTransformer[T]
	validator   validate.SliceValidator[T]
}

func (f SliceSpec[T]) fieldDefinition() erasedField {
	return eraseField(f.name, f.transformer, f.validator)
}
func (f SliceSpec[T]) Transform(v []T) ([]T, error) { return f.transformer.Transform(v) }
func (f SliceSpec[T]) TransformContext(ctx context.Context, v []T) ([]T, error) {
	return f.transformer.TransformContext(ctx, v)
}
func (f SliceSpec[T]) Validate(v []T) error { return f.validator.Validate(v) }
func (f SliceSpec[T]) ValidateContext(ctx context.Context, v []T) error {
	return f.validator.ValidateContext(ctx, v)
}
func (f SliceSpec[T]) ValidateFirst(v []T) error { return f.validator.ValidateFirst(v) }
func (f SliceSpec[T]) ValidateFirstContext(ctx context.Context, v []T) error {
	return f.validator.ValidateFirstContext(ctx, v)
}

func (f SliceSpec[T]) IfNull(v []T) SliceSpec[T] { f.transformer = f.transformer.IfNull(v); return f }
func (f SliceSpec[T]) Apply(values ...func([]T) ([]T, error)) SliceSpec[T] {
	f.transformer = f.transformer.Apply(values...)
	return f
}
func (f SliceSpec[T]) ApplyContext(values ...func(context.Context, []T) ([]T, error)) SliceSpec[T] {
	f.transformer = f.transformer.ApplyContext(values...)
	return f
}
func (f SliceSpec[T]) NotNull() SliceSpec[T]  { f.validator = f.validator.NotNull(); return f }
func (f SliceSpec[T]) NotEmpty() SliceSpec[T] { f.validator = f.validator.NotEmpty(); return f }
func (f SliceSpec[T]) Min(n int) SliceSpec[T] { f.validator = f.validator.Min(n); return f }
func (f SliceSpec[T]) Max(n int) SliceSpec[T] { f.validator = f.validator.Max(n); return f }
func (f SliceSpec[T]) Len(n int) SliceSpec[T] { f.validator = f.validator.Len(n); return f }
func (f SliceSpec[T]) Unique() SliceSpec[T]   { f.validator = f.validator.Unique(); return f }
func (f SliceSpec[T]) Refine(values ...func([]T) error) SliceSpec[T] {
	f.validator = f.validator.Refine(values...)
	return f
}
func (f SliceSpec[T]) RefineContext(values ...func(context.Context, []T) error) SliceSpec[T] {
	f.validator = f.validator.RefineContext(values...)
	return f
}
func (f SliceSpec[T]) Label(label string) SliceSpec[T] {
	f.validator = f.validator.Label(label)
	return f
}

func (f SliceSpec[T]) ParseJSON(source []byte, options ...JSONOptions) ([]T, error) {
	return parseJSON(f, source, options...)
}
func (f SliceSpec[T]) ParseJSONContext(ctx context.Context, source []byte, options ...JSONOptions) ([]T, error) {
	return parseJSONContext(ctx, f, source, options...)
}
func (f SliceSpec[T]) ParseJSONReader(reader io.Reader, options ...JSONOptions) ([]T, error) {
	return parseJSONReader(f, reader, options...)
}
func (f SliceSpec[T]) ParseJSONReaderContext(ctx context.Context, reader io.Reader, options ...JSONOptions) ([]T, error) {
	return parseJSONReaderContext(ctx, f, reader, options...)
}
func (f SliceSpec[T]) BindJSON(target *[]T, source []byte, options ...JSONOptions) error {
	return bindJSON(f, target, source, options...)
}
func (f SliceSpec[T]) BindJSONContext(ctx context.Context, target *[]T, source []byte, options ...JSONOptions) error {
	return bindJSONContext(ctx, f, target, source, options...)
}
func (f SliceSpec[T]) BindJSONReader(target *[]T, reader io.Reader, options ...JSONOptions) error {
	return bindJSONReader(f, target, reader, options...)
}
func (f SliceSpec[T]) BindJSONReaderContext(ctx context.Context, target *[]T, reader io.Reader, options ...JSONOptions) error {
	return bindJSONReaderContext(ctx, f, target, reader, options...)
}
