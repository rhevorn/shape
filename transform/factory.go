package transform

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"time"

	"github.com/rhevorn/shape/internal/transformpath"
	"github.com/rhevorn/shape/types"
)

// MapKey is the set of map key types supported by Map.
type MapKey interface {
	~string |
		~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

func Value[T any]() ValueTransformer[T] { return ValueTransformer[T]{} }

func String() StringTransformer { return StringTransformer{} }

func Number[N Numeric]() NumberTransformer[N] { return NumberTransformer[N]{} }

func Int() NumberTransformer[int]                 { return Number[int]() }
func Int64() NumberTransformer[int64]             { return Number[int64]() }
func Float64() NumberTransformer[float64]         { return Number[float64]() }
func Duration() NumberTransformer[types.Duration] { return Number[types.Duration]() }
func Bool() ValueTransformer[bool]                { return Value[bool]() }
func Time() ValueTransformer[time.Time]           { return Value[time.Time]() }

func Pointer[T any](inner Transformer[T]) PointerTransformer[T] {
	if inner == nil {
		panic("transform: nil pointer transformer")
	}
	return PointerTransformer[T]{value: Value[*T]().ApplyContext(func(ctx context.Context, value *T) (*T, error) {
		if value == nil {
			return nil, nil
		}
		out, err := inner.TransformContext(ctx, *value)
		if err != nil {
			return nil, err
		}
		return &out, nil
	})}
}

func Slice[T any](inner Transformer[T]) SliceTransformer[T] {
	if inner == nil {
		panic("transform: nil slice transformer")
	}
	return SliceTransformer[T]{value: Value[[]T]().ApplyContext(func(ctx context.Context, value []T) ([]T, error) {
		if value == nil {
			return nil, nil
		}
		out := make([]T, len(value))
		for i, item := range value {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			transformed, err := inner.TransformContext(ctx, item)
			if err != nil {
				return nil, transformpath.Index(err, i)
			}
			out[i] = transformed
		}
		return out, nil
	})}
}

func Map[K MapKey, V any](key Transformer[K], value Transformer[V]) MapTransformer[K, V] {
	if key == nil || value == nil {
		panic("transform: nil map transformer")
	}
	return MapTransformer[K, V]{value: Value[map[K]V]().ApplyContext(func(ctx context.Context, input map[K]V) (map[K]V, error) {
		if input == nil {
			return nil, nil
		}
		keys := make([]K, 0, len(input))
		for item := range input {
			keys = append(keys, item)
		}
		sort.Slice(keys, func(i, j int) bool { return lessMapKey(keys[i], keys[j]) })
		out := make(map[K]V, len(input))
		for _, item := range keys {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			newKey, err := key.TransformContext(ctx, item)
			if err != nil {
				return nil, transformpath.Key(err, item)
			}
			newValue, err := value.TransformContext(ctx, input[item])
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

func lessMapKey[K MapKey](a, b K) bool {
	av, bv := reflect.ValueOf(a), reflect.ValueOf(b)
	switch av.Kind() {
	case reflect.String:
		return av.String() < bv.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return av.Int() < bv.Int()
	default:
		return av.Uint() < bv.Uint()
	}
}
