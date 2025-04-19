package shape

import (
	"bytes"
	"context"
	"errors"
	"io"

	"github.com/rhevorn/shape/internal/jsondecode"
)

// ErrJSONTooLarge reports that JSON input exceeded JSONOptions.MaxBytes.
var ErrJSONTooLarge = errors.New("shape: JSON input too large")

type decodedTransformer[T any] interface {
	transformDecodedContext(context.Context, T) (T, error)
}

// JSONOptions controls the standard JSON decoding stage. MaxBytes zero means
// unlimited; a positive value rejects larger input with ErrJSONTooLarge.
type JSONOptions struct {
	DisallowUnknownFields bool
	MaxBytes              int64
}

// ParseJSON decodes, transforms, and validates one JSON value with schema.
// Use this for scalar or composite Schema roots. Prefer StructSpec/TaggedSpec
// methods when the Schema is already a named struct Schema.
func ParseJSON[T any](schema Schema[T], source []byte, options ...JSONOptions) (T, error) {
	return ParseJSONContext(context.Background(), schema, source, options...)
}

// ParseJSONContext is ParseJSON with cancellation and per-request locale.
func ParseJSONContext[T any](ctx context.Context, schema Schema[T], source []byte, options ...JSONOptions) (T, error) {
	return ParseJSONReaderContext(ctx, schema, bytes.NewReader(source), options...)
}

// ParseJSONReader decodes, transforms, and validates one value from reader.
func ParseJSONReader[T any](schema Schema[T], reader io.Reader, options ...JSONOptions) (T, error) {
	return ParseJSONReaderContext(context.Background(), schema, reader, options...)
}

// ParseJSONReaderContext is the context-aware reader form of ParseJSON.
func ParseJSONReaderContext[T any](ctx context.Context, schema Schema[T], reader io.Reader, options ...JSONOptions) (T, error) {
	var zero T
	if ctx == nil {
		panic("shape: nil context")
	}
	if len(options) > 1 {
		panic("shape: at most one JSONOptions")
	}
	var opts JSONOptions
	if len(options) == 1 {
		opts = options[0]
	}
	candidate, err := jsondecode.Decode[T](ctx, reader, jsondecode.Options{
		DisallowUnknownFields: opts.DisallowUnknownFields,
		MaxBytes:              opts.MaxBytes,
		TooLargeError:         ErrJSONTooLarge,
	})
	if err != nil {
		return zero, err
	}
	var out T
	if decoded, ok := schema.(decodedTransformer[T]); ok {
		out, err = decoded.transformDecodedContext(ctx, candidate)
	} else {
		out, err = schema.TransformContext(ctx, candidate)
	}
	if ctx.Err() != nil {
		return zero, ctx.Err()
	}
	if err != nil {
		return zero, err
	}
	if err := schema.ValidateContext(ctx, out); err != nil {
		return zero, err
	}
	return out, nil
}

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

func bindJSON[T any](schema Schema[T], target *T, source []byte, options ...JSONOptions) error {
	return bindJSONContext(context.Background(), schema, target, source, options...)
}

func bindJSONContext[T any](ctx context.Context, schema Schema[T], target *T, source []byte, options ...JSONOptions) error {
	if ctx == nil {
		panic("shape: nil context")
	}
	if target == nil {
		return errors.New("shape: nil bind target")
	}
	out, err := ParseJSONContext(ctx, schema, source, options...)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	*target = out
	return nil
}

func bindJSONReader[T any](schema Schema[T], target *T, reader io.Reader, options ...JSONOptions) error {
	return bindJSONReaderContext(context.Background(), schema, target, reader, options...)
}

func bindJSONReaderContext[T any](ctx context.Context, schema Schema[T], target *T, reader io.Reader, options ...JSONOptions) error {
	if ctx == nil {
		panic("shape: nil context")
	}
	if target == nil {
		return errors.New("shape: nil bind target")
	}
	out, err := ParseJSONReaderContext(ctx, schema, reader, options...)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	*target = out
	return nil
}
