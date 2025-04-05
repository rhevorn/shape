package transform

import (
	"context"
)

// MapTransformer transforms a map and each key and value in stable key order.
type MapTransformer[K MapKey, V any] struct {
	value ValueTransformer[map[K]V]
}

// IfNull replaces a nil map with an isolated snapshot of v.
func (t MapTransformer[K, V]) IfNull(v map[K]V) MapTransformer[K, V] {
	snapshot, err := cloneStrict(v)
	if err != nil {
		panic("transform: invalid IfNull value: " + err.Error())
	}
	t.value = t.value.ApplyContext(func(ctx context.Context, value map[K]V) (map[K]V, error) {
		if value != nil {
			return value, nil
		}
		return cloneContext(ctx, snapshot)
	})
	return t
}

// Apply appends custom whole-map transformations.
func (t MapTransformer[K, V]) Apply(fns ...func(map[K]V) (map[K]V, error)) MapTransformer[K, V] {
	t.value = t.value.Apply(fns...)
	return t
}

// ApplyContext appends context-aware whole-map transformations.
func (t MapTransformer[K, V]) ApplyContext(fns ...func(context.Context, map[K]V) (map[K]V, error)) MapTransformer[K, V] {
	t.value = t.value.ApplyContext(fns...)
	return t
}

// Then composes this transformer with subsequent map transformers.
func (t MapTransformer[K, V]) Then(vs ...Transformer[map[K]V]) Transformer[map[K]V] {
	return sequenceFrom[map[K]V](t, vs...)
}

// Transform runs the map pipeline with a background context.
func (t MapTransformer[K, V]) Transform(v map[K]V) (map[K]V, error) {
	return t.TransformContext(context.Background(), v)
}

// TransformContext runs the map pipeline and observes cancellation.
func (t MapTransformer[K, V]) TransformContext(ctx context.Context, v map[K]V) (map[K]V, error) {
	return t.value.TransformContext(ctx, v)
}

func (t MapTransformer[K, V]) transformOwnedContext(ctx context.Context, v map[K]V) (map[K]V, error) {
	return t.value.transformOwnedContext(ctx, v)
}
