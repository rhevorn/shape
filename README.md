# GoShape

**Type-safe, composable schema parsing and validation for Go.**

GoShape turns untrusted runtime data into validated, typed Go values without
struct tags or validation DSLs.

Requires Go 1.24 or newer. The module and bundled adapters use only the Go
standard library.

```go
schema := shape.String().
	Trim().
	Min(3).
	Max(50)

value, err := schema.Parse("  hello  ")
if err != nil {
	// Handle structured validation issues.
}

fmt.Println(value)
// hello
```

GoShape is moving toward a stable v1.0 API. No public release is tagged yet;
progress is tracked in [the v1 roadmap](docs/V1_ROADMAP.md). Until then, treat
the repository tip as the source of truth.

## Object schemas

```go
type User struct {
	Name  string
	Email string
	Age   int
}

f := shape.Fields[User]()
userSchema := shape.Object(
	f.Str("name", "姓名").Trim().Min(2).Max(50).Set(func(user *User, value string) {
		user.Name = value
	}),
	f.Email("email", "邮箱").Trim().Set(func(user *User, value string) {
		user.Email = value
	}),
	f.Int("age").Min(18).Max(120).Set(func(user *User, value int) {
		user.Age = value
	}),
).Strict()

user, err := userSchema.Parse(map[string]any{
	"name":  " Pong ",
	"email": "pong@example.com",
	"age":   30,
})
```

The second argument to `Str` / `Email` / `Int` / `Bool` is an optional display
label for validation messages. `Field(...)` remains available for custom
schemas. Fields are required by default. Use `Optional()` to permit a missing
field or `Default(value)` to assign a deeply immutable typed default.
Reference-bearing defaults use `DefaultFunc(func() T)` so every parse gets
fresh state. Objects strip unknown input keys by default; `Strict()` reports
them as `unknown_field` issues.

For real request bodies, prefer JSON helpers instead of building a `map` by
hand:

```go
user, err := shape.Parse(userSchema, requestBody)
// or
user, err := shape.Parse(userSchema, `{"name":"Pong","email":"pong@example.com","age":30}`)
```

## Collections and composition

```go
names := shape.Slice(shape.String().Trim().Min(1)).Min(1).Max(10)
labels := shape.Map(shape.String())

port := shape.Transform(
	shape.String().Trim(),
	func(value string) (int, error) { return strconv.Atoi(value) },
)
```

Use `Refine` methods for rules that keep the same output type. Object-level
refinements can validate relationships between fields.

GoShape also provides `Enum`, `Literal`, same-output-type `Union`/`OneOf`,
`Nullable`, fixed-length heterogeneous `Tuple`, typed-key `Record`, and
recursive `Lazy` schemas. `Nullable(schema)` returns `Schema[*T]`, preserving
the difference between JSON `null` and a non-null value. Typed `[]T` and map
inputs are supported where their input shape is unambiguous.

`Union` accepts the first successful alternative. `OneOf` requires exactly one
alternative to succeed, matching the distinction between JSON Schema `anyOf`
and `oneOf`.

`Slice.Unique` uses a linear equality path for scalar/comparable values. Types
that require deep comparison are capped at 128 items; add a smaller `Max` for
attacker-facing composite collections when appropriate.

Recursive schemas name their JSON Schema definition explicitly:

```go
type Node struct {
	Value    string
	Children []Node
}

var nodeSchema shape.Schema[Node]
nodeSchema = shape.Lazy("Node", func() shape.Schema[Node] {
	return shape.Object[Node](
		shape.Field("value", shape.String().NonEmpty(),
			func(node *Node, value string) { node.Value = value }),
		shape.Field("children", shape.Slice(nodeSchema),
			func(node *Node, children []Node) { node.Children = children }).Default(nil),
	).Strict()
})
```

`Lazy` limits recursive parsing to 64 active levels by default; use
`MaxDepth(n)` when a trusted data model needs a different positive bound.

## Structured errors

```go
value, err := schema.Parse(input)
if err != nil {
	var validation *shape.ValidationError
	if errors.As(err, &validation) {
		for _, issue := range validation.Issues {
			fmt.Println(issue.Code, issue.Path.String(), issue.Message)
		}
	}
}
```

