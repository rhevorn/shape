package transform

import (
	"context"
	"reflect"
)

// Transformer transforms a T without mutating caller-owned input.
type Transformer[T any] interface {
	Transform(T) (T, error)
	TransformContext(context.Context, T) (T, error)
}

// ownedTransformer is implemented by built-in transformers. The input passed
// to transformOwnedContext is already detached from caller-owned storage.
type ownedTransformer[T any] interface {
	transformerType() reflect.Type
	transformOwnedContext(context.Context, T) (T, error)
}

func runOwned[T any](ctx context.Context, transformer Transformer[T], value T) (T, error) {
	return ownedStep(transformer)(ctx, value)
}

func ownedStep[T any](transformer Transformer[T]) step[T] {
	if owned, ok := builtIn(transformer); ok {
		return owned.transformOwnedContext
	}
	return func(ctx context.Context, value T) (T, error) {
		out, err := transformer.TransformContext(ctx, value)
		if err != nil {
			return out, err
		}
		// External implementations may return storage they still own.
		return cloneContext(ctx, out)
	}
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
	owned, err := cloneContext(ctx, v)
	if err != nil {
		return zero, err
	}
	return s.transformOwnedContext(ctx, owned)
}

func (s sequence[T]) transformOwnedContext(ctx context.Context, v T) (T, error) {
	var zero T
	for _, current := range s.values {
		if err := ctx.Err(); err != nil {
			return zero, err
		}
		var err error
		v, err = runOwned(ctx, current, v)
		if ctx.Err() != nil {
			return zero, ctx.Err()
		}
		if err != nil {
			return zero, err
		}
	}
	return v, nil
}
func sequenceFrom[T any](first Transformer[T], rest ...Transformer[T]) Transformer[T] {
	items := make([]Transformer[T], 0, len(rest)+1)
	items = appendSequence(items, first)
	for _, v := range rest {
		if v == nil {
			panic("transform: nil transformer")
		}
		items = appendSequence(items, v)
	}
	return sequence[T]{items}
}

func appendSequence[T any](items []Transformer[T], transformer Transformer[T]) []Transformer[T] {
	if nested, ok := transformer.(sequence[T]); ok {
		return append(items, nested.values...)
	}
	return append(items, transformer)
}

// runComposite detaches once, runs outer steps, then transforms the resulting elements.
func runComposite[T any](ctx context.Context, outer ValueTransformer[T], elements step[T], value T, owned bool) (T, error) {
	var zero T
	if ctx == nil {
		panic("transform: nil context")
	}
	if elements == nil {
		panic("transform: composite must be built with Pointer, Slice or Map")
	}
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	if !owned {
		var err error
		value, err = cloneContext(ctx, value)
		if err != nil {
			return zero, err
		}
	}
	out, err := outer.transformOwnedContext(ctx, value)
	if err != nil {
		return zero, err
	}
	out, err = elements(ctx, out)
	if ctx.Err() != nil {
		return zero, ctx.Err()
	}
	if err != nil {
		return zero, err
	}
	return out, nil
}
