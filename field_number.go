package shape

import (
	"context"

	"github.com/rhevorn/shape/transform"
	"github.com/rhevorn/shape/validate"
)

// NumberSpec defines transform and validation behavior for a numeric value.
type NumberSpec[N Numeric] struct {
	transformer transform.NumberTransformer[N]
	validator   validate.NumberValidator[N]
}

func (f NumberSpec[N]) Transform(v N) (N, error) {
	return runSpecTransform(context.Background(), f.transformer, v)
}
func (f NumberSpec[N]) TransformContext(ctx context.Context, v N) (N, error) {
	return runSpecTransform(ctx, f.transformer, v)
}
func (f NumberSpec[N]) Validate(v N) error { return f.validator.Validate(v) }
func (f NumberSpec[N]) ValidateContext(ctx context.Context, v N) error {
	return f.validator.ValidateContext(ctx, v)
}
func (f NumberSpec[N]) ValidateFirst(v N) error { return f.validator.ValidateFirst(v) }
func (f NumberSpec[N]) ValidateFirstContext(ctx context.Context, v N) error {
	return f.validator.ValidateFirstContext(ctx, v)
}

func (f NumberSpec[N]) IfZero(v N) NumberSpec[N] { f.transformer = f.transformer.IfZero(v); return f }
func (f NumberSpec[N]) Apply(values ...func(N) (N, error)) NumberSpec[N] {
	f.transformer = f.transformer.Apply(values...)
	return f
}
func (f NumberSpec[N]) ApplyContext(values ...func(context.Context, N) (N, error)) NumberSpec[N] {
	f.transformer = f.transformer.ApplyContext(values...)
	return f
}
func (f NumberSpec[N]) Min(v N) NumberSpec[N] { f.validator = f.validator.Min(v); return f }
func (f NumberSpec[N]) Max(v N) NumberSpec[N] { f.validator = f.validator.Max(v); return f }
func (f NumberSpec[N]) Gt(v N) NumberSpec[N]  { f.validator = f.validator.Gt(v); return f }
func (f NumberSpec[N]) Gte(v N) NumberSpec[N] { f.validator = f.validator.Gte(v); return f }
func (f NumberSpec[N]) Lt(v N) NumberSpec[N]  { f.validator = f.validator.Lt(v); return f }
func (f NumberSpec[N]) Lte(v N) NumberSpec[N] { f.validator = f.validator.Lte(v); return f }
func (f NumberSpec[N]) Between(a, b N) NumberSpec[N] {
	f.validator = f.validator.Between(a, b)
	return f
}
func (f NumberSpec[N]) OneOf(values ...N) NumberSpec[N] {
	f.validator = f.validator.OneOf(values...)
	return f
}
func (f NumberSpec[N]) Positive() NumberSpec[N] { f.validator = f.validator.Positive(); return f }
func (f NumberSpec[N]) Negative() NumberSpec[N] { f.validator = f.validator.Negative(); return f }
func (f NumberSpec[N]) NonNegative() NumberSpec[N] {
	f.validator = f.validator.NonNegative()
	return f
}
func (f NumberSpec[N]) Refine(values ...func(N) error) NumberSpec[N] {
	f.validator = f.validator.Refine(values...)
	return f
}
func (f NumberSpec[N]) RefineContext(values ...func(context.Context, N) error) NumberSpec[N] {
	f.validator = f.validator.RefineContext(values...)
	return f
}
func (f NumberSpec[N]) Label(label string) NumberSpec[N] {
	f.validator = f.validator.Label(label)
	return f
}
func (f NumberSpec[N]) Pointer() PointerSpec[N] { return pointerSpec(f) }
func (f NumberSpec[N]) Slice() SliceSpec[N]     { return sliceSpec(f) }
