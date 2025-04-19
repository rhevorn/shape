package pipeline

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/rhevorn/shape/internal/reflectclone"
)

// Step transforms an entire schema value.
type Step[T any] func(context.Context, T) (T, error)

func Append[T any](current []Step[T], steps ...func(T) (T, error)) []Step[T] {
	if len(steps) == 0 {
		return current
	}
	out := append([]Step[T](nil), current...)
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

func AppendContext[T any](current []Step[T], steps ...func(context.Context, T) (T, error)) []Step[T] {
	if len(steps) == 0 {
		return current
	}
	out := append([]Step[T](nil), current...)
	for _, step := range steps {
		if step == nil {
			panic("shape: nil ApplyContext function")
		}
		out = append(out, step)
	}
	return out
}

func run[T any](ctx context.Context, steps []Step[T], value T) (T, error) {
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
// private copy. Both Schema implementations use this ownership policy.
func Apply[T any](
	ctx context.Context,
	transforms []Step[T],
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
			return zero, cloneError(err)
		}
		value = detached
	}
	out, err := run(ctx, transforms, value)
	if err != nil {
		return zero, err
	}
	return out, nil
}

func cloneWholeValue[T any](ctx context.Context, value T) (T, error) {
	var zero T
	out, err := reflectclone.Clone(ctx, reflect.ValueOf(&value).Elem(), false)
	if err != nil {
		return zero, err
	}
	return out.Interface().(T), nil
}

func cloneError(err error) error {
	switch {
	case errors.Is(err, reflectclone.ErrDepthExceeded):
		return errors.New("shape: copy depth exceeded (cyclic or deeply nested value)")
	case errors.Is(err, reflectclone.ErrUnsupportedType):
		return fmt.Errorf("shape: unsupported default type: %v", err)
	default:
		return err
	}
}
