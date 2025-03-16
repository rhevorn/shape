package shape

import (
	"context"
	"io"

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
func (f MapSpec[K, V]) Transform(v map[K]V) (map[K]V, error) { return f.transformer.Transform(v) }
func (f MapSpec[K, V]) TransformContext(ctx context.Context, v map[K]V) (map[K]V, error) {
	return f.transformer.TransformContext(ctx, v)
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

func (f MapSpec[K, V]) ParseJSON(source []byte, options ...JSONOptions) (map[K]V, error) {
	return parseJSON(f, source, options...)
}
func (f MapSpec[K, V]) ParseJSONContext(ctx context.Context, source []byte, options ...JSONOptions) (map[K]V, error) {
	return parseJSONContext(ctx, f, source, options...)
}
func (f MapSpec[K, V]) ParseJSONReader(reader io.Reader, options ...JSONOptions) (map[K]V, error) {
	return parseJSONReader(f, reader, options...)
}
func (f MapSpec[K, V]) ParseJSONReaderContext(ctx context.Context, reader io.Reader, options ...JSONOptions) (map[K]V, error) {
	return parseJSONReaderContext(ctx, f, reader, options...)
}
func (f MapSpec[K, V]) BindJSON(target *map[K]V, source []byte, options ...JSONOptions) error {
	return bindJSON(f, target, source, options...)
}
func (f MapSpec[K, V]) BindJSONContext(ctx context.Context, target *map[K]V, source []byte, options ...JSONOptions) error {
	return bindJSONContext(ctx, f, target, source, options...)
}
func (f MapSpec[K, V]) BindJSONReader(target *map[K]V, reader io.Reader, options ...JSONOptions) error {
	return bindJSONReader(f, target, reader, options...)
}
func (f MapSpec[K, V]) BindJSONReaderContext(ctx context.Context, target *map[K]V, reader io.Reader, options ...JSONOptions) error {
	return bindJSONReaderContext(ctx, f, target, reader, options...)
}
