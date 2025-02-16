package transform

import (
	"context"
	"errors"
	"reflect"
)

func clone[T any](v T) (T, error) { return cloneContext(context.Background(), v) }
func cloneContext[T any](ctx context.Context, v T) (T, error) {
	var zero T
	out, err := cloneReflect(ctx, reflect.ValueOf(&v).Elem(), 0)
	if err != nil {
		return zero, err
	}
	return out.Interface().(T), nil
}
func cloneReflect(ctx context.Context, v reflect.Value, depth int) (reflect.Value, error) {
	if err := ctx.Err(); err != nil {
		return reflect.Value{}, err
	}
	if depth >= 64 {
		return reflect.Value{}, errors.New("transform: copy depth exceeded")
	}
	out := reflect.New(v.Type()).Elem()
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			return out, nil
		}
		x, err := cloneReflect(ctx, v.Elem(), depth+1)
		if err != nil {
			return out, err
		}
		out.Set(reflect.New(v.Type().Elem()))
		out.Elem().Set(x)
	case reflect.Slice:
		if v.IsNil() {
			return out, nil
		}
		out = reflect.MakeSlice(v.Type(), v.Len(), v.Len())
		for i := 0; i < v.Len(); i++ {
			x, err := cloneReflect(ctx, v.Index(i), depth+1)
			if err != nil {
				return out, err
			}
			out.Index(i).Set(x)
		}
	case reflect.Map:
		if v.IsNil() {
			return out, nil
		}
		out = reflect.MakeMapWithSize(v.Type(), v.Len())
		it := v.MapRange()
		for it.Next() {
			k, err := cloneReflect(ctx, it.Key(), depth+1)
			if err != nil {
				return out, err
			}
			x, err := cloneReflect(ctx, it.Value(), depth+1)
			if err != nil {
				return out, err
			}
			out.SetMapIndex(k, x)
		}
	case reflect.Struct:
		out.Set(v)
		for i := 0; i < v.NumField(); i++ {
			if out.Field(i).CanSet() && v.Type().Field(i).PkgPath == "" {
				x, err := cloneReflect(ctx, v.Field(i), depth+1)
				if err != nil {
					return out, err
				}
				out.Field(i).Set(x)
			}
		}
	case reflect.Interface:
		if v.IsNil() {
			return out, nil
		}
		x, err := cloneReflect(ctx, v.Elem(), depth+1)
		if err != nil {
			return out, err
		}
		out.Set(x)
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			x, err := cloneReflect(ctx, v.Index(i), depth+1)
			if err != nil {
				return out, err
			}
			out.Index(i).Set(x)
		}
	default:
		out.Set(v)
	}
	return out, nil
}
