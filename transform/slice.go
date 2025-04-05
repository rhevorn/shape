package transform

import "context"

// SliceTransformer transforms a slice and each of its elements.
type SliceTransformer[T any] struct {
	value ValueTransformer[[]T]
}

// IfNull replaces a nil slice with an isolated snapshot of v.
func (t SliceTransformer[T]) IfNull(v []T) SliceTransformer[T] {
	snapshot, err := cloneStrict(v)
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

// Apply appends custom whole-slice transformations.
func (t SliceTransformer[T]) Apply(fns ...func([]T) ([]T, error)) SliceTransformer[T] {
	t.value = t.value.Apply(fns...)
	return t
}

// ApplyContext appends context-aware whole-slice transformations.
func (t SliceTransformer[T]) ApplyContext(fns ...func(context.Context, []T) ([]T, error)) SliceTransformer[T] {
	t.value = t.value.ApplyContext(fns...)
	return t
}

// Then composes this transformer with subsequent slice transformers.
func (t SliceTransformer[T]) Then(vs ...Transformer[[]T]) Transformer[[]T] {
	return sequenceFrom[[]T](t, vs...)
}

// Transform runs the slice pipeline with a background context.
func (t SliceTransformer[T]) Transform(v []T) ([]T, error) {
	return t.TransformContext(context.Background(), v)
}

// TransformContext runs the slice pipeline and observes cancellation.
func (t SliceTransformer[T]) TransformContext(ctx context.Context, v []T) ([]T, error) {
	return t.value.TransformContext(ctx, v)
}

func (t SliceTransformer[T]) transformOwnedContext(ctx context.Context, v []T) ([]T, error) {
	return t.value.transformOwnedContext(ctx, v)
}
