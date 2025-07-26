package shape

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/rhevorn/shape/internal/fieldmeta"
	"github.com/rhevorn/shape/internal/valuesdecode"
	"github.com/rhevorn/shape/validate"
)

// FormOptions controls form parameter decoding. Unknown fields are ignored by default.
type FormOptions struct {
	DisallowUnknownFields bool
}

// QueryOptions controls query parameter decoding. Unknown fields are ignored by default.
type QueryOptions struct {
	DisallowUnknownFields bool
}

// ParameterError identifies a form or query parameter that could not be decoded.
// Source is "form" or "query". Path uses that source's field names.
// Transform and validation failures retain TransformError and validate.Error.
type ParameterError struct {
	Source string
	Path   validate.Path
	Err    error
}

func (e *ParameterError) Error() string {
	if e == nil || e.Err == nil {
		return "shape: parameter decoding failed"
	}
	return fmt.Sprintf("shape: decode %s at %s: %v", e.Source, e.Path.String(), e.Err)
}

// Unwrap returns the original decoding error.
func (e *ParameterError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// ParseForm decodes form values into a new struct, then transforms and validates it.
// Field names use form tags, falling back to Go names. Repeated parameters
// require slices. Nested structs use dotted names. Invalid definitions panic.
func ParseForm[T any](schema Schema[T], values url.Values, options ...FormOptions) (T, error) {
	return ParseFormContext(context.Background(), schema, values, options...)
}

// ParseFormContext is ParseForm with cancellation and per-request locale.
func ParseFormContext[T any](ctx context.Context, schema Schema[T], values url.Values, options ...FormOptions) (T, error) {
	if len(options) > 1 {
		panic("shape: at most one FormOptions")
	}
	strict := len(options) == 1 && options[0].DisallowUnknownFields
	return parseParameters(ctx, schema, values, fieldmeta.Form, strict)
}

// ParseQuery decodes query values into a new struct, then transforms and validates it.
// Field names use query tags, falling back to Go names. It does not parse
// URL encoding; use url.ParseQuery and check its error before calling ParseQuery.
func ParseQuery[T any](schema Schema[T], values url.Values, options ...QueryOptions) (T, error) {
	return ParseQueryContext(context.Background(), schema, values, options...)
}

// ParseQueryContext is ParseQuery with cancellation and per-request locale.
func ParseQueryContext[T any](ctx context.Context, schema Schema[T], values url.Values, options ...QueryOptions) (T, error) {
	if len(options) > 1 {
		panic("shape: at most one QueryOptions")
	}
	strict := len(options) == 1 && options[0].DisallowUnknownFields
	return parseParameters(ctx, schema, values, fieldmeta.Query, strict)
}

func parseParameters[T any](ctx context.Context, schema Schema[T], values url.Values, source fieldmeta.Source, strict bool) (T, error) {
	var zero T
	if ctx == nil {
		panic("shape: nil context")
	}
	ctx = fieldmeta.WithSource(ctx, source)
	candidate, owned, err := decodeParameters[T](ctx, values, source, strict)
	if err != nil {
		return zero, err
	}
	return finishParse(ctx, schema, candidate, owned)
}

func decodeParameters[T any](ctx context.Context, values url.Values, source fieldmeta.Source, strict bool) (T, bool, error) {
	var zero T
	candidate, owned, err := valuesdecode.Decode[T](ctx, values, source, strict)
	if ctx.Err() != nil {
		return zero, false, ctx.Err()
	}
	if err != nil {
		var parameter *valuesdecode.Error
		if errors.As(err, &parameter) {
			return zero, false, &ParameterError{Source: source.String(), Path: append(validate.Path(nil), parameter.Path...), Err: parameter.Err}
		}
		return zero, false, err
	}
	return candidate, owned, nil
}

// BindForm derives the target's tagged Schema and replaces *target only after
// decoding, transformation, and validation succeed. It does not merge values.
func BindForm[T any](target *T, values url.Values, options ...FormOptions) error {
	return BindFormContext(context.Background(), target, values, options...)
}

// BindFormContext is the context-aware form of BindForm.
func BindFormContext[T any](ctx context.Context, target *T, values url.Values, options ...FormOptions) error {
	if ctx == nil {
		panic("shape: nil context")
	}
	if target == nil {
		return errors.New("shape: nil bind target")
	}
	out, err := ParseFormContext(ctx, FromTags[T](), values, options...)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	*target = out
	return nil
}

// BindQuery derives the target's tagged Schema and replaces *target only after
// decoding, transformation, and validation succeed. It does not merge values.
func BindQuery[T any](target *T, values url.Values, options ...QueryOptions) error {
	return BindQueryContext(context.Background(), target, values, options...)
}

// BindQueryContext is the context-aware form of BindQuery.
func BindQueryContext[T any](ctx context.Context, target *T, values url.Values, options ...QueryOptions) error {
	if ctx == nil {
		panic("shape: nil context")
	}
	if target == nil {
		return errors.New("shape: nil bind target")
	}
	out, err := ParseQueryContext(ctx, FromTags[T](), values, options...)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	*target = out
	return nil
}
