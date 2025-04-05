package transform

import (
	"context"
	"reflect"

	"github.com/rhevorn/shape/internal/reflectclone"
)

// ValueTransformer is an immutable transformation pipeline for any T.
type ValueTransformer[T any] struct{ steps []step[T] }

func (t ValueTransformer[T]) appendSteps(steps ...step[T]) ValueTransformer[T] {
	t.steps = append(append([]step[T](nil), t.steps...), steps...)
	return t
}

// Transform runs the pipeline with a background context.
func (t ValueTransformer[T]) Transform(v T) (T, error) {
	return t.TransformContext(context.Background(), v)
}

// TransformContext runs the pipeline and observes context cancellation.
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

// IfZero replaces the zero value with an isolated snapshot of fallback.
func (t ValueTransformer[T]) IfZero(fallback T) ValueTransformer[T] {
	snapshot, err := cloneStrict(fallback)
	if err != nil {
		panic("transform: invalid IfZero value: " + err.Error())
	}
	return t.appendSteps(func(ctx context.Context, v T) (T, error) {
		// reflectclone.IsZero, not reflect.Value.IsZero: the two disagree for a
		// zero time.Time carrying a non-nil Location.
		if !reflectclone.IsZero(reflect.ValueOf(&v).Elem()) {
			return v, nil
		}
		return cloneContext(ctx, snapshot)
	})
}

// Apply appends custom transformation steps in declaration order.
func (t ValueTransformer[T]) Apply(fns ...func(T) (T, error)) ValueTransformer[T] {
	for _, fn := range fns {
		if fn == nil {
			panic("transform: nil function")
		}
		t = t.appendSteps(function(fn))
	}
	return t
}

// ApplyContext appends context-aware custom transformation steps.
func (t ValueTransformer[T]) ApplyContext(fns ...func(context.Context, T) (T, error)) ValueTransformer[T] {
	for _, fn := range fns {
		if fn == nil {
			panic("transform: nil function")
		}
		t = t.appendSteps(functionContext(fn))
	}
	return t
}

// Then composes this transformer with subsequent transformers.
func (t ValueTransformer[T]) Then(values ...Transformer[T]) Transformer[T] {
	return sequenceFrom[T](t, values...)
}
