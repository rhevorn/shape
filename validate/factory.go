package validate

import (
	"context"
	"math"
	"reflect"
	"time"

	"github.com/rhevorn/shape/types"
)

// MapKey is the set of map key types supported by Map.
type MapKey interface {
	~string |
		~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

// Value creates a validator for any T.
func Value[T any]() ValueValidator[T] { return ValueValidator[T]{base: finiteCheck(base[T]{})} }

// String creates a string validator.
func String() StringValidator { return StringValidator{} }

// Number creates a validator for a supported numeric type.
func Number[N Numeric]() NumberValidator[N] { return NumberValidator[N]{base: finiteCheck(base[N]{})} }

// finiteCheck installs the non-finite rejection for float kinds. Value and
// Number share it so a float reaching a schema is rejected identically whether
// it was declared through Number[N] or through the generic Value[T] escape
// hatch; the tagged path applies the same guard to every walked node.
func finiteCheck[T any](v base[T]) base[T] {
	kind := reflect.TypeFor[T]().Kind()
	if kind != reflect.Float32 && kind != reflect.Float64 {
		return v
	}
	return v.add(func(_ context.Context, x T) error {
		n := reflect.ValueOf(x).Float()
		if math.IsNaN(n) || math.IsInf(n, 0) {
			return issueError(CodeInvalidNumber, "number.finite", nil, x)
		}
		return nil
	})
}

// Int creates an int validator.
func Int() NumberValidator[int] { return Number[int]() }

// Int64 creates an int64 validator.
func Int64() NumberValidator[int64] { return Number[int64]() }

// Float64 creates a float64 validator.
func Float64() NumberValidator[float64] { return Number[float64]() }

// Duration creates a validator for types.Duration.
func Duration() NumberValidator[types.Duration] { return Number[types.Duration]() }

// Bool creates a bool validator.
func Bool() ValueValidator[bool] { return Value[bool]() }

// Time creates a time.Time validator.
func Time() ValueValidator[time.Time] { return Value[time.Time]() }

// Pointer creates a validator for *T that applies inner to non-nil values.
func Pointer[T any](inner Validator[T]) PointerValidator[T] {
	if inner == nil {
		panic("validate: nil pointer validator")
	}
	return PointerValidator[T]{inner: inner}
}

// Slice creates a validator that applies inner to each element.
func Slice[T any](inner Validator[T]) SliceValidator[T] {
	if inner == nil {
		panic("validate: nil slice validator")
	}
	return SliceValidator[T]{inner: inner}
}

// Map creates a validator that checks keys and values in stable key order.
func Map[K MapKey, V any](key Validator[K], value Validator[V]) MapValidator[K, V] {
	if key == nil || value == nil {
		panic("validate: nil map validator")
	}
	return MapValidator[K, V]{key: key, value: value}
}
