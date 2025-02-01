# Changelog

All notable changes to GoShape will be documented in this file.

The format follows Keep a Changelog. The first public release is planned as
v1.0.0 and will use Semantic Versioning from that point onward.

## Unreleased

### Changed

- Renamed the root package and module path from `goshape` /
  `github.com/rhevorn/goshape` to `shape` / `github.com/rhevorn/shape`
- Renamed JSON Schema vendor extensions from `x-goshape-*` to `x-shape-*`

### Removed

- Dropped the `shape/http` (`shapehttp`) adapter; use `shape.Parse` /
  `ParseReaderLimitContext` and handle `*ValidationError` in the handler
- Dropped `docs/PRE_V1_MIGRATION.md`; the tip API is documented in the README
  and `docs/` without a migration trail

### Added

- Locale-aware validation messages via `SetLanguage`, `WithLocale`, and JSON
  message catalogs under `messages/` (built-in issue text comes only from
  catalogs; code no longer embeds natural-language messages)
- `Label` / `Field.Label` for human-readable names interpolated into issue
  messages as `{{.Label}}`
- Fluent object fields via `Fields[T]()`: `Str` / `Email` / `Int` / `Bool` with
  optional display label and terminal `Set`
- Package-level `Parse` / `ParseContext` / `ParseReader*` for JSON input
  (`string` or `[]byte`); `schema.Parse` remains for decoded Go values
- Generic `Schema[T]` contract with context-aware parsing
- Structured validation issues and typed field/index paths
- String, int, int64, float64, and bool schemas
- Slice, map, and typed object schemas
- Required, optional, defaulted, strict, and strip object behavior
- Value and object refinements
- Cross-type transforms
- Precision-preserving JSON parsing
- Unit, race, example, fuzz, and benchmark coverage
- Enum, literal, union, and nullable schemas
- Explicit scalar, time, and duration coercion
- Time, duration, URL, UUID, and IP schemas
- Additional string, number, and collection rules
- Schema metadata and JSON Schema Draft 2020-12 export
- OpenAPI 3.1 and dedicated JSON Schema adapters
- Reader- and context-aware JSON helpers
- Go 1.24 minimum version and generic `Number[T]`/`CoerceNumber[T]` schemas for
  named signed, unsigned, and floating-point types
- Fixed-length heterogeneous `Tuple` schemas with typed position setters
- Typed-key `Record[K, V]` schemas with deterministic key parsing
- Named recursive `Lazy` schemas with concurrency-safe one-time resolution
- JSON Schema `$defs`/`$ref` export for recursive schemas
- Distinct `Union` (at least one) and `OneOf` (exactly one) runtime and JSON
  Schema semantics
- Streaming JSON reader decoding with explicit byte-limited variants and
  `ErrJSONTooLarge`
- Bounded decimal integer parsing without arbitrary-precision allocation
- Linear uniqueness checks for scalar/comparable values with cancellable deep
  comparison fallback
- A 64-level default recursive parse bound with `Lazy.MaxDepth`
- A 100-issue composite aggregation bound and bounded HTTP error writer
- `DefaultFunc` for fresh mutable field defaults and detached metadata values
- Private vulnerability reporting guidance and a documented security scope
