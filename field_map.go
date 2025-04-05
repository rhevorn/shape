package shape

import (
	"context"

	"github.com/rhevorn/shape/transform"
	"github.com/rhevorn/shape/validate"
)

// MapSpec defines behavior for a map and each key and value.
type MapSpec[K MapKey, V any] struct {
	name        string
	transformer transform.MapTransformer[K, V]
	validator   validate.MapValidator[K, V]
}

func (f MapSpec[K, V]) fieldDefinition() erasedField {
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
func (f MapSpec[K, V]) NotNull() MapSpec[K, V]  { f.validator = f.validator.NotNull(); return f }
func (f MapSpec[K, V]) NotEmpty() MapSpec[K, V] { f.validator = f.validator.NotEmpty(); return f }
func (f MapSpec[K, V]) Min(n int) MapSpec[K, V] { f.validator = f.validator.Min(n); return f }
func (f MapSpec[K, V]) Max(n int) MapSpec[K, V] { f.validator = f.validator.Max(n); return f }
func (f MapSpec[K, V]) Len(n int) MapSpec[K, V] { f.validator = f.validator.Len(n); return f }
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
