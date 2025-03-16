package shape

import (
	"reflect"
	"time"

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
	compiled := compileProgramFields(typ, fields)
	return StructSpec[T]{
		transformer: programTransformer[T]{fields: compiled},
		validator:   programValidator[T]{fields: compiled},
	}
}

// Struct derives and caches a Schema for an ordinary value struct from its
// json and shape tags. Invalid program configuration panics during construction.
func Struct[T any]() TaggedSpec[T] { return taggedSchema[T]() }

// Value creates a Schema for any type. Omit name when using it as a Pointer,
// Slice, or Map element Schema.
func Value[T any](name ...string) ValueSpec[T] {
	return ValueSpec[T]{name: oneFieldName(name), transformer: transform.Value[T](), validator: validate.Value[T]()}
}

func String(name ...string) StringSpec {
	return StringSpec{name: oneFieldName(name), transformer: transform.String(), validator: validate.String()}
}

func Number[N Numeric](name ...string) NumberSpec[N] {
	return NumberSpec[N]{name: oneFieldName(name), transformer: transform.Number[N](), validator: validate.Number[N]()}
}

func Int(name ...string) NumberSpec[int]         { return Number[int](name...) }
func Int64(name ...string) NumberSpec[int64]     { return Number[int64](name...) }
func Float64(name ...string) NumberSpec[float64] { return Number[float64](name...) }
func Duration(name ...string) NumberSpec[types.Duration] {
	return Number[types.Duration](name...)
}
func Bool(name ...string) ValueSpec[bool]      { return Value[bool](name...) }
func Time(name ...string) ValueSpec[time.Time] { return Value[time.Time](name...) }

func Pointer[T any](name string, inner Schema[T]) PointerSpec[T] {
	return pointerSpec(name, inner)
}

func pointerSpec[T any](name string, inner Schema[T]) PointerSpec[T] {
	requireSchema(inner)
	return PointerSpec[T]{
		name: name, transformer: transform.Pointer[T](inner), validator: validate.Pointer[T](inner),
	}
}

func Slice[T any](name string, inner Schema[T]) SliceSpec[T] {
	return sliceSpec(name, inner)
}

func sliceSpec[T any](name string, inner Schema[T]) SliceSpec[T] {
	requireSchema(inner)
	return SliceSpec[T]{
		name: name, transformer: transform.Slice[T](inner), validator: validate.Slice[T](inner),
	}
}

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
