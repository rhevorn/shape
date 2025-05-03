package validate

import "context"

// Validator validates T either exhaustively or until the first issue.
type Validator[T any] interface {
	Validate(T) error
	ValidateContext(context.Context, T) error
	ValidateFirst(T) error
	ValidateFirstContext(context.Context, T) error
}

type check[T any] func(context.Context, T) error

type validationMode uint8

const (
	collectAll validationMode = iota
	stopAtFirst
)

type validationStep[T any] struct {
	check check[T]
	other Validator[T]
}

type base[T any] struct {
	steps []validationStep[T]
	label string
}

func (b base[T]) clone() base[T] {
	b.steps = append([]validationStep[T](nil), b.steps...)
	return b
}
func (b base[T]) add(fn check[T]) base[T] {
	b = b.clone()
	b.steps = append(b.steps, validationStep[T]{check: fn})
	return b
}

// Validate collects issues using a background context.
func (b base[T]) Validate(v T) error { return b.ValidateContext(context.Background(), v) }

// ValidateContext collects issues and observes cancellation.
func (b base[T]) ValidateContext(ctx context.Context, v T) error { return b.run(ctx, v, collectAll) }

// ValidateFirst stops after the first issue.
func (b base[T]) ValidateFirst(v T) error { return b.ValidateFirstContext(context.Background(), v) }

// ValidateFirstContext stops after the first issue and observes cancellation.
func (b base[T]) ValidateFirstContext(ctx context.Context, v T) error {
	return b.run(ctx, v, stopAtFirst)
}
func (b base[T]) run(ctx context.Context, v T, mode validationMode) error {
	if ctx == nil {
		panic("validate: nil context")
	}
	var issues []Issue
	for _, step := range b.steps {
		if err := ctx.Err(); err != nil {
			return err
		}
		var err error
		if step.check != nil {
			err = step.check(ctx, v)
		} else {
			err = runValidator(ctx, step.other, v, mode)
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil {
			var stop bool
			issues, stop = appendIssues(issues, withLabel(customIssue(err), b.label), mode == stopAtFirst)
			if stop {
				return finish(ctx, issues)
			}
		}
	}

	return finish(ctx, issues)
}

// Refine appends custom validation rules.
func (b base[T]) Refine(fns ...func(T) error) Validator[T] {
	for _, fn := range fns {
		if fn == nil {
			panic("validate: nil refine")
		}
		f := fn
		b = b.add(func(_ context.Context, v T) error { return f(v) })
	}
	return b
}

// RefineContext appends context-aware custom validation rules.
func (b base[T]) RefineContext(fns ...func(context.Context, T) error) Validator[T] {
	for _, fn := range fns {
		if fn == nil {
			panic("validate: nil refine")
		}
		b = b.add(fn)
	}
	return b
}

// andAll appends sibling validators to a clone of the receiver while retaining
// the receiver's checks and label.
func (b base[T]) andAll(vs ...Validator[T]) base[T] {
	b = b.clone()
	for _, v := range vs {
		if v == nil {
			panic("validate: nil validator")
		}
		b.steps = append(b.steps, validationStep[T]{other: v})
	}
	return b
}
func runValidator[T any](ctx context.Context, v Validator[T], value T, mode validationMode) error {
	if mode == stopAtFirst {
		return v.ValidateFirstContext(ctx, value)
	}
	return v.ValidateContext(ctx, value)
}