Paths retain typed field and index segments internally and format values such
as `users[3].address.zip` for display. Error codes are stable and do not require
parsing human-readable messages. Composite validation retains at most 100
issues by default and ends a truncated result with `too_many_issues`.

Human-readable messages come only from JSON catalogs under `messages/`
(for example `messages/en.json` and `messages/zh-CN.json`). Single-language
apps can call `shape.SetLanguage("zh-CN")` once at startup. Per-request
language uses `shape.WithLocale(ctx, "zh-CN")` with `ParseContext`.

Field and value display names are set directly with `Field(...).Label("姓名")`
or `shape.Label("年龄", shape.Int().Min(18))` and appear in issue messages as
`{{.Label}}`.

## JSON

```go
user, err := shape.Parse(userSchema, requestBody)
```

JSON is an adapter, not GoShape's core representation. `Parse` accepts
exactly one JSON value and retains numeric precision with `encoding/json.Number`.
Context and reader variants are available as `ParseContext`,
`ParseReader`, and `ParseReaderContext`. For an untrusted stream, use
`ParseReaderLimit` or `ParseReaderLimitContext` to enforce a byte limit.

## Explicit coercion

Strict constructors do not convert strings or unrelated Go scalar types. Use
the explicit constructors when conversion is intended:

```go
port := shape.CoerceInt().Min(1).Max(65535)
enabled := shape.CoerceBool()
timeout := shape.CoerceDuration()
createdAt := shape.CoerceTime()
```

Additional schemas include `Time`, `Duration`, `URL`, `UUID`, and `IP`.

## JSON Schema and OpenAPI

```go
document, err := shape.JSONSchema(userSchema)
requestBody, err := openapi.JSONRequestBody(userSchema, true)
```

The first call produces JSON Schema Draft 2020-12, including `$defs`/`$ref`
for named recursive schemas. The second produces an OpenAPI 3.1 request body.
Metadata can be attached to any schema:

```go
homepage := shape.Annotate(shape.URL()).
	Title("Homepage").
	Description("Absolute homepage URL").
	Example("https://example.com")
```

Custom refinements and transforms cannot be represented faithfully, so their
export returns `UnsupportedSchemaError` instead of silently losing behavior.

Dependency-free adapters are available at:

- `github.com/rhevorn/shape/jsonschema`
- `github.com/rhevorn/shape/openapi`

HTTP handlers should call `shape.Parse` / `shape.ParseReaderLimitContext`
directly; there is no separate `net/http` adapter package.

## Schemas and rules

- `String`: normalization, length, pattern, containment, and format rules
- `Int`, `Int64`, `Float64`: bounds, sign, allowed-value, and custom rules
- `Number[T]`: generic numbers, including named signed, unsigned, and floating
  types
- `Bool`, `Time`, `Duration`, `URL`, `UUID`, `IP`
- `Enum`, `Literal`, `Union`/`OneOf`, `Nullable`, `Lazy`
- `Slice`: `Min`, `Max`, `NonEmpty`, `Unique`, `Refine`
- `Map`: `Min`, `Max`, `NonEmpty`, `Refine`
- `Record`: typed key and value schemas, `Min`, `Max`, `NonEmpty`, `Refine`
- `Tuple`: fixed-length heterogeneous positions with typed setters
- `Object`: `Strict`, `Strip`, `Refine`
- Composition: `Fields` / `Field`, `Optional`, `Default`, `DefaultFunc`,
  `Label`, `Transform`, `Refine`, `RefineContext`, `Annotate`

## Design goals

- Typed, composable schemas
- Explicit normalization and transformation
- Structured, path-aware errors
- Immutable schemas that are safe for concurrent reuse
- No reflection on primitive parsing paths
- A dependency-free core
- Go 1.24+ with generics preferred for type relationships
- No struct-tag DSL

## Development

```sh
make check
make test-race
make fuzz-smoke
```

See [the v1 roadmap](docs/V1_ROADMAP.md), [public
contract](docs/COMPATIBILITY.md), [API sketch](docs/API_SKETCH.md),
[architecture](docs/ARCHITECTURE.md), [performance
baseline](docs/BENCHMARKS.md), [the original development
plan](docs/DEVELOPMENT_PLAN.md), [security policy](SECURITY.md), and
[contribution guide](CONTRIBUTING.md).

## License

MIT
