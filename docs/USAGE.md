# Shape usage guide

The recommended entry point is the root `shape` package. Use it to define a
Schema that can transform typed values, validate typed values, parse JSON, and
atomically bind JSON. Scalars, pointers, slices, maps, and structs use the same
`Schema[T]` interface.

The lower-level `validate` and `transform` packages are independent building
blocks. Use them directly when no struct Schema or JSON orchestration is needed.

```text
shape       Schema[T]: Transform + Validate + JSON for any supported T
validate    inspect T and return errors; never changes T
transform   convert T to a new T; never validates T
```

## 1. Install

```sh
go get github.com/rhevorn/shape
```

Shape requires Go 1.24 or newer. The runtime packages have no third-party
dependencies.

## 2. Explicit Schema: the primary API

### 2.1 Define once and reuse

```go
type User struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

var userSchema = shape.New[User](
	shape.String("Name").Trim().NotEmpty().MaxLength(50),
	shape.Int("Age").Min(18).Max(120),
)
```

The string passed to each field factory is the Go field name, not the JSON name.
Errors use the JSON name:

```go
type User struct {
	DisplayName string `json:"display_name"`
}

var schema = shape.New[User](
	shape.String("DisplayName").NotEmpty(),
)

// Error path: display_name
```

Construction checks that every configured field exists, is exported, has the
same Go type as its Schema, is not repeated, and is not excluded with
`json:"-"`. Invalid configuration panics immediately, which makes package-level
declarations fail during initialization.

Fields omitted from `shape.New` pass through unchanged and are not validated.
`shape` tags are ignored by an explicit Schema.

### 2.2 Transform and Validate typed values independently

```go
raw := User{Name: " Pong ", Age: 20}

user, err := userSchema.Transform(raw)
// user.Name == "Pong"
// raw.Name is still " Pong "

err = userSchema.Validate(user)      // collect all issues
err = userSchema.ValidateFirst(user) // stop at the first issue
```

`Validate(raw)` does not run `Trim`; it validates exactly the value supplied.
`Transform(raw)` does not run `NotEmpty` or `Min`.

Context-aware forms stop promptly when the request is cancelled:

```go
user, err = userSchema.TransformContext(ctx, raw)
err = userSchema.ValidateContext(ctx, user)
err = userSchema.ValidateFirstContext(ctx, user)
```

### 2.3 Parse and Bind JSON

```go
user, err := userSchema.ParseJSON(data)
```

Schema JSON operations always run:

```text
read exactly one JSON value
    → encoding/json decode into a fresh zero T
    → Transform
    → Validate
    → return T
```

Bind writes only after the complete operation succeeds:

```go
target := User{Name: "keep", Age: 99}
err := userSchema.BindJSON(&target, invalidData)
// target remains unchanged
```

Byte and reader forms:

```go
user, err := userSchema.ParseJSON(data)
user, err = userSchema.ParseJSONContext(ctx, data)
user, err = userSchema.ParseJSONReader(reader)
user, err = userSchema.ParseJSONReaderContext(ctx, reader)

err = userSchema.BindJSON(&user, data)
err = userSchema.BindJSONContext(ctx, &user, data)
err = userSchema.BindJSONReader(&user, reader)
err = userSchema.BindJSONReaderContext(ctx, &user, reader)
```

Mutation targets intentionally appear before input sources.

### 2.4 Scalar and standalone Schemas

Every root Spec is a complete `Schema[T]`; it does not need to be placed inside
`shape.New`. A standalone scalar or composite supports the same transform,
validation, ParseJSON, and BindJSON operations:

```go
nameSchema := shape.String().Trim().NotEmpty()
name, err := nameSchema.ParseJSON([]byte(`" Pong "`))
// name == "Pong"

scoreSchema := shape.Int().NonNegative().Slice().NotEmpty()
scores, err := scoreSchema.ParseJSON([]byte(`[1, 2, 3]`))
```

Decoding follows `encoding/json` and never performs implicit coercion. JSON
number `123` can decode into `int`; JSON string `"123"` cannot.

The same factories define fields when given a Go field name:

```go
type Settings struct {
	Name     string
	Enabled  bool
	Retries  uint8
	Page     int
	Total    int64
	Ratio    float64
	Created  time.Time
	Timeout  types.Duration
	Metadata Metadata
}

var settingsSchema = shape.New[Settings](
	shape.String("Name").Trim().NotEmpty(),
	shape.Bool("Enabled"),
	shape.Number[uint8]("Retries").Between(1, 5),
	shape.Int("Page").Positive(),
	shape.Int64("Total").NonNegative(),
	shape.Float64("Ratio").Between(0, 1),
	shape.Time("Created").Refine(requireTime),
	shape.Duration("Timeout").Positive(),
	shape.Value[Metadata]("Metadata").Refine(validateMetadata),
)
```

