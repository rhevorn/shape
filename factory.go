package shape

import (
	"reflect"
	"time"

	"github.com/rhevorn/shape/internal/program"
	"github.com/rhevorn/shape/internal/tagged"
	"github.com/rhevorn/shape/transform"
	"github.com/rhevorn/shape/types"
	"github.com/rhevorn/shape/validate"
)

// New creates an explicit reusable Schema for an ordinary value struct.
// Fields not listed in fields pass through unchanged and are not validated.
// New does not inspect shape tags.
func New[T any](fields ...FieldSpec) StructSpec[T] {
	typ := reflect.TypeFor[T]()
	if typ.Kind() != reflect.Struct || typ == reflect.TypeFor[time.Time]() {
		panic("shape: New requires an ordinary value struct")
	}
	definitions := make([]program.Definition, len(fields))
	for i, field := range fields {
		if field == nil {
			panic("shape: nil field")
		}
		definitions[i] = field.fieldDefinition()
	}
	compiled := program.Compile(typ, definitions)
	return StructSpec[T]{
		transformer: program.NewTransformer[T](compiled),
		validator:   program.NewValidator[T](compiled),
	}
}

// Struct derives and caches a Schema for an ordinary value struct from its
// json and shape tags. Invalid program configuration panics during construction.
func Struct[T any]() TaggedSpec[T] {
	typ := reflect.TypeFor[T]()
	plan, err := tagged.Compile(typ)
	if err != nil {
		panic(err)
	}
	return newTaggedSpec[T](plan)
}

// Value creates a Schema for any type. Omit name when using it as a Pointer,
// Slice, or Map element Schema.
func Value[T any](name ...string) ValueSpec[T] {
	return ValueSpec[T]{name: oneFieldName(name), transformer: transform.Value[T](), validator: validate.Value[T]()}
}

// String creates a string Schema, optionally named for use in New.
func String(name ...string) StringSpec {
	return StringSpec{name: oneFieldName(name), transformer: transform.String(), validator: validate.String()}
}

// Number creates a numeric Schema, optionally named for use in New.
func Number[N Numeric](name ...string) NumberSpec[N] {
	return NumberSpec[N]{name: oneFieldName(name), transformer: transform.Number[N](), validator: validate.Number[N]()}
}

// Int creates an int Schema.
func Int(name ...string) NumberSpec[int] { return Number[int](name...) }

// Int64 creates an int64 Schema.
func Int64(name ...string) NumberSpec[int64] { return Number[int64](name...) }

// Float64 creates a float64 Schema.
func Float64(name ...string) NumberSpec[float64] { return Number[float64](name...) }

// Duration creates a Schema for types.Duration.
func Duration(name ...string) NumberSpec[types.Duration] {
	return Number[types.Duration](name...)
}

// Bool creates a bool Schema.
func Bool(name ...string) ValueSpec[bool] { return Value[bool](name...) }

// Time creates a time.Time Schema.
func Time(name ...string) ValueSpec[time.Time] { return Value[time.Time](name...) }

// Pointer creates a pointer Schema with inner behavior for non-nil values.
func Pointer[T any](name string, inner Schema[T]) PointerSpec[T] {
	return pointerSpec(name, inner)
}

func pointerSpec[T any](name string, inner Schema[T]) PointerSpec[T] {
	requireSchema(inner)
	return PointerSpec[T]{
		name: name, transformer: transform.Pointer[T](inner), validator: validate.Pointer[T](inner),
	}
}

// Slice creates a slice Schema that applies inner to every element.
func Slice[T any](name string, inner Schema[T]) SliceSpec[T] {
	return sliceSpec(name, inner)
}

func sliceSpec[T any](name string, inner Schema[T]) SliceSpec[T] {
	requireSchema(inner)
	return SliceSpec[T]{
		name: name, transformer: transform.Slice[T](inner), validator: validate.Slice[T](inner),
	}
}

// Map creates a map Schema that applies key and value in stable key order.
func Map[K MapKey, V any](name string, key Schema[K], value Schema[V]) MapSpec[K, V] {
	requireSchema(key)
	requireSchema(value)
	return MapSpec[K, V]{
		name: name, transformer: transform.Map[K, V](key, value), validator: validate.Map[K, V](key, value),
	}
}

// Field uses another Schema for a nested field.
func Field[T any](name string, schema Schema[T]) FieldSpec {
	if schema == nil {
		panic("shape: nil field schema")
	}
	return explicitField[T]{name: name, transformer: schema, validator: schema}
}
