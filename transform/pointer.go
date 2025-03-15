package transform

import "context"

type PointerTransformer[T any] struct {
	value ValueTransformer[*T]
}

func (t PointerTransformer[T]) IfNull(v *T) PointerTransformer[T] {
	snapshot, err := clone(v)
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
func (t PointerTransformer[T]) Apply(fns ...func(*T) (*T, error)) PointerTransformer[T] {
	t.value = t.value.Apply(fns...)
	return t
}
func (t PointerTransformer[T]) ApplyContext(fns ...func(context.Context, *T) (*T, error)) PointerTransformer[T] {
	t.value = t.value.ApplyContext(fns...)
	return t
}
func (t PointerTransformer[T]) Then(vs ...Transformer[*T]) Transformer[*T] {
	return sequenceFrom[*T](t, vs...)
}
func (t PointerTransformer[T]) Transform(v *T) (*T, error) {
	return t.TransformContext(context.Background(), v)
}
func (t PointerTransformer[T]) TransformContext(ctx context.Context, v *T) (*T, error) {
	return t.value.TransformContext(ctx, v)
}

func (t PointerTransformer[T]) transformOwnedContext(ctx context.Context, v *T) (*T, error) {
	return t.value.transformOwnedContext(ctx, v)
}
