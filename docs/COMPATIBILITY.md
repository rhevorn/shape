# Public Contract

GoShape has not tagged a release yet. Until `v1.0.0`, the API may still change.
This file describes the **current** intentional contracts so docs, tests, and
implementations stay aligned.

## Toolchains and dependencies

- Minimum Go version: 1.24.
- Root module and bundled `jsonschema` / `openapi` packages use only the Go
  standard library.
- HTTP integration uses root helpers (`Parse`, `ParseReaderLimitContext`, …);
  there is no `shape/http` package.
- Third-party framework integrations, if added later, live in separate modules.

## Behavioral contracts

- Exported identifiers and the `Schema[T]` parse contract
- Strict constructors versus explicit `Coerce*` conversion
- Required, optional, default, strict, and strip object behavior
- Stable validation issue **codes** and the JSON shape of `Issue`, `Path`, and
  `ValidationError`
- Default 100-issue aggregation cap and terminal `too_many_issues`
- Default 64-level `Lazy` recursion cap and `MaxDepth` override
- 128-item deep-comparison fallback cap for `Slice.Unique`
- `ErrJSONTooLarge` for size-limited JSON readers
- `Union` = at-least-one (first match); `OneOf` = exactly-one
- JSON Schema Draft 2020-12 and OpenAPI 3.1 export dialects
- Immutability and safe concurrent reuse of fully constructed schemas
- Message catalogs under `messages/`; `SetLanguage` process default;
  `WithLocale` per-context override
- Display labels via `Fields` optional label, `Field.Label`, or `Label`

Schema constructors and builder methods may panic for invalid configuration
(negative lengths, empty alternatives, nil callbacks, duplicate field names).
Untrusted values passed to a valid schema return errors; they do not panic.

Constructor functions are the supported way to initialize builders. Primitive
zero values currently behave like their strict constructors; composites that
need a child schema, field, item, alternative, or lazy provider do not promise
a useful zero value.

Composite schemas preserve child issue codes and prefix paths without mutating
the reusable child error. Map and record issue order is deterministic.

## Flexible details

These are not machine API:

- Catalog message wording and available locales
- Performance outside documented regression thresholds
- Map iteration and serialized JSON object member order
- New builders, rules, metadata, issue codes, and document annotations
- More precise errors for inputs that already fail

Custom refinements and transforms that cannot be represented faithfully make
JSON Schema / OpenAPI export return `UnsupportedSchemaError` instead of
silently dropping behavior.

`Field.Default` accepts only deeply immutable values. Maps, non-nil slices,
pointers, channels, functions, and structs containing such values must use
`DefaultFunc`. Objects that use `DefaultFunc` cannot be exported to JSON Schema.
