package validate

import "context"

type Numeric interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~float32 | ~float64
}
type NumberValidator[N Numeric] struct{ base[N] }

func (v NumberValidator[N]) add(fn check[N]) NumberValidator[N] { v.base = v.base.add(fn); return v }
func (v NumberValidator[N]) Min(n N) NumberValidator[N]         { return v.Gte(n) }
func (v NumberValidator[N]) Max(n N) NumberValidator[N]         { return v.Lte(n) }
func (v NumberValidator[N]) Gt(n N) NumberValidator[N] {
	return v.add(func(_ context.Context, x N) error {
		if !(x > n) {
			return issue(CodeTooSmall, "number.gt", n, x)
		}
		return nil
	})
}
func (v NumberValidator[N]) Gte(n N) NumberValidator[N] {
	return v.add(func(_ context.Context, x N) error {
		if !(x >= n) {
			return issue(CodeTooSmall, "number.gte", n, x)
		}
		return nil
	})
}
func (v NumberValidator[N]) Lt(n N) NumberValidator[N] {
	return v.add(func(_ context.Context, x N) error {
		if !(x < n) {
			return issue(CodeTooBig, "number.lt", n, x)
		}
		return nil
	})
}
func (v NumberValidator[N]) Lte(n N) NumberValidator[N] {
	return v.add(func(_ context.Context, x N) error {
		if !(x <= n) {
			return issue(CodeTooBig, "number.lte", n, x)
		}
		return nil
	})
}
func (v NumberValidator[N]) Between(a, b N) NumberValidator[N] {
	// a != a is false for every integer kind and true only for a NaN float.
	// Without it a NaN bound makes both comparisons below false, turning the
	// rule into a silent no-op. Infinities stay legal: Between(0, +Inf) is a
	// meaningful "no upper bound".
	if a > b || a != a || b != b {
		panic("validate: invalid range")
	}
	return v.add(func(_ context.Context, x N) error {
		if x < a || x > b {
			return issue(CodeInvalidValue, "number.between", []N{a, b}, x)
		}
		return nil
	})
}
func (v NumberValidator[N]) OneOf(values ...N) NumberValidator[N] {
	if len(values) == 0 {
		panic("validate: OneOf requires values")
	}
	values = append([]N(nil), values...)
	return v.add(func(_ context.Context, x N) error {
		for _, n := range values {
			if x == n {
				return nil
			}
		}
		return issue(CodeInvalidEnum, "invalid_enum", values, x)
	})
}
func (v NumberValidator[N]) Positive() NumberValidator[N]    { var z N; return v.Gt(z) }
func (v NumberValidator[N]) Negative() NumberValidator[N]    { var z N; return v.Lt(z) }
func (v NumberValidator[N]) NonNegative() NumberValidator[N] { var z N; return v.Gte(z) }
func (v NumberValidator[N]) Refine(fns ...func(N) error) NumberValidator[N] {
	for _, fn := range fns {
		if fn == nil {
			panic("validate: nil refine")
		}
		f := fn
		v = v.add(func(_ context.Context, x N) error { return f(x) })
	}
	return v
}
func (v NumberValidator[N]) RefineContext(fns ...func(context.Context, N) error) NumberValidator[N] {
	for _, fn := range fns {
		if fn == nil {
			panic("validate: nil refine")
		}
		v = v.add(fn)
	}
	return v
}
func (v NumberValidator[N]) And(vs ...Validator[N]) NumberValidator[N] {
	v.base = v.base.andAll(vs...)
	return v
}
func (v NumberValidator[N]) Label(s string) NumberValidator[N] {
	v.base = v.base.clone()
	v.base.label = s
	return v
}
