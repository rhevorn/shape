# Compatibility Policy

This policy takes effect with GoShape v1.0.0. Pre-v1 commits may still change
while the release gates are open.

## Supported toolchains and dependencies

- GoShape v1.x supports Go 1.24 and newer.
- The root module and bundled `jsonschema`, `openapi`, and `http` packages use
  only the Go standard library.
- Framework integrations that need third-party modules will not add those
  dependencies to the root module.

## Stable public contracts

Within v1.x, semantic versioning covers:

- Exported identifiers and function/method signatures
- The `Schema[T]` parsing contract
- Strict-versus-coercing constructor behavior
- Required, optional, default, strict, and strip object behavior
- Stable validation issue codes and the JSON shape of `Issue`, `Path`, and
  `ValidationError`
- The default 100-issue aggregation cap and terminal `too_many_issues` issue
- The default 64-level `Lazy` recursion cap and `MaxDepth` override
- The 128-item deep-comparison fallback cap for `Slice.Unique`
- `ErrJSONTooLarge` for explicitly size-limited JSON reader parsing
- `Union` as at-least-one matching and `OneOf` as exactly-one matching
- JSON Schema Draft 2020-12 and OpenAPI 3.1 target dialects
- Immutability and safe concurrent reuse of fully constructed schemas

Schema constructors and builder methods may panic for invalid schema
configuration, such as negative lengths, empty alternatives, nil callbacks, or
duplicate field names. Untrusted values passed to a valid schema return errors;
they do not cause configuration panics.

Constructor functions are the supported way to initialize concrete schema
builders. Primitive schema zero values currently behave like their strict
constructors, but v1 does not promise useful zero values for composite builders
that require a child schema, field, item, alternative, or lazy provider.

Composite schemas preserve child issue codes and prefix paths without mutating
the reusable child error. Map and record issue order is deterministic.

## Intentionally flexible details

The following may change in compatible v1 releases:

- Human-readable error messages; applications should use issue codes and paths
- Performance characteristics outside published regression thresholds
- Map iteration and serialized JSON object member order
- New schema builders, rules, metadata, issue codes, and optional document
  annotations
- More precise errors for inputs that already fail

Custom refinements and transforms are runtime behavior and are rejected by the
JSON Schema exporter when they cannot be represented faithfully. Exporters do
not silently discard those operations.

GoShape bounds each composite validation result to 100 retained issues and each
named `Lazy` schema to 64 active recursive calls by default. The final retained
issue is `too_many_issues` when sibling failures were omitted. `MaxDepth`
provides an explicit positive override for trusted recursive models. JSON
reader and HTTP byte limits, collection `Max`, and context cancellation remain
independent resource controls.

`Field.Default` accepts only deeply immutable values. Maps, non-nil slices,
pointers, channels, functions, and structs containing such values must use
`DefaultFunc`, which creates a fresh value for every missing-field parse.
Because a dynamic default cannot be represented faithfully, exporting an
object containing `DefaultFunc` returns `UnsupportedSchemaError`.

## Deprecation and removal

Compatible replacements may be added and old APIs deprecated during v1.x.
Removing or incompatibly changing a public contract requires v2, except for a
security or correctness emergency that cannot be addressed compatibly.
