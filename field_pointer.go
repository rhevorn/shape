package shape

import (
	"context"

	"github.com/rhevorn/shape/transform"
	"github.com/rhevorn/shape/validate"
)

// PointerSpec defines behavior for a pointer and its non-nil inner value.
type PointerSpec[T any] struct {
	name        string
	transformer transform.PointerTransformer[T]
	validator   validate.PointerValidator[T]
}

func (f PointerSpec[T]) fieldDefinition() erasedField {
	return eraseField(f.name, f.transformer, f.validator)
}
func (f PointerSpec[T]) Transform(v *T) (*T, error) { return f.transformer.Transform(v) }
func (f PointerSpec[T]) TransformContext(ctx context.Context, v *T) (*T, error) {
	return f.transformer.TransformContext(ctx, v)
}
func (f PointerSpec[T]) Validate(v *T) error { return f.validator.Validate(v) }
func (f PointerSpec[T]) ValidateContext(ctx context.Context, v *T) error {
	return f.validator.ValidateContext(ctx, v)
}
func (f PointerSpec[T]) ValidateFirst(v *T) error { return f.validator.ValidateFirst(v) }
func (f PointerSpec[T]) ValidateFirstContext(ctx context.Context, v *T) error {
	return f.validator.ValidateFirstContext(ctx, v)
}

func (f PointerSpec[T]) IfNull(v *T) PointerSpec[T] {
	f.transformer = f.transformer.IfNull(v)
	return f
}
func (f PointerSpec[T]) Apply(values ...func(*T) (*T, error)) PointerSpec[T] {
	f.transformer = f.transformer.Apply(values...)
	return f
}
func (f PointerSpec[T]) ApplyContext(values ...func(context.Context, *T) (*T, error)) PointerSpec[T] {
	f.transformer = f.transformer.ApplyContext(values...)
	return f
}
func (f PointerSpec[T]) NotNull() PointerSpec[T]  { f.validator = f.validator.NotNull(); return f }
func (f PointerSpec[T]) NotEmpty() PointerSpec[T] { f.validator = f.validator.NotEmpty(); return f }
func (f PointerSpec[T]) Refine(values ...func(*T) error) PointerSpec[T] {
	f.validator = f.validator.Refine(values...)
	return f
}
func (f PointerSpec[T]) RefineContext(values ...func(context.Context, *T) error) PointerSpec[T] {
	f.validator = f.validator.RefineContext(values...)
	return f
}
func (f PointerSpec[T]) Label(label string) PointerSpec[T] {
	f.validator = f.validator.Label(label)
	return f
}
