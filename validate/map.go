package validate

import (
	"context"

	"github.com/rhevorn/shape/internal/maporder"
)

// MapValidator validates a map and each key and value in stable key order.
type MapValidator[K MapKey, V any] struct {
	base[map[K]V]
	key   Validator[K]
	value Validator[V]
}

func (v MapValidator[K, V]) add(fn check[map[K]V]) MapValidator[K, V] {
	v.base = v.base.add(fn)
	return v
}

// NotNull rejects nil maps.
func (v MapValidator[K, V]) NotNull() MapValidator[K, V] {
	return v.add(func(_ context.Context, x map[K]V) error {
		if x == nil {
			return issueError(CodeInvalidValue, "not_null", nil, x)
		}
		return nil
	})
}

// NotEmpty rejects nil and zero-length maps.
func (v MapValidator[K, V]) NotEmpty() MapValidator[K, V] {
	return v.add(func(_ context.Context, x map[K]V) error {
		if len(x) == 0 {
			return issueError(CodeInvalidValue, "not_empty.collection", nil, x)
		}
		return nil
	})
}

// Min requires at least n entries.
func (v MapValidator[K, V]) Min(n int) MapValidator[K, V] {
	if n < 0 {
		panic("validate: negative length")
	}
	return v.add(func(_ context.Context, x map[K]V) error {
		if len(x) < n {
			return issueError(CodeTooSmall, "too_small.collection", n, len(x))
		}
		return nil
	})
}

// Max allows at most n entries.
func (v MapValidator[K, V]) Max(n int) MapValidator[K, V] {
	if n < 0 {
		panic("validate: negative length")
	}
	return v.add(func(_ context.Context, x map[K]V) error {
		if len(x) > n {
			return issueError(CodeTooBig, "too_big.collection", n, len(x))
		}
		return nil
	})
}

// Len requires exactly n entries.
func (v MapValidator[K, V]) Len(n int) MapValidator[K, V] {
	if n < 0 {
		panic("validate: negative length")
	}
	return v.add(func(_ context.Context, x map[K]V) error {
		if len(x) != n {
			return issueError(CodeInvalidValue, "collection.len", n, len(x))
		}
		return nil
	})
}

// Validate collects issues using a background context.
func (v MapValidator[K, V]) Validate(x map[K]V) error {
	return v.ValidateContext(context.Background(), x)
}

// ValidateContext collects issues and observes cancellation.
func (v MapValidator[K, V]) ValidateContext(ctx context.Context, x map[K]V) error {
	return v.validateMode(ctx, x, collectAll)
}

// ValidateFirst stops after the first issue.
func (v MapValidator[K, V]) ValidateFirst(x map[K]V) error {
	return v.ValidateFirstContext(context.Background(), x)
}

// ValidateFirstContext stops after the first issue and observes cancellation.
func (v MapValidator[K, V]) ValidateFirstContext(ctx context.Context, x map[K]V) error {
	return v.validateMode(ctx, x, stopAtFirst)
}
func (v MapValidator[K, V]) validateMode(ctx context.Context, x map[K]V, mode validationMode) error {
	if v.key == nil || v.value == nil {
		panic("validate: MapValidator must be built with validate.Map")
	}
	var issues []Issue
	if err := v.base.run(ctx, x, mode); err != nil {
		issues = customIssue(err)
		if mode == stopAtFirst {
			return err
		}
	}
	keys, err := maporder.Sorted(ctx, x)
	if err != nil {
		return err
	}
	for _, k := range keys {
		if err := ctx.Err(); err != nil {
			return err
		}
		segment := MapKeyPath(k)
		keyErr := runValidator(ctx, v.key, k, mode)
		if keyErr != nil {
			var stop bool
			issues, stop = appendIssues(issues, prefix(withLabel(customIssue(keyErr), mapPartLabel(v.base.label, "key")), segment), mode == stopAtFirst)
			if stop {
				return finish(ctx, issues)
			}
		}
		valueErr := runValidator(ctx, v.value, x[k], mode)
		if valueErr != nil {
			var stop bool
			issues, stop = appendIssues(issues, prefix(withLabel(customIssue(valueErr), mapPartLabel(v.base.label, "value")), segment), mode == stopAtFirst)
			if stop {
				return finish(ctx, issues)
			}
		}
	}
	return finish(ctx, issues)
}

// Refine appends custom validation rules.
func (v MapValidator[K, V]) Refine(fns ...func(map[K]V) error) MapValidator[K, V] {
	for _, fn := range fns {
		if fn == nil {
			panic("validate: nil refine")
		}
		f := fn
		v.base = v.base.add(func(_ context.Context, x map[K]V) error { return f(x) })
	}
	return v
}

// RefineContext appends context-aware custom validation rules.
func (v MapValidator[K, V]) RefineContext(fns ...func(context.Context, map[K]V) error) MapValidator[K, V] {
	for _, fn := range fns {
		if fn == nil {
			panic("validate: nil refine")
		}
		v.base = v.base.add(fn)
	}
	return v
}

// And appends validators that run after this validator's rules.
func (v MapValidator[K, V]) And(vs ...Validator[map[K]V]) MapValidator[K, V] {
	v.base = v.base.andAll(vs...)
	return v
}

// mapPartLabel supplies a label so a map key failure and a map value
// failure for the same entry can be told apart. Both share one path segment
// (the key), so without this the two issues are byte-identical.
func mapPartLabel(label, part string) string {
	if label != "" {
		return label + "." + part
	}
	return part
}

// Label sets a parent label; key and value issues append their role.
func (v MapValidator[K, V]) Label(s string) MapValidator[K, V] {
	v.base = v.base.clone()
	v.base.label = s
	return v
}
