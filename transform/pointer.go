package transform

import "context"

// PointerTransformer transforms an optional *T and its non-nil value.
type PointerTransformer[T any] struct {
	value ValueTransformer[*T]
}

// IfNull replaces nil with an isolated snapshot of v.
func (t PointerTransformer[T]) IfNull(v *T) PointerTransformer[T] {
	snapshot, err := cloneStrict(v)
	if err != nil {
		panic("transform: invalid IfNull value: " + err.Error())
	}
	t.value = t.value.ApplyContext(func(ctx context.Context, value *T) (*T, error) {
		if value != nil {
			return value, nil
		}
		return cloneContext(ctx, snapshot)
	})
	return t
}

// Apply appends custom pointer transformations.
func (t PointerTransformer[T]) Apply(fns ...func(*T) (*T, error)) PointerTransformer[T] {
	t.value = t.value.Apply(fns...)
	return t
}

// ApplyContext appends context-aware custom pointer transformations.
func (t PointerTransformer[T]) ApplyContext(fns ...func(context.Context, *T) (*T, error)) PointerTransformer[T] {
	t.value = t.value.ApplyContext(fns...)
	return t
}

// Then composes this transformer with subsequent pointer transformers.
func (t PointerTransformer[T]) Then(vs ...Transformer[*T]) Transformer[*T] {
	return sequenceFrom[*T](t, vs...)
}

// Transform runs the pointer pipeline with a background context.
func (t PointerTransformer[T]) Transform(v *T) (*T, error) {
	return t.TransformContext(context.Background(), v)
}

// TransformContext runs the pointer pipeline and observes cancellation.
func (t PointerTransformer[T]) TransformContext(ctx context.Context, v *T) (*T, error) {
	return t.value.TransformContext(ctx, v)
}

func (t PointerTransformer[T]) transformOwnedContext(ctx context.Context, v *T) (*T, error) {
	return t.value.transformOwnedContext(ctx, v)
}
