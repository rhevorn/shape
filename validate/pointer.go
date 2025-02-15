package validate

import "context"

type PointerValidator[T any] struct {
	base[*T]
	inner Validator[T]
}

func (v PointerValidator[T]) add(fn check[*T]) PointerValidator[T] { v.base = v.base.add(fn); return v }
func (v PointerValidator[T]) NotNull() PointerValidator[T] {
	return v.add(func(_ context.Context, x *T) error {
		if x == nil {
			return issue(CodeInvalidValue, "not_null", nil, x)
		}
		return nil
	})
}
func (v PointerValidator[T]) NotEmpty() PointerValidator[T] {
	return v.add(func(_ context.Context, x *T) error {
		if x == nil {
			return issue(CodeInvalidValue, "not_empty.pointer", nil, x)
		}
		return nil
	})
}
func (v PointerValidator[T]) Validate(x *T) error { return v.ValidateContext(context.Background(), x) }
func (v PointerValidator[T]) ValidateContext(ctx context.Context, x *T) error {
	return v.runAll(ctx, x, false)
}
func (v PointerValidator[T]) ValidateFirst(x *T) error {
	return v.ValidateFirstContext(context.Background(), x)
}
func (v PointerValidator[T]) ValidateFirstContext(ctx context.Context, x *T) error {
	return v.runAll(ctx, x, true)
}
func (v PointerValidator[T]) runAll(ctx context.Context, x *T, first bool) error {
	var issues []Issue
	if err := v.base.run(ctx, x, first); err != nil {
		issues = customIssue(err)
		if first {
			return finish(ctx, issues)
		}
	}
	if x == nil {
		return finish(ctx, issues)
	}
	if err := runValidator(ctx, v.inner, *x, first); err != nil {
		var stop bool
		issues, stop = appendIssues(issues, withLabel(customIssue(err), v.base.label), first)
		if stop {
			return finish(ctx, issues)
		}
	}
	return finish(ctx, issues)
}
func (v PointerValidator[T]) Refine(fns ...func(*T) error) PointerValidator[T] {
	for _, fn := range fns {
		if fn == nil {
			panic("validate: nil refine")
		}
		f := fn
		v.base = v.base.add(func(_ context.Context, x *T) error { return f(x) })
	}
	return v
}
func (v PointerValidator[T]) RefineContext(fns ...func(context.Context, *T) error) PointerValidator[T] {
	for _, fn := range fns {
		if fn == nil {
			panic("validate: nil refine")
		}
		v.base = v.base.add(fn)
	}
	return v
}
func (v PointerValidator[T]) And(vs ...Validator[*T]) Validator[*T] {
	return joinValidators[*T](v, vs...)
}
func (v PointerValidator[T]) Label(s string) PointerValidator[T] {
	v.base = v.base.clone()
	v.base.label = s
	return v
}
