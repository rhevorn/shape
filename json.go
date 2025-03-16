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

func parseJSON[T any](schema Schema[T], source []byte, options ...JSONOptions) (T, error) {
	return parseJSONContext(context.Background(), schema, source, options...)
}

func parseJSONContext[T any](ctx context.Context, schema Schema[T], source []byte, options ...JSONOptions) (T, error) {
	return parseJSONReaderContext(ctx, schema, bytes.NewReader(source), options...)
}

func parseJSONReader[T any](schema Schema[T], reader io.Reader, options ...JSONOptions) (T, error) {
	return parseJSONReaderContext(context.Background(), schema, reader, options...)
}

func parseJSONReaderContext[T any](ctx context.Context, schema Schema[T], reader io.Reader, options ...JSONOptions) (T, error) {
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
	out, err := schema.ParseJSONContext(ctx, source, options...)
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
	out, err := schema.ParseJSONReaderContext(ctx, reader, options...)
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
	return parseJSON(s, source, options...)
}

// ParseJSONContext is ParseJSON with cancellation and per-request locale.
func (s structSchema[T]) ParseJSONContext(ctx context.Context, source []byte, options ...JSONOptions) (T, error) {
	return parseJSONContext(ctx, s, source, options...)
}

// ParseJSONReader decodes, transforms, and validates one value from reader.
func (s structSchema[T]) ParseJSONReader(reader io.Reader, options ...JSONOptions) (T, error) {
	return parseJSONReader(s, reader, options...)
}

// ParseJSONReaderContext is the context-aware reader form.
func (s structSchema[T]) ParseJSONReaderContext(ctx context.Context, reader io.Reader, options ...JSONOptions) (T, error) {
	return parseJSONReaderContext(ctx, s, reader, options...)
}

// BindJSON atomically replaces target only after decode, transform, and
// validation all succeed. The target comes first to make the mutation clear.
func (s structSchema[T]) BindJSON(target *T, source []byte, options ...JSONOptions) error {
	return bindJSON(s, target, source, options...)
}

// BindJSONContext is the context-aware form of BindJSON.
func (s structSchema[T]) BindJSONContext(ctx context.Context, target *T, source []byte, options ...JSONOptions) error {
	return bindJSONContext(ctx, s, target, source, options...)
}

// BindJSONReader atomically binds one JSON value read from reader.
func (s structSchema[T]) BindJSONReader(target *T, reader io.Reader, options ...JSONOptions) error {
	return bindJSONReader(s, target, reader, options...)
}

// BindJSONReaderContext is the context-aware reader bind form.
func (s structSchema[T]) BindJSONReaderContext(ctx context.Context, target *T, reader io.Reader, options ...JSONOptions) error {
	return bindJSONReaderContext(ctx, s, target, reader, options...)
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
