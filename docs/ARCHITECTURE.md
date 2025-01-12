# Architecture and Behavioral Decisions

This document records the current pre-v1 implementation contract. It should
change only with an intentional API or behavior decision.

## Public shape

The central contract is:

```go
type Schema[T any] interface {
	Parse(value any) (T, error)
	ParseContext(ctx context.Context, value any) (T, error)
}
```

Concrete builders are small values. Chained methods return a modified copy;
they never mutate a previously constructed schema. Any internal slices must be
copied before append so two derived schemas cannot share writable backing
storage.

Primitive parsing uses type assertions, not reflection. Limited reflection may
be considered later for convenience input shapes, but it is not part of v0.1's
primitive core.

## Parse pipeline

Schemas follow this order where applicable:

1. Check context and input type
2. Decode an adapter-specific representation, such as `json.Number`
3. Normalize
4. Apply built-in rules in declaration order
5. Apply refinements in declaration order
6. Transform the successfully parsed value

Normal `Parse` delegates to `ParseContext(context.Background(), value)`.
Context cancellation is returned as `ctx.Err()` rather than converted into a
validation issue.

## Error model

Validation failures return `*ValidationError`, containing one or more `Issue`
values. Error codes are exported constants so callers do not depend on message
text. `errors.As` must work for `*ValidationError`.

Paths are stored as typed field/index segments and formatted only for display.
Composite schemas prefix child issues by copying the path; they never mutate an
error returned by a reusable child schema.

Primitive schemas generally emit one issue. Slices, maps, and objects collect
issues from all independently parseable children in deterministic input/schema
order. An object does not run its object-level refinement if any field failed,
because the partially populated value is not a valid refinement input.

An arbitrary error returned by a value or object refinement becomes a `custom`
issue. An `Issue` or `ValidationError` returned by a refinement preserves its
structured data. An arbitrary transform error becomes `transform_failed`.

## Strings

`Min`, `Max`, and `Len` count Unicode code points, not bytes. Normalization runs
before length, pattern, email, and refinement checks. Invalid rule parameters,
such as a negative length, should fail at schema construction time with a clear
panic; this catches programmer errors while keeping parse calls allocation-light.

Email validation should use a documented, pragmatic policy suitable for common
application input. It should not claim full RFC mailbox deliverability.

## Numbers and JSON

Direct primitive constructors are strict:

- `Int` accepts `int`
- `Int64` accepts `int64`
- `Float64` accepts `float64`
- `Bool` accepts `bool`

`ParseJSON` decodes using `json.Decoder.UseNumber`. Numeric schemas accept the
standard library's `json.Number` representation and reject overflow,
non-integral values for integers, and malformed numbers. This is explicit
support for a numeric type used by the JSON adapter, not general string
coercion. User-defined schemas passed to `ParseJSON` also receive ordinary
`json.Number` values rather than a private adapter type.

`ParseJSON` must consume exactly one JSON value and reject trailing non-space
input. JSON syntax/decoding failures are returned as ordinary errors, while
schema failures remain `*ValidationError`.

## Collections

`Slice` and `Map` consume the natural untrusted representations `[]any` and
`map[string]any`. They also accept their exact typed output representations,
`[]T` and `map[string]T`, without reflection.

Collection size rules evaluate after type checking and before child parsing.
When size is valid, child errors are aggregated and prefixed with their index or
key. Map issue order must be deterministic; keys are sorted before parsing.

## Objects and fields

`Field` returns a concrete generic field builder that satisfies an object-field
interface. The setter's signature allows Go to infer both the object type and
field value type:

```go
Field("age", Int(), func(user *User, age int) { user.Age = age })
```

Fields are required by default. Semantics:

| Input state | Plain field | `Optional()` | `Default(v)` |
| --- | --- | --- | --- |
| Missing | `required` issue | no setter | setter receives `v` |
| Present and valid | setter receives parsed value | same | same |
| Present and invalid | schema issue | same | same |
| Present `nil` | parsed normally and usually `invalid_type` | same | same |

`Default(v)` implies optional-on-missing behavior; requiring an additional
`Optional()` call would add ceremony without changing meaning. Defaults are
already typed and are assigned directly; they are not re-parsed.

Objects default to strip mode. Strip means unknown keys are ignored while the
typed output naturally contains only declared fields. `Strict` emits one
`unknown_field` issue per unknown key, ordered lexicographically. Setters run
only for successfully parsed fields.

Duplicate field names are programmer errors and should panic during object
construction. Empty field names and nil setters are also construction errors.

## Refinement and transformation

Changing a schema's output type uses a top-level helper:

```go
Transform(source, func(A) (B, error)) Schema[B]
```

Value-preserving refinement should be available on concrete schema builders
where ergonomic, backed by a shared generic helper if needed. Cross-type method
generic parameters are not used because Go methods cannot introduce them.

Refinement and transform callbacks are treated as immutable schema definition
state. Callers are responsible for making captured state concurrency-safe.

## Dependency and compatibility policy

The module targets Go 1.24 and newer and uses only the Go standard library.
Generics are the preferred mechanism for preserving relationships between
schema output, collection elements, object fields, refinements, and transforms.
Concrete primitive fast paths remain appropriate when they avoid reflection or
materially reduce allocations. Reflection is limited to adapter boundaries
where Go generics cannot construct a named runtime type directly.

The confirmed canonical module path is `github.com/rhevorn/goshape`.

## Recursive schemas

`Lazy(name, provider)` is the explicit recursion boundary. Its provider is
resolved at most once and the resolved schema is safely published to concurrent
parsers. Construction remains explicit: recursion does not depend on reflection,
global registries, or mutable package state.

The name is used only as the JSON Schema `$defs` key. Names must be unique per
exported document, and JSON Pointer escaping is applied when constructing a
`$ref`. Recursive exports share one build context so direct and mutually
recursive graphs terminate without dropping definitions.

## Alternative schemas

`Union` means that at least one alternative must parse successfully and returns
the first successful result. `OneOf` evaluates every alternative and succeeds
only when exactly one parses successfully. Their JSON Schema representations
are `anyOf` and `oneOf`, respectively.

## Public compatibility decisions

- Email validation accepts plain mailbox addresses parsed by `net/mail`; display
  names are rejected, and deliverability is explicitly out of scope.
- Error codes and structured fields are API. Human-readable messages may improve
  between minor releases and should not be parsed by applications.
- Invalid schema construction parameters panic because they are programmer
  errors; untrusted input never causes a construction panic.

The repository is licensed under MIT.

## Extended implementation status

The self-contained v0.2 features are implemented: enum, literal, same-output
unions, nullable values, explicit coercion, time/duration, URL/UUID/IP, and
context-aware refinements. v0.3 metadata and JSON Schema Draft 2020-12 export
are implemented. Standard-library v0.4 adapters cover OpenAPI 3.1 and
`net/http` without adding dependencies.

Framework-specific adapters remain separate future work because adding Gin,
Echo, Fiber, or similar frameworks to the root module would violate the
dependency-light core and requires choosing concrete external API versions.
