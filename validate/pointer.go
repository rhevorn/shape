package validate

import "context"

// PointerValidator validates an optional *T and its non-nil value.
type PointerValidator[T any] struct {
	base[*T]
	inner Validator[T]
}

func (v PointerValidator[T]) add(fn check[*T]) PointerValidator[T] { v.base = v.base.add(fn); return v }

// NotNull rejects nil pointers.
func (v PointerValidator[T]) NotNull() PointerValidator[T] {
	return v.add(func(_ context.Context, x *T) error {
		if x == nil {
			return issueError(CodeInvalidValue, "not_null", nil, x)
		}
		return nil
	})
}

// NotEmpty rejects nil pointers.
func (v PointerValidator[T]) NotEmpty() PointerValidator[T] {
	return v.add(func(_ context.Context, x *T) error {
		if x == nil {
			return issueError(CodeInvalidValue, "not_empty.pointer", nil, x)
		}
		return nil
	})
}

// Validate collects issues using a background context.
func (v PointerValidator[T]) Validate(x *T) error { return v.ValidateContext(context.Background(), x) }

// ValidateContext collects issues and observes cancellation.
func (v PointerValidator[T]) ValidateContext(ctx context.Context, x *T) error {
	return v.validateMode(ctx, x, collectAll)
}

// ValidateFirst stops after the first issue.
func (v PointerValidator[T]) ValidateFirst(x *T) error {
	return v.ValidateFirstContext(context.Background(), x)
}

// ValidateFirstContext stops after the first issue and observes cancellation.
func (v PointerValidator[T]) ValidateFirstContext(ctx context.Context, x *T) error {
	return v.validateMode(ctx, x, stopAtFirst)
}
func (v PointerValidator[T]) validateMode(ctx context.Context, x *T, mode validationMode) error {
	// Guard before the nil-pointer short circuit below: otherwise a zero-value
	// validator reports success for nil and dereferences nil for anything else.
	if v.inner == nil {
		panic("validate: PointerValidator must be built with validate.Pointer")
	}
	var issues []Issue
	if err := v.base.run(ctx, x, mode); err != nil {
		issues = customIssue(err)
		if mode == stopAtFirst {
			return finish(ctx, issues)
		}
	}
	if x == nil {
		return finish(ctx, issues)
	}
	if err := runValidator(ctx, v.inner, *x, mode); err != nil {
		var stop bool
		issues, stop = appendIssues(issues, withLabel(customIssue(err), v.base.label), mode == stopAtFirst)
		if stop {
			return finish(ctx, issues)
		}
	}
	return finish(ctx, issues)
}

// Refine appends custom validation rules.
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

// RefineContext appends context-aware custom validation rules.
func (v PointerValidator[T]) RefineContext(fns ...func(context.Context, *T) error) PointerValidator[T] {
	for _, fn := range fns {
		if fn == nil {
			panic("validate: nil refine")
		}
		v.base = v.base.add(fn)
	}
	return v
}

// And appends validators that run after this validator's rules.
func (v PointerValidator[T]) And(vs ...Validator[*T]) PointerValidator[T] {
	v.base = v.base.andAll(vs...)
	return v
}

// Label sets the human-readable label on otherwise unlabeled issues.
func (v PointerValidator[T]) Label(s string) PointerValidator[T] {
	v.base = v.base.clone()
	v.base.label = s
	return v
}
