package transform

import (
	"context"
	"reflect"

	"github.com/rhevorn/shape/internal/reflectclone"
)

type ValueTransformer[T any] struct{ steps []step[T] }

func (t ValueTransformer[T]) append(steps ...step[T]) ValueTransformer[T] {
	t.steps = append(append([]step[T](nil), t.steps...), steps...)
	return t
}

func (t ValueTransformer[T]) Transform(v T) (T, error) {
	return t.TransformContext(context.Background(), v)
}

func (t ValueTransformer[T]) TransformContext(ctx context.Context, v T) (T, error) {
	if ctx == nil {
		panic("transform: nil context")
	}
	var zero T
	owned, err := cloneContext(ctx, v)
	if err != nil {
		return zero, err
	}
	return t.transformOwnedContext(ctx, owned)
}

func (t ValueTransformer[T]) transformOwnedContext(ctx context.Context, v T) (T, error) {
	var zero T
	for _, step := range t.steps {
		if err := ctx.Err(); err != nil {
			return zero, err
		}
		var err error
		v, err = step(ctx, v)
		if ctx.Err() != nil {
			return zero, ctx.Err()
		}
		if err != nil {
			return zero, err
		}
	}
	return v, nil
}

func (t ValueTransformer[T]) IfZero(fallback T) ValueTransformer[T] {
	snapshot, err := clone(fallback)
	if err != nil {
		panic("transform: invalid IfZero value: " + err.Error())
	}
	return t.append(func(ctx context.Context, v T) (T, error) {
		// reflectclone.IsZero, not reflect.Value.IsZero: the two disagree for a
		// zero time.Time carrying a non-nil Location.
		if !reflectclone.IsZero(reflect.ValueOf(&v).Elem()) {
			return v, nil
		}
		return cloneContext(ctx, snapshot)
	})
}

func (t ValueTransformer[T]) Apply(fns ...func(T) (T, error)) ValueTransformer[T] {
	for _, fn := range fns {
		if fn == nil {
			panic("transform: nil function")
		}
		t = t.append(function(fn))
	}
	return t
}

func (t ValueTransformer[T]) ApplyContext(fns ...func(context.Context, T) (T, error)) ValueTransformer[T] {
	for _, fn := range fns {
		if fn == nil {
			panic("transform: nil function")
		}
		t = t.append(functionContext(fn))
	}
	return t
}

func (t ValueTransformer[T]) Then(values ...Transformer[T]) Transformer[T] {
	return sequenceFrom[T](t, values...)
}