Use `Number[N]` for other integer, unsigned integer, floating-point, or named
number types. Use `Value[T]` for arbitrary same-type custom behavior.

`types.Duration` decodes duration strings such as `"30s"`. Standard
`time.Duration` remains a named `int64`; use `shape.Number[time.Duration]` when
its ordinary JSON number representation is desired.

### 2.5 String behavior

Transforms:

```go
shape.String("Name").
	IfZero("guest").
	Trim().
	LTrim("_").
	RTrim("-").
	ToLower()
```

- `Trim()` removes Unicode whitespace from both sides.
- `Trim(chars)` removes runes contained in `chars` from both sides.
- `LTrim` and `RTrim` affect one side.
- `ToLower` and `ToUpper` use Unicode case conversion.

Rules:

```go
shape.String("Email").
	NotEmpty().
	MinLength(3).
	MaxLength(254).
	Email()
```

Available string rules are `NotEmpty`, `MinLength`, `MaxLength`, `Len`,
`OneOf`, `Pattern`, `StartsWith`, `EndsWith`, `Contains`, `Email`, `URL`,
`UUID`, and `IP`. Length uses Unicode rune count.

`NotEmpty` means `value != ""`; whitespace is not implicitly empty:

```go
shape.String("Name").NotEmpty().Validate("   ") // succeeds
```

When JSON processing should reject whitespace-only input, declare the transform:

```go
shape.String("Name").Trim().NotEmpty()
```

### 2.6 Number behavior

```go
shape.Int("Age").Min(18).Max(120)
shape.Float64("Ratio").Gt(0).Lte(1)
shape.Number[uint16]("Port").Between(1, 65535)
```

Number rules are `Min`/`Gte`, `Max`/`Lte`, `Gt`, `Lt`, `Between`, `OneOf`,
`Positive`, `Negative`, and `NonNegative`. Float validators always reject NaN
and positive/negative infinity.

### 2.7 Zero and null fallbacks

`IfZero` is a transform on scalar/value Specs:

```go
shape.Int("Page").IfZero(1)
shape.String("Mode").IfZero("read")
```

After `encoding/json` decoding, a missing key, JSON `null`, and an explicit zero
often produce the same Go zero value. `IfZero` intentionally treats them the
same; Shape does not reconstruct information discarded by `encoding/json`.

`IfNull` applies only to pointer, slice, and map Specs:

```go
fallback := "guest"
shape.Pointer("Nickname", shape.String().Trim()).IfNull(&fallback)
shape.Slice("Tags", shape.String()).IfNull([]string{})
shape.Map("Scores", shape.String(), shape.Int()).IfNull(map[string]int{})
```

It changes nil only. A non-nil empty slice/map stays empty.

For scalar element Schemas, the common pointer/slice forms also have fluent
shortcuts:

```go
shape.String("Nickname").Trim().Pointer().NotNull()
shape.Int("Scores").NonNegative().Slice().NotEmpty()
shape.Value[Metadata]("Items").Slice()
```

Use the factory form for nested composites and maps. Go rejects unsupported
key types at compile time; map keys are named or unnamed string/integer types.

### 2.8 Pointer, Slice, and Map

```go
var schema = shape.New[Request](
	shape.Pointer(
		"Nickname",
		shape.String().Trim().NotEmpty(),
	).NotNull(),

	shape.Slice(
		"Tags",
		shape.String().Trim().NotEmpty(),
	).NotEmpty().Min(1).Max(10).Unique(),

	shape.Map(
		"Scores",
		shape.String().Trim().NotEmpty(),
		shape.Int().NonNegative(),
	).NotNull().Max(100),
)
```

- Pointer rules run on the pointer. A non-nil value then runs the inner Schema.
- Slice rules run on the slice. The inner Schema runs in ascending index order.
- Map rules run on the map. Key and value Schemas run in deterministic key order.
- `NotEmpty` rejects nil as well as an empty collection/pointer.
- `NotNull` rejects nil but accepts an empty slice/map.
- Map keys are string or integer types. Transformed duplicate keys return an error.

Collection error paths include their location:

```text
tags[2]
scores.admin
```

### 2.9 Nested structs

Reuse an ordinary nested Schema with `Field`:

