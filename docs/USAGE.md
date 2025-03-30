# Shape usage guide

```text
shape.New[T](...)     explicit Schema (primary)
shape.BindJSON(...)   tag DTO, no Schema variable
shape.Struct[T]()     reusable tagged Schema
shape.ParseJSON(...)  any Schema as JSON root
validate / transform  standalone packages
```

Method inventories live in [API.md](API.md). Tag grammar lives in [TAGS.md](TAGS.md).

## 1. Install

```sh
go get github.com/rhevorn/shape
```

Requires Go 1.24+. Runtime packages have no third-party dependencies.

## 2. Explicit Schema

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

Factory strings are **Go field names**; error paths use **JSON names**.
Construction panics on missing, unexported, duplicate, type-mismatched, or
`json:"-"` fields. Omitted fields pass through unchanged. `shape` tags are
ignored by `New`.

### Transform and Validate

```go
user, err := userSchema.Transform(raw) // does not validate
err = userSchema.Validate(user)       // does not transform
err = userSchema.ValidateFirst(user)  // stop at first issue
```

Context forms: `TransformContext`, `ValidateContext`, `ValidateFirstContext`.

### Parse JSON

```go
user, err := userSchema.ParseJSON(data)
user, err = userSchema.ParseJSONReader(reader)
```

Pipeline: decode one JSON value → Transform → Validate → return `T`.
Also: `ParseJSONContext`, `ParseJSONReaderContext`, optional `JSONOptions`.

### Fields and composition

```go
shape.String("Name").Trim().NotEmpty()
shape.Int("Age").Min(18)
shape.Bool("Enabled")
shape.Pointer("Nickname", shape.String().Trim()).NotNull()
shape.Slice("Tags", shape.String().Trim()).NotEmpty().Unique()
shape.Map("Scores", shape.String(), shape.Int().NonNegative())
shape.Field("Address", addressSchema)
```

Scalars implement `Schema[T]` (Transform/Validate), not `JSONSchema`.
Collection JSON roots use package helpers:

```go
scores, err := shape.ParseJSON(shape.Int().NonNegative().Slice(), data)
```

`Apply` transforms (`T → T`); `Refine` validates (`T → error`). Whole-struct
callbacks chain on `New` / `Struct` after field work in the same phase.

See [API.md](API.md) for every method on each Spec type.

## 3. Tag Schema

```go
type Request struct {
	Name string `json:"name" shape:"ifzero=guest,trim,notempty,maxlength=50"`
	Age  int    `json:"age" shape:"min=18,max=120"`
}

var request Request
err := shape.BindJSON(&request, data)
```

No Schema variable. Bind writes `*target` only on full success.
Reuse / Parse / export: `shape.Struct[Request]()`. Cross-field:

```go
var requestSchema = shape.Struct[Request]().Apply(normalize).Refine(check)
```

Full tag grammar and type matrix: [TAGS.md](TAGS.md).

## 4. JSON options

```go
options := shape.JSONOptions{
	DisallowUnknownFields: true,
	MaxBytes:              1 << 20,
}
```

`MaxBytes == 0` is unlimited. Exactly one top-level JSON value; trailing junk
is rejected. Missing / null / zero follow `encoding/json` (no separate presence
bits). Use pointers when nil must differ from a value.

## 5. Errors and languages

```go
var validationError *validate.Error
if errors.As(err, &validationError) {
	for _, issue := range validationError.Issues {
		fmt.Println(issue.Code, issue.Path.String(), issue.Message)
	}
}
```

Default language is English. Global: `validate.SetLanguage(...)`.
Per request: `validate.WithLocale(ctx, validate.SimplifiedChinese)`.
Transform failures from a struct Schema use `*shape.TransformError`.

## 6. Standalone packages

```go
err := validate.String().NotEmpty().MinLength(3).Validate(value)
name, err := transform.String().Trim().ToLower().Transform(input)
```

Full surfaces: [API.md](API.md) §2–3.

## 7. Export and shapevet

```go
document, err := jsonschema.Export(shape.Struct[Request]())
```

Only representable tagged plans export; unsupported behavior returns an error
rather than silent omission.

```sh
go install github.com/rhevorn/shape/tools/shapevet@latest
go vet -vettool="$(which shapevet)" ./...
```

## 8. Choosing an API

```text
Explicit reusable Schema     shape.New[T] → .ParseJSON
Tag DTO bind                 shape.BindJSON(&req, data)
Reusable tagged Schema       shape.Struct[T]
Non-struct JSON root         shape.ParseJSON(schema, data)
Validate only                validate
Transform only               transform
```

See [API.md](API.md) and [examples](../examples).
