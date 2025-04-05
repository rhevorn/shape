// Package maporder provides the deterministic map traversal order shared by
// Shape's tagged, validation, and transformation paths.
package maporder

import (
	"context"
	"reflect"
)

// Key is the set of map-key types supported by Shape collections.
type Key interface {
	~string |
		~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

// Sorted returns the keys in ascending order and observes cancellation while
// collecting and sorting them.
func Sorted[K Key, V any](ctx context.Context, values map[K]V) ([]K, error) {
	keys := make([]K, 0, len(values))
	for key := range values {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	buffer := make([]K, len(keys))
	if err := mergeSort(ctx, keys, buffer, func(a, b K) bool {
		return less(reflect.ValueOf(a), reflect.ValueOf(b))
	}); err != nil {
		return nil, err
	}
	return keys, nil
}

// Reflect returns the keys of a reflected supported map in ascending order.
func Reflect(ctx context.Context, value reflect.Value) ([]reflect.Value, error) {
	keys := make([]reflect.Value, 0, value.Len())
	iterator := value.MapRange()
	for iterator.Next() {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		keys = append(keys, iterator.Key())
	}
	buffer := make([]reflect.Value, len(keys))
	if err := mergeSort(ctx, keys, buffer, less); err != nil {
		return nil, err
	}
	return keys, nil
}

func mergeSort[T any](ctx context.Context, values, buffer []T, less func(T, T) bool) error {
	for width := 1; width < len(values); width *= 2 {
		for start := 0; start < len(values); start += 2 * width {
			middle := min(start+width, len(values))
			end := min(start+2*width, len(values))
			left, right := start, middle
			for index := start; index < end; index++ {
				if err := ctx.Err(); err != nil {
					return err
				}
				if left < middle && (right >= end || less(values[left], values[right])) {
					buffer[index] = values[left]
					left++
				} else {
					buffer[index] = values[right]
					right++
				}
			}
		}
		copy(values, buffer)
	}
	return nil
}

func less(a, b reflect.Value) bool {
	switch a.Kind() {
	case reflect.String:
		return a.String() < b.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return a.Int() < b.Int()
	default:
		return a.Uint() < b.Uint()
	}
}
