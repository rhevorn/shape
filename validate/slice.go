package validate

import (
	"context"
	"reflect"
)

// SliceValidator validates a slice and each of its elements.
type SliceValidator[T any] struct {
	base[[]T]
	inner Validator[T]
}

func (v SliceValidator[T]) add(fn check[[]T]) SliceValidator[T] { v.base = v.base.add(fn); return v }

// NotNull rejects nil slices.
func (v SliceValidator[T]) NotNull() SliceValidator[T] {
	return v.add(func(_ context.Context, x []T) error {
		if x == nil {
			return issueError(CodeInvalidValue, "not_null", nil, x)
		}
		return nil
	})
}

// NotEmpty rejects nil and zero-length slices.
func (v SliceValidator[T]) NotEmpty() SliceValidator[T] {
	return v.add(func(_ context.Context, x []T) error {
		if len(x) == 0 {
			return issueError(CodeInvalidValue, "not_empty.collection", nil, x)
		}
		return nil
	})
}

// Min requires at least n elements.
func (v SliceValidator[T]) Min(n int) SliceValidator[T] {
	if n < 0 {
		panic("validate: negative length")
	}
	return v.add(func(_ context.Context, x []T) error {
		if len(x) < n {
			return issueError(CodeTooSmall, "too_small.collection", n, len(x))
		}
		return nil
	})
}

// Max allows at most n elements.
func (v SliceValidator[T]) Max(n int) SliceValidator[T] {
	if n < 0 {
		panic("validate: negative length")
	}
	return v.add(func(_ context.Context, x []T) error {
		if len(x) > n {
			return issueError(CodeTooBig, "too_big.collection", n, len(x))
		}
		return nil
	})
}

// Len requires exactly n elements.
func (v SliceValidator[T]) Len(n int) SliceValidator[T] {
	if n < 0 {
		panic("validate: negative length")
	}
	return v.add(func(_ context.Context, x []T) error {
		if len(x) != n {
			return issueError(CodeInvalidValue, "collection.len", n, len(x))
		}
		return nil
	})
}

// Unique rejects duplicate elements using value equality.
func (v SliceValidator[T]) Unique() SliceValidator[T] {
	return v.add(func(ctx context.Context, x []T) error {
		typ := reflect.TypeFor[T]()
		fast := deepEqualMatchesComparable(typ)
		if !fast && len(x) > MaxDeepUniqueItems {
			return issueError(CodeUniqueLimit, "unique_limit", MaxDeepUniqueItems, len(x))
		}
		if fast {
			seen := make(map[any]struct{}, len(x))
			for _, item := range x {
				if err := ctx.Err(); err != nil {
					return err
				}
				key := any(item)
				if _, exists := seen[key]; exists {
					return issueError(CodeInvalidValue, "unique", nil, item)
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
					return issueError(CodeInvalidValue, "unique", nil, x[i])
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

// Validate collects issues using a background context.
func (v SliceValidator[T]) Validate(x []T) error { return v.ValidateContext(context.Background(), x) }

// ValidateContext collects issues and observes cancellation.
func (v SliceValidator[T]) ValidateContext(ctx context.Context, x []T) error {
	return v.validateMode(ctx, x, collectAll)
}

// ValidateFirst stops after the first issue.
func (v SliceValidator[T]) ValidateFirst(x []T) error {
	return v.ValidateFirstContext(context.Background(), x)
}

// ValidateFirstContext stops after the first issue and observes cancellation.
func (v SliceValidator[T]) ValidateFirstContext(ctx context.Context, x []T) error {
	return v.validateMode(ctx, x, stopAtFirst)
}
func (v SliceValidator[T]) validateMode(ctx context.Context, x []T, mode validationMode) error {
	// The zero value is writable from outside the package, so a nil inner is
	// reachable. Report it as the configuration mistake it is instead of
	// dereferencing nil — and do so regardless of the data, so the same
	// validator cannot panic for one value and silently pass for another.
	if v.inner == nil {
		panic("validate: SliceValidator must be built with validate.Slice")
	}
	var issues []Issue
	if err := v.base.run(ctx, x, mode); err != nil {
		issues = customIssue(err)
		if mode == stopAtFirst {
			return err
		}
	}
	for i, item := range x {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := runValidator(ctx, v.inner, item, mode)
		if err != nil {
			var stop bool
			issues, stop = appendIssues(issues, prefix(withLabel(customIssue(err), v.base.label), IndexPath(i)), mode == stopAtFirst)
			if stop {
				return finish(ctx, issues)
			}
		}
	}
	return finish(ctx, issues)
}

// Refine appends custom validation rules.
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

// RefineContext appends context-aware custom validation rules.
func (v SliceValidator[T]) RefineContext(fns ...func(context.Context, []T) error) SliceValidator[T] {
	for _, fn := range fns {
		if fn == nil {
			panic("validate: nil refine")
		}
		v.base = v.base.add(fn)
	}
	return v
}

// And appends validators that run after this validator's rules.
func (v SliceValidator[T]) And(vs ...Validator[[]T]) SliceValidator[T] {
	v.base = v.base.andAll(vs...)
	return v
}

// Label sets the human-readable label on otherwise unlabeled issues.
func (v SliceValidator[T]) Label(s string) SliceValidator[T] {
	v.base = v.base.clone()
	v.base.label = s
	return v
}
