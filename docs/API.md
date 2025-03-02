# Public API and behavior contract

This document defines the public surface intended to remain source-compatible
after the first release. New methods and rule types may be added in minor
versions. Removing, renaming, changing signatures, changing documented order,
or changing null/zero semantics requires a major version.

## 1. Root package: `shape`

The root package is the recommended API for complete struct contracts.

### Construction

```go
func New[T any](fields ...FieldSpec) StructSpec[T]
func Struct[T any]() TaggedSpec[T]
```

- `New[T]` accepts an ordinary value struct and explicit fields.
- An empty field list is valid for a Schema containing only whole-struct
  `Apply`/`Refine` callbacks.
- `Struct[T]` compiles `json` and `shape` tags and caches the plan by Go type.
- Invalid program configuration panics during construction.
- Both returned values are immutable and safe for concurrent reuse.

Explicit field factories:

```go
Value[T](name ...string) ValueSpec[T]
String(name ...string) StringSpec
Bool(name ...string) ValueSpec[bool]
Number[N Numeric](name ...string) NumberSpec[N]
Int(name ...string) NumberSpec[int]
Int64(name ...string) NumberSpec[int64]
Float64(name ...string) NumberSpec[float64]
Time(name ...string) ValueSpec[time.Time]
Duration(name ...string) NumberSpec[types.Duration]

Pointer[T](name string, inner Contract[T]) PointerSpec[T]
Slice[T](name string, inner Contract[T]) SliceSpec[T]
Map[K MapKey, V any](name string, key Contract[K], value Contract[V]) MapSpec[K, V]
Field[T](name string, schema Schema[T]) FieldSpec
```

Names are Go field names. A name may be omitted from scalar factories when the
result is used as a Pointer/Slice/Map inner contract. `New` rejects unnamed,
missing, unexported, duplicate, type-mismatched, and `json:"-"` fields.

`Numeric` includes named forms of all integers except `uintptr` and
`float32`/`float64`. Complex numbers are excluded. `MapKey` includes named
string, signed-integer, and unsigned-integer types; `uintptr` is excluded.

### Field methods

Every Spec implements `Contract[T]`, so it has:

```go
type Contract[T any] interface {
    transform.Transformer[T]
    validate.Validator[T]
}

Transform(T) (T, error)
TransformContext(context.Context, T) (T, error)
Validate(T) error
ValidateContext(context.Context, T) error
ValidateFirst(T) error
ValidateFirstContext(context.Context, T) error
```

`ValueSpec[T]`:

```text
IfZero  Apply  ApplyContext  Refine  RefineContext  Label  Pointer  Slice
```

`StringSpec`:

```text
IfZero  Trim  LTrim  RTrim  ToLower  ToUpper
Apply  ApplyContext
NotEmpty  MinLength  MaxLength  Len  OneOf  Pattern
StartsWith  EndsWith  Contains  Email  URL  UUID  IP
Refine  RefineContext  Label  Pointer  Slice
```

`NumberSpec[N]`:

```text
IfZero  Apply  ApplyContext
Min  Max  Gt  Gte  Lt  Lte  Between  OneOf
Positive  Negative  NonNegative
Refine  RefineContext  Label  Pointer  Slice
```

`PointerSpec[T]`:

```text
IfNull  Apply  ApplyContext  NotNull  NotEmpty
Refine  RefineContext  Label
```

`SliceSpec[T]`:

```text
IfNull  Apply  ApplyContext  NotNull  NotEmpty
Min  Max  Len  Unique  Refine  RefineContext  Label
```

`MapSpec[K,V]`:

```text
IfNull  Apply  ApplyContext  NotNull  NotEmpty
Min  Max  Len  Refine  RefineContext  Label
```

### Whole-struct methods

Both `StructSpec[T]` and `TaggedSpec[T]` add whole-struct callbacks after field
processing. Each method returns the same concrete Spec type:

```go
Apply(steps ...func(T) (T, error)) StructSpec[T]
ApplyContext(steps ...func(context.Context, T) (T, error)) StructSpec[T]
Refine(rules ...func(T) error) StructSpec[T]
RefineContext(rules ...func(context.Context, T) error) StructSpec[T]
```

Export rejects a tagged Spec after whole-struct callbacks are added because
JSON Schema cannot execute those callbacks faithfully.

### Schema interface

```go
type Schema[T any] interface {
    Transform(T) (T, error)
    TransformContext(context.Context, T) (T, error)
    Validate(T) error
    ValidateContext(context.Context, T) error
    ValidateFirst(T) error
    ValidateFirstContext(context.Context, T) error

    ParseJSON([]byte, ...JSONOptions) (T, error)
    ParseJSONContext(context.Context, []byte, ...JSONOptions) (T, error)
    ParseJSONReader(io.Reader, ...JSONOptions) (T, error)
    ParseJSONReaderContext(context.Context, io.Reader, ...JSONOptions) (T, error)
    BindJSON(*T, []byte, ...JSONOptions) error
    BindJSONContext(context.Context, *T, []byte, ...JSONOptions) error
    BindJSONReader(*T, io.Reader, ...JSONOptions) error
    BindJSONReaderContext(context.Context, *T, io.Reader, ...JSONOptions) error
}
```

Package-level tag Bind shortcuts:

```go
BindJSON[T](target *T, source []byte, options ...JSONOptions) error
BindJSONContext[T](ctx context.Context, target *T, source []byte, options ...JSONOptions) error
BindJSONReader[T](target *T, reader io.Reader, options ...JSONOptions) error
BindJSONReaderContext[T](ctx context.Context, target *T, reader io.Reader, options ...JSONOptions) error
```

