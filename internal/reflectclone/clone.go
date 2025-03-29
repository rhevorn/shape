// Package reflectclone holds the single deep-copy implementation shared by the
// root package and transform.
//
// Both packages used to carry hand-synced copies of this logic, and the copies
// had drifted: the transform copy never cloned unexported fields, so a private
// slice or map stayed aliased to the caller's storage, while the two packages
// disagreed about the recursion limit and the zero test for time.Time.
package reflectclone

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"
	"unsafe"
)

// MaxDepth bounds recursion so a cyclic or pathologically nested value fails
// with an error instead of exhausting the stack.
const MaxDepth = 64

// ErrDepthExceeded reports a value nested deeper than MaxDepth, or a cycle.
var ErrDepthExceeded = errors.New("copy depth exceeded")

// ErrUnsupportedType reports a type that cannot be snapshotted safely.
var ErrUnsupportedType = errors.New("unsupported type")

// IsZero reports whether v holds its type's zero value.
//
// reflect.Value.IsZero and time.Time.IsZero disagree for a zero instant that
// carries a non-nil Location: the former compares struct fields and sees a
// non-nil pointer, the latter does not. Shape treats "zero instant" as zero
// everywhere, so time.Time is special-cased.
func IsZero(v reflect.Value) bool {
	if v.Type() == reflect.TypeFor[time.Time]() {
		return v.Interface().(time.Time).IsZero()
	}
	return v.IsZero()
}

// ImmutableType reports whether values of t can be shared between copies
// instead of being duplicated.
func ImmutableType(t reflect.Type) bool {
	if t == reflect.TypeFor[time.Time]() {
		return true
	}
	switch t.Kind() {
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			if !ImmutableType(t.Field(i).Type) {
				return false
			}
		}
		return true
	case reflect.Array:
		return ImmutableType(t.Elem())
	case reflect.Pointer, reflect.Slice, reflect.Map, reflect.Interface, reflect.Func, reflect.Chan, reflect.UnsafePointer:
		return false
	}
	return true
}

// Clone returns a deep copy of v, detached from the caller's storage.
//
// In strict mode, types that cannot be snapshotted safely (interface, func,
// chan, unsafe pointer) return ErrUnsupportedType instead of being shared.
// Schemas use strict mode for fallback values, which are stored and reused
// across calls and must not alias anything the caller can still mutate.
func Clone(ctx context.Context, v reflect.Value, strict bool) (reflect.Value, error) {
	return clone(ctx, v, 0, strict)
}

func clone(ctx context.Context, v reflect.Value, depth int, strict bool) (reflect.Value, error) {
	if err := ctx.Err(); err != nil {
		return reflect.Value{}, err
	}
	if depth >= MaxDepth {
		return reflect.Value{}, ErrDepthExceeded
	}
	if ImmutableType(v.Type()) {
		return v, nil
	}
	out := reflect.New(v.Type()).Elem()
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			return out, nil
		}
		x, err := clone(ctx, v.Elem(), depth+1, strict)
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
		if ImmutableType(v.Type().Elem()) {
			reflect.Copy(out, v)
			return out, nil
		}
		for i := 0; i < v.Len(); i++ {
			x, err := clone(ctx, v.Index(i), depth+1, strict)
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
		iter := v.MapRange()
		for iter.Next() {
			k, err := clone(ctx, iter.Key(), depth+1, strict)
			if err != nil {
				return out, err
			}
			x, err := clone(ctx, iter.Value(), depth+1, strict)
			if err != nil {
				return out, err
			}
			out.SetMapIndex(k, x)
		}
	case reflect.Struct:
		out.Set(v)
		for i := 0; i < v.NumField(); i++ {
			field, err := writable(out.Field(i))
			if err != nil {
				return out, err
			}
			x, err := clone(ctx, field, depth+1, strict)
			if err != nil {
				return out, err
			}
			field.Set(x)
		}
	case reflect.Interface:
		if v.IsNil() {
			return out, nil
		}
		if strict {
			return out, unsupported(v.Type())
		}
		x, err := clone(ctx, v.Elem(), depth+1, strict)
		if err != nil {
			return out, err
		}
		out.Set(x)
	case reflect.Func, reflect.Chan, reflect.UnsafePointer:
		if strict {
			return out, unsupported(v.Type())
		}
		out.Set(v)
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			x, err := clone(ctx, v.Index(i), depth+1, strict)
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

// writable returns a settable view of f.
//
// Exported fields are already settable. Unexported fields are not, which is
// why they used to be skipped and left aliasing the caller's storage. The
// enclosing struct was built with reflect.New, so it is addressable and
// reflect.NewAt can re-derive a writable view over the same memory. Without
// this, a promoted exported field over a private slice or map leaks caller
// storage into the working copy.
func writable(f reflect.Value) (reflect.Value, error) {
	if f.CanSet() {
		return f, nil
	}
	if !f.CanAddr() {
		return reflect.Value{}, fmt.Errorf("%w: unaddressable field", ErrUnsupportedType)
	}
	return reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr())).Elem(), nil
}

func unsupported(t reflect.Type) error {
	return fmt.Errorf("%w: %s", ErrUnsupportedType, t)
}
