# v0.1 API Sketch

This file is a design target, not yet a compatibility promise.

```go
type Schema[T any] interface {
	Parse(any) (T, error)
	ParseContext(context.Context, any) (T, error)
}

func String() StringSchema
func Int() IntSchema
func Int64() Int64Schema
func Float64() Float64Schema
func Bool() BoolSchema

func Slice[T any](element Schema[T]) SliceSchema[T]
func Map[T any](value Schema[T]) MapSchema[T]

func Object[T any](fields ...ObjectField[T]) ObjectSchema[T]
func Field[T, V any](
	name string,
	schema Schema[V],
	setter func(*T, V),
) FieldDef[T, V]

func Transform[A, B any](
	schema Schema[A],
	fn func(A) (B, error),
) Schema[B]

func ParseJSON[T any](schema Schema[T], data []byte) (T, error)
```

Representative chaining surface:

```go
String().Trim().Min(1).Max(100).Len(8).Pattern(re).Email().Refine(fn)
Int().Min(0).Max(10).Gt(0).Gte(1).Lt(10).Lte(9).Refine(fn)
Slice(String()).Min(1).Max(10).Refine(fn)
Field("name", String(), setter).Optional().Default("anonymous")
Object[User](fields...).Strict().Strip().Refine(fn)
```

`Strict` and `Strip` are mutually overriding copy methods; the last call wins.
`Default` alone is sufficient for a missing field. Calling `Optional` before or
after `Default` is permitted and does not erase the default.

The initial exported error-code constants should cover:

```text
invalid_type required too_small too_big invalid_format invalid_value
invalid_email unknown_field custom transform_failed
```

Other codes from the broader product brief should be added only when a v0.1
schema actually emits them.
