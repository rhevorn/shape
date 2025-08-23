package shape

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"time"

	shapetypes "github.com/rhevorn/shape/types"
)

type Schema[T any] interface {
	Transform(T) (T, error)
}

type JSONOptions struct{}

func FromTags[T any]() StructSpec[T] { return StructSpec[T]{} }

func BindJSON[T any](target *T, source []byte, _ ...JSONOptions) error         { return nil }
func BindJSONContext[T any](context.Context, *T, []byte, ...JSONOptions) error { return nil }
func BindJSONReader[T any](*T, io.Reader, ...JSONOptions) error                { return nil }
func BindJSONReaderContext[T any](context.Context, *T, io.Reader, ...JSONOptions) error {
	return nil
}

type FieldSpec interface{ field() }

type fieldSpec struct{}

func (fieldSpec) field() {}

type ValueSpec[T any] struct{}

func (ValueSpec[T]) Transform(v T) (T, error) { return v, nil }

type StringSpec struct{}

func (StringSpec) Trim(...string) StringSpec        { return StringSpec{} }
func (StringSpec) NotEmpty() StringSpec             { return StringSpec{} }
func (StringSpec) Transform(string) (string, error) { return "", nil }

type Numeric interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}

type NumberSpec[N Numeric] struct{}

func (NumberSpec[N]) Min(N) NumberSpec[N]    { return NumberSpec[N]{} }
func (NumberSpec[N]) Transform(N) (N, error) { var zero N; return zero, nil }

type PointerSpec[T any] struct{}

func (PointerSpec[T]) Transform(*T) (*T, error) { return nil, nil }

type SliceSpec[T any] struct{}

func (SliceSpec[T]) Transform([]T) ([]T, error) { return nil, nil }

type MapSpec[K comparable, V any] struct{}

func (MapSpec[K, V]) Transform(map[K]V) (map[K]V, error) { return nil, nil }

func Value[T any]() ValueSpec[T]                { return ValueSpec[T]{} }
func String() StringSpec                        { return StringSpec{} }
func Number[N Numeric]() NumberSpec[N]          { return NumberSpec[N]{} }
func Int() NumberSpec[int]                      { return NumberSpec[int]{} }
func Int64() NumberSpec[int64]                  { return NumberSpec[int64]{} }
func Float64() NumberSpec[float64]              { return NumberSpec[float64]{} }
func Duration() NumberSpec[shapetypes.Duration] { return NumberSpec[shapetypes.Duration]{} }
func Bool() ValueSpec[bool]                     { return ValueSpec[bool]{} }
func Time() ValueSpec[time.Time]                { return ValueSpec[time.Time]{} }
func Pointer[T any](Schema[T]) PointerSpec[T]   { return PointerSpec[T]{} }
func Slice[T any](Schema[T]) SliceSpec[T]       { return SliceSpec[T]{} }
func Map[K comparable, V any](Schema[K], Schema[V]) MapSpec[K, V] {
	return MapSpec[K, V]{}
}
func Field[T any](string, Schema[T]) FieldSpec { return fieldSpec{} }
func New[T any](...FieldSpec) StructSpec[T]    { return StructSpec[T]{} }

type FormOptions struct{}
type QueryOptions struct{}
type StructSpec[T any] struct{}

func (StructSpec[T]) Transform(v T) (T, error) { return v, nil }

func BindForm[T any](*T, url.Values, ...FormOptions) error                         { return nil }
func BindFormContext[T any](context.Context, *T, url.Values, ...FormOptions) error { return nil }
func ParseForm[T any](Schema[T], url.Values, ...FormOptions) (T, error)            { var zero T; return zero, nil }
func ParseFormContext[T any](context.Context, Schema[T], url.Values, ...FormOptions) (T, error) {
	var zero T
	return zero, nil
}
func (StructSpec[T]) ParseForm(url.Values, ...FormOptions) (T, error) { var zero T; return zero, nil }
func (StructSpec[T]) ParseFormContext(context.Context, url.Values, ...FormOptions) (T, error) {
	var zero T
	return zero, nil
}

func BindQuery[T any](*T, url.Values, ...QueryOptions) error                         { return nil }
func BindQueryContext[T any](context.Context, *T, url.Values, ...QueryOptions) error { return nil }
func ParseQuery[T any](Schema[T], url.Values, ...QueryOptions) (T, error) {
	var zero T
	return zero, nil
}
func ParseQueryContext[T any](context.Context, Schema[T], url.Values, ...QueryOptions) (T, error) {
	var zero T
	return zero, nil
}
func (StructSpec[T]) ParseQuery(url.Values, ...QueryOptions) (T, error) { var zero T; return zero, nil }
func (StructSpec[T]) ParseQueryContext(context.Context, url.Values, ...QueryOptions) (T, error) {
	var zero T
	return zero, nil
}

type RequestOptions struct{}

func BindRequest[T any](*T, *http.Request, ...RequestOptions) error { return nil }
