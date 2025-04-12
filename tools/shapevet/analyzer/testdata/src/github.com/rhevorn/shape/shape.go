package shape

import (
	"context"
	"io"
	"time"

	shapetypes "github.com/rhevorn/shape/types"
)

type Schema[T any] interface {
	Transform(T) (T, error)
}

type JSONOptions struct{}

func Struct[T any]() int { return 0 }

func BindJSON[T any](target *T, source []byte, _ ...JSONOptions) error         { return nil }
func BindJSONContext[T any](context.Context, *T, []byte, ...JSONOptions) error { return nil }
func BindJSONReader[T any](*T, io.Reader, ...JSONOptions) error                { return nil }
func BindJSONReaderContext[T any](context.Context, *T, io.Reader, ...JSONOptions) error {
	return nil
}

type FieldSpec interface{ field() }

type ValueSpec[T any] struct{}

func (ValueSpec[T]) field()                   {}
func (ValueSpec[T]) Transform(v T) (T, error) { return v, nil }

type StringSpec struct{}

func (StringSpec) field()                           {}
func (StringSpec) Trim(...string) StringSpec        { return StringSpec{} }
func (StringSpec) NotEmpty() StringSpec             { return StringSpec{} }
func (StringSpec) Transform(string) (string, error) { return "", nil }

type Numeric interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}

type NumberSpec[N Numeric] struct{}

func (NumberSpec[N]) field()                 {}
func (NumberSpec[N]) Min(N) NumberSpec[N]    { return NumberSpec[N]{} }
func (NumberSpec[N]) Transform(N) (N, error) { var zero N; return zero, nil }

type PointerSpec[T any] struct{}

func (PointerSpec[T]) field()                   {}
func (PointerSpec[T]) Transform(*T) (*T, error) { return nil, nil }

type SliceSpec[T any] struct{}

func (SliceSpec[T]) field()                     {}
func (SliceSpec[T]) Transform([]T) ([]T, error) { return nil, nil }

type MapSpec[K comparable, V any] struct{}

func (MapSpec[K, V]) field()                             {}
func (MapSpec[K, V]) Transform(map[K]V) (map[K]V, error) { return nil, nil }

func Value[T any](...string) ValueSpec[T]                { return ValueSpec[T]{} }
func String(...string) StringSpec                        { return StringSpec{} }
func Number[N Numeric](...string) NumberSpec[N]          { return NumberSpec[N]{} }
func Int(...string) NumberSpec[int]                      { return NumberSpec[int]{} }
func Int64(...string) NumberSpec[int64]                  { return NumberSpec[int64]{} }
func Float64(...string) NumberSpec[float64]              { return NumberSpec[float64]{} }
func Duration(...string) NumberSpec[shapetypes.Duration] { return NumberSpec[shapetypes.Duration]{} }
func Bool(...string) ValueSpec[bool]                     { return ValueSpec[bool]{} }
func Time(...string) ValueSpec[time.Time]                { return ValueSpec[time.Time]{} }
func Pointer[T any](string, Schema[T]) PointerSpec[T]    { return PointerSpec[T]{} }
func Slice[T any](string, Schema[T]) SliceSpec[T]        { return SliceSpec[T]{} }
func Map[K comparable, V any](string, Schema[K], Schema[V]) MapSpec[K, V] {
	return MapSpec[K, V]{}
}
func Field[T any](string, Schema[T]) FieldSpec { return ValueSpec[T]{} }
func New[T any](...FieldSpec) int              { return 0 }
