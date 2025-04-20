package shape

import (
	"context"
	"reflect"

	"github.com/rhevorn/shape/internal/program"
	"github.com/rhevorn/shape/transform"
	"github.com/rhevorn/shape/validate"
)

// Numeric is the set of scalar number types supported by Number.
type Numeric interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}

// MapKey is the set of key types supported by Map. Named string and integer
// types are included through the underlying-type constraints.
type MapKey interface {
	~string |
		~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

// FieldSpec is an explicit field binding created by Field and accepted by New.
// Its unexported method intentionally limits implementations to this package.
type FieldSpec interface{ fieldDefinition() program.Definition }

type explicitField[T any] struct {
	name        string
	transformer transform.Transformer[T]
	validator   validate.Validator[T]
}

func (f explicitField[T]) fieldDefinition() program.Definition {
	return eraseField(f.name, f.transformer, f.validator)
}

func eraseField[T any](name string, transformer transform.Transformer[T], validator validate.Validator[T]) program.Definition {
	return program.Definition{
		Name: name,
		Type: reflect.TypeFor[T](),
		Transform: func(ctx context.Context, value reflect.Value) (reflect.Value, error) {
			out, err := transformer.TransformContext(ctx, value.Interface().(T))
			if err != nil {
				return reflect.Value{}, err
			}
			return reflect.ValueOf(&out).Elem(), nil
		},
		ValidateAll: func(ctx context.Context, value reflect.Value) error {
			return validator.ValidateContext(ctx, value.Interface().(T))
		},
		ValidateOne: func(ctx context.Context, value reflect.Value) error {
			return validator.ValidateFirstContext(ctx, value.Interface().(T))
		},
	}
}

func requireSchema[T any](schema Schema[T]) {
	if schema == nil {
		panic("shape: nil schema")
	}
}

func runSpecTransform[T any](ctx context.Context, transformer transform.Transformer[T], value T) (T, error) {
	out, err := transformer.TransformContext(ctx, value)
	if err != nil {
		var zero T
		return zero, normalizeTransformError(ctx, err)
	}
	return out, nil
}

// ValueSpec defines transform and validation behavior for a value of any type.
type ValueSpec[T any] struct {
	transformer transform.ValueTransformer[T]
	validator   validate.ValueValidator[T]
}

func (f ValueSpec[T]) Transform(v T) (T, error) {
	return runSpecTransform(context.Background(), f.transformer, v)
}
func (f ValueSpec[T]) TransformContext(ctx context.Context, v T) (T, error) {
	return runSpecTransform(ctx, f.transformer, v)
}
func (f ValueSpec[T]) Validate(v T) error { return f.validator.Validate(v) }
func (f ValueSpec[T]) ValidateContext(ctx context.Context, v T) error {
	return f.validator.ValidateContext(ctx, v)
}
func (f ValueSpec[T]) ValidateFirst(v T) error { return f.validator.ValidateFirst(v) }
func (f ValueSpec[T]) ValidateFirstContext(ctx context.Context, v T) error {
	return f.validator.ValidateFirstContext(ctx, v)
}
func (f ValueSpec[T]) IfZero(v T) ValueSpec[T] {
	f.transformer = f.transformer.IfZero(v)
	return f
}
func (f ValueSpec[T]) Apply(values ...func(T) (T, error)) ValueSpec[T] {
	f.transformer = f.transformer.Apply(values...)
	return f
}
func (f ValueSpec[T]) ApplyContext(values ...func(context.Context, T) (T, error)) ValueSpec[T] {
	f.transformer = f.transformer.ApplyContext(values...)
	return f
}
func (f ValueSpec[T]) Refine(values ...func(T) error) ValueSpec[T] {
	f.validator = f.validator.Refine(values...)
	return f
}
func (f ValueSpec[T]) RefineContext(values ...func(context.Context, T) error) ValueSpec[T] {
	f.validator = f.validator.RefineContext(values...)
	return f
}
func (f ValueSpec[T]) Label(label string) ValueSpec[T] {
	f.validator = f.validator.Label(label)
	return f
}
func (f ValueSpec[T]) Pointer() PointerSpec[T] { return pointerSpec(f) }
func (f ValueSpec[T]) Slice() SliceSpec[T]     { return sliceSpec(f) }
