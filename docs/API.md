# Public API and compatibility guarantees

This document defines the public surface intended to remain source-compatible
after the first release. New methods and rule types may be added in minor
versions. Removing, renaming, changing signatures, changing documented order,
or changing null/zero semantics requires a major version. See §6 for the full
compatibility policy.

## 1. Root package: `shape`

The root package is the recommended API for reusable Schemas. Every root Spec
is a `Schema[T]` (Transform + Validate). `StructSpec` and `TaggedSpec` also
implement `JSONSchema[T]` for ParseJSON. Tag-driven in-place bind is only the
package-level `BindJSON*`, `BindForm*`, `BindQuery*`, and `BindRequest` helpers.

### Construction

```go
func New[T any](fields ...FieldSpec) StructSpec[T]
func FromTags[T any]() TaggedSpec[T]
```

- `New[T]` accepts an ordinary value struct and explicit fields.
- An empty field list is valid for a Schema containing only whole-struct
  `Apply`/`Refine` callbacks.
- `FromTags[T]` compiles field mappings and `shape` tags and caches the plan by Go type.
- Invalid program configuration panics during construction.
- Both returned values are immutable and safe for concurrent reuse.

Value Schema factories:

```go
Value[T]() ValueSpec[T]
String() StringSpec
Bool() ValueSpec[bool]
Number[N Numeric]() NumberSpec[N]
Int() NumberSpec[int]
Int64() NumberSpec[int64]
Float64() NumberSpec[float64]
Time() ValueSpec[time.Time]
Duration() NumberSpec[types.Duration]

Pointer[T](inner Schema[T]) PointerSpec[T]
Slice[T](inner Schema[T]) SliceSpec[T]
Map[K MapKey, V any](key Schema[K], value Schema[V]) MapSpec[K, V]
```

Explicit struct binding:

```go
Field[T](name string, schema Schema[T]) FieldSpec
```

Value factories never accept field names, so the same Schema composes at the
root, as a collection element, or as a struct field. `Field` binds a value
Schema to one direct Go field name and is the only way to construct a
`FieldSpec`. Every argument to `New` must be a `Field(...)`. The name is the
exact, case-sensitive name of a direct exported Go field, not its JSON name.
`New` rejects empty, missing, unexported, duplicate, and type-mismatched fields.
Input exclusion tags do not prevent explicit rule bindings.

`Numeric` includes named forms of signed and unsigned integer types except
`uintptr`, plus `float32` and `float64`. Complex numbers are excluded. `MapKey` includes named
string, signed-integer, and unsigned-integer types; `uintptr` is excluded.

### Spec methods by type

Every Spec below implements `Schema[T]` run methods:

```text
Transform  TransformContext
Validate  ValidateContext
ValidateFirst  ValidateFirstContext
```

`Apply` changes a value (`T → T`). `Refine` only checks (`T → error`).
Fluent methods return a new Spec value; receivers are never mutated.
Root Specs do **not** expose `validate.And` or `transform.Then` — those exist
only in the subpackages.

#### `StringSpec` — `shape.String`

```text
Transform
  IfZero  Trim  LTrim  RTrim  ToLower  ToUpper
  Apply  ApplyContext

Validate
  NotEmpty  MinLength  MaxLength  Len
  OneOf  Pattern  StartsWith  EndsWith  Contains
  Email  URL  UUID  IP
  Refine  RefineContext  Label

Compose
  Pointer  Slice
```

`Trim`/`LTrim`/`RTrim` with no args trim Unicode whitespace; with args they trim
runes from the provided character set. Length rules use Unicode rune count.

#### `NumberSpec[N]` — `shape.Number` / `Int` / `Int64` / `Float64` / `Duration`

```text
Transform
  IfZero  Apply  ApplyContext

Validate
  Min  Max  Gt  Gte  Lt  Lte  Between  OneOf
  Positive  Negative  NonNegative
  Refine  RefineContext  Label

Compose
  Pointer  Slice
```

`Min` is an alias of `Gte`; `Max` is an alias of `Lte`. Float validators reject
NaN and ±Inf. `Duration` is `NumberSpec[types.Duration]`.

#### `ValueSpec[T]` — `shape.Value` / `Bool` / `Time`

