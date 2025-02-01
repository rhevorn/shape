package shape

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

// ErrJSONTooLarge is returned when a reader exceeds an explicit byte limit.
var ErrJSONTooLarge = errors.New("shape: JSON input too large")

// jsonText is JSON input as either UTF-8 text or raw bytes.
type jsonText interface {
	~string | ~[]byte
}

// Parse decodes exactly one JSON value and validates it with schema.
// data may be a string or []byte. For already-decoded Go values, call
// schema.Parse instead.
//
// JSON numbers retain their lexical representation until a numeric schema
// handles them, avoiding float64 precision loss.
func Parse[T any, B jsonText](schema Schema[T], data B) (T, error) {
	return ParseContext(context.Background(), schema, data)
}

// ParseContext is Parse with context propagation.
func ParseContext[T any, B jsonText](ctx context.Context, schema Schema[T], data B) (T, error) {
	if schema == nil {
		panic("shape: JSON schema must not be nil")
	}
	return parseJSONReader(ctx, schema, openJSONText(data))
}

func openJSONText[B jsonText](data B) io.Reader {
	switch value := any(data).(type) {
	case string:
		return strings.NewReader(value)
	case []byte:
		return bytes.NewReader(value)
	default:
		return bytes.NewReader([]byte(data))
	}
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

// ParseReader reads, decodes, and parses exactly one JSON value.
func ParseReader[T any](schema Schema[T], reader io.Reader) (T, error) {
	return ParseReaderContext(context.Background(), schema, reader)
}

// ParseReaderContext is ParseReader with context propagation.
func ParseReaderContext[T any](ctx context.Context, schema Schema[T], reader io.Reader) (T, error) {
	var zero T
	if reader == nil {
		return zero, errors.New("shape: JSON reader must not be nil")
	}
	if schema == nil {
		panic("shape: JSON schema must not be nil")
	}
	return parseJSONReader(ctx, schema, reader)
}

// ParseReaderLimit reads, decodes, and parses exactly one JSON value while
// rejecting input streams larger than maxBytes.
func ParseReaderLimit[T any](schema Schema[T], reader io.Reader, maxBytes int64) (T, error) {
	return ParseReaderLimitContext(context.Background(), schema, reader, maxBytes)
}

// ParseReaderLimitContext is ParseReaderLimit with context propagation.
func ParseReaderLimitContext[T any](ctx context.Context, schema Schema[T], reader io.Reader, maxBytes int64) (T, error) {
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
