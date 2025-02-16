package transform

import (
	"context"
)

type MapTransformer[K MapKey, V any] struct {
	value ValueTransformer[map[K]V]
}

func (t MapTransformer[K, V]) IfNull(v map[K]V) MapTransformer[K, V] {
	snapshot, err := clone(v)
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
func (t MapTransformer[K, V]) Apply(fns ...func(map[K]V) (map[K]V, error)) MapTransformer[K, V] {
	t.value = t.value.Apply(fns...)
	return t
}
func (t MapTransformer[K, V]) ApplyContext(fns ...func(context.Context, map[K]V) (map[K]V, error)) MapTransformer[K, V] {
	t.value = t.value.ApplyContext(fns...)
	return t
}
func (t MapTransformer[K, V]) Then(vs ...Transformer[map[K]V]) Transformer[map[K]V] {
	return sequenceFrom[map[K]V](t, vs...)
}
func (t MapTransformer[K, V]) Transform(v map[K]V) (map[K]V, error) {
	return t.TransformContext(context.Background(), v)
}
func (t MapTransformer[K, V]) TransformContext(ctx context.Context, v map[K]V) (map[K]V, error) {
	return t.value.TransformContext(ctx, v)
}
