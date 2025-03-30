# shape

[![CI](https://github.com/rhevorn/shape/actions/workflows/ci.yml/badge.svg)](https://github.com/rhevorn/shape/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/rhevorn/shape.svg)](https://pkg.go.dev/github.com/rhevorn/shape)
[![Go Report Card](https://goreportcard.com/badge/github.com/rhevorn/shape)](https://goreportcard.com/report/github.com/rhevorn/shape)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Type-safe validation, transformation, and JSON binding for Go.

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

Pre-1.0: the public surface is documented in [`docs/API.md`](docs/API.md), but
breaking changes may still happen before a stable release.

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

## Explicit Schema

Field behavior written in Go with `shape.New`:

```go
type User struct {
	Name string   `json:"name"`
	Age  int      `json:"age"`
	Tags []string `json:"tags"`
}

var userSchema = shape.New[User](
	shape.String("Name").Trim().NotEmpty().MaxLength(50),
	shape.Int("Age").Min(18).Max(120),
	shape.Slice("Tags", shape.String().Trim().NotEmpty()).NotEmpty().Unique(),
)

user, err := userSchema.ParseJSON([]byte(`{
	"name": " Pong ",
	"age": 20,
	"tags": [" go ", "shape"]
}`))
// user.Name == "Pong"
```

The same schema also supports `Transform` and `Validate` on typed values.

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
