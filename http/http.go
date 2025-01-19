// Package goshapehttp provides small net/http adapters for GoShape.
package goshapehttp

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/rhevorn/goshape"
)

// DefaultMaxBodyBytes is the default request-body limit used by DecodeJSON.
const DefaultMaxBodyBytes int64 = 1 << 20

// ErrBodyTooLarge is returned when a request exceeds the configured limit.
var ErrBodyTooLarge = errors.New("goshapehttp: request body too large")

// DecodeJSON decodes and validates one JSON request body with a 1 MiB limit.
func DecodeJSON[T any](request *http.Request, schema goshape.Schema[T]) (T, error) {
	return DecodeJSONLimit(request, schema, DefaultMaxBodyBytes)
}

// DecodeJSONLimit decodes and validates one JSON request body. The request
// context is propagated to the schema.
func DecodeJSONLimit[T any](request *http.Request, schema goshape.Schema[T], maxBytes int64) (T, error) {
	var zero T
	if request == nil {
		return zero, errors.New("goshapehttp: request must not be nil")
	}
	if request.Body == nil {
		return zero, errors.New("goshapehttp: request body must not be nil")
	}
	if maxBytes < 0 {
		return zero, errors.New("goshapehttp: max body bytes must not be negative")
	}
	const maxInt64 = int64(1<<63 - 1)
	reader := io.Reader(request.Body)
	if maxBytes != maxInt64 {
		reader = io.LimitReader(request.Body, maxBytes+1)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return zero, fmt.Errorf("goshapehttp: read request body: %w", err)
	}
	if maxBytes != maxInt64 && int64(len(data)) > maxBytes {
		return zero, ErrBodyTooLarge
	}
	return goshape.ParseJSONContext(request.Context(), schema, data)
}

// WriteValidationError writes err as a JSON issue response and returns true
// when err is a GoShape ValidationError. Other errors are left to the caller.
func WriteValidationError(response http.ResponseWriter, status int, err error) bool {
	var validation *goshape.ValidationError
	if !errors.As(err, &validation) {
		return false
	}
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(map[string]any{"issues": validation.Issues})
	return true
}
