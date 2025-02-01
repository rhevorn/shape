# Current API Sketch

Authoritative examples live in the README and `examples/`. This file is a
compact map of the exported surface.

```go
type Schema[T any] interface {
	Parse(any) (T, error)
	ParseContext(context.Context, any) (T, error)
}

// Scalars
func String() StringSchema
func Bool() BoolSchema
func Int() IntSchema
func Int64() Int64Schema
func Float64() Float64Schema
func Number[N Numeric]() NumberSchema[N]
func Time() TimeSchema
func Duration() DurationSchema
func URL() StringSchema  // also String().URL()
func UUID() StringSchema // also String().UUID()
func IP() StringSchema   // also String().IP()

// Explicit coercion (not hidden conversion)
func CoerceString() StringSchema
func CoerceBool() BoolSchema
func CoerceInt() IntSchema
func CoerceInt64() Int64Schema
func CoerceFloat64() Float64Schema // CoerceFloat is an alias
func CoerceNumber[N Numeric]() NumberSchema[N]
func CoerceTime(layouts ...string) TimeSchema
func CoerceDuration() DurationSchema

// Composition
func Slice[T any](element Schema[T]) SliceSchema[T]
func Map[T any](value Schema[T]) MapSchema[T]
func Record[K comparable, V any](key Schema[K], value Schema[V]) RecordSchema[K, V]
func Tuple[T any](items ...TupleElement[T]) TupleSchema[T]
func TupleItem[T, V any](schema Schema[V], setter func(*T, V)) TupleItemDef[T, V]
func Object[T any](fields ...ObjectField[T]) ObjectSchema[T]
func Field[T, V any](name string, schema Schema[V], setter func(*T, V)) FieldDef[T, V]
func Fields[T any]() FieldFactory[T] // f.Str / f.Email / f.Int / f.Bool ... .Set
func Enum[T comparable](values ...T) EnumSchema[T]
func Literal[T comparable](value T) LiteralSchema[T]
func Union[T any](alternatives ...Schema[T]) Schema[T]   // first match
func OneOf[T any](alternatives ...Schema[T]) Schema[T]   // exactly one
func Nullable[T any](schema Schema[T]) Schema[*T]
func Lazy[T any](name string, provider func() Schema[T]) LazySchema[T]
func Transform[A, B any](schema Schema[A], fn func(A) (B, error)) Schema[B]
func Label[T any](label string, schema Schema[T]) Schema[T]
func Annotate[T any](schema Schema[T]) AnnotatedSchema[T]

// JSON input (string or []byte); schema.Parse remains for decoded Go values
func Parse[T any, B ~string | ~[]byte](schema Schema[T], data B) (T, error)
func ParseContext[T any, B ~string | ~[]byte](ctx context.Context, schema Schema[T], data B) (T, error)
func ParseReader[T any](schema Schema[T], reader io.Reader) (T, error)
func ParseReaderContext[T any](ctx context.Context, schema Schema[T], reader io.Reader) (T, error)
func ParseReaderLimit[T any](schema Schema[T], reader io.Reader, maxBytes int64) (T, error)
func ParseReaderLimitContext[T any](ctx context.Context, schema Schema[T], reader io.Reader, maxBytes int64) (T, error)

// Locale (messages come from messages/*.json catalogs)
func SetLanguage(lang string)
func Language() string
func WithLocale(ctx context.Context, lang string) context.Context
func Localize(err error, lang string) error
```

Fluent object fields (preferred for common scalars):

```go
f := Fields[User]()
Object(
	f.Str("name", "姓名").Trim().Min(2).Set(func(u *User, v string) { u.Name = v }),
	f.Email("email", "邮箱").Trim().Set(func(u *User, v string) { u.Email = v }),
	f.Int("age").Min(18).Set(func(u *User, v int) { u.Age = v }),
).Strict()
```

`Field(...)` remains for arbitrary nested schemas. Optional display labels also
attach via `Field(...).Label("…")` or `Label("…", schema)`.

Representative chaining:

```go
String().Trim().Min(1).Max(100).Len(8).Pattern(re).Email().Refine(fn)
Int().Min(0).Max(10).Gt(0).Gte(1).Lt(10).Lte(9).Refine(fn)
Slice(String()).Min(1).Max(10).NonEmpty().Unique().Refine(fn)
Field("name", String(), setter).Optional().Default("anonymous").Label("姓名")
Field("labels", Map(String()), setter).DefaultFunc(factory)
Object[User](fields...).Strict().Strip().Refine(fn)
```

`Strict` and `Strip` are mutually overriding copy methods; the last call wins.
`Default` alone is enough for a missing field. Mutable defaults use
`DefaultFunc`.

Issue codes currently include:

```text
invalid_type required too_small too_big too_deep invalid_format invalid_value
invalid_enum invalid_union invalid_string invalid_number invalid_email
invalid_url invalid_uuid invalid_ip unknown_field too_many_issues custom
transform_failed
```

Adapters (separate packages, stdlib-only):

- `github.com/rhevorn/shape/jsonschema` — Draft 2020-12 export
- `github.com/rhevorn/shape/openapi` — OpenAPI 3.1 fragments

HTTP handlers use the root JSON helpers (`Parse`, `ParseReaderLimitContext`,
…). There is no dedicated `net/http` package.
