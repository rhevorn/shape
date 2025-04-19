// Package jsondecode owns Shape's standard JSON input stage.
package jsondecode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// Options controls decoding behavior.
type Options struct {
	DisallowUnknownFields bool
	MaxBytes              int64
	TooLargeError         error
}

// Decode reads exactly one JSON value while observing ctx.
func Decode[T any](ctx context.Context, reader io.Reader, options Options) (T, error) {
	var zero T
	if ctx == nil {
		panic("shape: nil context")
	}
	if reader == nil {
		return zero, errors.New("shape: nil JSON reader")
	}
	if options.MaxBytes < 0 {
		return zero, errors.New("shape: MaxBytes must not be negative")
	}
	if options.MaxBytes > 0 {
		reader = &limitedReader{
			reader: reader, remaining: options.MaxBytes, tooLargeError: options.TooLargeError,
		}
	}
	if err := ctx.Err(); err != nil {
		return zero, err
	}

	decoder := json.NewDecoder(contextReader{ctx: ctx, reader: reader})
	if options.DisallowUnknownFields {
		decoder.DisallowUnknownFields()
	}
	var value T
	if err := decoder.Decode(&value); err != nil {
		if ctx.Err() != nil {
			return zero, ctx.Err()
		}
		return zero, fmt.Errorf("shape: decode JSON: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if ctx.Err() != nil {
			return zero, ctx.Err()
		}
		if err == nil {
			return zero, errors.New("shape: expected exactly one JSON value")
		}
		return zero, fmt.Errorf("shape: trailing JSON: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	return value, nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	n, err := r.reader.Read(p)
	if contextErr := r.ctx.Err(); contextErr != nil {
		return n, contextErr
	}
	return n, err
}

type limitedReader struct {
	reader        io.Reader
	remaining     int64
	tooLargeError error
}

func (r *limitedReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if r.remaining == 0 {
		var one [1]byte
		n, err := r.reader.Read(one[:])
		if n > 0 {
			if r.tooLargeError != nil {
				return 0, r.tooLargeError
			}
			return 0, errors.New("shape: JSON input too large")
		}
		return 0, err
	}
	if int64(len(p)) > r.remaining {
		p = p[:r.remaining]
	}
	n, err := r.reader.Read(p)
	r.remaining -= int64(n)
	return n, err
}
