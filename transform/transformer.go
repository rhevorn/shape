package transform

import "context"

type Transformer[T any] interface {
	Transform(T) (T, error)
	TransformContext(context.Context, T) (T, error)
}

// ownedTransformer is implemented by built-in transformers. The input passed
// to transformOwnedContext is already detached from caller-owned storage.
type ownedTransformer[T any] interface {
	transformOwnedContext(context.Context, T) (T, error)
}

func runOwned[T any](ctx context.Context, transformer Transformer[T], value T) (T, error) {
	if owned, ok := transformer.(ownedTransformer[T]); ok {
		return owned.transformOwnedContext(ctx, value)
	}
	// A foreign transformer detaches its input (its TransformContext clones)
	// but promises nothing about what it returns: it may hand back storage it
	// still holds. The rest of the pipeline treats the result as private, so
	// detach it here rather than let an alias escape.
	out, err := transformer.TransformContext(ctx, value)
	if err != nil {
		return out, err
	}
	return cloneContext(ctx, out)
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
