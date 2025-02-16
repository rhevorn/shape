package transform

import "context"

type SliceTransformer[T any] struct {
	value ValueTransformer[[]T]
}

func (t SliceTransformer[T]) IfNull(v []T) SliceTransformer[T] {
	snapshot, err := clone(v)
	if err != nil {
		panic("transform: invalid IfNull value: " + err.Error())
	}
	t.value = t.value.ApplyContext(func(ctx context.Context, value []T) ([]T, error) {
		if value != nil {
			return value, nil
		}
		return cloneContext(ctx, snapshot)
	})
	return t
}
func (t SliceTransformer[T]) Apply(fns ...func([]T) ([]T, error)) SliceTransformer[T] {
	t.value = t.value.Apply(fns...)
	return t
}
func (t SliceTransformer[T]) ApplyContext(fns ...func(context.Context, []T) ([]T, error)) SliceTransformer[T] {
	t.value = t.value.ApplyContext(fns...)
	return t
}
func (t SliceTransformer[T]) Then(vs ...Transformer[[]T]) Transformer[[]T] {
	return sequenceFrom[[]T](t, vs...)
}
func (t SliceTransformer[T]) Transform(v []T) ([]T, error) {
	return t.TransformContext(context.Background(), v)
}
func (t SliceTransformer[T]) TransformContext(ctx context.Context, v []T) ([]T, error) {
	return t.value.TransformContext(ctx, v)
}
