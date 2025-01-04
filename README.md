# GoShape

**Type-safe, composable schema parsing and validation for Go.**

GoShape turns untrusted runtime data into validated, typed Go values without
struct tags or validation DSLs.

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

The v0.1 API is implemented and preparing for its first release. The behavior
contract is recorded in [the architecture notes](docs/ARCHITECTURE.md).

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
`Default(value)` to assign a typed default. Objects strip unknown input keys by
default; `Strict()` reports them as `unknown_field` issues.

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
parsing human-readable messages.

## JSON

```go
user, err := goshape.ParseJSON(userSchema, requestBody)
```

JSON is an adapter, not GoShape's core representation. `ParseJSON` accepts
exactly one JSON value and retains numeric precision with `encoding/json.Number`.

## v0.1 schemas and rules

- `String`: `Trim`, `Min`, `Max`, `Len`, `Pattern`, `Email`, `Refine`
- `Int`, `Int64`, `Float64`: `Min`, `Max`, `Gt`, `Gte`, `Lt`, `Lte`, `Refine`
- `Bool`: `Refine`
- `Slice`: `Min`, `Max`, `Refine`
- `Map`: `Refine`
- `Object`: `Strict`, `Strip`, `Refine`
- Composition: `Field`, `Optional`, `Default`, `Transform`, generic `Refine`

## Design goals

- Typed, composable schemas
- Explicit normalization and transformation
- Structured, path-aware errors
- Immutable schemas that are safe for concurrent reuse
- No reflection on primitive parsing paths
- A dependency-free core
- No struct-tag DSL

## Development

```sh
make check
make test-race
make fuzz-smoke
```

See [the development plan](docs/DEVELOPMENT_PLAN.md) and
[contribution guide](CONTRIBUTING.md) for scope and release checks.

## License

MIT
