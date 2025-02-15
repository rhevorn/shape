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

func Value[T any]() ValueValidator[T] { return ValueValidator[T]{} }

func String() StringValidator { return StringValidator{} }

func Number[N Numeric]() NumberValidator[N] {
	v := NumberValidator[N]{}
	typ := reflect.TypeFor[N]()
	if typ.Kind() == reflect.Float32 || typ.Kind() == reflect.Float64 {
		v = v.add(func(_ context.Context, x N) error {
			n := reflect.ValueOf(x).Float()
			if math.IsNaN(n) || math.IsInf(n, 0) {
				return issue(CodeInvalidNumber, "number.finite", nil, x)
			}
			return nil
		})
	}
	return v
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