```text
Transform
  IfZero  Apply  ApplyContext

Validate
  Refine  RefineContext  Label

Compose
  Pointer  Slice
```

No built-in domain rules. `Bool` and `Time` are `ValueSpec` aliases.

#### `PointerSpec[T]` — `shape.Pointer` or `.Pointer()`

```text
Transform
  IfNull  Apply  ApplyContext

Validate
  NotNull
  Refine  RefineContext  Label
```

Outer rules run on the pointer; a non-nil value then runs the inner Schema.
`NotNull` rejects nil. Apply content rules to the inner Schema; pointers do not expose `NotEmpty`.

#### `SliceSpec[T]` — `shape.Slice` or `.Slice()`

```text
Transform
  IfNull  Apply  ApplyContext

Validate
  NotNull  NotEmpty  Min  Max  Len  Unique
  Refine  RefineContext  Label
```

Outer rules run on the slice; the inner Schema runs in ascending index order.

#### `MapSpec[K,V]` — `shape.Map`

```text
Transform
  IfNull  Apply  ApplyContext

Validate
  NotNull  NotEmpty  Min  Max  Len
  Refine  RefineContext  Label
```

No `Unique`. Outer rules run on the map; key and value Schemas run in
deterministic key order.

#### `StructSpec[T]` — `shape.New`

```text
Transform
  Apply  ApplyContext

Validate
  Refine  RefineContext

JSON (also JSONSchema[T])
  ParseJSON  ParseJSONContext
  ParseJSONReader  ParseJSONReaderContext

Parameters
  ParseForm  ParseFormContext
  ParseQuery  ParseQueryContext
```

Whole-struct callbacks run after all field callbacks in the same phase.

#### `TaggedSpec[T]` — `shape.FromTags`

```text
Transform
  Apply  ApplyContext

Validate
  Refine  RefineContext

JSON (also JSONSchema[T])
  ParseJSON  ParseJSONContext
  ParseJSONReader  ParseJSONReaderContext

Parameters
  ParseForm  ParseFormContext
  ParseQuery  ParseQueryContext
```

Same whole-struct callback timing as `StructSpec`. Adding `Apply`/`Refine`
marks the Spec non-exportable for JSON Schema/OpenAPI.

### Schema and JSONSchema

```go
type Schema[T any] interface {
    Transform(T) (T, error)
    TransformContext(context.Context, T) (T, error)
    Validate(T) error
    ValidateContext(context.Context, T) error
    ValidateFirst(T) error
    ValidateFirstContext(context.Context, T) error
}

type JSONSchema[T any] interface {
    Schema[T]

    ParseJSON([]byte, ...JSONOptions) (T, error)
    ParseJSONContext(context.Context, []byte, ...JSONOptions) (T, error)
    ParseJSONReader(io.Reader, ...JSONOptions) (T, error)
    ParseJSONReaderContext(context.Context, io.Reader, ...JSONOptions) (T, error)
}
```

`StructSpec` and `TaggedSpec` implement `JSONSchema`. Scalar and composite Specs
implement `Schema` only. Package-level `ParseJSON*` is the uniform entry point
for every root type; struct methods are convenience delegates.

Package-level Parse for any Schema root (including scalars and collections):

```go
ParseJSON[T](schema Schema[T], source []byte, options ...JSONOptions) (T, error)
ParseJSONContext[T](ctx context.Context, schema Schema[T], source []byte, options ...JSONOptions) (T, error)
ParseJSONReader[T](schema Schema[T], reader io.Reader, options ...JSONOptions) (T, error)
ParseJSONReaderContext[T](ctx context.Context, schema Schema[T], reader io.Reader, options ...JSONOptions) (T, error)
```

JSON decoding is strict and follows `encoding/json`. For example,
`shape.ParseJSON(shape.Int(), []byte("123"))` succeeds, while a JSON string
containing `"123"` is not coerced to an integer.

Package-level tag Bind (no Schema variable; mutation target first):

```go
BindJSON[T](target *T, source []byte, options ...JSONOptions) error
BindJSONContext[T](ctx context.Context, target *T, source []byte, options ...JSONOptions) error
BindJSONReader[T](target *T, reader io.Reader, options ...JSONOptions) error
BindJSONReaderContext[T](ctx context.Context, target *T, reader io.Reader, options ...JSONOptions) error
```

