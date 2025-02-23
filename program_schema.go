package shape

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/rhevorn/shape/validate"
)

type programField struct {
	index       int
	name        string
	transform   func(context.Context, reflect.Value) (reflect.Value, error)
	validateAll func(context.Context, reflect.Value) error
	validateOne func(context.Context, reflect.Value) error
}

func compileProgramFields(owner reflect.Type, specs []FieldSpec) []programField {
	fields := make([]programField, 0, len(specs))
	seen := make(map[string]struct{}, len(specs))
	for _, spec := range specs {
		if spec == nil {
			panic("shape: nil field")
		}
		definition := spec.fieldDefinition()
		if definition.name == "" {
			panic("shape: field name must not be empty")
		}
		if _, ok := seen[definition.name]; ok {
			panic("shape: duplicate field " + definition.name)
		}
		seen[definition.name] = struct{}{}
		field, ok := owner.FieldByName(definition.name)
		if !ok || len(field.Index) != 1 || field.Anonymous {
			panic(fmt.Sprintf("shape: %v has no direct field %s", owner, definition.name))
		}
		if field.PkgPath != "" {
			panic(fmt.Sprintf("shape: field %s is not exported", definition.name))
		}
		if field.Type != definition.typ {
			panic(fmt.Sprintf("shape: field %s has type %v, contract has type %v", definition.name, field.Type, definition.typ))
		}
		jsonName := strings.Split(field.Tag.Get("json"), ",")[0]
		if jsonName == "-" {
			panic(fmt.Sprintf("shape: field %s is excluded from JSON", definition.name))
		}
		if jsonName == "" {
			jsonName = field.Name
		}
		fields = append(fields, programField{
			index: field.Index[0], name: jsonName,
			transform: definition.transform, validateAll: definition.validateAll, validateOne: definition.validateOne,
		})
	}
	return fields
}

type programTransformer[T any] struct{ fields []programField }

func (t programTransformer[T]) Transform(value T) (T, error) {
	return t.TransformContext(context.Background(), value)
}

func (t programTransformer[T]) TransformContext(ctx context.Context, value T) (T, error) {
	var zero T
	if ctx == nil {
		panic("shape: nil context")
	}
	out := reflect.New(reflect.TypeFor[T]()).Elem()
	out.Set(reflect.ValueOf(&value).Elem())
	for _, field := range t.fields {
		if err := ctx.Err(); err != nil {
			return zero, err
		}
		item, err := field.transform(ctx, out.Field(field.index))
		if err != nil {
			err = normalizeTransformError(ctx, err)
			if ctx.Err() != nil {
				return zero, ctx.Err()
			}
			path := validate.Path{validate.FieldPath(field.name)}
			var nested *TransformError
			if errors.As(err, &nested) && nested != nil {
				path = append(path, nested.Path...)
				err = nested.Err
			}
			return zero, &TransformError{Path: path, Err: err}
		}
		out.Field(field.index).Set(item)
	}
	return out.Interface().(T), nil
}

type programValidator[T any] struct{ fields []programField }

func (v programValidator[T]) Validate(value T) error {
	return v.ValidateContext(context.Background(), value)
}
func (v programValidator[T]) ValidateContext(ctx context.Context, value T) error {
	return v.run(ctx, value, false)
}
func (v programValidator[T]) ValidateFirst(value T) error {
	return v.ValidateFirstContext(context.Background(), value)
}
func (v programValidator[T]) ValidateFirstContext(ctx context.Context, value T) error {
	return v.run(ctx, value, true)
}

func (v programValidator[T]) run(ctx context.Context, value T, first bool) error {
	if ctx == nil {
		panic("shape: nil context")
	}
	root := reflect.ValueOf(&value).Elem()
	issues := make([]validate.Issue, 0)
	for _, field := range v.fields {
		if err := ctx.Err(); err != nil {
			return err
		}
		var err error
		if first {
			err = field.validateOne(ctx, root.Field(field.index))
		} else {
			err = field.validateAll(ctx, root.Field(field.index))
		}
		if err == nil {
			continue
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		fieldIssues := validationIssues(err)
		for _, issue := range fieldIssues {
			issue.Path = append(validate.Path{validate.FieldPath(field.name)}, issue.Path...)
			if appendSchemaIssue(&issues, issue, first) {
				return &validate.Error{Issues: issues}
			}
		}
	}
	if len(issues) == 0 {
		return nil
	}
	return &validate.Error{Issues: issues}
}

func validationIssues(err error) []validate.Issue {
	var validationError *validate.Error
	if errors.As(err, &validationError) && validationError != nil {
		return append([]validate.Issue(nil), validationError.Issues...)
	}
	return []validate.Issue{{Code: validate.CodeCustom, Message: err.Error()}}
}
