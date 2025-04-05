package validate

import "context"

// ValueValidator validates any T with custom rules and composed validators.
type ValueValidator[T any] struct{ base[T] }

// Validate collects issues using a background context.
func (v ValueValidator[T]) Validate(value T) error { return v.base.Validate(value) }

// ValidateContext collects issues and observes cancellation.
func (v ValueValidator[T]) ValidateContext(ctx context.Context, value T) error {
	return v.base.ValidateContext(ctx, value)
}

// ValidateFirst stops after the first issue.
func (v ValueValidator[T]) ValidateFirst(value T) error { return v.base.ValidateFirst(value) }

// ValidateFirstContext stops after the first issue and observes cancellation.
func (v ValueValidator[T]) ValidateFirstContext(ctx context.Context, value T) error {
	return v.base.ValidateFirstContext(ctx, value)
}

// Refine appends custom validation rules.
func (v ValueValidator[T]) Refine(fns ...func(T) error) ValueValidator[T] {
	for _, fn := range fns {
		if fn == nil {
			panic("validate: nil refine")
		}
		f := fn
		v.base = v.base.add(func(_ context.Context, x T) error { return f(x) })
	}
	return v
}

// RefineContext appends context-aware custom validation rules.
func (v ValueValidator[T]) RefineContext(fns ...func(context.Context, T) error) ValueValidator[T] {
	for _, fn := range fns {
		if fn == nil {
			panic("validate: nil refine")
		}
		v.base = v.base.add(fn)
	}
	return v
}

// And appends validators that run after this validator's rules.
func (v ValueValidator[T]) And(vs ...Validator[T]) ValueValidator[T] {
	v.base = v.base.andAll(vs...)
	return v
}

// Label sets the human-readable label on otherwise unlabeled issues.
func (v ValueValidator[T]) Label(s string) ValueValidator[T] {
	v.base = v.base.clone()
	v.base.label = s
	return v
}
