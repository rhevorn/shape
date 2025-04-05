package transform

import (
	"context"
	"fmt"
	"time"

	"github.com/rhevorn/shape/internal/maporder"
	"github.com/rhevorn/shape/internal/transformpath"
	"github.com/rhevorn/shape/types"
)

// MapKey is the set of map key types supported by Map.
type MapKey interface {
	~string |
		~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

// Value creates a transformer for any T.
func Value[T any]() ValueTransformer[T] { return ValueTransformer[T]{} }

// String creates a string transformer.
func String() StringTransformer { return StringTransformer{} }

// Number creates a transformer for a supported numeric type.
func Number[N Numeric]() NumberTransformer[N] { return NumberTransformer[N]{} }

// Int creates an int transformer.
func Int() NumberTransformer[int] { return Number[int]() }

// Int64 creates an int64 transformer.
func Int64() NumberTransformer[int64] { return Number[int64]() }

// Float64 creates a float64 transformer.
func Float64() NumberTransformer[float64] { return Number[float64]() }

// Duration creates a transformer for types.Duration.
func Duration() NumberTransformer[types.Duration] { return Number[types.Duration]() }

// Bool creates a bool transformer.
func Bool() ValueTransformer[bool] { return Value[bool]() }

// Time creates a time.Time transformer.
func Time() ValueTransformer[time.Time] { return Value[time.Time]() }

// Pointer creates a transformer for *T that applies inner to non-nil values.
func Pointer[T any](inner Transformer[T]) PointerTransformer[T] {
	if inner == nil {
		panic("transform: nil pointer transformer")
	}
	return PointerTransformer[T]{value: Value[*T]().ApplyContext(func(ctx context.Context, value *T) (*T, error) {
		if value == nil {
			return nil, nil
		}
		out, err := runOwned(ctx, inner, *value)
		if err != nil {
			return nil, err
		}
		*value = out
		return value, nil
	})}
}

// Slice creates a transformer that applies inner to each element.
func Slice[T any](inner Transformer[T]) SliceTransformer[T] {
	if inner == nil {
		panic("transform: nil slice transformer")
	}
	return SliceTransformer[T]{value: Value[[]T]().ApplyContext(func(ctx context.Context, value []T) ([]T, error) {
		if value == nil {
			return nil, nil
		}
		for i, item := range value {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			transformed, err := runOwned(ctx, inner, item)
			if err != nil {
				return nil, transformpath.Index(err, i)
			}
			value[i] = transformed
		}
		return value, nil
	})}
}

// Map creates a transformer that applies key and value in stable key order.
func Map[K MapKey, V any](key Transformer[K], value Transformer[V]) MapTransformer[K, V] {
	if key == nil || value == nil {
		panic("transform: nil map transformer")
	}
	return MapTransformer[K, V]{value: Value[map[K]V]().ApplyContext(func(ctx context.Context, input map[K]V) (map[K]V, error) {
		if input == nil {
			return nil, nil
		}
		keys, err := maporder.Sorted(ctx, input)
		if err != nil {
			return nil, err
		}
		out := make(map[K]V, len(input))
		for _, item := range keys {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			newKey, err := runOwned(ctx, key, item)
			if err != nil {
				return nil, transformpath.Key(err, item)
			}
			newValue, err := runOwned(ctx, value, input[item])
			if err != nil {
				return nil, transformpath.Key(err, item)
			}
			if _, exists := out[newKey]; exists {
				return nil, transformpath.Key(fmt.Errorf("transform: duplicate map key %v", newKey), item)
			}
			out[newKey] = newValue
		}
		return out, nil
	})}
}
