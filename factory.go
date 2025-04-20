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

// New creates an explicit reusable Schema for an ordinary value struct from
// field bindings created by Field. Fields not listed in fields pass through
// unchanged and are not validated.
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

// Value creates a Schema for any type.
func Value[T any]() ValueSpec[T] {
	return ValueSpec[T]{transformer: transform.Value[T](), validator: validate.Value[T]()}
}

// String creates a string Schema.
func String() StringSpec {
	return StringSpec{transformer: transform.String(), validator: validate.String()}
}

// Number creates a numeric Schema.
func Number[N Numeric]() NumberSpec[N] {
	return NumberSpec[N]{transformer: transform.Number[N](), validator: validate.Number[N]()}
}

// Int creates an int Schema.
func Int() NumberSpec[int] { return Number[int]() }

// Int64 creates an int64 Schema.
func Int64() NumberSpec[int64] { return Number[int64]() }

// Float64 creates a float64 Schema.
func Float64() NumberSpec[float64] { return Number[float64]() }

// Duration creates a Schema for types.Duration.
func Duration() NumberSpec[types.Duration] {
	return Number[types.Duration]()
}

// Bool creates a bool Schema.
func Bool() ValueSpec[bool] { return Value[bool]() }

// Time creates a time.Time Schema.
func Time() ValueSpec[time.Time] { return Value[time.Time]() }

// Pointer creates a pointer Schema with inner behavior for non-nil values.
func Pointer[T any](inner Schema[T]) PointerSpec[T] {
	return pointerSpec(inner)
}

func pointerSpec[T any](inner Schema[T]) PointerSpec[T] {
	requireSchema(inner)
	return PointerSpec[T]{
		transformer: transform.Pointer[T](inner), validator: validate.Pointer[T](inner),
	}
}

// Slice creates a slice Schema that applies inner to every element.
func Slice[T any](inner Schema[T]) SliceSpec[T] {
	return sliceSpec(inner)
}

func sliceSpec[T any](inner Schema[T]) SliceSpec[T] {
	requireSchema(inner)
	return SliceSpec[T]{
		transformer: transform.Slice[T](inner), validator: validate.Slice[T](inner),
	}
}

// Map creates a map Schema that applies key and value in stable key order.
func Map[K MapKey, V any](key Schema[K], value Schema[V]) MapSpec[K, V] {
	requireSchema(key)
	requireSchema(value)
	return MapSpec[K, V]{
		transformer: transform.Map[K, V](key, value), validator: validate.Map[K, V](key, value),
	}
}

// Field binds a value Schema to a direct Go struct field.
func Field[T any](name string, schema Schema[T]) FieldSpec {
	if schema == nil {
		panic("shape: nil field schema")
	}
	return explicitField[T]{name: name, transformer: schema, validator: schema}
}
