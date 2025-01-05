package goshape

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// ParseJSON decodes exactly one JSON value and parses it with schema. JSON
// numbers retain their lexical representation until a numeric schema handles
// them, avoiding float64 precision loss.
func ParseJSON[T any](schema Schema[T], data []byte) (T, error) {
	return ParseJSONContext(context.Background(), schema, data)
}

// ParseJSONContext is ParseJSON with context propagation.
func ParseJSONContext[T any](ctx context.Context, schema Schema[T], data []byte) (T, error) {
	var zero T
	if schema == nil {
		panic("goshape: JSON schema must not be nil")
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var input any
	if err := decoder.Decode(&input); err != nil {
		return zero, fmt.Errorf("goshape: decode JSON: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return zero, fmt.Errorf("goshape: decode JSON: expected exactly one value")
		}
		return zero, fmt.Errorf("goshape: decode trailing JSON: %w", err)
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
		return zero, errors.New("goshape: JSON reader must not be nil")
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return zero, fmt.Errorf("goshape: read JSON: %w", err)
	}
	return ParseJSONContext(ctx, schema, data)
}
