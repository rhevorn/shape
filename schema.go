package shape

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/rhevorn/shape/internal/program"
	"github.com/rhevorn/shape/internal/tagged"
	"github.com/rhevorn/shape/internal/transformpath"
	"github.com/rhevorn/shape/validate"
)

// Schema describes transformation and validation for T.
// Transformation and validation remain independently callable.
// JSON decode lives on StructSpec/TaggedSpec (JSONSchema) and package-level
// ParseJSON*; tag-driven BindJSON* is package-level only.
type Schema[T any] interface {
	Transform(T) (T, error)
	TransformContext(context.Context, T) (T, error)
	Validate(T) error
	ValidateContext(context.Context, T) error
	ValidateFirst(T) error
	ValidateFirstContext(context.Context, T) error
}

// JSONSchema is the struct Schema surface that owns ParseJSON.
// Scalar and composite Specs implement Schema only; use package-level ParseJSON*
// when a non-struct Schema is the JSON root. For tag-driven in-place bind, use
// package-level BindJSON* — it needs no Schema variable.
type JSONSchema[T any] interface {
	Schema[T]

	ParseJSON([]byte, ...JSONOptions) (T, error)
	ParseJSONContext(context.Context, []byte, ...JSONOptions) (T, error)
	ParseJSONReader(io.Reader, ...JSONOptions) (T, error)
	ParseJSONReaderContext(context.Context, io.Reader, ...JSONOptions) (T, error)
}

// TransformError reports the struct field or collection element whose
// transform failed. Unwrap returns the original transform error.
type TransformError struct {
	Path validate.Path
	Err  error
}

// Error formats the failed path and underlying error.
func (e *TransformError) Error() string {
	if e == nil || e.Err == nil {
		return "shape: transform failed"
	}
	if len(e.Path) == 0 {
		return "shape: transform failed: " + e.Err.Error()
	}
	return fmt.Sprintf("shape: transform failed at %s: %v", e.Path.String(), e.Err)
}

// Unwrap returns the underlying transform error.
func (e *TransformError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func normalizeTransformError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	path, cause := transformErrorParts(err)
	return &TransformError{Path: path, Err: cause}
}

func transformErrorParts(err error) (validate.Path, error) {
	var programError *program.TransformError
	if errors.As(err, &programError) && programError != nil {
		path, cause := transformErrorParts(programError.Err)
		return appendPath(programError.Path, path), cause
	}
	var taggedError *tagged.TransformError
	if errors.As(err, &taggedError) && taggedError != nil {
		path, cause := transformErrorParts(taggedError.Err)
		return appendPath(taggedError.Path, path), cause
	}
	var internal *transformpath.Error
	if errors.As(err, &internal) && internal != nil && len(internal.Segments) > 0 {
		path := make(validate.Path, 0, len(internal.Segments))
		for _, segment := range internal.Segments {
			if segment.IsIndex {
				path = append(path, validate.IndexPath(segment.Index))
			} else {
				path = append(path, validate.MapKeyPath(segment.Key))
			}
		}
		nested, cause := transformErrorParts(internal.Err)
		return appendPath(path, nested), cause
	}
	var public *TransformError
	if errors.As(err, &public) && public != nil {
		path, cause := transformErrorParts(public.Err)
		return appendPath(public.Path, path), cause
	}
	return nil, err
}

func appendPath(prefix, suffix validate.Path) validate.Path {
	out := make(validate.Path, 0, len(prefix)+len(suffix))
	out = append(out, prefix...)
	out = append(out, suffix...)
	return out
}
