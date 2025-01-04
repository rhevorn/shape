package goshape

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// ParseJSON decodes exactly one JSON value and parses it with schema. JSON
// numbers retain their lexical representation until a numeric schema handles
// them, avoiding float64 precision loss.
func ParseJSON[T any](schema Schema[T], data []byte) (T, error) {
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

	return schema.Parse(input)
}
