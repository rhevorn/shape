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

func Value[T any]() ValueValidator[T] { return ValueValidator[T]{base: finiteCheck(base[T]{})} }

func String() StringValidator { return StringValidator{} }

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
			return issue(CodeInvalidNumber, "number.finite", nil, x)
		}
		return nil
	})
}

func Int() NumberValidator[int]                 { return Number[int]() }
func Int64() NumberValidator[int64]             { return Number[int64]() }
func Float64() NumberValidator[float64]         { return Number[float64]() }
func Duration() NumberValidator[types.Duration] { return Number[types.Duration]() }
func Bool() ValueValidator[bool]                { return Value[bool]() }
func Time() ValueValidator[time.Time]           { return Value[time.Time]() }

func Pointer[T any](inner Validator[T]) PointerValidator[T] {
	if inner == nil {
		panic("validate: nil pointer validator")
	}
	return PointerValidator[T]{inner: inner}
}

func Slice[T any](inner Validator[T]) SliceValidator[T] {
	if inner == nil {
		panic("validate: nil slice validator")
	}
	return SliceValidator[T]{inner: inner}
}

func Map[K MapKey, V any](key Validator[K], value Validator[V]) MapValidator[K, V] {
	if key == nil || value == nil {
		panic("validate: nil map validator")
	}
	return MapValidator[K, V]{key: key, value: value}
}
