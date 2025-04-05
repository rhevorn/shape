package transform

import "context"

// Numeric is the set of supported signed, unsigned, and floating-point types.
type Numeric interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~float32 | ~float64
}

// NumberTransformer transforms a numeric value of type N.
type NumberTransformer[N Numeric] struct{ ValueTransformer[N] }

// IfZero replaces numeric zero with v.
func (t NumberTransformer[N]) IfZero(v N) NumberTransformer[N] {
	t.ValueTransformer = t.ValueTransformer.IfZero(v)
	return t
}

// Apply appends custom numeric transformations.
func (t NumberTransformer[N]) Apply(fns ...func(N) (N, error)) NumberTransformer[N] {
	t.ValueTransformer = t.ValueTransformer.Apply(fns...)
	return t
}

// ApplyContext appends context-aware custom numeric transformations.
func (t NumberTransformer[N]) ApplyContext(fns ...func(context.Context, N) (N, error)) NumberTransformer[N] {
	t.ValueTransformer = t.ValueTransformer.ApplyContext(fns...)
	return t
}
