package validate

import "context"

// Numeric is the set of supported signed, unsigned, and floating-point types.
type Numeric interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~float32 | ~float64
}

// NumberValidator validates values of a supported numeric type.
type NumberValidator[N Numeric] struct{ base[N] }

func (v NumberValidator[N]) add(fn check[N]) NumberValidator[N] { v.base = v.base.add(fn); return v }

// Validate collects issues using a background context.
func (v NumberValidator[N]) Validate(value N) error { return v.base.Validate(value) }

// ValidateContext collects issues and observes cancellation.
func (v NumberValidator[N]) ValidateContext(ctx context.Context, value N) error {
	return v.base.ValidateContext(ctx, value)
}

// ValidateFirst stops after the first issue.
func (v NumberValidator[N]) ValidateFirst(value N) error { return v.base.ValidateFirst(value) }

// ValidateFirstContext stops after the first issue and observes cancellation.
func (v NumberValidator[N]) ValidateFirstContext(ctx context.Context, value N) error {
	return v.base.ValidateFirstContext(ctx, value)
}

// Min requires a value greater than or equal to n.
func (v NumberValidator[N]) Min(n N) NumberValidator[N] { return v.Gte(n) }

// Max requires a value less than or equal to n.
func (v NumberValidator[N]) Max(n N) NumberValidator[N] { return v.Lte(n) }

// Gt requires a value strictly greater than n.
func (v NumberValidator[N]) Gt(n N) NumberValidator[N] {
	return v.add(func(_ context.Context, x N) error {
		if !(x > n) {
			return issueError(CodeTooSmall, "number.gt", n, x)
		}
		return nil
	})
}

// Gte requires a value greater than or equal to n.
func (v NumberValidator[N]) Gte(n N) NumberValidator[N] {
	return v.add(func(_ context.Context, x N) error {
		if !(x >= n) {
			return issueError(CodeTooSmall, "number.gte", n, x)
		}
		return nil
	})
}

// Lt requires a value strictly less than n.
func (v NumberValidator[N]) Lt(n N) NumberValidator[N] {
	return v.add(func(_ context.Context, x N) error {
		if !(x < n) {
			return issueError(CodeTooBig, "number.lt", n, x)
		}
		return nil
	})
}

// Lte requires a value less than or equal to n.
func (v NumberValidator[N]) Lte(n N) NumberValidator[N] {
	return v.add(func(_ context.Context, x N) error {
		if !(x <= n) {
			return issueError(CodeTooBig, "number.lte", n, x)
		}
		return nil
	})
}

// Between requires an inclusive value between a and b.
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
			return issueError(CodeInvalidValue, "number.between", []N{a, b}, x)
		}
		return nil
	})
}

// OneOf requires equality with one listed value.
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
		return issueError(CodeInvalidEnum, "invalid_enum", values, x)
	})
}

// Positive requires a value greater than zero.
func (v NumberValidator[N]) Positive() NumberValidator[N] { var z N; return v.Gt(z) }

// Negative requires a value less than zero.
func (v NumberValidator[N]) Negative() NumberValidator[N] { var z N; return v.Lt(z) }

// NonNegative requires a value greater than or equal to zero.
func (v NumberValidator[N]) NonNegative() NumberValidator[N] { var z N; return v.Gte(z) }

// Refine appends custom validation rules.
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

// RefineContext appends context-aware custom validation rules.
func (v NumberValidator[N]) RefineContext(fns ...func(context.Context, N) error) NumberValidator[N] {
	for _, fn := range fns {
		if fn == nil {
			panic("validate: nil refine")
		}
		v = v.add(fn)
	}
	return v
}

// And appends validators that run after this validator's rules.
func (v NumberValidator[N]) And(vs ...Validator[N]) NumberValidator[N] {
	v.base = v.base.andAll(vs...)
	return v
}

// Label sets the human-readable label on otherwise unlabeled issues.
func (v NumberValidator[N]) Label(s string) NumberValidator[N] {
	v.base = v.base.clone()
	v.base.label = s
	return v
}