Bind derives a cached tagged Schema from `T`, runs the same decode → transform →
validate pipeline, and writes `*target` only on complete success.

### Form and query parameters

Both `StructSpec[T]` and `TaggedSpec[T]` expose:

```go
ParseForm(url.Values, ...FormOptions) (T, error)
ParseFormContext(context.Context, url.Values, ...FormOptions) (T, error)
ParseQuery(url.Values, ...QueryOptions) (T, error)
ParseQueryContext(context.Context, url.Values, ...QueryOptions) (T, error)
```

Package-level calls accept any `Schema[T]` with an ordinary value struct `T`:

```go
ParseForm[T](Schema[T], url.Values, ...FormOptions) (T, error)
ParseFormContext[T](context.Context, Schema[T], url.Values, ...FormOptions) (T, error)
ParseQuery[T](Schema[T], url.Values, ...QueryOptions) (T, error)
ParseQueryContext[T](context.Context, Schema[T], url.Values, ...QueryOptions) (T, error)
BindForm[T](*T, url.Values, ...FormOptions) error
BindFormContext[T](context.Context, *T, url.Values, ...FormOptions) error
BindQuery[T](*T, url.Values, ...QueryOptions) error
BindQueryContext[T](context.Context, *T, url.Values, ...QueryOptions) error

type FormOptions struct { DisallowUnknownFields bool }
type QueryOptions struct { DisallowUnknownFields bool }
type ParameterError struct {
    Source string
    Path   validate.Path
    Err    error
}
func (*ParameterError) Error() string
func (*ParameterError) Unwrap() error
```

These methods do not widen `Schema[T]` or `JSONSchema[T]`. Invalid input
returns errors and no partial value; Bind replaces the target only on success.
Input exclusion affects decoding only. All Parse/Bind calls and direct
Transform/Validate process the same Schema fields and rules. Whole-struct callbacks
always run. Built-in error paths use the selected input's names.

The complete naming, duplicate, empty-value, conversion, nesting, error-order,
and unsupported-type contracts are in [PARAMETERS.md](PARAMETERS.md). Input maps
and slices must not be mutated concurrently by callers. Plans are immutable
and cached separately for each input and Go type.

### Automatic HTTP binding

```go
BindRequest[T](*T, *http.Request, ...RequestOptions) error

type RequestPrecedence uint8
const (
    RejectConflicts RequestPrecedence = iota
    QueryFirst
    BodyFirst
)
const DefaultMaxRequestBytes int64 = 1 << 20

type RequestOptions struct {
    DisallowUnknownFields bool
    MaxBytes              int64
    Precedence            RequestPrecedence
}
type SourceConflictError struct {
    Field   string
    Sources []string
}
func (*SourceConflictError) Error() string
var ErrRequestTooLarge error
var ErrUnsupportedContentType error
var ErrRequestFiles error
```

BindRequest uses the request context, derives a tagged Schema, and combines
Query and the body before one Transform/Validate pass. Content-Type selects
JSON or Form; decoding never guesses another format after a failure. Conflicts
are errors by default. Explicit precedence operates on submitted top-level Go
fields, including zero/empty/null values; nested objects and slices are not
recursively merged. All submitted sources must decode successfully.

