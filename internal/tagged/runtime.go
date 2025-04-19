package tagged

import (
	"context"
	"errors"
	"reflect"

	"github.com/rhevorn/shape/internal/maporder"
	"github.com/rhevorn/shape/internal/validationlocale"
	"github.com/rhevorn/shape/internal/validationmsg"
	"github.com/rhevorn/shape/validate"
)

// TransformError carries a tag-runtime path to the public facade.
type TransformError struct {
	Path validate.Path
	Err  error
}

func (e *TransformError) Error() string {
	if e == nil || e.Err == nil {
		return "transform failed"
	}
	return e.Err.Error()
}

func (e *TransformError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// Runtime executes one immutable compiled tag plan.
type Runtime[T any] struct{ Plan *Plan }

func (s Runtime[T]) Transform(value T) (T, error) {
	return s.TransformContext(context.Background(), value)
}

func (s Runtime[T]) TransformContext(ctx context.Context, value T) (T, error) {
	return s.transformContext(ctx, value, false)
}

func (s Runtime[T]) transformContext(ctx context.Context, value T, owned bool) (T, error) {
	var zero T
	if ctx == nil {
		panic("shape: nil context")
	}
	if s.Plan == nil {
		return zero, errors.New("shape: uninitialized schema; use shape.Struct")
	}
	ownership := borrowedValue
	if owned {
		ownership = ownedValue
	}
	out, err := transformTagPlan(ctx, s.Plan, reflect.ValueOf(&value).Elem(), nil, 0, ownership)
	if err != nil {
		return zero, err
	}
	return out.Interface().(T), nil
}

func (s Runtime[T]) Validate(value T) error {
	return s.ValidateContext(context.Background(), value)
}

func (s Runtime[T]) ValidateContext(ctx context.Context, value T) error {
	return s.validate(ctx, value, false)
}

func (s Runtime[T]) ValidateFirst(value T) error {
	return s.ValidateFirstContext(context.Background(), value)
}

func (s Runtime[T]) ValidateFirstContext(ctx context.Context, value T) error {
	return s.validate(ctx, value, true)
}

func (s Runtime[T]) validate(ctx context.Context, value T, first bool) error {
	if ctx == nil {
		panic("shape: nil context")
	}
	if s.Plan == nil {
		return errors.New("shape: uninitialized schema; use shape.Struct")
	}
	issues := make([]validate.Issue, 0)
	_, err := validatePlan(ctx, s.Plan, reflect.ValueOf(&value).Elem(), nil, 0, first, &issues)
	if err != nil {
		return err
	}
	if len(issues) == 0 {
		return nil
	}
	return &validate.Error{Issues: issues}
}

type valueOwnership uint8

const (
	borrowedValue valueOwnership = iota
	ownedValue
)

func transformTagPlan(ctx context.Context, p *Plan, value reflect.Value, path validate.Path, depth int, ownership valueOwnership) (reflect.Value, error) {
	owned := ownership == ownedValue
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
		case tagStepOption:
			if (step.option == "ifzero" && isZero(value)) || (step.option == "ifnull" && value.IsNil()) {
				value, err = cloneValue(ctx, step.fallback, true)
				if err != nil {
					return reflect.Value{}, &TransformError{Path: cloneValidatePath(path), Err: err}
				}
			}
		case tagStepTransform:
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
		inner, err := transformTagPlan(ctx, p.element, value.Elem(), path, depth+1, ownership)
		if err != nil {
			return reflect.Value{}, err
		}
		if owned {
			value.Elem().Set(inner)
			return value, nil
		}
		out := reflect.New(value.Type().Elem())
		out.Elem().Set(inner)
		return out, nil
	case reflect.Struct:
		if p.fields == nil {
			return value, nil
		}
		out := value
		if !owned {
			out = reflect.New(value.Type()).Elem()
			out.Set(value)
		}
		for _, field := range p.fields {
			fieldPath := appendValidatePath(path, validate.FieldPath(field.name))
			item, err := transformTagPlan(ctx, field.plan, value.Field(field.index), fieldPath, depth+1, ownership)
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
		out := value
		if !owned {
			out = reflect.MakeSlice(value.Type(), value.Len(), value.Len())
		}
		for i := 0; i < value.Len(); i++ {
			item, err := transformTagPlan(ctx, p.element, value.Index(i), appendValidatePath(path, validate.IndexPath(i)), depth+1, ownership)
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
		keys, err := maporder.Reflect(ctx, value)
		if err != nil {
			return reflect.Value{}, err
		}
		out := reflect.MakeMapWithSize(value.Type(), value.Len())
		for _, key := range keys {
			itemPath := appendValidatePath(path, validate.MapKeyPath(key.Interface()))
			mapValue := value.MapIndex(key)
			valueInput := reflect.New(mapValue.Type()).Elem()
			valueInput.Set(mapValue)
			item, err := transformTagPlan(ctx, p.element, valueInput, itemPath, depth+1, ownership)
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

func validatePlan(ctx context.Context, p *Plan, value reflect.Value, path validate.Path, depth int, first bool, issues *[]validate.Issue) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if depth >= defaultMaxRecursiveDepth {
		return appendSchemaIssue(ctx, issues, validate.Issue{
			Code: validate.CodeTooDeep, Path: cloneValidatePath(path),
			Message:  validationMessage(ctx, "too_deep", "", defaultMaxRecursiveDepth),
			Expected: defaultMaxRecursiveDepth, Received: depth,
		}, first), nil
	}
	if !finite(value) {
		if appendSchemaIssue(ctx, issues, validate.Issue{
			Code: validate.CodeInvalidNumber, Path: cloneValidatePath(path),
			Message:  validationMessage(ctx, "number.finite", p.label, nil),
			Received: value.Interface(), Label: p.label,
		}, first) {
			return true, nil
		}
	}

	for _, step := range p.steps {
		if step.kind != tagStepRule {
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
		issue, ok := ruleIssue(ctx, err, path, p.label)
		if !ok {
			issue = validate.Issue{
				Code: validate.CodeCustom, Path: cloneValidatePath(path),
				Message: err.Error(), Label: p.label,
			}
		}
		if appendSchemaIssue(ctx, issues, issue, first) {
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
			keys, err := maporder.Reflect(ctx, value)
			if err != nil {
				return false, err
			}
			for _, key := range keys {
				stop, err := validatePlan(ctx, p.element, value.MapIndex(key), appendValidatePath(path, validate.MapKeyPath(key.Interface())), depth+1, first, issues)
				if err != nil || stop {
					return stop, err
				}
			}
		}
	}
	return false, nil
}

func appendSchemaIssue(ctx context.Context, issues *[]validate.Issue, issue validate.Issue, first bool) bool {
	if len(*issues) >= validate.DefaultMaxIssues {
		(*issues)[validate.DefaultMaxIssues-1] = validate.Issue{
			Code:     validate.CodeTooManyIssues,
			Message:  validationMessage(ctx, "too_many_issues", "", validate.DefaultMaxIssues),
			Expected: validate.DefaultMaxIssues,
		}
		return true
	}
	*issues = append(*issues, issue)
	return first
}

func validationMessage(ctx context.Context, id, label string, expected any) string {
	return validationmsg.Render(validationlocale.Get(ctx), id, label, expected)
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
