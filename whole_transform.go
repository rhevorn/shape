package shape

import (
	"context"
	"reflect"
)

type wholeTransformStep[T any] func(context.Context, T) (T, error)

func appendWholeApply[T any](current []wholeTransformStep[T], steps ...func(T) (T, error)) []wholeTransformStep[T] {
	if len(steps) == 0 {
		return current
	}
	out := append([]wholeTransformStep[T](nil), current...)
	for _, step := range steps {
		if step == nil {
			panic("shape: nil Apply function")
		}
		out = append(out, func(_ context.Context, value T) (T, error) {
			return step(value)
		})
	}
	return out
}

func appendWholeApplyContext[T any](current []wholeTransformStep[T], steps ...func(context.Context, T) (T, error)) []wholeTransformStep[T] {
	if len(steps) == 0 {
		return current
	}
	out := append([]wholeTransformStep[T](nil), current...)
	for _, step := range steps {
		if step == nil {
			panic("shape: nil ApplyContext function")
		}
		out = append(out, step)
	}
	return out
}

func runWholeTransforms[T any](ctx context.Context, steps []wholeTransformStep[T], value T) (T, error) {
	var zero T
	for _, step := range steps {
		if err := ctx.Err(); err != nil {
			return zero, err
		}
		var err error
		value, err = step(ctx, value)
		if ctx.Err() != nil {
			return zero, ctx.Err()
		}
		if err != nil {
			return zero, err
		}
	}
	return value, nil
}

// applyWholeTransforms runs the whole-struct steps that follow a base
// transform, detaching the value first unless the caller already handed us a
// private copy.
//
// This is the single policy both Schema types use. They used to implement it
// separately, which is how the tagged path came to skip the detach step: its
// compiled plan only covers the fields it walks, so json:"-" and unexported
// storage reaches a whole-struct Apply still aliased to the caller.
func applyWholeTransforms[T any](
	ctx context.Context,
	transforms []wholeTransformStep[T],
	value T,
	owned bool,
) (T, error) {
	var zero T
	if len(transforms) == 0 {
		return value, nil
	}
	if !owned {
		detached, err := cloneWholeValue(ctx, value)
		if err != nil {
			return zero, normalizeTransformError(ctx, err)
		}
		value = detached
	}
	out, err := runWholeTransforms(ctx, transforms, value)
	if err != nil {
		return zero, normalizeTransformError(ctx, err)
	}
	return out, nil
}

func cloneWholeValue[T any](ctx context.Context, value T) (T, error) {
	var zero T
	out, err := cloneValue(ctx, reflect.ValueOf(&value).Elem(), false)
	if err != nil {
		return zero, err
	}
	return out.Interface().(T), nil
}