```go
var addressSchema = shape.New[Address](
	shape.String("City").Trim().NotEmpty(),
)

var userSchema = shape.New[User](
	shape.Field("Address", addressSchema),
	shape.Pointer("BillingAddress", addressSchema),
	shape.Slice("PreviousAddresses", addressSchema),
	shape.Map("AddressesByKind", shape.String(), addressSchema),
)
```

Nested issues are prefixed automatically, for example
`previous_addresses[1].city`.

### 2.10 Custom field and cross-field behavior

Field-level same-type transforms use `Apply`:

```go
shape.String("Name").Apply(removeControlCharacters, canonicalizeName)
```

Field-level custom rules use `Refine` and return ordinary errors:

```go
shape.String("Name").Refine(checkReserved, checkBusinessPolicy)
```

Context-aware callbacks:

```go
shape.String("Name").
	ApplyContext(normalizeWithRepository).
	RefineContext(checkUniqueName)
```

For cross-field behavior, chain on `shape.New`:

```go
var periodSchema = shape.New[Period](
	shape.Time("Start"),
	shape.Time("End"),
).Apply(normalizePeriod).Refine(func(period Period) error {
	if period.End.Before(period.Start) {
		return errors.New("end must not be before start")
	}
	return nil
})
```

Whole-struct callbacks run after all field callbacks in the same phase.
`Apply`/`Refine` accept multiple callbacks and preserve method-call order.

## 3. Tag Schema: convenience API

Use tags for simple DTOs where field behavior is static and expressible by the
tag language:

```go
type Request struct {
	Name string `json:"name" shape:"ifzero=guest,trim,notempty,maxlength=50"`
	Age  int    `json:"age" shape:"min=18,max=120"`
}
```

Direct binding requires no Schema variable:

```go
var request Request
err := shape.BindJSON(&request, data)
err = shape.BindJSONContext(ctx, &request, data)
err = shape.BindJSONReader(&request, reader)
err = shape.BindJSONReaderContext(ctx, &request, reader)
```

Store a tag Schema when Parse, independent Transform/Validate, reuse, or export
is needed:

```go
var requestSchema = shape.Struct[Request]()

request, err := requestSchema.ParseJSON(data)
request, err = requestSchema.Transform(request)
err = requestSchema.Validate(request)
```

Tagged schemas can add whole-struct transforms and validations without giving
up the concise field tags:

```go
var requestSchema = shape.Struct[Request]().
	Apply(normalizeRequest).
	Refine(checkRequest)
```

Plans are compiled once and cached by Go type. Nested structs, pointers to
supported scalar/struct types, slices, maps, `time.Time`, and `types.Duration`
are traversed automatically. See [TAGS.md](TAGS.md) for the exact grammar and
type matrix.

Prefer explicit `shape.New[T]` when custom callbacks, cross-field behavior, or a
Schema independent of struct tags is required.

## 4. JSON behavior

### Strict fields and size limits

```go
options := shape.JSONOptions{
	DisallowUnknownFields: true,
	MaxBytes:              1 << 20,
}

request, err := schema.ParseJSONReaderContext(ctx, reader, options)
```

- `DisallowUnknownFields` delegates to `encoding/json.Decoder`.
- `MaxBytes == 0` means unlimited.
- Negative MaxBytes returns an error.
- More than one options value is a programming error and panics.
- Trailing non-whitespace JSON values are rejected.
- Cancellation before or during read/traversal returns `ctx.Err()`.

### Missing, null, and zero

Shape decodes directly into `T` before any transform or validation. Therefore:

| Go field | missing key | JSON null | explicit zero |
| --- | --- | --- | --- |
| scalar | zero | zero | zero |
| pointer | nil | nil | pointer or nil |
| slice | nil | nil | empty/non-empty slice |
| map | nil | nil | empty/non-empty map |

Use pointers when nil must remain distinguishable from a value. Shape does not
track key presence separately from `encoding/json`.

## 5. Errors, paths, and languages

Built-in validation returns `*validate.Error`:

```go
var validationError *validate.Error
if errors.As(err, &validationError) {
	for _, issue := range validationError.Issues {
		fmt.Println(issue.Code)
		fmt.Println(issue.Path.String())
		fmt.Println(issue.Message)
		fmt.Println(issue.Expected, issue.Received)
	}
}
```

Use `Code` and `Path` for program logic. Messages may change for language or
clarity. The root path renders as `$`; fields, indexes, and map keys produce
paths such as `users[2].email`.

`Validate` aggregates up to `validate.DefaultMaxIssues`. `ValidateFirst` stops
without evaluating later rules or children and returns at most one issue.

Global language:

```go
validate.SetLanguage(validate.SimplifiedChinese)
```

Per-request language:

