package transform

import "context"

type Transformer[T any] interface {
	Transform(T) (T, error)
	TransformContext(context.Context, T) (T, error)
}
type sequence[T any] struct{ values []Transformer[T] }

func (s sequence[T]) Transform(v T) (T, error) {
	return s.TransformContext(context.Background(), v)
}
func (s sequence[T]) TransformContext(ctx context.Context, v T) (T, error) {
	var zero T
	if ctx == nil {
		panic("transform: nil context")
	}
	for _, current := range s.values {
		var err error
		v, err = current.TransformContext(ctx, v)
		if err != nil {
			return zero, err
		}
	}
	return v, nil
}
func sequenceFrom[T any](first Transformer[T], rest ...Transformer[T]) Transformer[T] {
	items := []Transformer[T]{first}
	for _, v := range rest {
		if v == nil {
			panic("transform: nil transformer")
		}
		items = append(items, v)
	}
	return sequence[T]{items}
}
