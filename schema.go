package shape

import (
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"

	"github.com/rhevorn/shape/internal/transformpath"
	"github.com/rhevorn/shape/validate"
)

// Schema is the high-level contract produced by New or Struct. Transformation
// and validation remain independently callable; JSON methods compose decode,
// transform, and validation in that order.
type Schema[T any] interface {
	Transform(T) (T, error)
	TransformContext(context.Context, T) (T, error)
	Validate(T) error
	ValidateContext(context.Context, T) error
	ValidateFirst(T) error
	ValidateFirstContext(context.Context, T) error

	ParseJSON([]byte, ...JSONOptions) (T, error)
	ParseJSONContext(context.Context, []byte, ...JSONOptions) (T, error)
	ParseJSONReader(io.Reader, ...JSONOptions) (T, error)
	ParseJSONReaderContext(context.Context, io.Reader, ...JSONOptions) (T, error)
	BindJSON(*T, []byte, ...JSONOptions) error
	BindJSONContext(context.Context, *T, []byte, ...JSONOptions) error
	BindJSONReader(*T, io.Reader, ...JSONOptions) error
	BindJSONReaderContext(context.Context, *T, io.Reader, ...JSONOptions) error
}

// TransformError reports the struct field or collection element whose tag
// transform failed. Unwrap returns the original transform error.
type TransformError struct {
	Path validate.Path
	Err  error
}

func (e *TransformError) Error() string {
	if e == nil || e.Err == nil {
		return "shape: transform failed"
	}
	if len(e.Path) == 0 {
		return "shape: transform failed: " + e.Err.Error()
	}
	return fmt.Sprintf("shape: transform failed at %s: %v", e.Path.String(), e.Err)
}

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
	var public *TransformError
	if errors.As(err, &public) && public != nil {
		return public
	}
	var internal *transformpath.Error
	if errors.As(err, &internal) && internal != nil {
		path := make(validate.Path, 0, len(internal.Segments))
		for _, segment := range internal.Segments {
			if segment.IsIndex {
				path = append(path, validate.IndexPath(segment.Index))
			} else {
				path = append(path, validate.FieldPath(segment.Key))
			}
		}
		return &TransformError{Path: path, Err: internal.Err}
	}
	return &TransformError{Err: err}
}

// structSchema is the immutable, concurrency-safe implementation returned by
// Struct. Users only need the Schema interface.
type structSchema[T any] struct{ p *valuePlan }

var _ Schema[struct{}] = structSchema[struct{}]{}

func (s structSchema[T]) plan() *valuePlan { return s.p }

func (s structSchema[T]) Transform(value T) (T, error) {
	return s.TransformContext(context.Background(), value)
}

func (s structSchema[T]) TransformContext(ctx context.Context, value T) (T, error) {
	var zero T
	if ctx == nil {
		panic("shape: nil context")
	}
	if s.p == nil {
		return zero, errors.New("shape: uninitialized schema; use shape.Struct")
	}
	out, err := transformPlan(ctx, s.p, reflect.ValueOf(&value).Elem(), nil, 0)
	if err != nil {
		return zero, err
	}
	return out.Interface().(T), nil
}

func (s structSchema[T]) Validate(value T) error {
	return s.ValidateContext(context.Background(), value)
}

func (s structSchema[T]) ValidateContext(ctx context.Context, value T) error {
	return s.validate(ctx, value, false)
}

func (s structSchema[T]) ValidateFirst(value T) error {
	return s.ValidateFirstContext(context.Background(), value)
}

func (s structSchema[T]) ValidateFirstContext(ctx context.Context, value T) error {
	return s.validate(ctx, value, true)
}

func (s structSchema[T]) validate(ctx context.Context, value T, first bool) error {
	if ctx == nil {
		panic("shape: nil context")
	}
	if s.p == nil {
		return errors.New("shape: uninitialized schema; use shape.Struct")
	}
	issues := make([]validate.Issue, 0)
	_, err := validatePlan(ctx, s.p, reflect.ValueOf(&value).Elem(), nil, 0, first, &issues)
	if err != nil {
		return err
	}
	if len(issues) == 0 {
		return nil
	}
	return validate.Localize(&validate.Error{Issues: issues}, validate.LocaleFromContext(ctx))
}