```go
ctx := validate.WithLocale(context.Background(), validate.SimplifiedChinese)
err := schema.ValidateContext(ctx, value)
```

Supported constants are `validate.English` and
`validate.SimplifiedChinese`. Language strings are not accepted. Custom
`Refine` errors use code `custom` and preserve `err.Error()` without translation.

Transform failures from a struct Schema use `*shape.TransformError`, containing
the JSON path and original error. `errors.Is`/`errors.As` work through Unwrap.

## 6. Independent `validate` package

Import only validation when the value is already typed and no transformation or
JSON orchestration is needed:

```go
import "github.com/rhevorn/shape/validate"
```

```go
var username = validate.String().
	NotEmpty().
	MinLength(3).
	MaxLength(20).
	Pattern(`^[a-z0-9_]+$`)

err := username.Validate(value)
err = username.ValidateFirst(value)
```

Factories:

```text
Value[T]  String  Bool  Number[N]  Int  Int64  Float64
Time  Duration  Pointer  Slice  Map
```

Composition:

```go
var identifier = validate.String().NotEmpty().MaxLength(50)

var username = validate.String().
	Pattern(`^[a-z0-9_]+$`).
	And(identifier)
```

Custom rule:

```go
var orderValidator = validate.Value[Order]().Refine(func(order Order) error {
	if order.Total < 0 {
		return errors.New("total must not be negative")
	}
	return nil
})
```

Pointer, slice, and map validators run inner validators and produce structured
paths without requiring a root Schema.

## 7. Independent `transform` package

Import only transformation when no validation or JSON orchestration is needed:

```go
import "github.com/rhevorn/shape/transform"
```

```go
var canonicalName = transform.String().
	IfZero("guest").
	Trim().
	ToLower()

name, err := canonicalName.Transform(input)
```

Factories:

```text
Value[T]  String  Bool  Number[N]  Int  Int64  Float64
Time  Duration  Pointer  Slice  Map
```

Custom same-type transforms:

```go
var slug = transform.String().Apply(toSlug, truncateSlug)
var order = transform.Value[Order]().Apply(normalizeOrder)
```

Reuse a complete transformer with `Then`:

```go
var canonicalText = transform.String().Trim().ToLower()
var username = transform.String().IfZero("guest").Then(canonicalText)
```

Pointer, slice, and map transformers copy their outer value and transform inner
values. They do not mutate the caller's value. Map key collisions after
transformation return an error rather than overwriting data.

Transform is strictly same-type (`T → T`). Shape intentionally provides no
implicit string-to-number coercion.

## 8. JSON Schema and OpenAPI

Currently exporters operate on representable tag Schemas:

```go
schema := shape.Struct[Request]()
document, err := jsonschema.Export(schema)
requestBody, err := openapi.JSONRequestBody(schema, true)
response, err := openapi.JSONResponse("ok", schema)
```

JSON Schema output uses Draft 2020-12; OpenAPI output targets 3.1 Schema Object
semantics. Transforms, fallbacks, custom callbacks, and rules that cannot be
represented faithfully return `UnsupportedSchemaError`. Export never silently
drops runtime behavior.

## 9. Static checking with `shapevet`

Go itself verifies method calls and ordinary types, but it does not understand
struct-tag contents or connect a field-name string to a generic struct.

Runtime construction always checks configuration. CI can additionally check
statically visible tag Schemas and direct explicit fields:

```sh
go install github.com/rhevorn/shape/tools/shapevet@latest
go vet -vettool="$(which shapevet)" ./...
```

`shapevet` checks:

- unknown, malformed, or type-incompatible tags;
- unsupported tag field graphs and duplicate JSON names;
- `Struct[T]()` and package-level Bind target types;
- direct `shape.New[T](shape.String("Name"), ...)` field existence, type,
  duplication, visibility, and JSON exclusion.

Definitions hidden behind variables or dynamic expressions remain runtime-only
checks. `shapevet` is optional and is not a runtime dependency.

## 10. Choosing an API

```text
Explicit reusable struct Schema                shape.New[T]
Scalar or composite Schema                     shape.String / Int / Pointer / Slice / Map
Simple fixed DTO tags                          shape.BindJSON / shape.Struct[T]
Validate an existing typed value only          validate
Transform an existing typed value only         transform
Human-readable JSON duration                   types.Duration
Generate JSON Schema/OpenAPI from tags          jsonschema / openapi
```

See [API.md](API.md) for the frozen method inventory and exact compatibility
guarantees, [ARCHITECTURE.md](ARCHITECTURE.md) for implementation boundaries, and
[the examples](../examples) for runnable programs.
