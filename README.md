# GoShape

**Type-safe, composable schema parsing and validation for Go.**

GoShape turns untrusted runtime data into validated, typed Go values without
struct tags or validation DSLs.

Requires Go 1.24 or newer. The module and bundled adapters use only the Go
standard library.

```go
schema := goshape.String().
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

GoShape is moving directly toward a stable v1.0 API. No public release is
planned before the v1 compatibility, hardening, and documentation gates are
complete. Progress and remaining gates are tracked in
[the v1 roadmap](docs/V1_ROADMAP.md).

## Object schemas

```go
type User struct {
	Name  string
	Email string
	Age   int
}

userSchema := goshape.Object[User](
	goshape.Field(
		"name",
		goshape.String().Trim().Min(2).Max(50),
		func(user *User, value string) { user.Name = value },
	),
	goshape.Field(
		"email",
		goshape.String().Trim().Email(),
		func(user *User, value string) { user.Email = value },
	),
	goshape.Field(
		"age",
		goshape.Int().Min(18).Max(120),
		func(user *User, value int) { user.Age = value },
	),
).Strict()

user, err := userSchema.Parse(map[string]any{
	"name":  " Pong ",
	"email": "pong@example.com",
	"age":   30,
})
```

Fields are required by default. Use `Optional()` to permit a missing field or
`Default(value)` to assign a deeply immutable typed default. Reference-bearing
defaults use `DefaultFunc(func() T)` so every parse gets fresh state. Objects
strip unknown input keys by default; `Strict()` reports them as
`unknown_field` issues.

## Collections and composition

```go
names := goshape.Slice(goshape.String().Trim().Min(1)).Min(1).Max(10)
labels := goshape.Map(goshape.String())

port := goshape.Transform(
	goshape.String().Trim(),
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

var nodeSchema goshape.Schema[Node]
nodeSchema = goshape.Lazy("Node", func() goshape.Schema[Node] {
	return goshape.Object[Node](
		goshape.Field("value", goshape.String().NonEmpty(),
			func(node *Node, value string) { node.Value = value }),
		goshape.Field("children", goshape.Slice(nodeSchema),
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
	var validation *goshape.ValidationError
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

## JSON

```go
user, err := goshape.ParseJSON(userSchema, requestBody)
```

JSON is an adapter, not GoShape's core representation. `ParseJSON` accepts
exactly one JSON value and retains numeric precision with `encoding/json.Number`.
Context and reader variants are available as `ParseJSONContext`,
`ParseJSONReader`, and `ParseJSONReaderContext`. For an untrusted stream, use
`ParseJSONReaderLimit` or `ParseJSONReaderLimitContext` to enforce a byte limit.

## Explicit coercion

Strict constructors do not convert strings or unrelated Go scalar types. Use
the explicit constructors when conversion is intended:

```go
port := goshape.CoerceInt().Min(1).Max(65535)
enabled := goshape.CoerceBool()
timeout := goshape.CoerceDuration()
createdAt := goshape.CoerceTime()
```

Additional schemas include `Time`, `Duration`, `URL`, `UUID`, and `IP`.

## JSON Schema and OpenAPI

```go
document, err := goshape.JSONSchema(userSchema)
requestBody, err := openapi.JSONRequestBody(userSchema, true)
```

The first call produces JSON Schema Draft 2020-12, including `$defs`/`$ref`
for named recursive schemas. The second produces an OpenAPI 3.1 request body.
Metadata can be attached to any schema:

```go
homepage := goshape.Annotate(goshape.URL()).
	Title("Homepage").
	Description("Absolute homepage URL").
	Example("https://example.com")
```

Custom refinements and transforms cannot be represented faithfully, so their
export returns `UnsupportedSchemaError` instead of silently losing behavior.

Dependency-free adapters are available at:

- `github.com/rhevorn/goshape/jsonschema`
- `github.com/rhevorn/goshape/openapi`
- `github.com/rhevorn/goshape/http` (package name `goshapehttp`)

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
- Composition: `Field`, `Optional`, `Default`, `Transform`, `Refine`,
  `DefaultFunc`, `RefineContext`

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

See [the v1 roadmap](docs/V1_ROADMAP.md), [compatibility
policy](docs/COMPATIBILITY.md), [performance baseline](docs/BENCHMARKS.md),
[pre-v1 migration notes](docs/PRE_V1_MIGRATION.md), [the original development
plan](docs/DEVELOPMENT_PLAN.md), [security policy](SECURITY.md), and
[contribution guide](CONTRIBUTING.md) for scope and release checks.

## License

MIT
