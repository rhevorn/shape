package shape

import (
	"context"
	"io"
)

// BindJSON derives the target's cached tagged struct Schema and atomically
// replaces *target only after decode, transform, and validation all succeed.
// No Schema variable is required; field behavior comes from shape tags.
func BindJSON[T any](target *T, source []byte, options ...JSONOptions) error {
	return bindJSON(Struct[T](), target, source, options...)
}

// BindJSONContext is the context-aware form of BindJSON.
func BindJSONContext[T any](ctx context.Context, target *T, source []byte, options ...JSONOptions) error {
	return bindJSONContext(ctx, Struct[T](), target, source, options...)
}

// BindJSONReader derives the target's cached tagged struct Schema and
// atomically binds one JSON value from reader.
func BindJSONReader[T any](target *T, reader io.Reader, options ...JSONOptions) error {
	return bindJSONReader(Struct[T](), target, reader, options...)
}

// BindJSONReaderContext is the context-aware reader form of BindJSON.
func BindJSONReaderContext[T any](ctx context.Context, target *T, reader io.Reader, options ...JSONOptions) error {
	return bindJSONReaderContext(ctx, Struct[T](), target, reader, options...)
}
