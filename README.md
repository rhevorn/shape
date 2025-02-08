# Shape

**Type-safe, composable schema parsing and validation for Go.**

Turn untrusted input—especially JSON—into typed values without struct tags or
a validation DSL. Requires Go 1.24+. The core and bundled adapters use only the
standard library.

```sh
go get github.com/rhevorn/shape
```

## Quick start

```go
type User struct {
	Name  string
	Email string
	Age   int
}

f := shape.Fields[User]()
userSchema := shape.Object(
	f.Str("name", "姓名").Trim().Min(2).Max(50).Set(func(u *User, v string) { u.Name = v }),
	f.Email("email", "邮箱").Trim().Set(func(u *User, v string) { u.Email = v }),
	f.Int("age").Min(18).Max(120).Set(func(u *User, v int) { u.Age = v }),
).Strict()

user, err := shape.Parse(userSchema, `{"name":" Pong ","email":"pong@example.com","age":30}`)
```

`shape.Parse` accepts `string` or `[]byte` JSON. For already-decoded Go values,
call `schema.Parse` instead. HTTP bodies: `ParseReaderLimitContext(ctx, schema, r.Body, 1<<20)`.

Runnable samples: `[examples/](examples/)`.

## Objects and fields

`Fields[T]()` covers common scalars (`Str`, `Email`, `Int`, `Bool`, `Int64`,
`Float64`, `Time`, `Duration`). Optional second argument is a display label for
messages. Use `Field(...)` for nested or custom schemas.

Fields are required by default. `Optional()`, `Default(v)` (immutable values
only), and `DefaultFunc` cover missing input. Unknown keys are stripped unless
you call `Strict()`.

```go
users := shape.Slice(userSchema).Min(1) // []User
```

## Collections and composition

Build larger schemas by wrapping smaller ones. The element/value schema comes
first; size or shape rules chain afterward.

**List — `Slice`** (same element type, output `[]T`):

```go
tags := shape.Slice(shape.String().Trim().Min(1)).Min(1).Max(10)
// JSON: ["go","shape"]  →  []string{"go","shape"}

users := shape.Slice(userSchema).Min(1)
// JSON: [{...},{...}]  →  []User
```

**Map — `Map(key, value)`** → `map[K]V` (key schema first, value schema second).
JSON objects always arrive with string keys; the key schema parses each one:

```go
labels := shape.Map(shape.String(), shape.String().Min(1))
// map[string]string

loose := shape.Map(shape.String(), shape.Any())
// map[string]any

ids := shape.Map(shape.String().ToLower(), shape.Int())
// {"User":1} → map[string]int{"user":1}

ports := shape.Map(
	shape.Transform(shape.String().Trim(), strconv.Atoi),
	shape.Bool(),
)
// {"8080":true} → map[int]bool{8080:true}
```

**Change type — `Transform`** (A → B after a successful parse):

```go
port := shape.Transform(shape.String().Trim(), strconv.Atoi)
// JSON: "8080"  →  int(8080)
```

**Same type, extra rule — `Refine`**:

```go
even := shape.Int().Refine(func(n int) error {
	if n%2 != 0 {
		return shape.NewIssue("not_even", "must be even")
	}
	return nil
})
```

**One of several shapes:**

```go
// First match wins (like JSON Schema anyOf)
contact := shape.Union(shape.String().Email(), shape.UUID())

// Exactly one alternative must match (like oneOf)
id := shape.OneOf(shape.String().Email(), shape.UUID())
```

**JSON `null` — `Nullable`** (output `*T`; missing null vs value stay distinct):

```go
nickname := shape.Nullable(shape.String().Trim())
// null → (*string)(nil)    "Ada" → &"Ada"
```

Less common: `Tuple` (fixed-length positions into a struct) and `Lazy`
(recursion). Details and limits are in [architecture](docs/ARCHITECTURE.md).

## Errors and locale

```go
if err != nil {
	var validation *shape.ValidationError
	if errors.As(err, &validation) {
		for _, issue := range validation.Issues {
			fmt.Println(issue.Code, issue.Path, issue.Message)
		}
	}
}
```

Prefer `issue.Code` and `issue.Path` over parsing message text. Catalogs live
under `messages/` (`en`, `zh-CN`, …). Set a process default with
`SetLanguage("zh-CN")`, or per call with `WithLocale(ctx, "zh-CN")`.

## Coercion

Strict constructors do not convert types. Use explicit helpers when you want
that:

```go
shape.CoerceInt().Min(1).Max(65535)
shape.CoerceBool()
shape.CoerceTime()
shape.CoerceDuration()
```

## JSON Schema and OpenAPI

```go
import (
	"github.com/rhevorn/shape/jsonschema"
	"github.com/rhevorn/shape/openapi"
)

document, err := jsonschema.Export(userSchema)
body, err := openapi.JSONRequestBody(userSchema, true)
```

Attach export-only metadata with `shape.Annotate(schema).Title(...).Description(...)`.
Unsupported operations (custom refine/transform) return `UnsupportedSchemaError`.

## Feature map


| Area               | Builders                                                                                                 |
| ------------------ | -------------------------------------------------------------------------------------------------------- |
| Scalars            | `String`, `Bool`, `Int`, `Int64`, `Float64`, `Number[T]`, `Any`, `Time`, `Duration`, `URL`, `UUID`, `IP` |
| Objects            | `Object`, `Fields`, `Field`, `Optional`, `Default`, `DefaultFunc`, `Strict` / `Strip`                    |
| Collections        | `Slice`, `Map(key, value)`, `Tuple`                                                                     |
| Choice / recursion | `Enum`, `Literal`, `Union`, `OneOf`, `Nullable`, `Lazy`                                                  |
| Pipeline           | `Transform`, `Refine`, `RefineContext`, `Label`, `Annotate`                                              |
| Input              | `Parse`, `ParseContext`, `ParseReader`, `ParseReaderLimit` (+ Context)                                   |


## Design

- Compile-time types via generics; no struct-tag DSL
- Explicit normalize / coerce / transform
- Path-aware structured errors
- Immutable schemas, safe to reuse concurrently
- No reflection on strict primitive parse paths
- Stdlib-only core

## Development

```sh
make check
make test-race
make fuzz-smoke
```

See [architecture](docs/ARCHITECTURE.md), [security](SECURITY.md), and
[contributing](CONTRIBUTING.md).

## License

MIT