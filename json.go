package shape

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	if reader == nil {
		return zero, errors.New("shape: nil JSON reader")
	}
	if len(options) > 1 {
		panic("shape: at most one JSONOptions")
	}
	var opts JSONOptions
	if len(options) == 1 {
		opts = options[0]
	}
	if opts.MaxBytes < 0 {
		return zero, errors.New("shape: MaxBytes must not be negative")
	}
	if opts.MaxBytes > 0 {
		reader = &limitedJSONReader{r: reader, remaining: opts.MaxBytes}
	}
	if e := ctx.Err(); e != nil {
		return zero, e
	}
	decoder := json.NewDecoder(contextReader{ctx, reader})
	if opts.DisallowUnknownFields {
		decoder.DisallowUnknownFields()
	}
	var candidate T
	if e := decoder.Decode(&candidate); e != nil {
		if ctx.Err() != nil {
			return zero, ctx.Err()
		}
		return zero, fmt.Errorf("shape: decode JSON: %w", e)
	}
	var extra any
	if e := decoder.Decode(&extra); e != io.EOF {
		if ctx.Err() != nil {
			return zero, ctx.Err()
		}
		if e == nil {
			return zero, errors.New("shape: expected exactly one JSON value")
		}
		return zero, fmt.Errorf("shape: trailing JSON: %w", e)
	}
	if e := ctx.Err(); e != nil {
		return zero, e
	}
	var out T
	var err error
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

// ParseJSON decodes, transforms, and validates one JSON value.
func (s structSchema[T]) ParseJSON(source []byte, options ...JSONOptions) (T, error) {
	return ParseJSON(s, source, options...)
}

// ParseJSONContext is ParseJSON with cancellation and per-request locale.
func (s structSchema[T]) ParseJSONContext(ctx context.Context, source []byte, options ...JSONOptions) (T, error) {
	return ParseJSONContext(ctx, s, source, options...)
}

// ParseJSONReader decodes, transforms, and validates one value from reader.
func (s structSchema[T]) ParseJSONReader(reader io.Reader, options ...JSONOptions) (T, error) {
	return ParseJSONReader(s, reader, options...)
}

// ParseJSONReaderContext is the context-aware reader form.
func (s structSchema[T]) ParseJSONReaderContext(ctx context.Context, reader io.Reader, options ...JSONOptions) (T, error) {
	return ParseJSONReaderContext(ctx, s, reader, options...)
}

type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if e := r.ctx.Err(); e != nil {
		return 0, e
	}
	n, e := r.r.Read(p)
	if ce := r.ctx.Err(); ce != nil {
		return n, ce
	}
	return n, e
}

type limitedJSONReader struct {
	r         io.Reader
	remaining int64
}

func (r *limitedJSONReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if r.remaining == 0 {
		var b [1]byte
		n, e := r.r.Read(b[:])
		if n > 0 {
			return 0, ErrJSONTooLarge
		}
		return 0, e
	}
	if int64(len(p)) > r.remaining {
		p = p[:r.remaining]
	}
	n, e := r.r.Read(p)
	r.remaining -= int64(n)
	return n, e
}
