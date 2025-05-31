# shape

[![CI](https://github.com/rhevorn/shape/actions/workflows/ci.yml/badge.svg)](https://github.com/rhevorn/shape/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/rhevorn/shape.svg)](https://pkg.go.dev/github.com/rhevorn/shape)
[![Go Report Card](https://goreportcard.com/badge/github.com/rhevorn/shape)](https://goreportcard.com/report/github.com/rhevorn/shape)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Typed validation, transformation, and JSON binding for Go.

Describe a struct once, then transform, validate, or decode JSON with the same
definition. Invalid field configuration panics at construction; bad input
returns errors.

## Features

- **Struct tags or explicit Schemas** — `shape.BindJSON` for DTOs; `shape.New` when rules belong in Go
- **Transform and validate stay separate** — JSON always runs decode → transform → validate
- **Typed errors** — stable issue codes and JSON paths; English / 简体中文 messages
- **Stdlib only at runtime** — no third-party dependencies for the library itself
- **Optional `shapevet`** — static checks for tags and explicit field names

## Status

Preparing the first public release. The public surface and behavioral contract
are specified in [`docs/API.md`](docs/API.md); after the first public tag, incompatible changes
require a major version.

## Install

```sh
go get github.com/rhevorn/shape
```

Requires **Go 1.24** or newer.

## Struct tags

For simple DTOs, put rules on the struct and bind with no Schema variable:

```go
type CreateUserRequest struct {
	Name string `json:"name" shape:"trim,notempty,maxlength=50"`
	Age  int    `json:"age" shape:"min=18,max=120"`
}

var request CreateUserRequest
err := shape.BindJSON(&request, data)
```

`BindJSON` writes the target only after the full pipeline succeeds.
Use `shape.FromTags[CreateUserRequest]()` to reuse the tagged Schema explicitly.

## Explicit Schema

Field behavior written in Go with `shape.New`:

```go
type User struct {
	Name string   `json:"name"`
	Age  int      `json:"age"`
	Tags []string `json:"tags"`
}

var userSchema = shape.New[User](
	shape.Field("Name", shape.String().Trim().NotEmpty().MaxLength(50)),
	shape.Field("Age", shape.Int().Min(18).Max(120)),
	shape.Field("Tags", shape.Slice(shape.String().Trim().NotEmpty()).NotEmpty().Unique()),
)

data := []byte(`{
	"name": " Pong ",
	"age": 20,
	"tags": [" go ", "shape"]
}`)

user, err := userSchema.ParseJSON(data)
// user.Name == "Pong"
```

The equivalent package-level call is:

```go
user, err := shape.ParseJSON(userSchema, data)
```

Both forms decode, transform, and validate in one call. Schemas created with
`shape.New[T]` or `shape.FromTags[T]` support the method form; the package
function accepts any `Schema[T]`, including scalars and collections.

`Field` names and their target types are checked at construction; generics
check value and collection composition. Optional `shapevet` catches statically
visible binding mistakes before runtime.

The same schema also supports `Transform` and `Validate` on typed values.
Value factories such as `String`, `Slice`, and `Map` never take a field name;
`Field` is the single explicit bridge from a value Schema to a struct field.

## Scalars

Standalone string Specs work the same way without a struct:

```go
name := shape.String().Trim().ToLower().NotEmpty()

out, err := name.Transform(" Pong ")
// out == "pong", err == nil

err = name.Validate(out)
```

## Documentation

- [Usage guide](docs/USAGE.md)
- [Public API](docs/API.md)
- [Struct tags](docs/TAGS.md)
- [Examples](examples)
- [Contributing](CONTRIBUTING.md)

## License

[MIT](LICENSE)
