# shape

[![CI](https://github.com/rhevorn/shape/actions/workflows/ci.yml/badge.svg)](https://github.com/rhevorn/shape/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/rhevorn/shape.svg)](https://pkg.go.dev/github.com/rhevorn/shape)
[![Go Report Card](https://goreportcard.com/badge/github.com/rhevorn/shape)](https://goreportcard.com/report/github.com/rhevorn/shape)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Typed validation, transformation, and JSON, form, and query binding for Go.

Describe a struct once, then transform, validate, or parse input with the same
definition. Invalid field configuration panics; bad input returns errors.

## Features

- **Struct tags or explicit Schemas** — `shape.BindJSON` for DTOs; `shape.New` when rules belong in Go
- **Transform and validate stay separate** — parsing always runs decode → transform → validate
- **Typed errors** — stable issue codes and field paths; English / 简体中文 messages
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
	Name string `json:"name" form:"name" query:"name" shape:"trim,notempty,maxlength=50"`
	Age  int    `json:"age" form:"age" query:"age" shape:"min=18,max=120"`
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
	Name string   `json:"name" form:"name" query:"name"`
	Age  int      `json:"age" form:"age" query:"age"`
	Tags []string `json:"tags" form:"tags" query:"tags"`
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

## Forms and query parameters

The same struct Schema accepts `url.Values`:

```go
values := url.Values{
    "name": {" Pong "},
    "age":  {"20"},
    "tags": {"go", "shape"},
}

// Schema methods.
user, err := userSchema.ParseForm(values)
user, err = userSchema.ParseQuery(values)

// Equivalent package functions.
user, err = shape.ParseForm(userSchema, values)
user, err = shape.ParseQuery(userSchema, values)
```

For tagged DTOs, use `shape.BindForm(&request, values)` or
`shape.BindQuery(&request, values)`. All calls have context variants and Bind
updates its target only on success.

Each input uses its own tag (`json`, `form`, or `query`), falling back to the
Go field name. Input tags never change the Schema’s transformation or validation rules. Repeated parameters bind to slices; repeated scalars return an error.
Nested structs use names such as `address.city`.

For HTTP forms, parse the request and pass `r.PostForm` or
`r.MultipartForm.Value`. For queries, use `url.ParseQuery(r.URL.RawQuery)` and
check its error before passing the values. See [Forms and query parameters](docs/PARAMETERS.md)
for decoding rules, strict options, and HTTP examples.

## HTTP requests

For automatic Query and body binding:

```go
var request CreateUserRequest
err := shape.BindRequest(&request, r, shape.RequestOptions{
    DisallowUnknownFields: true,
})
```

Query uses `query` tags. Content-Type selects JSON (`json` tags) or Form (`form`
tags) for the body. A field submitted by both sources returns a conflict error;
set `Precedence: shape.QueryFirst` or `shape.BodyFirst` for explicit precedence.
Objects and slices are replaced as whole top-level fields, never recursively
merged. The combined value is transformed and validated once before replacing
the target. The default body limit is 1 MiB; multipart file parts require manual
handling. See [HTTP binding](docs/PARAMETERS.md#automatic-http-binding).

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
- [Forms and query parameters](docs/PARAMETERS.md)
- [Examples](examples)
- [Contributing](CONTRIBUTING.md)

## License

[MIT](LICENSE)
