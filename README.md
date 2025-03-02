# shape

Type-safe validation, transformation, and JSON binding for Go.

`shape` lets you describe a struct once, then reuse that definition to:

- transform an existing Go value;
- validate an existing Go value;
- decode, transform, and validate JSON;
- collect every validation issue or stop at the first one.

The public API uses Go generics. Invalid field names and incompatible field
types are rejected when a schema is created, while normal method and argument
mistakes are caught by the Go compiler.

## Install

```sh
go get github.com/rhevorn/shape
```

Requires Go 1.24 or newer. Runtime packages use only the standard library.

## Quick start

This is a complete program:

```go
package main

import (
	"fmt"

	"github.com/rhevorn/shape"
)

type User struct {
	Name string   `json:"name"`
	Age  int      `json:"age"`
	Tags []string `json:"tags"`
}

var userSchema = shape.New[User](
	shape.String("Name").Trim().NotEmpty().MaxLength(50),
	shape.Int("Age").Min(18).Max(120),
	shape.Slice("Tags", shape.String().Trim().NotEmpty()).
		NotEmpty().
		Unique(),
)

func main() {
	user, err := userSchema.ParseJSON([]byte(`{
		"name": " Pong ",
		"age": 20,
		"tags": [" go ", "shape"]
	}`))
	if err != nil {
		panic(err)
	}

	fmt.Printf("%#v\n", user)
	// main.User{Name:"Pong", Age:20, Tags:[]string{"go", "shape"}}
}
```

Transformation steps keep the order in which methods are called, and validation
rules do the same. A schema JSON operation completes all transformation
steps before it starts validation, so `NotEmpty` sees the trimmed string.

## One schema, three operations

Use the same schema with typed Go values or JSON:

```go
raw := User{Name: " Pong ", Age: 20, Tags: []string{" go "}}

user, err := userSchema.Transform(raw)
err = userSchema.Validate(user)

user, err = userSchema.ParseJSON([]byte(`{
	"name": " Pong ",
	"age": 20,
	"tags": [" go "]
}`))
```

The responsibilities are deliberately separate:

- `Transform` returns a transformed copy and does not validate it.
- `Validate` checks a typed value and collects all issues.
- `ValidateFirst` checks a typed value and stops at the first issue.
- `ParseJSON` decodes one JSON value, transforms it, validates it, and returns it.

All operations also have context-aware forms.

## Struct tags

Tags are a shorter alternative for straightforward request structs:

```go
type CreateUserRequest struct {
	Name string `json:"name" shape:"trim,notempty,maxlength=50"`
	Age  int    `json:"age" shape:"min=18,max=120"`
}

func decodeRequest(data []byte) (CreateUserRequest, error) {
	var request CreateUserRequest
	err := shape.BindJSON(&request, data)
	return request, err
}
```

`BindJSON` changes the destination only after decoding, transformation, and
validation all succeed. Use `shape.Struct[CreateUserRequest]()` when the tagged
schema needs to be stored or reused.

Tagged schemas can add cross-field behavior with
`shape.Struct[T]().Apply(...).Refine(...)`. Use explicit `shape.New[T](...)`
schemas when field behavior should be written in Go rather than tags.

## Validation without a schema

The `validate` package works independently:

```go
username := validate.String().
	NotEmpty().
	MinLength(3).
	Pattern(`^[a-z0-9_]+$`)

err := username.Validate("pong")
```

It provides validators for strings, numbers, booleans, times, durations,
pointers, slices, maps, and custom values.

## Transformation without a schema

The `transform` package also works independently:

```go
canonicalName := transform.String().
	Trim().
	ToLower()

name, err := canonicalName.Transform(" Pong ")
// name is "pong"
```

Transformers never perform validation. Custom transformations use `Apply` or
`ApplyContext`.

## JSON options

JSON decoding can reject unknown object fields and limit input size:

```go
options := shape.JSONOptions{
	DisallowUnknownFields: true,
	MaxBytes:              1 << 20,
}

user, err := userSchema.ParseJSON(data, options)
```

Reader, context, Parse, and Bind variants are listed in the
[API contract](docs/API.md).

## Errors and languages

Validation errors contain stable issue codes and paths:

```go
var validationError *validate.Error
if errors.As(err, &validationError) {
	for _, issue := range validationError.Issues {
		fmt.Println(issue.Code, issue.Path.String(), issue.Message)
	}
}
```

Built-in messages support `validate.English` and
`validate.SimplifiedChinese`. A locale can be attached to a context with
`validate.WithLocale`.

## Static checking

Go checks fluent method names, generic types, callback signatures, and argument
types. The optional `shapevet` analyzer additionally checks statically visible
struct tags and explicit schema field names:

```sh
go install github.com/rhevorn/shape/tools/shapevet@latest
go vet -vettool="$(which shapevet)" ./...
```

`shapevet` is optional. Schema construction still rejects invalid definitions
at runtime when the analyzer is not installed.

## Documentation

- [Complete usage guide](docs/USAGE.md)
- [Public API and behavior contract](docs/API.md)
- [Struct tag reference](docs/TAGS.md)
- [Architecture and compatibility contract](docs/ARCHITECTURE.md)
- [Runnable examples](examples)

JSON Schema Draft 2020-12 and OpenAPI 3.1 adapters are available in the
`jsonschema` and `openapi` packages. See the [export example](examples/shape/export).

## GitHub topics

`go`, `golang`, `validation`, `validator`, `schema-validation`,
`data-validation`, `struct-tags`, `json-validation`, `data-normalization`,
`type-safe`

## License

MIT
