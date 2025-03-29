package validate

import (
	"context"
	"reflect"
)

type SliceValidator[T any] struct {
	base[[]T]
	inner Validator[T]
}

func (v SliceValidator[T]) add(fn check[[]T]) SliceValidator[T] { v.base = v.base.add(fn); return v }
func (v SliceValidator[T]) NotNull() SliceValidator[T] {
	return v.add(func(_ context.Context, x []T) error {
		if x == nil {
			return issue(CodeInvalidValue, "not_null", nil, x)
		}
		return nil
	})
}
func (v SliceValidator[T]) NotEmpty() SliceValidator[T] {
	return v.add(func(_ context.Context, x []T) error {
		if len(x) == 0 {
			return issue(CodeInvalidValue, "not_empty.collection", nil, x)
		}
		return nil
	})
}
func (v SliceValidator[T]) Min(n int) SliceValidator[T] {
	if n < 0 {
		panic("validate: negative length")
	}
	return v.add(func(_ context.Context, x []T) error {
		if len(x) < n {
			return issue(CodeTooSmall, "too_small.collection", n, len(x))
		}
		return nil
	})
}
func (v SliceValidator[T]) Max(n int) SliceValidator[T] {
	if n < 0 {
		panic("validate: negative length")
	}
	return v.add(func(_ context.Context, x []T) error {
		if len(x) > n {
			return issue(CodeTooBig, "too_big.collection", n, len(x))
		}
		return nil
	})
}
func (v SliceValidator[T]) Len(n int) SliceValidator[T] {
	if n < 0 {
		panic("validate: negative length")
	}
	return v.add(func(_ context.Context, x []T) error {
		if len(x) != n {
			return issue(CodeInvalidValue, "collection.len", n, len(x))
		}
		return nil
	})
}
func (v SliceValidator[T]) Unique() SliceValidator[T] {
	return v.add(func(ctx context.Context, x []T) error {
		typ := reflect.TypeFor[T]()
		fast := deepEqualMatchesComparable(typ)
		if !fast && len(x) > MaxDeepUniqueItems {
			return issue(CodeUniqueLimit, "unique_limit", MaxDeepUniqueItems, len(x))
		}
		if fast {
			seen := make(map[any]struct{}, len(x))
			for _, item := range x {
				if err := ctx.Err(); err != nil {
					return err
				}
				key := any(item)
				if _, exists := seen[key]; exists {
					return issue(CodeInvalidValue, "unique", nil, item)
				}
				seen[key] = struct{}{}
			}
			return nil
		}
		for i := 0; i < len(x); i++ {
			for j := 0; j < i; j++ {
				if err := ctx.Err(); err != nil {
					return err
				}
				if reflect.DeepEqual(x[i], x[j]) {
					return issue(CodeInvalidValue, "unique", nil, x[i])
				}
			}
		}
		return nil
	})
}

func deepEqualMatchesComparable(t reflect.Type) bool {
	if t == nil || !t.Comparable() {
		return false
	}
	switch t.Kind() {
	case reflect.Pointer, reflect.Interface:
		return false
	case reflect.Array:
		return deepEqualMatchesComparable(t.Elem())
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			if !deepEqualMatchesComparable(t.Field(i).Type) {
				return false
			}
		}
	}
	return true
}
func (v SliceValidator[T]) Validate(x []T) error { return v.ValidateContext(context.Background(), x) }
func (v SliceValidator[T]) ValidateContext(ctx context.Context, x []T) error {
	return v.runAll(ctx, x, false)
}
func (v SliceValidator[T]) ValidateFirst(x []T) error {
	return v.ValidateFirstContext(context.Background(), x)
}
func (v SliceValidator[T]) ValidateFirstContext(ctx context.Context, x []T) error {
	return v.runAll(ctx, x, true)
}
func (v SliceValidator[T]) runAll(ctx context.Context, x []T, first bool) error {
	// The zero value is writable from outside the package, so a nil inner is
	// reachable. Report it as the configuration mistake it is instead of
	// dereferencing nil — and do so regardless of the data, so the same
	// validator cannot panic for one value and silently pass for another.
	if v.inner == nil {
		panic("validate: SliceValidator must be built with validate.Slice")
	}
	var issues []Issue
	if err := v.base.run(ctx, x, first); err != nil {
		issues = customIssue(err)
		if first {
			return err
		}
	}
	for i, item := range x {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := runValidator(ctx, v.inner, item, first)
		if err != nil {
			var stop bool
			issues, stop = appendIssues(issues, prefix(withLabel(customIssue(err), v.base.label), IndexPath(i)), first)
			if stop {
				return finish(ctx, issues)
			}
		}
	}
	return finish(ctx, issues)
}
func (v SliceValidator[T]) Refine(fns ...func([]T) error) SliceValidator[T] {
	for _, fn := range fns {
		if fn == nil {
			panic("validate: nil refine")
		}
		f := fn
		v.base = v.base.add(func(_ context.Context, x []T) error { return f(x) })
	}
	return v
}
func (v SliceValidator[T]) RefineContext(fns ...func(context.Context, []T) error) SliceValidator[T] {
	for _, fn := range fns {
		if fn == nil {
			panic("validate: nil refine")
		}
		v.base = v.base.add(fn)
	}
	return v
}
func (v SliceValidator[T]) And(vs ...Validator[[]T]) SliceValidator[T] {
	v.base = v.base.andAll(vs...)
	return v
}
func (v SliceValidator[T]) Label(s string) SliceValidator[T] {
	v.base = v.base.clone()
	v.base.label = s
	return v
}