func transformPlan(ctx context.Context, p *valuePlan, value reflect.Value, path validate.Path, depth int) (reflect.Value, error) {
	if err := ctx.Err(); err != nil {
		return reflect.Value{}, err
	}
	if depth >= defaultMaxRecursiveDepth {
		return reflect.Value{}, &TransformError{Path: cloneValidatePath(path), Err: errors.New("maximum traversal depth exceeded")}
	}

	var err error
	for _, step := range p.steps {
		if err = ctx.Err(); err != nil {
			return reflect.Value{}, err
		}
		switch step.kind {
		case valueStepOption:
			if (step.option == "ifzero" && zeroValue(value)) || (step.option == "ifnull" && value.IsNil()) {
				value, err = cloneValue(ctx, step.fallback, 0, true)
				if err != nil {
					return reflect.Value{}, &TransformError{Path: cloneValidatePath(path), Err: err}
				}
			}
		case valueStepTransform:
			value, err = step.transform(ctx, value)
			if err != nil {
				if ctx.Err() != nil {
					return reflect.Value{}, ctx.Err()
				}
				return reflect.Value{}, &TransformError{Path: cloneValidatePath(path), Err: err}
			}
		}
	}

	switch p.typ.Kind() {
	case reflect.Pointer:
		if value.IsNil() || p.element == nil {
			return value, nil
		}
		inner, err := transformPlan(ctx, p.element, value.Elem(), path, depth+1)
		if err != nil {
			return reflect.Value{}, err
		}
		out := reflect.New(value.Type().Elem())
		out.Elem().Set(inner)
		return out, nil
	case reflect.Struct:
		if p.fields == nil {
			return value, nil
		}
		out := reflect.New(value.Type()).Elem()
		out.Set(value)
		for _, field := range p.fields {
			fieldPath := appendValidatePath(path, validate.FieldPath(field.name))
			item, err := transformPlan(ctx, field.plan, value.Field(field.index), fieldPath, depth+1)
			if err != nil {
				return reflect.Value{}, err
			}
			out.Field(field.index).Set(item)
		}
		return out, nil
	case reflect.Slice:
		if value.IsNil() || p.element == nil {
			return value, nil
		}
		out := reflect.MakeSlice(value.Type(), value.Len(), value.Len())
		for i := 0; i < value.Len(); i++ {
			item, err := transformPlan(ctx, p.element, value.Index(i), appendValidatePath(path, validate.IndexPath(i)), depth+1)
			if err != nil {
				return reflect.Value{}, err
			}
			out.Index(i).Set(item)
		}
		return out, nil
	case reflect.Map:
		if value.IsNil() || p.element == nil {
			return value, nil
		}
		keys, err := orderedKeys(ctx, value)
		if err != nil {
			return reflect.Value{}, err
		}
		out := reflect.MakeMapWithSize(value.Type(), value.Len())
		for _, key := range keys {
			itemPath := appendValidatePath(path, validate.FieldPath(fmt.Sprint(key.Interface())))
			item, err := transformPlan(ctx, p.element, value.MapIndex(key), itemPath, depth+1)
			if err != nil {
				return reflect.Value{}, err
			}
			out.SetMapIndex(key, item)
		}
		return out, nil
	default:
		return value, nil
	}
}

func validatePlan(ctx context.Context, p *valuePlan, value reflect.Value, path validate.Path, depth int, first bool, issues *[]validate.Issue) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if depth >= defaultMaxRecursiveDepth {
		return appendSchemaIssue(issues, validate.Issue{
			Code: validate.CodeTooDeep, Path: cloneValidatePath(path),
			MessageID: "too_deep",
			Expected:  defaultMaxRecursiveDepth, Received: depth,
		}, first), nil
	}
	if !finite(value) {
		if appendSchemaIssue(issues, validate.Issue{
			Code: validate.CodeInvalidNumber, Path: cloneValidatePath(path),
			MessageID: "number.finite", Received: value.Interface(), Label: p.label,
		}, first) {
			return true, nil
		}
	}

	for _, step := range p.steps {
		if step.kind != valueStepRule {
			continue
		}
		if err := ctx.Err(); err != nil {
			return false, err
		}
		err := step.check(ctx, value)
		if ctx.Err() != nil {
			return false, ctx.Err()
		}
		if err == nil {
			continue
		}
		issue, ok := ruleIssue(err, path, p.label)
		if !ok {
			issue = validate.Issue{
				Code: validate.CodeCustom, Path: cloneValidatePath(path),
				Message: err.Error(), Label: p.label,
			}
		}
		if appendSchemaIssue(issues, issue, first) {
			return true, nil
		}
	}

	switch p.typ.Kind() {
	case reflect.Pointer:
		if !value.IsNil() && p.element != nil {
			return validatePlan(ctx, p.element, value.Elem(), path, depth+1, first, issues)
		}
	case reflect.Struct:
		for _, field := range p.fields {
			stop, err := validatePlan(ctx, field.plan, value.Field(field.index), appendValidatePath(path, validate.FieldPath(field.name)), depth+1, first, issues)
			if err != nil || stop {
				return stop, err
			}
		}
	case reflect.Slice:
		if p.element != nil {
			for i := 0; i < value.Len(); i++ {
				stop, err := validatePlan(ctx, p.element, value.Index(i), appendValidatePath(path, validate.IndexPath(i)), depth+1, first, issues)
				if err != nil || stop {
					return stop, err
				}
			}
		}
	case reflect.Map:
		if p.element != nil && !value.IsNil() {
			keys, err := orderedKeys(ctx, value)
			if err != nil {
				return false, err
			}
			for _, key := range keys {
				stop, err := validatePlan(ctx, p.element, value.MapIndex(key), appendValidatePath(path, validate.FieldPath(fmt.Sprint(key.Interface()))), depth+1, first, issues)
				if err != nil || stop {
					return stop, err
				}
			}
		}
	}
	return false, nil
}

func appendSchemaIssue(issues *[]validate.Issue, issue validate.Issue, first bool) bool {
	if len(*issues) >= validate.DefaultMaxIssues {
		(*issues)[validate.DefaultMaxIssues-1] = validate.Issue{
			Code:      validate.CodeTooManyIssues,
			MessageID: "too_many_issues",
			Expected:  validate.DefaultMaxIssues,
		}
		return true
	}
	*issues = append(*issues, issue)
	return first
}

func appendValidatePath(path validate.Path, segment validate.PathSegment) validate.Path {
	out := make(validate.Path, len(path)+1)
	copy(out, path)
	out[len(path)] = segment
	return out
}

func cloneValidatePath(path validate.Path) validate.Path {
	return append(validate.Path(nil), path...)
}
