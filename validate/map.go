package validate

import (
	"context"
	"fmt"
	"reflect"
	"sort"
)

type MapValidator[K MapKey, V any] struct {
	base[map[K]V]
	key   Validator[K]
	value Validator[V]
}

func (v MapValidator[K, V]) add(fn check[map[K]V]) MapValidator[K, V] {
	v.base = v.base.add(fn)
	return v
}
func (v MapValidator[K, V]) NotNull() MapValidator[K, V] {
	return v.add(func(_ context.Context, x map[K]V) error {
		if x == nil {
			return issue(CodeInvalidValue, "not_null", nil, x)
		}
		return nil
	})
}
func (v MapValidator[K, V]) NotEmpty() MapValidator[K, V] {
	return v.add(func(_ context.Context, x map[K]V) error {
		if len(x) == 0 {
			return issue(CodeInvalidValue, "not_empty.collection", nil, x)
		}
		return nil
	})
}
func (v MapValidator[K, V]) Min(n int) MapValidator[K, V] {
	if n < 0 {
		panic("validate: negative length")
	}
	return v.add(func(_ context.Context, x map[K]V) error {
		if len(x) < n {
			return issue(CodeTooSmall, "too_small.collection", n, len(x))
		}
		return nil
	})
}
func (v MapValidator[K, V]) Max(n int) MapValidator[K, V] {
	if n < 0 {
		panic("validate: negative length")
	}
	return v.add(func(_ context.Context, x map[K]V) error {
		if len(x) > n {
			return issue(CodeTooBig, "too_big.collection", n, len(x))
		}
		return nil
	})
}
func (v MapValidator[K, V]) Len(n int) MapValidator[K, V] {
	if n < 0 {
		panic("validate: negative length")
	}
	return v.add(func(_ context.Context, x map[K]V) error {
		if len(x) != n {
			return issue(CodeInvalidValue, "collection.len", n, len(x))
		}
		return nil
	})
}
func (v MapValidator[K, V]) Validate(x map[K]V) error {
	return v.ValidateContext(context.Background(), x)
}
func (v MapValidator[K, V]) ValidateContext(ctx context.Context, x map[K]V) error {
	return v.runAll(ctx, x, false)
}
func (v MapValidator[K, V]) ValidateFirst(x map[K]V) error {
	return v.ValidateFirstContext(context.Background(), x)
}
func (v MapValidator[K, V]) ValidateFirstContext(ctx context.Context, x map[K]V) error {
	return v.runAll(ctx, x, true)
}
func (v MapValidator[K, V]) runAll(ctx context.Context, x map[K]V, first bool) error {
	if v.key == nil || v.value == nil {
		panic("validate: MapValidator must be built with validate.Map")
	}
	var issues []Issue
	if err := v.base.run(ctx, x, first); err != nil {
		issues = customIssue(err)
		if first {
			return err
		}
	}
	keys := make([]K, 0, len(x))
	for k := range x {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return lessMapKey(keys[i], keys[j]) })
	for _, k := range keys {
		if err := ctx.Err(); err != nil {
			return err
		}
		segment := FieldPath(fmt.Sprint(k))
		keyErr := runValidator(ctx, v.key, k, first)
		if keyErr != nil {
			var stop bool
			issues, stop = appendIssues(issues, prefix(withLabel(customIssue(keyErr), labelOr(v.base.label, "key")), segment), first)
			if stop {
				return finish(ctx, issues)
			}
		}
		valueErr := runValidator(ctx, v.value, x[k], first)
		if valueErr != nil {
			var stop bool
			issues, stop = appendIssues(issues, prefix(withLabel(customIssue(valueErr), labelOr(v.base.label, "value")), segment), first)
			if stop {
				return finish(ctx, issues)
			}
		}
	}
	return finish(ctx, issues)
}
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
func (v MapValidator[K, V]) RefineContext(fns ...func(context.Context, map[K]V) error) MapValidator[K, V] {
	for _, fn := range fns {
		if fn == nil {
			panic("validate: nil refine")
		}
		v.base = v.base.add(fn)
	}
	return v
}
func (v MapValidator[K, V]) And(vs ...Validator[map[K]V]) MapValidator[K, V] {
	v.base = v.base.andAll(vs...)
	return v
}

// labelOr supplies a default label so a map key failure and a map value
// failure for the same entry can be told apart. Both share one path segment
// (the key), so without this the two issues are byte-identical.
func labelOr(label, fallback string) string {
	if label != "" {
		return label
	}
	return fallback
}

func (v MapValidator[K, V]) Label(s string) MapValidator[K, V] {
	v.base = v.base.clone()
	v.base.label = s
	return v
}

func lessMapKey[K MapKey](a, b K) bool {
	av, bv := reflect.ValueOf(a), reflect.ValueOf(b)
	switch av.Kind() {
	case reflect.String:
		return av.String() < bv.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return av.Int() < bv.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return av.Uint() < bv.Uint()
	default:
		return av.Uint() < bv.Uint()
	}
}
