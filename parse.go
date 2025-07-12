package shape

import "context"

// finishParse preserves public overrides and commits no result on failure.
func finishParse[T any](ctx context.Context, schema Schema[T], candidate T, owned bool) (T, error) {
	var zero T
	var err error
	var out T
	// Only exact built-in types may bypass the public transformation method.
	switch builtIn := schema.(type) {
	case TaggedSpec[T]:
		out, err = builtIn.transformContext(ctx, candidate, owned)
	case *TaggedSpec[T]:
		out, err = builtIn.transformContext(ctx, candidate, owned)
	default:
		out, err = schema.TransformContext(ctx, candidate)
	}

	if ctx.Err() != nil {
		return zero, ctx.Err()
	}
	if err != nil {
		return zero, err
	}
	err = schema.ValidateContext(ctx, out)
	if ctx.Err() != nil {
		return zero, ctx.Err()
	}
	if err != nil {
		return zero, err
	}
	return out, nil
}