Mutation targets remain before input sources. Bind is atomic: it writes only on
complete success.

### JSON configuration

```go
type JSONOptions struct {
    DisallowUnknownFields bool
    MaxBytes              int64
}

var ErrJSONTooLarge error
```

At most one `JSONOptions` value is accepted. `MaxBytes == 0` is unlimited;
negative values fail. Exactly one JSON value must be present.

### Root errors

```go
type TransformError struct {
    Path validate.Path
    Err  error
}

type UnsupportedSchemaError struct {
    Operation string
}

ExportDocument[T](schema Schema[T]) (map[string]any, error)
```

Every Schema transform failure, including a whole-struct `Apply` failure, uses
`TransformError`; collection paths include indexes or map keys. It supports
`errors.Unwrap`. Validation failures use
`*validate.Error`. JSON syntax/type failures wrap the `encoding/json` error.
`ExportDocument` is the low-level adapter hook; applications normally call the
`jsonschema` or `openapi` package.

## 2. Package `validate`

```go
type Validator[T any] interface {
    Validate(T) error
    ValidateContext(context.Context, T) error
    ValidateFirst(T) error
    ValidateFirstContext(context.Context, T) error
}
```

Factories:

```text
Value[T]  String  Bool  Number[N]  Int  Int64  Float64  Time  Duration
Pointer  Slice  Map
```

Common composition/customization:

```text
Refine  RefineContext  And  Label
```

String rules:

```text
NotEmpty  MinLength  MaxLength  Len  OneOf  Pattern
StartsWith  EndsWith  Contains  Email  URL  UUID  IP
```

Number rules:

```text
Min  Max  Gt  Gte  Lt  Lte  Between  OneOf
Positive  Negative  NonNegative
```

Pointer rules: `NotNull`, `NotEmpty`.

Slice rules: `NotNull`, `NotEmpty`, `Min`, `Max`, `Len`, `Unique`.

Map rules: `NotNull`, `NotEmpty`, `Min`, `Max`, `Len`.

Errors and paths:

```go
type Error struct { Issues []Issue }
type Issue struct {
    Code string
    Path Path
    Message string
    MessageID string // built-in localization key; omitted from JSON
    Label string
    Expected any
    Received any
}

FieldPath(string) PathSegment
IndexPath(int) PathSegment

type PathKind uint8
const PathField PathKind
const PathIndex PathKind

type PathSegment struct {
    Kind PathKind
    Key string
    Index int
}

func (Path) String() string
func (Path) MarshalJSON() ([]byte, error)
```

Stable codes:

```text
too_small  too_big  invalid_format  invalid_value  invalid_enum
invalid_number  invalid_email  invalid_url  invalid_uuid  invalid_ip
too_deep  too_many_issues  custom
```

Locale API:

```go
type Language uint8
const English Language
const SimplifiedChinese Language

SetLanguage(Language)
CurrentLanguage() Language
WithLocale(context.Context, Language) context.Context
LocaleFromContext(context.Context) Language
Localize(error, Language) error
LocalizeContext(context.Context, error) error
```

## 3. Package `transform`

```go
type Transformer[T any] interface {
    Transform(T) (T, error)
    TransformContext(context.Context, T) (T, error)
}
```

Factories:

```text
Value[T]  String  Bool  Number[N]  Int  Int64  Float64  Time  Duration
Pointer  Slice  Map
```

Value/scalar methods: `IfZero`, `Apply`, `ApplyContext`, `Then`.

String additionally provides:

```text
Trim  LTrim  RTrim  ToLower  ToUpper
```

Pointer/Slice/Map provide `IfNull`, `Apply`, `ApplyContext`, and `Then`.
Their inner element transformation is the first constructed step; fluent
`IfNull`/`Apply` calls then run in method-call order.

All transforms return a value of the same Go type. Cross-type coercion is not
part of this API.

## 4. Type and export packages

`types.Duration` is a signed nanosecond duration whose JSON representation is a
duration string such as `"30s"`. JSON `null` decodes to zero. It provides
`String`, `MarshalJSON`, and pointer-receiver `UnmarshalJSON` methods.

```go
type jsonschema.Document map[string]any
func (jsonschema.Document) Bytes() ([]byte, error)
type jsonschema.UnsupportedError = shape.UnsupportedSchemaError
jsonschema.Export(schema) (jsonschema.Document, error)

openapi.Schema(schema)
openapi.JSONRequestBody(schema, required)
openapi.JSONResponse(description, schema)
```

Export is intentionally conservative. Currently only representable tagged
`shape.Struct[T]()` plans are exported. Transforms, custom callbacks, fallbacks,
custom JSON representations, and unsupported rules return
`UnsupportedSchemaError`; they are never omitted silently.

## 5. Frozen behavior

- `Transform` never validates.
- `Validate` never transforms.
- Schema JSON operations run Decode → Transform → Validate.
- `Validate` aggregates up to `validate.DefaultMaxIssues`.
- `ValidateFirst` returns at most one issue and stops evaluation.
- Transform steps preserve the order in which their fluent methods are called.
- Validation rules preserve the order in which their fluent methods are called.
- Explicit Schema fields use `shape.New` argument order.
- Tagged Schema fields use Go declaration order.
- Slices use ascending indexes; maps use deterministic sorted keys.
- `NotEmpty` treats nil pointer/slice/map as empty.
- `IfZero` cannot distinguish missing JSON, JSON null, and an explicit zero after decoding.
- `IfNull` applies only to nilable values and preserves nil/empty distinction.
- Context cancellation stops traversal and returns the context error.
- Transformers, Validators, and Schemas are immutable and concurrency-safe.
- User input errors return errors; invalid program configuration panics at construction.
