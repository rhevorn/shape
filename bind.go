package shape

import (
	"context"
	"io"
)

// BindJSON derives the target's cached tagged struct Schema and atomically
// binds one JSON value.
func BindJSON[T any](target *T, source []byte, options ...JSONOptions) error {
	return Struct[T]().BindJSON(target, source, options...)
}

// BindJSONContext is the context-aware form of BindJSON.
func BindJSONContext[T any](ctx context.Context, target *T, source []byte, options ...JSONOptions) error {
	return Struct[T]().BindJSONContext(ctx, target, source, options...)
}

// BindJSONReader derives the target's cached tagged struct Schema and
// atomically binds one JSON value from reader.
func BindJSONReader[T any](target *T, reader io.Reader, options ...JSONOptions) error {
	return Struct[T]().BindJSONReader(target, reader, options...)
}

// BindJSONReaderContext is the context-aware reader form of BindJSON.
func BindJSONReaderContext[T any](ctx context.Context, target *T, reader io.Reader, options ...JSONOptions) error {
	return Struct[T]().BindJSONReaderContext(ctx, target, reader, options...)
}
