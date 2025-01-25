# Changelog

All notable changes to GoShape will be documented in this file.

The format follows Keep a Changelog. The first public release is planned as
v1.0.0 and will use Semantic Versioning from that point onward.

## Unreleased

### Added

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
- OpenAPI 3.1, net/http, and dedicated JSON Schema adapters
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