The default body limit is 1 MiB. It consumes the body without closing it and
must run before another body parser. Multipart text fields are supported;
file parts and root custom JSON unmarshaling require the explicit input APIs.
Both Form and Query type definitions are checked before reading input; exclude
unsupported fields with the corresponding input tags.
See [PARAMETERS.md](PARAMETERS.md#automatic-http-binding) for the full contract.

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
    Feature string
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

Every concrete validator also exposes run methods
`Validate` / `ValidateContext` / `ValidateFirst` / `ValidateFirstContext`.

#### `StringValidator`

```text
NotEmpty  MinLength  MaxLength  Len
OneOf  Pattern  StartsWith  EndsWith  Contains
Email  URL  UUID  IP
Refine  RefineContext  And  Label
```

#### `NumberValidator[N]`

```text
Min  Max  Gt  Gte  Lt  Lte  Between  OneOf
Positive  Negative  NonNegative
Refine  RefineContext  And  Label
```

`Min` ≡ `Gte`, `Max` ≡ `Lte`.

#### `ValueValidator[T]` — also `Bool` / `Time`

```text
Refine  RefineContext  And  Label
```

#### `PointerValidator[T]`

```text
NotNull
Refine  RefineContext  And  Label
```

Pointer presence uses `NotNull`; inner validators check the value.

#### `SliceValidator[T]`

```text
NotNull  NotEmpty  Min  Max  Len  Unique
Refine  RefineContext  And  Label
```

#### `MapValidator[K,V]`

```text
NotNull  NotEmpty  Min  Max  Len
Refine  RefineContext  And  Label
```

`And` returns the concrete family type, so further rule methods can follow it.
A label set before `And` applies to the appended validators' issues too.
`And` occupies its declaration position among outer rules; a later `Refine`
executes after it. Container elements execute after all outer rules.

Errors and paths:

```go
type Error struct { Issues []Issue }
type Issue struct {
    Target IssueTarget
    Code string
    Path Path
    Message string
    Label string
    Expected any
    Received any
}

type IssueTarget string
const TargetValue IssueTarget = ""
const TargetKey IssueTarget = "key"

FieldPath(string) PathSegment
IndexPath(int) PathSegment
MapKeyPath(any) PathSegment

type PathKind uint8
const PathField PathKind
const PathIndex PathKind
const PathMapKey PathKind

type PathSegment struct {
    Kind PathKind
    Key string
    Index int
    MapKey any
}

func (Path) String() string
func (Path) MarshalJSON() ([]byte, error)
```

Map-key segments preserve the concrete key value: integer key `3` renders as
`[3]` and marshals as JSON number `3`, while string key `"3"` renders as
`["3"]` and marshals as a JSON string.

`Issue.Target` identifies what failed at `Path`: `TargetValue` (the zero value)
means the value, and `TargetKey` means the map key at the final segment. Key
issues encode `"target":"key"`; value issues omit `target`. This distinction
survives custom labels and nested maps. `Label` is display text only.

Stable codes:

```text
too_small  too_big  invalid_format  invalid_value  invalid_enum
invalid_number  invalid_email  invalid_url  invalid_uuid  invalid_ip
too_deep  too_many_issues  custom  unique_limit
```

`unique_limit` is a resource bound, not a verdict about the data: `Unique`
switches to pairwise deep comparison for element types that are not safely
comparable with `==`, and a collection larger than `validate.MaxDeepUniqueItems`
(1024) reports `unique_limit` instead of being compared. Keeping it distinct
from `too_big` means a consumer cannot mistake it for a `Max` length violation.
Comparable element types use a hash set and are not bounded by it.

Locale API:

```go
type Language uint8
const English Language
const SimplifiedChinese Language

SetLanguage(Language)
WithLocale(context.Context, Language) context.Context
```

Built-in messages use the global language, or the `WithLocale` language for a
context-aware call, when the error is created. Existing errors are not
translated afterward. Custom `Refine` messages are returned unchanged.

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

Every concrete transformer also exposes
`Transform` / `TransformContext`.

#### `StringTransformer`

```text
IfZero  Trim  LTrim  RTrim  ToLower  ToUpper
Apply  ApplyContext  Then
```

`Then` is inherited from the embedded `ValueTransformer[string]`.

#### `NumberTransformer[N]` — also `Int` / `Int64` / `Float64` / `Duration`

```text
IfZero  Apply  ApplyContext  Then
```

#### `ValueTransformer[T]` — also `Bool` / `Time`

```text
IfZero  Apply  ApplyContext  Then
```

#### `PointerTransformer[T]`

```text
IfNull  Apply  ApplyContext  Then
```

#### `SliceTransformer[T]`

```text
IfNull  Apply  ApplyContext  Then
```

#### `MapTransformer[K,V]`

```text
IfNull  Apply  ApplyContext  Then
```

For Pointer/Slice/Map, outer `IfNull`/`Apply` calls run in method-call order,
then the resulting non-nil elements run their inner transformers. Defaults and
elements inserted by `Apply` are therefore transformed too. `Then` composes
complete pipelines in order.

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
`shape.FromTags[T]()` plans are exported. Transforms, custom callbacks, fallbacks,
custom JSON representations, `json:",string"`, byte slices, Go duration strings,
float32 values, URL rules, and unsupported rules return `UnsupportedSchemaError`;
they are never omitted silently. Repeated rules intersect, and exported documents are detached from the
cached plan. Patterns export only when the supported ASCII Go regexp subset
can be translated to ECMA-262; unsupported flags and Unicode classes fail.

`unique` exports only for scalar elements whose decoded equality matches JSON
equality. Scalar pointers to strings, booleans, and integers preserve the
null/value distinction; ordinary scalar slices that accept both null and zero,
struct slices, and nested collections are rejected. Exported pointer uniqueness
also includes the runtime deep-comparison limit as `maxItems`.

Only exact `TaggedSpec[T]` values and pointers export. Wrapping or embedding a
Spec does not make custom behavior exportable.

The adapters describe constraints on JSON values, not every lexical detail of
`encoding/json` (for example integer token spelling or duplicate object keys).
Consumers must enable format assertions when relying on `format` annotations.
The input accepts null where decoding to zero passes validation; the same
value-constraint document is used by `JSONResponse`, which does not serialize
values or validate a response body.

## 5. Frozen behavior

- `Transform` never validates.
- `Validate` never transforms.
- JSON, form, and query operations run Decode → Transform → Validate.
- `Validate` aggregates up to `validate.DefaultMaxIssues`.
- `ValidateFirst` returns at most one issue and stops evaluation.
- Transform steps preserve the order in which their fluent methods are called.
- Validation rules preserve the order in which their fluent methods are called.
- Explicit Schema fields use `shape.New` argument order.
- Tagged Schema fields use Go declaration order.
- Slices use ascending indexes; maps use deterministic sorted keys.
- `NotEmpty` rejects empty strings and nil/empty slices/maps. Pointers use `NotNull`.
- `IfZero` cannot distinguish missing JSON, JSON null, and an explicit zero after decoding.
- `IfNull` applies only to nilable values and preserves nil/empty distinction.
- Context cancellation stops traversal and returns the context error.
- Transformers, Validators, and Schemas are immutable and concurrency-safe.
- User input errors return errors; invalid program configuration panics at construction
  or, for a form/query decoding plan, its first use.

## 6. Compatibility policy

After the first public release, the following require a major version:

- removing or renaming a documented public identifier;
- changing a public signature or generic constraint incompatibly;
- changing Decode → Transform → Validate order;
- making Transform validate or Validate transform;
- changing aggregate/fail-fast semantics or traversal order;
- changing documented zero, null, empty, path, or atomic Bind behavior;
- changing an existing stable validation code to mean something different;
- adding implicit coercion to an existing strict operation.

Minor versions may add new factories, rules, transforms, languages, issue codes,
or optional adapters when existing programs retain their behavior.

Intentionally absent from the initial contract:

```text
implicit coercion
Optional
Default
SkipNull
Nullable
NotBlank
cross-type Transform
ordinary-value Parse or in-place Bind
```

`IfZero`, `IfNull`, pointers, and explicit custom callbacks cover the intended
cases without hiding JSON state or mixing validation with transformation.

## 7. Ownership and callbacks

A constructed Schema owns its rule configuration and fallback snapshots. Error
`Expected` lists and exported documents are independent copies and may be edited
without changing later calls. `Received` can refer to caller-supplied values;
errors do not promise an arbitrary deep copy of user data.

Built-in transforms detach the mutable data they process before invoking
callbacks. Explicit fields omitted from `New` pass through unchanged and may
share storage; a whole-struct `Apply` detaches the data it receives. Tagged JSON
parsing may reuse decoded storage when the type graph contains no custom JSON
or text decoders and no interface fields. Otherwise transforms detach the data
before modifying it. Every fallback is snapshotted during construction and
copied for use; the caller's later edits cannot change it.

Package-level parsing and collection composition call the public methods
of external Schema implementations, including wrappers that embed built-in
Specs. Internal fast paths apply only to exact built-in types.

`Value[T]` is intended for data values. Copying live synchronization objects,
resource handles, closures or channels does not provide transactional isolation.
Cycles or values beyond the copy depth limit return an error. Private data
fields are copied as well; this is not a general-purpose object cloning API.
Custom callbacks must be safe for concurrent reuse, validators must not mutate
inputs, and transform callbacks must transfer ownership of mutable values they
return. Side effects and external storage accessed by callbacks remain the
caller's responsibility.
