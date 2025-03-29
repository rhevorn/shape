package validate

import "context"

type ValueValidator[T any] struct{ base[T] }

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
func (v ValueValidator[T]) RefineContext(fns ...func(context.Context, T) error) ValueValidator[T] {
	for _, fn := range fns {
		if fn == nil {
			panic("validate: nil refine")
		}
		v.base = v.base.add(fn)
	}
	return v
}
func (v ValueValidator[T]) And(vs ...Validator[T]) ValueValidator[T] {
	v.base = v.base.andAll(vs...)
	return v
}
func (v ValueValidator[T]) Label(s string) ValueValidator[T] {
	v.base = v.base.clone()
	v.base.label = s
	return v
}
