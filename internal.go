package shape

import (
	"context"
	"reflect"
)

func appendCopy[T any](values []T, value T) []T {
	result := make([]T, len(values)+1)
	copy(result, values)
	result[len(values)] = value
	return result
}

func orderedKeys(ctx context.Context, v reflect.Value) ([]reflect.Value, error) {
	keys := make([]reflect.Value, 0, v.Len())
	it := v.MapRange()
	for it.Next() {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		keys = append(keys, it.Key())
	}
	buf := make([]reflect.Value, len(keys))
	less := func(a, b reflect.Value) bool {
		if a.Kind() == reflect.String {
			return a.String() < b.String()
		}
		return compareNumber(a, b) < 0
	}
	for width := 1; width < len(keys); width *= 2 {
		for start := 0; start < len(keys); start += 2 * width {
			mid := min(start+width, len(keys))
			end := min(start+2*width, len(keys))
			a, b := start, mid
			for n := start; n < end; n++ {
				if err := ctx.Err(); err != nil {
					return nil, err
				}
				if a < mid && (b >= end || less(keys[a], keys[b])) {
					buf[n] = keys[a]
					a++
				} else {
					buf[n] = keys[b]
					b++
				}
			}
		}
		keys, buf = buf, keys
	}
	return keys, nil
}
