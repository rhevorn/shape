// Package shapehttp provides small net/http adapters for GoShape.
package shapehttp

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/rhevorn/shape"
)

// DefaultMaxBodyBytes is the default request-body limit used by DecodeJSON.
const DefaultMaxBodyBytes int64 = 1 << 20

// DefaultMaxResponseIssues is the maximum issue count emitted by
// WriteValidationError.
const DefaultMaxResponseIssues = shape.DefaultMaxIssues

// ErrBodyTooLarge is returned when a request exceeds the configured limit.
var ErrBodyTooLarge = errors.New("shapehttp: request body too large")

// DecodeJSON decodes and validates one JSON request body with a 1 MiB limit.
func DecodeJSON[T any](request *http.Request, schema shape.Schema[T]) (T, error) {
	return DecodeJSONLimit(request, schema, DefaultMaxBodyBytes)
}

// DecodeJSONLimit decodes and validates one JSON request body. The request
// context is propagated to the schema.
func DecodeJSONLimit[T any](request *http.Request, schema shape.Schema[T], maxBytes int64) (T, error) {
	var zero T
	if request == nil {
		return zero, errors.New("shapehttp: request must not be nil")
	}
	if request.Body == nil {
		return zero, errors.New("shapehttp: request body must not be nil")
	}
	if maxBytes < 0 {
		return zero, errors.New("shapehttp: max body bytes must not be negative")
	}
	const maxInt64 = int64(1<<63 - 1)
	reader := io.Reader(request.Body)
	if maxBytes != maxInt64 {
		reader = io.LimitReader(request.Body, maxBytes+1)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return zero, fmt.Errorf("shapehttp: read request body: %w", err)
	}
	if maxBytes != maxInt64 && int64(len(data)) > maxBytes {
		return zero, ErrBodyTooLarge
	}
	return shape.ParseJSONContext(request.Context(), schema, data)
}

// WriteValidationError writes err as a JSON issue response and returns true
// when err is a GoShape ValidationError. Other errors are left to the caller.
func WriteValidationError(response http.ResponseWriter, status int, err error) bool {
	return WriteValidationErrorLimit(response, status, err, DefaultMaxResponseIssues)
}

// WriteValidationErrorLimit writes at most maxIssues issues. When validation
// produced more, the final emitted issue reports truncation.
func WriteValidationErrorLimit(response http.ResponseWriter, status int, err error, maxIssues int) bool {
	if maxIssues <= 0 {
		panic("shapehttp: maximum response issues must be positive")
	}
	var validation *shape.ValidationError
	if !errors.As(err, &validation) || validation == nil {
		return false
	}
	issues := validation.Issues
	if len(issues) > maxIssues {
		issues = append([]shape.Issue(nil), issues[:maxIssues]...)
		issues[maxIssues-1] = shape.Issue{
			Code:    shape.CodeTooManyIssues,
			Message: "additional validation issues were omitted",
		}
	}
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(map[string]any{"issues": issues})
	return true
}
