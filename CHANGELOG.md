# Changelog

## [Unreleased]

### Added

- A pure `validate.Validator[T]` API with aggregate and fail-fast calls,
  structured paths, stable issue codes, localization, composition, and
  context-aware refinements.
- A pure `transform.Transformer[T]` API with same-type fluent transforms,
  immutable composition, conditional `IfZero`/`IfNull`, and context-aware
  custom `Apply` callbacks.
- Code-defined struct schemas through typed `shape.New[T](...)` field contracts,
  including scalar, pointer, slice, map, and nested Schema fields.
- Cached tag schemas through `shape.Struct[T]()` and inferred package-level
  `shape.BindJSON*` helpers.
- Schema JSON methods using a fixed decode → transform → validate workflow,
  atomic `BindJSON` methods, unknown-field rejection, byte limits, and context
  cancellation.
- Nested struct, pointer, slice, map, `time.Time`, and human-readable
  `types.Duration` tag support.
- JSON Schema Draft 2020-12 and OpenAPI 3.1 adapters, plus the optional
  `shapevet` analyzer.
- A Shape-first example suite plus complete usage, tag, architecture, and frozen
  public API contracts.

### Removed

- The pre-release `Option`, `Rule`, scalar Schema, `Object`, and `Fields`,
  typed `Parse`, and typed `Bind` APIs.
- Ambiguous coercion, `Optional`, `Default`, `SkipNull`, `NotBlank`, union, and
  loose `any` parsing APIs.

[Unreleased]: https://github.com/rhevorn/shape/commits/HEAD
