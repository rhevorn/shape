package shape

import (
	"context"

	"github.com/rhevorn/shape/internal/program"
	"github.com/rhevorn/shape/transform"
	"github.com/rhevorn/shape/validate"
)

// PointerSpec defines behavior for a pointer and its non-nil inner value.
type PointerSpec[T any] struct {
	name        string
	transformer transform.PointerTransformer[T]
	validator   validate.PointerValidator[T]
}

func (f PointerSpec[T]) fieldDefinition() program.Definition {
	return eraseField(f.name, f.transformer, f.validator)
}
func (f PointerSpec[T]) Transform(v *T) (*T, error) {
	return runSpecTransform(context.Background(), f.transformer, v)
}
func (f PointerSpec[T]) TransformContext(ctx context.Context, v *T) (*T, error) {
	return runSpecTransform(ctx, f.transformer, v)
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

// SliceSpec defines behavior for a slice and each of its elements.
type SliceSpec[T any] struct {
	name        string
	transformer transform.SliceTransformer[T]
	validator   validate.SliceValidator[T]
}

func (f SliceSpec[T]) fieldDefinition() program.Definition {
	return eraseField(f.name, f.transformer, f.validator)
}
func (f SliceSpec[T]) Transform(v []T) ([]T, error) {
	return runSpecTransform(context.Background(), f.transformer, v)
}
func (f SliceSpec[T]) TransformContext(ctx context.Context, v []T) ([]T, error) {
	return runSpecTransform(ctx, f.transformer, v)
}
func (f SliceSpec[T]) Validate(v []T) error { return f.validator.Validate(v) }
func (f SliceSpec[T]) ValidateContext(ctx context.Context, v []T) error {
	return f.validator.ValidateContext(ctx, v)
}
func (f SliceSpec[T]) ValidateFirst(v []T) error { return f.validator.ValidateFirst(v) }
func (f SliceSpec[T]) ValidateFirstContext(ctx context.Context, v []T) error {
	return f.validator.ValidateFirstContext(ctx, v)
}

func (f SliceSpec[T]) IfNull(v []T) SliceSpec[T] {
	f.transformer = f.transformer.IfNull(v)
	return f
}
func (f SliceSpec[T]) Apply(values ...func([]T) ([]T, error)) SliceSpec[T] {
	f.transformer = f.transformer.Apply(values...)
	return f
}
func (f SliceSpec[T]) ApplyContext(values ...func(context.Context, []T) ([]T, error)) SliceSpec[T] {
	f.transformer = f.transformer.ApplyContext(values...)
	return f
}
func (f SliceSpec[T]) NotNull() SliceSpec[T] {
	f.validator = f.validator.NotNull()
	return f
}
func (f SliceSpec[T]) NotEmpty() SliceSpec[T] {
	f.validator = f.validator.NotEmpty()
	return f
}
func (f SliceSpec[T]) Min(n int) SliceSpec[T] {
	f.validator = f.validator.Min(n)
	return f
}
func (f SliceSpec[T]) Max(n int) SliceSpec[T] {
	f.validator = f.validator.Max(n)
	return f
}
func (f SliceSpec[T]) Len(n int) SliceSpec[T] {
	f.validator = f.validator.Len(n)
	return f
}
func (f SliceSpec[T]) Unique() SliceSpec[T] {
	f.validator = f.validator.Unique()
	return f
}
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

// MapSpec defines behavior for a map and each key and value.
type MapSpec[K MapKey, V any] struct {
	name        string
	transformer transform.MapTransformer[K, V]
	validator   validate.MapValidator[K, V]
}

func (f MapSpec[K, V]) fieldDefinition() program.Definition {
	return eraseField(f.name, f.transformer, f.validator)
}
func (f MapSpec[K, V]) Transform(v map[K]V) (map[K]V, error) {
	return runSpecTransform(context.Background(), f.transformer, v)
}
func (f MapSpec[K, V]) TransformContext(ctx context.Context, v map[K]V) (map[K]V, error) {
	return runSpecTransform(ctx, f.transformer, v)
}
func (f MapSpec[K, V]) Validate(v map[K]V) error { return f.validator.Validate(v) }
func (f MapSpec[K, V]) ValidateContext(ctx context.Context, v map[K]V) error {
	return f.validator.ValidateContext(ctx, v)
}
func (f MapSpec[K, V]) ValidateFirst(v map[K]V) error { return f.validator.ValidateFirst(v) }
func (f MapSpec[K, V]) ValidateFirstContext(ctx context.Context, v map[K]V) error {
	return f.validator.ValidateFirstContext(ctx, v)
}

func (f MapSpec[K, V]) IfNull(v map[K]V) MapSpec[K, V] {
	f.transformer = f.transformer.IfNull(v)
	return f
}
func (f MapSpec[K, V]) Apply(values ...func(map[K]V) (map[K]V, error)) MapSpec[K, V] {
	f.transformer = f.transformer.Apply(values...)
	return f
}
func (f MapSpec[K, V]) ApplyContext(values ...func(context.Context, map[K]V) (map[K]V, error)) MapSpec[K, V] {
	f.transformer = f.transformer.ApplyContext(values...)
	return f
}
func (f MapSpec[K, V]) NotNull() MapSpec[K, V] {
	f.validator = f.validator.NotNull()
	return f
}
func (f MapSpec[K, V]) NotEmpty() MapSpec[K, V] {
	f.validator = f.validator.NotEmpty()
	return f
}
func (f MapSpec[K, V]) Min(n int) MapSpec[K, V] {
	f.validator = f.validator.Min(n)
	return f
}
func (f MapSpec[K, V]) Max(n int) MapSpec[K, V] {
	f.validator = f.validator.Max(n)
	return f
}
func (f MapSpec[K, V]) Len(n int) MapSpec[K, V] {
	f.validator = f.validator.Len(n)
	return f
}
func (f MapSpec[K, V]) Refine(values ...func(map[K]V) error) MapSpec[K, V] {
	f.validator = f.validator.Refine(values...)
	return f
}
func (f MapSpec[K, V]) RefineContext(values ...func(context.Context, map[K]V) error) MapSpec[K, V] {
	f.validator = f.validator.RefineContext(values...)
	return f
}
func (f MapSpec[K, V]) Label(label string) MapSpec[K, V] {
	f.validator = f.validator.Label(label)
	return f
}
