# Architecture and Behavioral Decisions

This document records the current implementation contract. Update it whenever
the public API or parse behavior changes intentionally.

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

Primitive parsing uses type assertions, not reflection. Reflection is allowed
only at explicit adapter boundaries when Go cannot construct a named runtime
type directly.

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

Built-in issue messages come only from JSON catalogs under `messages/`
(for example `en.json`, `zh-CN.json`). Templates may use `{{.Label}}`,
`{{.Expected}}`, and similar placeholders. Custom `NewIssue` text and plain
errors keep the caller-supplied message.

`SetLanguage` sets the process-default catalog (the one intentional process
default). `WithLocale(ctx, lang)` overrides it for a parse. `Localize` /
`LocalizeContext` rewrite an existing `*ValidationError` for display.

`Issue.Label` holds an optional display name. Attach it with
`Field(...).Label("姓名")`, `shape.Label("年龄", schema)`, or the optional
label argument on `Fields[T]().Str` / `Email` / `Int` / `Bool` / `Int64` /
`Float64` / `Time` / `Duration`.

Paths are stored as typed field/index segments and formatted only for display.
Composite schemas prefix child issues by copying the path; they never mutate an
error returned by a reusable child schema.

Primitive schemas generally emit one issue. Slices, maps, and objects collect
issues from independently parseable children in deterministic input/schema
order, up to `DefaultMaxIssues`. When more failures exist, the last retained
issue has code `too_many_issues` and sibling traversal stops. An object does not
run its object-level refinement if any field failed, because the partially
populated value is not a valid refinement input.

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

Package-level `Parse` / `ParseContext` accept JSON as `string` or `[]byte` and
decode with `json.Decoder.UseNumber`. Numeric schemas accept `json.Number` and
reject overflow, non-integral values for integers, and malformed numbers. This
is support for the JSON adapter's number type, not general string coercion.
Already-decoded Go values use `schema.Parse` / `schema.ParseContext` instead;
those entry points do not auto-decode JSON text (a string schema must keep
seeing strings).

`Parse` must consume exactly one JSON value and reject trailing non-space
input. JSON syntax/decoding failures are returned as ordinary errors, while
schema failures remain `*ValidationError`.

Reader helpers decode directly from the stream rather than buffering the full
input. `ParseReaderLimit` and its context variant use an explicit maximum
byte count and return `ErrJSONTooLarge` when exceeded. HTTP handlers typically
pass `1 << 20` (1 MiB) or another app-specific limit.

Composite results retain at most 100 issues, and each named `Lazy` schema has a
64-level default recursion limit. These local immutable limits require no
request-specific mutable state. JSON decoder nesting protection, byte-limited
readers, collection `Max`, and context cancellation provide additional
independent controls.

## Collections

`Slice` and `Map` consume the natural untrusted representations `[]any` and
`map[string]any`. `Map(key, value)` parses each JSON string key through the key
schema and each value through the value schema, producing `map[K]V`. They also
accept typed `[]T` and `map[string]V` inputs without reflection where the shape
is unambiguous.

Collection size rules evaluate after type checking and before child parsing.
When size is valid, child errors are aggregated and prefixed with their index or
key. Map issue order must be deterministic; keys are sorted before parsing.
`Slice.Unique` uses hash-map equality for types where it is equivalent to
`reflect.DeepEqual`. The cancellable deep-comparison fallback is limited to 128
items so non-comparable inputs cannot restore unbounded quadratic work.

## Objects and fields

Preferred object construction for common scalars uses `Fields[T]()`:

```go
f := Fields[User]()
Object(
	f.Str("name", "姓名").Trim().Min(2).Set(func(user *User, value string) {
		user.Name = value
	}),
	f.Int("age").Min(18).Set(func(user *User, age int) { user.Age = age }),
)
```

`Field` remains for arbitrary nested schemas. The setter's signature allows Go
to infer both the object type and field value type:

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
`Optional()` call would add ceremony without changing meaning. It accepts only
deeply immutable values. `DefaultFunc(func() V)` supports reference-bearing
defaults by creating a fresh value per parse. Dynamic defaults are not
exportable to JSON Schema because invoking a factory during documentation
generation would not faithfully describe runtime behavior.

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

## Dependency policy

The module targets Go 1.24 and newer and uses only the Go standard library.
Generics are the preferred mechanism for preserving relationships between
schema output, collection elements, object fields, refinements, and transforms.
Concrete primitive fast paths remain appropriate when they avoid reflection or
materially reduce allocations. Reflection is limited to adapter boundaries
where Go generics cannot construct a named runtime type directly.

The confirmed canonical module path is `github.com/rhevorn/shape`.

## Recursive schemas

`Lazy(name, provider)` is the explicit recursion boundary. Its provider is
resolved at most once and the resolved schema is safely published to concurrent
parsers. Construction remains explicit: recursion does not depend on reflection,
global registries, or mutable package state.

Parsing tracks active calls to each Lazy identity in parse-local context state.
The state is synchronized for safe context propagation and is never stored on
the reusable schema. The default limit is 64 and `MaxDepth` returns a configured
schema copy. This terminates cyclic in-memory graphs and bounds error-path
amplification without changing recursive JSON Schema export.

The name is used only as the JSON Schema `$defs` key. Names must be unique per
exported document, and JSON Pointer escaping is applied when constructing a
`$ref`. Recursive exports share one build context so direct and mutually
recursive graphs terminate without dropping definitions.

## Alternative schemas

`Union` means that at least one alternative must parse successfully and returns
the first successful result. `OneOf` evaluates every alternative and succeeds
only when exactly one parses successfully. Their JSON Schema representations
are `anyOf` and `oneOf`, respectively.

## Intentional decisions

- Email validation accepts plain mailbox addresses parsed by `net/mail`; display
  names are rejected, and deliverability is out of scope.
- Error codes, paths, and structured `Issue` fields are the machine API.
  Catalog message text is for humans and may change; prefer codes and paths.
- Invalid schema construction parameters panic because they are programmer
  errors; untrusted input never causes a construction panic.
- Concrete builders should be created with their constructor functions. A useful
  zero value is not promised for composites that require child definitions.
- `CoerceFloat` is removed; use `CoerceFloat64`.
- There is no dedicated `net/http` adapter; handlers call root JSON helpers.
- JSON Schema / OpenAPI export live in the `jsonschema` and `openapi`
  packages. The root exposes `ExportDocument` only as an adapter hook; prefer
  those packages' public APIs. Schema metadata uses `Annotate`, not methods on
  concrete builders.

The repository is licensed under MIT.

## Current surface

Implemented in the root module: primitives, generic numbers, collections,
typed objects (`Field` and `Fields`), transforms, refinements, explicit
coercion, time/duration, URL/UUID/IP, enum/literal/union/oneOf/nullable,
lazy recursion, JSON parse helpers, metadata/`Annotate`, locale catalogs, and
labels. JSON Schema and OpenAPI export live in the `jsonschema` and `openapi`
packages (root `ExportDocument` is the shared adapter hook).

Framework-specific integrations (Gin, Echo, Fiber, and similar) stay out of
the root module so ordinary users do not inherit third-party dependency graphs.
