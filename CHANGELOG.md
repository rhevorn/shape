# Changelog

All notable changes to GoShape will be documented in this file.

The format follows Keep a Changelog, and the project intends to use Semantic
Versioning after the first public release.

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
