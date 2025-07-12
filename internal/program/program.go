package program

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/rhevorn/shape/internal/fieldmeta"
	"github.com/rhevorn/shape/internal/jsonfields"
	"github.com/rhevorn/shape/internal/validationlocale"
	"github.com/rhevorn/shape/internal/validationmsg"
	"github.com/rhevorn/shape/validate"
)

// Definition is the erased form of one explicit field specification.
type Definition struct {
	Name        string
	Type        reflect.Type
	Transform   func(context.Context, reflect.Value) (reflect.Value, error)
	ValidateAll func(context.Context, reflect.Value) error
	ValidateOne func(context.Context, reflect.Value) error
}

type programField struct {
	index       int
	names       fieldmeta.Names
	transform   func(context.Context, reflect.Value) (reflect.Value, error)
	validateAll func(context.Context, reflect.Value) error
	validateOne func(context.Context, reflect.Value) error
}

// Compiled is an immutable set of fields bound to one struct type.
type Compiled struct{ fields []programField }

// Compile validates and binds field definitions to owner indexes.
func Compile(owner reflect.Type, specs []Definition) Compiled {
	fields := make([]programField, 0, len(specs))
	seen := make(map[string]struct{}, len(specs))
	paths := make(map[string]bool, len(specs))
	jsonFields := directJSONFields(owner)
	for _, spec := range specs {
		definition := spec
		if definition.Name == "" {
			panic("shape: field name must not be empty")
		}
		if _, ok := seen[definition.Name]; ok {
			panic("shape: duplicate field " + definition.Name)
		}
		seen[definition.Name] = struct{}{}
		field, ok := owner.FieldByName(definition.Name)
		if !ok || len(field.Index) != 1 || field.Anonymous {
			panic(fmt.Sprintf("shape: %v has no direct field %s", owner, definition.Name))
		}
		if field.PkgPath != "" {
			panic(fmt.Sprintf("shape: field %s is not exported", definition.Name))
		}
		if field.Type != definition.Type {
			panic(fmt.Sprintf("shape: field %s has type %v, schema has type %v", definition.Name, field.Type, definition.Type))
		}
		names := fieldmeta.Resolve(field.Name, field.Tag)
		if names.JSON != "" && jsonFields[names.JSON] != field.Index[0] {
			panic("shape: ambiguous or shadowed JSON field " + names.JSON)
		}

		if paths[names.Default] {
			panic("shape: duplicate field path " + names.Default)
		}
		paths[names.Default] = true

		fields = append(fields, programField{
			index: field.Index[0], names: names,
			transform: definition.Transform, validateAll: definition.ValidateAll, validateOne: definition.ValidateOne,
		})
	}
	return Compiled{fields: fields}
}

// Transformer executes compiled explicit field transforms.
type Transformer[T any] struct{ compiled Compiled }

// NewTransformer creates a transformer over compiled fields.
func NewTransformer[T any](compiled Compiled) Transformer[T] {
	return Transformer[T]{compiled: compiled}
}

func (t Transformer[T]) Transform(value T) (T, error) {
	return t.TransformContext(context.Background(), value)
}

func (t Transformer[T]) TransformContext(ctx context.Context, value T) (T, error) {
	var zero T
	if ctx == nil {
		panic("shape: nil context")
	}
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	out := reflect.New(reflect.TypeFor[T]()).Elem()
	out.Set(reflect.ValueOf(&value).Elem())
	source := fieldmeta.FromContext(ctx)
	for _, field := range t.compiled.fields {
		name := field.names.PathName(source)
		if err := ctx.Err(); err != nil {
			return zero, err
		}
		item, err := field.transform(ctx, out.Field(field.index))
		if err != nil {
			if ctx.Err() != nil {
				return zero, ctx.Err()
			}
			return zero, &TransformError{Path: validate.Path{validate.FieldPath(name)}, Err: err}
		}
		out.Field(field.index).Set(item)
	}
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	return out.Interface().(T), nil
}

// Validator executes compiled explicit field validation.
type Validator[T any] struct{ compiled Compiled }

// NewValidator creates a validator over compiled fields.
func NewValidator[T any](compiled Compiled) Validator[T] {
	return Validator[T]{compiled: compiled}
}

func (v Validator[T]) Validate(value T) error {
	return v.ValidateContext(context.Background(), value)
}
func (v Validator[T]) ValidateContext(ctx context.Context, value T) error {
	return v.run(ctx, value, false)
}
func (v Validator[T]) ValidateFirst(value T) error {
	return v.ValidateFirstContext(context.Background(), value)
}
func (v Validator[T]) ValidateFirstContext(ctx context.Context, value T) error {
	return v.run(ctx, value, true)
}

func (v Validator[T]) run(ctx context.Context, value T, first bool) error {
	if ctx == nil {
		panic("shape: nil context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	root := reflect.ValueOf(&value).Elem()
	issues := make([]validate.Issue, 0)
	source := fieldmeta.FromContext(ctx)
	for _, field := range v.compiled.fields {
		name := field.names.PathName(source)
		if err := ctx.Err(); err != nil {
			return err
		}
		var err error
		if first {
			err = field.validateOne(ctx, root.Field(field.index))
		} else {
			err = field.validateAll(ctx, root.Field(field.index))
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err == nil {
			continue
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		fieldIssues := validationIssues(err)
		for _, issue := range fieldIssues {
			issue.Path = append(validate.Path{validate.FieldPath(name)}, issue.Path...)
			if appendSchemaIssue(ctx, &issues, issue, first) {
				return &validate.Error{Issues: issues}
			}
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(issues) == 0 {
		return nil
	}
	return &validate.Error{Issues: issues}
}

// TransformError carries an explicit field path to the public facade.
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

func appendSchemaIssue(ctx context.Context, issues *[]validate.Issue, issue validate.Issue, first bool) bool {
	if len(*issues) >= validate.DefaultMaxIssues {
		(*issues)[validate.DefaultMaxIssues-1] = validate.Issue{
			Code:     validate.CodeTooManyIssues,
			Message:  validationmsg.Render(validationlocale.Get(ctx), "too_many_issues", "", validate.DefaultMaxIssues),
			Expected: validate.DefaultMaxIssues,
		}
		return true
	}
	*issues = append(*issues, issue)
	return first
}

func validationIssues(err error) []validate.Issue {
	var validationError *validate.Error
	if errors.As(err, &validationError) && validationError != nil && len(validationError.Issues) != 0 {
		return append([]validate.Issue(nil), validationError.Issues...)
	}
	return []validate.Issue{{Code: validate.CodeCustom, Message: err.Error()}}
}

func directJSONFields(owner reflect.Type) map[string]int {
	candidates := make([]jsonfields.Candidate, 0, owner.NumField())
	for i := 0; i < owner.NumField(); i++ {
		field := owner.Field(i)
		typ := field.Type
		if typ.Kind() == reflect.Pointer {
			typ = typ.Elem()
		}
		if field.PkgPath != "" && (!field.Anonymous || typ.Kind() != reflect.Struct) || field.Tag.Get("json") == "-" {
			continue
		}
		name, tagged := jsonfields.Name(field.Name, field.Tag.Get("json"))
		if field.Anonymous && !tagged && typ.Kind() == reflect.Struct {
			continue
		}
		candidates = append(candidates, jsonfields.Candidate{Index: i, Name: name, Tagged: tagged})
	}
	return jsonfields.Resolve(candidates)
}
