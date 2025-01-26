package shape

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// ErrJSONTooLarge is returned when a reader exceeds an explicit byte limit.
var ErrJSONTooLarge = errors.New("shape: JSON input too large")

// ParseJSON decodes exactly one JSON value and parses it with schema. JSON
// numbers retain their lexical representation until a numeric schema handles
// them, avoiding float64 precision loss.
func ParseJSON[T any](schema Schema[T], data []byte) (T, error) {
	return ParseJSONContext(context.Background(), schema, data)
}

// ParseJSONContext is ParseJSON with context propagation.
func ParseJSONContext[T any](ctx context.Context, schema Schema[T], data []byte) (T, error) {
	if schema == nil {
		panic("shape: JSON schema must not be nil")
	}
	return parseJSONReader(ctx, schema, bytes.NewReader(data))
}

func parseJSONReader[T any](ctx context.Context, schema Schema[T], reader io.Reader) (T, error) {
	var zero T
	decoder := json.NewDecoder(reader)
	decoder.UseNumber()
	var input any
	if err := decoder.Decode(&input); err != nil {
		return zero, fmt.Errorf("shape: decode JSON: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return zero, fmt.Errorf("shape: decode JSON: expected exactly one value")
		}
		return zero, fmt.Errorf("shape: decode trailing JSON: %w", err)
	}

	return schema.ParseContext(ctx, input)
}

// ParseJSONReader reads, decodes, and parses exactly one JSON value.
func ParseJSONReader[T any](schema Schema[T], reader io.Reader) (T, error) {
	return ParseJSONReaderContext(context.Background(), schema, reader)
}

// ParseJSONReaderContext is ParseJSONReader with context propagation.
func ParseJSONReaderContext[T any](ctx context.Context, schema Schema[T], reader io.Reader) (T, error) {
	var zero T
	if reader == nil {
		return zero, errors.New("shape: JSON reader must not be nil")
	}
	if schema == nil {
		panic("shape: JSON schema must not be nil")
	}
	return parseJSONReader(ctx, schema, reader)
}

// ParseJSONReaderLimit reads, decodes, and parses exactly one JSON value while
// rejecting input streams larger than maxBytes.
func ParseJSONReaderLimit[T any](schema Schema[T], reader io.Reader, maxBytes int64) (T, error) {
	return ParseJSONReaderLimitContext(context.Background(), schema, reader, maxBytes)
}

// ParseJSONReaderLimitContext is ParseJSONReaderLimit with context propagation.
func ParseJSONReaderLimitContext[T any](ctx context.Context, schema Schema[T], reader io.Reader, maxBytes int64) (T, error) {
	var zero T
	if reader == nil {
		return zero, errors.New("shape: JSON reader must not be nil")
	}
	if maxBytes < 0 {
		return zero, errors.New("shape: max JSON bytes must not be negative")
	}
	if schema == nil {
		panic("shape: JSON schema must not be nil")
	}
	const maxInt64 = int64(1<<63 - 1)
	if maxBytes == maxInt64 {
		return parseJSONReader(ctx, schema, reader)
	}
	limited := &io.LimitedReader{R: reader, N: maxBytes + 1}
	parsed, err := parseJSONReader(ctx, schema, limited)
	if limited.N == 0 {
		return zero, ErrJSONTooLarge
	}
	return parsed, err
}
